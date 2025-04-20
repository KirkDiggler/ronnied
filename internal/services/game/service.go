package game

import (
	"context"
	"errors"

	"fmt"
	"log"

	"github.com/KirkDiggler/ronnied/internal/common/clock"
	"github.com/KirkDiggler/ronnied/internal/common/uuid"
	"github.com/KirkDiggler/ronnied/internal/dice"
	"github.com/KirkDiggler/ronnied/internal/models"
	ledgerRepo "github.com/KirkDiggler/ronnied/internal/repositories/drink_ledger"
	gameRepo "github.com/KirkDiggler/ronnied/internal/repositories/game"
	playerRepo "github.com/KirkDiggler/ronnied/internal/repositories/player"
)

// service implements the Service interface
type service struct {
	// Configuration parameters
	maxPlayers         int
	diceSides          int
	criticalHitValue   int
	criticalFailValue  int
	maxConcurrentGames int

	// Repository dependencies
	gameRepo        gameRepo.Repository
	playerRepo      playerRepo.Repository
	drinkLedgerRepo ledgerRepo.Repository

	// Service dependencies
	diceRoller dice.Roller
	clock      clock.Clock
	uuid       uuid.UUID
}

// New creates a new game service
func New(cfg *Config) (*service, error) {
	// Validate config
	if cfg == nil {
		return nil, ErrNilConfig
	}

	if cfg.GameRepo == nil {
		return nil, ErrNilGameRepo
	}

	if cfg.PlayerRepo == nil {
		return nil, ErrNilPlayerRepo
	}

	if cfg.DrinkLedgerRepo == nil {
		return nil, ErrNilDrinkLedgerRepo
	}

	if cfg.DiceRoller == nil {
		return nil, ErrNilDiceRoller
	}

	if cfg.Clock == nil {
		return nil, ErrNilClock
	}

	if cfg.UUIDGenerator == nil {
		return nil, ErrNilUUIDGenerator
	}

	// Set default values for configuration parameters if not provided
	maxPlayers := cfg.MaxConcurrentGames
	if maxPlayers <= 0 {
		maxPlayers = 10
	}

	diceSides := cfg.DiceSides
	if diceSides <= 0 {
		diceSides = 6
	}

	criticalHitValue := cfg.CriticalHitValue
	if criticalHitValue <= 0 {
		criticalHitValue = 6
	}

	criticalFailValue := cfg.CriticalFailValue
	if criticalFailValue <= 0 {
		criticalFailValue = 1
	}

	maxConcurrentGames := cfg.MaxConcurrentGames
	if maxConcurrentGames <= 0 {
		maxConcurrentGames = 100
	}

	return &service{
		// Configuration parameters
		maxPlayers:         maxPlayers,
		diceSides:          diceSides,
		criticalHitValue:   criticalHitValue,
		criticalFailValue:  criticalFailValue,
		maxConcurrentGames: maxConcurrentGames,

		// Repository dependencies
		gameRepo:        cfg.GameRepo,
		playerRepo:      cfg.PlayerRepo,
		drinkLedgerRepo: cfg.DrinkLedgerRepo,

		// Service dependencies
		diceRoller: cfg.DiceRoller,
		clock:      cfg.Clock,
		uuid:       cfg.UUIDGenerator,
	}, nil
}

func (s *service) CreateGame(ctx context.Context, input *CreateGameInput) (*CreateGameOutput, error) {
	// Create a new game using the repository
	createGameOutput, err := s.gameRepo.CreateGame(ctx, &gameRepo.CreateGameInput{
		ChannelID: input.ChannelID,
		CreatorID: input.CreatorID,
		Status:    models.GameStatusWaiting,
	})
	if err != nil {
		return nil, err
	}

	// Create the creator as a participant
	_, err = s.gameRepo.CreateParticipant(ctx, &gameRepo.CreateParticipantInput{
		GameID:     createGameOutput.Game.ID,
		PlayerID:   input.CreatorID,
		PlayerName: input.CreatorName,
		Status:     models.ParticipantStatusWaitingToRoll,
	})
	if err != nil {
		return nil, err
	}

	return &CreateGameOutput{
		GameID: createGameOutput.Game.ID,
	}, nil
}

// StartGame transitions a game from waiting to active state
func (s *service) StartGame(ctx context.Context, input *StartGameInput) (*StartGameOutput, error) {
	// Get the game
	game, err := s.gameRepo.GetGame(ctx, &gameRepo.GetGameInput{
		GameID: input.GameID,
	})
	if err != nil {
		return nil, ErrGameNotFound
	}

	// Verify the player is the game creator
	if game.CreatorID != input.PlayerID {
		return nil, ErrNotCreator
	}

	// Ensure the game is in waiting status
	if game.Status != models.GameStatusWaiting {
		return nil, ErrInvalidGameState
	}

	// Ensure there is at least 1 player (the creator)
	if len(game.Participants) < 1 {
		return nil, ErrNotEnoughPlayers
	}

	// Update game status to active
	game.Status = models.GameStatusActive
	game.UpdatedAt = s.clock.Now()

	// Save the updated game
	err = s.gameRepo.SaveGame(ctx, &gameRepo.SaveGameInput{
		Game: game,
	})
	if err != nil {
		return nil, err
	}

	// Check if the game is ready to complete (all players have rolled and assigned drinks)
	if game.IsReadyToComplete() {
		log.Printf("Game %s is ready to complete immediately after starting", game.ID)

		// Try to end the game
		endGameOutput, err := s.EndGame(ctx, &EndGameInput{
			Game: game,
		})

		if err != nil {
			// Log the error but don't fail the start game operation
			log.Printf("Error ending game after start: %v", err)
		} else if endGameOutput.NeedsRollOff {
			// A roll-off is needed, log this information
			log.Printf("Game %s needs a roll-off after immediate completion", game.ID)
		}
	}

	return &StartGameOutput{
		Success: true,
	}, nil
}

// JoinGame adds a player to an existing game
func (s *service) JoinGame(ctx context.Context, input *JoinGameInput) (*JoinGameOutput, error) {
	// Get the game
	game, err := s.gameRepo.GetGame(ctx, &gameRepo.GetGameInput{
		GameID: input.GameID,
	})
	if err != nil {
		return nil, ErrGameNotFound
	}

	// Check if player is already in the game
	playerAlreadyInGame := false
	for _, participant := range game.Participants {
		if participant.PlayerID == input.PlayerID {
			playerAlreadyInGame = true
			break
		}
	}

	// If player is not already in the game, check if they can join based on game state
	if !playerAlreadyInGame {
		// Return specific error based on game state
		switch game.Status {
		case models.GameStatusActive:
			return nil, ErrGameActive
		case models.GameStatusRollOff:
			return nil, ErrGameRollOff
		case models.GameStatusCompleted:
			return nil, ErrGameCompleted
		case models.GameStatusWaiting:
			// Check if the game is full
			if len(game.Participants) >= s.maxPlayers {
				return nil, ErrGameFull
			}
			// Game is waiting and not full, so player can join
		default:
			// Unknown game status
			return nil, ErrInvalidGameState
		}
	}

	// If player is already in the game, just return success
	if playerAlreadyInGame {
		return &JoinGameOutput{
			Success:       true,
			AlreadyJoined: true,
		}, nil
	}

	// Check if player already exists
	existingPlayer, err := s.playerRepo.GetPlayer(ctx, &playerRepo.GetPlayerInput{
		PlayerID: input.PlayerID,
	})

	// If player exists, check if they're already in a game
	if err == nil {
		if existingPlayer.CurrentGameID != "" {
			// They're in another game, update their game ID
			err = s.playerRepo.UpdatePlayerGame(ctx, &playerRepo.UpdatePlayerGameInput{
				PlayerID: input.PlayerID,
				GameID:   input.GameID,
			})
			if err != nil {
				return nil, err
			}
		} else {
			// Update the player's current game
			existingPlayer.CurrentGameID = input.GameID
			err = s.playerRepo.SavePlayer(ctx, &playerRepo.SavePlayerInput{
				Player: existingPlayer,
			})
			if err != nil {
				return nil, err
			}
		}
	} else {
		// Create a new player
		now := s.clock.Now()
		player := &models.Player{
			ID:            input.PlayerID,
			Name:          input.PlayerName,
			CurrentGameID: input.GameID,
			LastRoll:      0,
			LastRollTime:  now,
		}

		err = s.playerRepo.SavePlayer(ctx, &playerRepo.SavePlayerInput{
			Player: player,
		})
		if err != nil {
			return nil, err
		}
	}

	// Use the repository to create a participant with a generated UUID
	_, err = s.gameRepo.CreateParticipant(ctx, &gameRepo.CreateParticipantInput{
		GameID:     input.GameID,
		PlayerID:   input.PlayerID,
		PlayerName: input.PlayerName,
		Status:     models.ParticipantStatusWaitingToRoll,
	})
	if err != nil {
		return nil, err
	}

	return &JoinGameOutput{
		Success: true,
	}, nil
}

// RollDice performs a dice roll for a player
func (s *service) RollDice(ctx context.Context, input *RollDiceInput) (*RollDiceOutput, error) {
	// Validate input
	if input == nil {
		return nil, errors.New("input cannot be nil")
	}

	if input.GameID == "" {
		return nil, errors.New("game ID cannot be empty")
	}

	if input.PlayerID == "" {
		return nil, errors.New("player ID cannot be empty")
	}

	// Get the game
	game, err := s.gameRepo.GetGame(ctx, &gameRepo.GetGameInput{
		GameID: input.GameID,
	})
	if err != nil {
		// Return the actual error instead of swallowing it
		return nil, fmt.Errorf("failed to get game: %w", err)
	}

	// If this is a roll-off game, check if there's a nested roll-off the player should be in
	if game.Status == models.GameStatusRollOff && game.ParentGameID != "" {
		rollOffGame, err := s.FindActiveRollOffGame(ctx, input.PlayerID, input.GameID)
		if err != nil && !errors.Is(err, ErrRollOffGameNotFound) {
			return nil, fmt.Errorf("failed to check for nested roll-off games: %w", err)
		}

		// If a nested roll-off game was found, use that instead
		if rollOffGame != nil {
			input.GameID = rollOffGame.ID
			game = rollOffGame
		}
	}

	// Check if game is in a valid state for rolling
	if !isValidGameStateForRolling(game.Status) {
		return nil, fmt.Errorf("%w: game status is %s", ErrInvalidGameState, game.Status)
	}

	// Find the participant in the game
	participant := game.GetParticipant(input.PlayerID)
	if participant == nil {
		return nil, ErrPlayerNotInGame
	}

	// Check if the participant has already rolled
	if participant.RollTime != nil {
		return nil, fmt.Errorf("player %s has already rolled in this game", participant.PlayerName)
	}

	// Roll the dice
	rollValue := s.diceRoller.Roll(s.diceSides)
	now := s.clock.Now()

	// Update the participant's roll
	participant.RollValue = rollValue
	participant.RollTime = &now

	// Check if the roll is a critical hit or fail
	isCriticalHit := rollValue == s.criticalHitValue
	isCriticalFail := rollValue == s.criticalFailValue

	// Update participant status based on roll
	if isCriticalHit {
		participant.Status = models.ParticipantStatusNeedsToAssign
	} else {
		participant.Status = models.ParticipantStatusActive

		// If it's a critical fail, automatically assign a drink to self
		if isCriticalFail {
			// Create a new drink record using the repository
			_, err = s.drinkLedgerRepo.CreateDrinkRecord(ctx, &ledgerRepo.CreateDrinkRecordInput{
				GameID:       input.GameID,
				FromPlayerID: input.PlayerID,
				ToPlayerID:   input.PlayerID,
				Reason:       models.DrinkReasonCriticalFail,
				Timestamp:    now,
				SessionID:    s.getSessionIDForChannel(ctx, game.ChannelID),
			})

			if err != nil {
				log.Printf("Error saving critical fail drink record: %v", err)
				// Don't return the error, continue with the roll
			}
		}
	}

	// Update the game
	game.UpdatedAt = now
	err = s.gameRepo.SaveGame(ctx, &gameRepo.SaveGameInput{
		Game: game,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to save game: %w", err)
	}

	// Check if all players have rolled
	allPlayersRolled := true
	for _, p := range game.Participants {
		if p.RollTime == nil {
			allPlayersRolled = false
			break
		}
	}

	// If all players have rolled and no players need to assign drinks, try to end the game
	var endGameOutput *EndGameOutput
	needsRollOff := false
	rollOffType := ""
	rollOffGameID := ""

	if allPlayersRolled {
		// Check if any players need to assign drinks
		allDrinksAssigned := true
		for _, p := range game.Participants {
			if p.Status == models.ParticipantStatusNeedsToAssign {
				allDrinksAssigned = false
				break
			}
		}

		// Only try to end the game if all drinks are assigned
		if allDrinksAssigned {
			endGameOutput, err = s.EndGame(ctx, &EndGameInput{
				Game: game,
			})

			if err == nil {
				if endGameOutput.NeedsRollOff {
					needsRollOff = true
					rollOffType = string(endGameOutput.RollOffType)
					rollOffGameID = endGameOutput.RollOffGameID
				}
			} else {
				// Log the error but don't return it to the caller
				log.Printf("Error ending game after all players rolled: %v", err)
			}
		}
	}

	// Prepare domain result information
	result := ""
	details := ""
	activeRollOffGameID := ""
	var eligiblePlayers []PlayerOption

	// Get the player name
	playerName := ""
	for _, p := range game.Participants {
		if p.PlayerID == input.PlayerID {
			playerName = p.PlayerName
			break
		}
	}

	// Set result and details based on roll result
	if isCriticalHit {
		result = fmt.Sprintf("You Rolled a %d! Critical Hit!", rollValue)
		details = "Select a player to assign a drink:"

		// Get eligible players for drink assignment
		for _, p := range game.Participants {
			isCurrentPlayer := p.PlayerID == input.PlayerID

			// For critical hits, include all players except the current player initially
			if !isCurrentPlayer {
				eligiblePlayers = append(eligiblePlayers, PlayerOption{
					PlayerID:        p.PlayerID,
					PlayerName:      p.PlayerName,
					IsCurrentPlayer: false,
				})
			}
		}

		// If there are no other players, include the current player
		if len(eligiblePlayers) == 0 {
			// Find the current player
			for _, p := range game.Participants {
				if p.PlayerID == input.PlayerID {
					eligiblePlayers = append(eligiblePlayers, PlayerOption{
						PlayerID:        p.PlayerID,
						PlayerName:      p.PlayerName + " (You)",
						IsCurrentPlayer: true,
					})
					break
				}
			}
			details += "\n\nYou're the only player, so you'll have to drink yourself!"
		}
	} else if isCriticalFail {
		result = "You Rolled a 1! Critical Fail!"
		details = "Drink up! 🍺"
	} else {
		result = fmt.Sprintf("You Rolled a %d", rollValue)
		details = "Your roll has been recorded."
	}

	// Check if the player should be redirected to a roll-off game
	if game.Status == models.GameStatusRollOff && game.ParentGameID != "" {
		activeRollOffGameID = game.ID
	}

	return &RollDiceOutput{
		// Basic roll information
		Value:            rollValue,
		RollValue:        rollValue, // Alias for Value to maintain compatibility
		PlayerID:         input.PlayerID,
		PlayerName:       playerName,
		IsCriticalHit:    isCriticalHit,
		IsCriticalFail:   isCriticalFail,
		AllPlayersRolled: allPlayersRolled,
		NeedsRollOff:     needsRollOff,
		RollOffType:      RollOffType(rollOffType),
		RollOffGameID:    rollOffGameID,

		// Domain result information
		Result:              result,
		Details:             details,
		ActiveRollOffGameID: activeRollOffGameID,
		EligiblePlayers:     eligiblePlayers,
		Game:                game,
	}, nil
}

// isValidGameStateForRolling checks if a game state allows dice rolling
func isValidGameStateForRolling(status models.GameStatus) bool {
	return status == models.GameStatusActive ||
		status == models.GameStatusRollOff ||
		status == models.GameStatusWaiting
}

// AssignDrink records that one player has assigned a drink to another
func (s *service) AssignDrink(ctx context.Context, input *AssignDrinkInput) (*AssignDrinkOutput, error) {
	// Validate input
	if input == nil {
		return nil, errors.New("input cannot be nil")
	}

	if input.GameID == "" {
		return nil, errors.New("game ID cannot be empty")
	}

	if input.FromPlayerID == "" {
		return nil, errors.New("from player ID cannot be empty")
	}

	if input.ToPlayerID == "" {
		return nil, errors.New("to player ID cannot be empty")
	}

	// Get the game
	game, err := s.gameRepo.GetGame(ctx, &gameRepo.GetGameInput{
		GameID: input.GameID,
	})
	if err != nil {
		return nil, ErrGameNotFound
	}

	// Check if game is active or waiting
	if game.Status != models.GameStatusActive && game.Status != models.GameStatusRollOff && game.Status != models.GameStatusWaiting {
		return nil, ErrInvalidGameState
	}

	// Find the assigning participant in the game
	assigningParticipant := game.GetParticipant(input.FromPlayerID)
	if assigningParticipant == nil {
		return nil, ErrPlayerNotInGame
	}

	// Check if the assigning participant is allowed to assign a drink
	if assigningParticipant.Status != models.ParticipantStatusNeedsToAssign {
		return nil, errors.New("player is not eligible to assign a drink")
	}

	// Find the target participant in the game
	targetParticipant := game.GetParticipant(input.ToPlayerID)
	if targetParticipant == nil {
		return nil, errors.New("target player is not in the game")
	}

	// Create a drink record using the repository
	_, err = s.drinkLedgerRepo.CreateDrinkRecord(ctx, &ledgerRepo.CreateDrinkRecordInput{
		GameID:       input.GameID,
		FromPlayerID: input.FromPlayerID,
		ToPlayerID:   input.ToPlayerID,
		Reason:       models.DrinkReason(input.Reason),
		Timestamp:    s.clock.Now(),
		SessionID:    s.getSessionIDForChannel(ctx, game.ChannelID),
	})
	if err != nil {
		return nil, err
	}

	// Update the assigning participant's status
	assigningParticipant.Status = models.ParticipantStatusActive

	// Update the game
	game.UpdatedAt = s.clock.Now()
	err = s.gameRepo.SaveGame(ctx, &gameRepo.SaveGameInput{
		Game: game,
	})
	if err != nil {
		return nil, err
	}

	// Check if all players have completed their actions and the game can be ended
	allPlayersRolled := true
	allDrinksAssigned := true
	for _, participant := range game.Participants {
		if participant.RollTime == nil {
			allPlayersRolled = false
			break
		}

		if participant.Status == models.ParticipantStatusNeedsToAssign {
			allDrinksAssigned = false
			break
		}
	}

	// If all players have rolled and all drinks are assigned, attempt to end the game
	var endGameOutput *EndGameOutput
	if allPlayersRolled && allDrinksAssigned {
		endGameOutput, err = s.EndGame(ctx, &EndGameInput{
			Game: game,
		})
		if err == nil {
		} else {
			// Log the error but don't return it to the caller
			log.Printf("Error ending game after drink assignment: %v", err)
		}
	}

	return &AssignDrinkOutput{
		Success:       true,
		GameEnded:     allPlayersRolled && allDrinksAssigned,
		EndGameOutput: endGameOutput,
	}, nil
}

// EndGame concludes a game session
func (s *service) EndGame(ctx context.Context, input *EndGameInput) (*EndGameOutput, error) {
	// Get the game
	game := input.Game

	// Check if this is a roll-off game
	var parentGame *models.Game
	var isRollOffGame bool
	if game.ParentGameID != "" {
		isRollOffGame = true
		// Get the parent game
		var err error
		parentGame, err = s.gameRepo.GetGame(ctx, &gameRepo.GetGameInput{
			GameID: game.ParentGameID,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to get parent game: %w", err)
		}
	}

	// Check if game is active
	if game.Status != models.GameStatusActive && game.Status != models.GameStatusRollOff {
		return nil, ErrInvalidGameState
	}

	// For roll-off games, we always mark them as completed when EndGame is called
	if isRollOffGame {
		game.Status = models.GameStatusCompleted
	}

	// Check if all participants have completed their actions
	for _, participant := range game.Participants {
		// Check if everyone has rolled
		if participant.RollTime == nil {
			return nil, errors.New("not all players have rolled yet")
		}

		// Check if anyone still needs to assign a drink
		if participant.Status == models.ParticipantStatusNeedsToAssign {
			return nil, errors.New("some players still need to assign drinks")
		}
	}

	// Get drink records for this game
	drinkRecords, err := s.drinkLedgerRepo.GetDrinkRecordsForGame(ctx, &ledgerRepo.GetDrinkRecordsForGameInput{
		GameID: game.ID,
	})
	if err != nil {
		return nil, err
	}

	// Build a map of player ID to player stats
	playerStatsMap := make(map[string]*PlayerStats)

	// Initialize stats for all participants
	for _, participant := range game.Participants {
		// Initialize player stats
		playerStatsMap[participant.PlayerID] = &PlayerStats{
			PlayerID:       participant.PlayerID,
			PlayerName:     participant.PlayerName,
			DrinksAssigned: 0,
			DrinksReceived: 0,
			LastRoll:       participant.RollValue,
			LastRollTime:   *participant.RollTime,
		}
	}

	// Tally up drinks assigned and received
	for _, record := range drinkRecords.Records {
		// Increment drinks assigned counter for the assigner
		if stats, ok := playerStatsMap[record.FromPlayerID]; ok {
			stats.DrinksAssigned++
		}

		// Increment drinks received counter for the assignee
		if stats, ok := playerStatsMap[record.ToPlayerID]; ok {
			stats.DrinksReceived++
		}
	}

	// Find players with the lowest roll
	lowestRoll := s.diceSides + 1 // Start with a value higher than possible
	lowestRollPlayerIDs := []string{}

	// Find players with the highest roll
	highestRoll := 0
	highestRollPlayerIDs := []string{}

	// First pass: find the highest and lowest roll values
	for _, participant := range game.Participants {
		// Track highest rolls
		if participant.RollValue > highestRoll {
			highestRoll = participant.RollValue
		}

		// Track lowest rolls
		if participant.RollValue < lowestRoll {
			lowestRoll = participant.RollValue
		}
	}

	// Second pass: find the players with the lowest and highest roll values
	for _, participant := range game.Participants {
		// Track lowest rolls
		if participant.RollValue == lowestRoll {
			lowestRollPlayerIDs = append(lowestRollPlayerIDs, participant.PlayerID)
		}

		// Track highest rolls
		if participant.RollValue == highestRoll {
			highestRollPlayerIDs = append(highestRollPlayerIDs, participant.PlayerID)
		}
	}

	// Variables to track roll-off information
	var needsHighestRollOff bool
	var highestRollOffGameID string
	var highestRollOffPlayerIDs []string
	var needsLowestRollOff bool
	var lowestRollOffGameID string
	var lowestRollOffPlayerIDs []string

	// Check for ties with the highest roll (critical hits)
	if len(highestRollPlayerIDs) > 1 {
		// Multiple players tied for highest roll, create a roll-off game

		// Create a map of player IDs to names for the roll-off game
		playerNames := make(map[string]string)
		for _, participant := range game.Participants {
			for _, playerID := range highestRollPlayerIDs {
				if participant.PlayerID == playerID {
					playerNames[playerID] = participant.PlayerName
					break
				}
			}
		}

		// Create the roll-off game with the repository
		rollOffGameOutput, err := s.gameRepo.CreateRollOffGame(ctx, &gameRepo.CreateRollOffGameInput{
			ChannelID:    game.ChannelID,
			CreatorID:    game.CreatorID,
			ParentGameID: game.ID,
			PlayerIDs:    highestRollPlayerIDs,
			PlayerNames:  playerNames,
		})

		if err != nil {
			return nil, fmt.Errorf("failed to create roll-off game for highest rollers: %w", err)
		}

		// Update the parent game with the roll-off game ID
		game.HighestRollOffGameID = rollOffGameOutput.Game.ID
		game.RollOffGameID = rollOffGameOutput.Game.ID // For backward compatibility
		game.Status = models.GameStatusRollOff
		game.UpdatedAt = s.clock.Now()

		// Update the players' current game ID
		for _, playerID := range highestRollPlayerIDs {
			player, err := s.playerRepo.GetPlayer(ctx, &playerRepo.GetPlayerInput{
				PlayerID: playerID,
			})
			if err != nil {
				return nil, err
			}

			player.CurrentGameID = rollOffGameOutput.Game.ID

			err = s.playerRepo.SavePlayer(ctx, &playerRepo.SavePlayerInput{
				Player: player,
			})
			if err != nil {
				return nil, err
			}
		}

		// Store the highest roll-off information
		needsHighestRollOff = true
		highestRollOffGameID = rollOffGameOutput.Game.ID
		highestRollOffPlayerIDs = highestRollPlayerIDs
	}

	// Check for lowest roll ties or single lowest roller
	if len(lowestRollPlayerIDs) == 1 && !needsHighestRollOff {
		// If there's only one player with the lowest roll and we don't need a highest roll-off,
		// we can complete the game and assign a drink
		lowestPlayerID := lowestRollPlayerIDs[0]

		// Determine which game ID to use for the drink record
		targetGameID := game.ID
		if isRollOffGame {
			// If this is a roll-off game, assign the drink to the parent game
			targetGameID = game.ParentGameID
		}

		// Create a drink record for the player with the lowest roll using the repository
		_, err = s.drinkLedgerRepo.CreateDrinkRecord(ctx, &ledgerRepo.CreateDrinkRecordInput{
			GameID:     targetGameID,
			ToPlayerID: lowestPlayerID,
			Reason:     models.DrinkReasonLowestRoll,
			Timestamp:  s.clock.Now(),
			SessionID:  s.getSessionIDForChannel(ctx, game.ChannelID),
		})

		if err != nil {
			log.Printf("Error saving lowest roll drink record: %v", err)
			// Don't return the error, continue with ending the game
		}
	} else if len(lowestRollPlayerIDs) > 1 {
		// Multiple players tied for lowest roll, create a roll-off game
		// Only create a lowest roll-off if we don't already have a highest roll-off
		// This matches the current test expectations

		// Create a map of player IDs to names for the roll-off game
		playerNames := make(map[string]string)
		for _, participant := range game.Participants {
			for _, playerID := range lowestRollPlayerIDs {
				if participant.PlayerID == playerID {
					playerNames[playerID] = participant.PlayerName
					break
				}
			}
		}

		// Create the roll-off game with the repository
		rollOffGameOutput, err := s.gameRepo.CreateRollOffGame(ctx, &gameRepo.CreateRollOffGameInput{
			ChannelID:    game.ChannelID,
			CreatorID:    game.CreatorID,
			ParentGameID: game.ID,
			PlayerIDs:    lowestRollPlayerIDs,
			PlayerNames:  playerNames,
		})

		if err != nil {
			return nil, fmt.Errorf("failed to create roll-off game for lowest rollers: %w", err)
		}

		// Update the parent game with the roll-off game ID
		game.LowestRollOffGameID = rollOffGameOutput.Game.ID
		game.RollOffGameID = rollOffGameOutput.Game.ID // For backward compatibility
		game.Status = models.GameStatusRollOff
		game.UpdatedAt = s.clock.Now()
		// Update the players' current game ID
		for _, playerID := range lowestRollPlayerIDs {
			player, err := s.playerRepo.GetPlayer(ctx, &playerRepo.GetPlayerInput{
				PlayerID: playerID,
			})
			if err != nil {
				return nil, err
			}

			player.CurrentGameID = rollOffGameOutput.Game.ID

			err = s.playerRepo.SavePlayer(ctx, &playerRepo.SavePlayerInput{
				Player: player,
			})
			if err != nil {
				return nil, err
			}
		}

		// Store the lowest roll-off information
		needsLowestRollOff = true
		lowestRollOffGameID = rollOffGameOutput.Game.ID
		lowestRollOffPlayerIDs = lowestRollPlayerIDs
	}

	// Convert map to slice for output
	playerStats := make([]*PlayerStats, 0, len(playerStatsMap))
	for _, stats := range playerStatsMap {
		playerStats = append(playerStats, stats)
	}

	// Update game status to completed if no roll-offs are needed
	if !needsHighestRollOff && !needsLowestRollOff {
		game.Status = models.GameStatusCompleted
		game.UpdatedAt = s.clock.Now()

		// Save the updated game
		err = s.gameRepo.SaveGame(ctx, &gameRepo.SaveGameInput{
			Game: game,
		})
		if err != nil {
			return nil, err
		}

		// If this is a roll-off game, update the parent game as well
		if isRollOffGame && parentGame != nil {
			// Check if the parent game has any other active roll-offs
			hasOtherActiveRollOffs := false

			// If the parent game has a highest roll-off that's not this game
			if parentGame.HighestRollOffGameID != "" && parentGame.HighestRollOffGameID != game.ID {
				// Check if that roll-off is still active
				highestRollOffGame, err := s.gameRepo.GetGame(ctx, &gameRepo.GetGameInput{
					GameID: parentGame.HighestRollOffGameID,
				})
				if err == nil && highestRollOffGame.Status != models.GameStatusCompleted {
					hasOtherActiveRollOffs = true
				}
			}

			// If the parent game has a lowest roll-off that's not this game
			if parentGame.LowestRollOffGameID != "" && parentGame.LowestRollOffGameID != game.ID {
				// Check if that roll-off is still active
				lowestRollOffGame, err := s.gameRepo.GetGame(ctx, &gameRepo.GetGameInput{
					GameID: parentGame.LowestRollOffGameID,
				})
				if err == nil && lowestRollOffGame.Status != models.GameStatusCompleted {
					hasOtherActiveRollOffs = true
				}
			}

			// If there are no other active roll-offs, mark the parent game as completed
			if !hasOtherActiveRollOffs {
				parentGame.Status = models.GameStatusCompleted
				parentGame.UpdatedAt = s.clock.Now()

				// Save the updated parent game
				err = s.gameRepo.SaveGame(ctx, &gameRepo.SaveGameInput{
					Game: parentGame,
				})
				if err != nil {
					log.Printf("Error updating parent game status: %v", err)
					// Don't return the error, continue with ending the game
				}
			}
		}
	} else {
		// If there are roll-offs, mark the game as roll-off
		game.Status = models.GameStatusRollOff
		game.UpdatedAt = s.clock.Now()

		// Save the updated game
		err = s.gameRepo.SaveGame(ctx, &gameRepo.SaveGameInput{
			Game: game,
		})
		if err != nil {
			return nil, err
		}
	}

	// Prepare the output
	output := &EndGameOutput{
		Success:                 !needsHighestRollOff && !needsLowestRollOff,
		FinalLeaderboard:        playerStats,
		NeedsHighestRollOff:     needsHighestRollOff,
		HighestRollOffGameID:    highestRollOffGameID,
		HighestRollOffPlayerIDs: highestRollOffPlayerIDs,
		NeedsLowestRollOff:      needsLowestRollOff,
		LowestRollOffGameID:     lowestRollOffGameID,
		LowestRollOffPlayerIDs:  lowestRollOffPlayerIDs,
	}

	// Set backward compatibility fields
	if needsHighestRollOff {
		output.NeedsRollOff = true
		output.RollOffType = RollOffTypeHighest
		output.RollOffGameID = highestRollOffGameID
		output.RollOffPlayerIDs = highestRollOffPlayerIDs
	} else if needsLowestRollOff {
		output.NeedsRollOff = true
		output.RollOffType = RollOffTypeLowest
		output.RollOffGameID = lowestRollOffGameID
		output.RollOffPlayerIDs = lowestRollOffPlayerIDs
	}

	// Get the session ID for the channel
	sessionID := s.getSessionIDForChannel(ctx, game.ChannelID)
	output.SessionID = sessionID

	// Only fetch the session leaderboard if the game is actually ending (no roll-offs needed)
	if !needsHighestRollOff && !needsLowestRollOff {
		// Get the session leaderboard
		sessionLeaderboardOutput, err := s.GetSessionLeaderboard(ctx, &GetSessionLeaderboardInput{
			SessionID: sessionID,
		})
		if err == nil && sessionLeaderboardOutput != nil {
			output.SessionLeaderboard = sessionLeaderboardOutput.Entries
		}
	}

	return output, nil
}

// HandleRollOff manages roll-offs for tied players
func (s *service) HandleRollOff(ctx context.Context, input *HandleRollOffInput) (*HandleRollOffOutput, error) {
	// Validate input
	if input == nil {
		return nil, errors.New("input cannot be nil")
	}

	if input.ParentGameID == "" {
		return nil, errors.New("parent game ID cannot be empty")
	}

	if input.RollOffGameID == "" {
		return nil, errors.New("roll-off game ID cannot be empty")
	}

	if len(input.PlayerIDs) < 2 {
		return nil, errors.New("at least 2 players are required for a roll-off")
	}

	if input.Type != RollOffTypeHighest && input.Type != RollOffTypeLowest {
		return nil, errors.New("invalid roll-off type")
	}

	// Get the roll-off game
	rollOffGame, err := s.gameRepo.GetGame(ctx, &gameRepo.GetGameInput{
		GameID: input.RollOffGameID,
	})
	if err != nil {
		return nil, ErrGameNotFound
	}

	// Ensure the roll-off game is in the correct state
	if rollOffGame.Status != models.GameStatusRollOff {
		return nil, ErrInvalidGameState
	}

	// Ensure the roll-off game has the correct parent
	if rollOffGame.ParentGameID != input.ParentGameID {
		return nil, errors.New("roll-off game does not belong to the specified parent game")
	}

	// Check if all players in the roll-off have rolled
	allRolled := true
	var highestValue int
	var lowestValue int = s.diceSides + 1 // Initialize to a value higher than possible

	// Track players with highest/lowest rolls
	highestPlayers := []string{}
	lowestPlayers := []string{}

	// First pass: check if all have rolled and find highest/lowest values
	for _, participant := range rollOffGame.Participants {
		// Check if this participant is part of the roll-off
		isInRollOff := false
		for _, playerID := range input.PlayerIDs {
			if participant.PlayerID == playerID {
				isInRollOff = true
				break
			}
		}

		if !isInRollOff {
			continue
		}

		// Check if player has rolled
		if participant.RollTime == nil {
			allRolled = false
			break
		}

		// Update highest/lowest values
		if participant.RollValue > highestValue {
			highestValue = participant.RollValue
		}

		if participant.RollValue < lowestValue {
			lowestValue = participant.RollValue
		}
	}

	// If not all players have rolled, we can't determine winners yet
	if !allRolled {
		return &HandleRollOffOutput{
			Success:             true,
			NeedsAnotherRollOff: false,
		}, nil
	}

	// Second pass: identify players with highest/lowest rolls
	for _, participant := range rollOffGame.Participants {
		// Check if this participant is part of the roll-off
		isInRollOff := false
		for _, playerID := range input.PlayerIDs {
			if participant.PlayerID == playerID {
				isInRollOff = true
				break
			}
		}

		if !isInRollOff {
			continue
		}

		if participant.RollValue == highestValue {
			highestPlayers = append(highestPlayers, participant.PlayerID)
		}

		if participant.RollValue == lowestValue {
			lowestPlayers = append(lowestPlayers, participant.PlayerID)
		}
	}

	// Determine winners based on roll-off type
	var winners []string
	var needsAnotherRollOff bool
	var nextRollOffGameID string

	if input.Type == RollOffTypeHighest {
		// For highest roll-off, winners are those with the highest roll
		winners = highestPlayers

		// If there's still a tie for highest, we need another roll-off
		if len(highestPlayers) > 1 {
			needsAnotherRollOff = true
		}
	} else { // RollOffTypeLowest
		// For lowest roll-off, winners (or rather "losers") are those with the lowest roll
		winners = lowestPlayers

		// If there's still a tie for lowest, we need another roll-off
		if len(lowestPlayers) > 1 {
			needsAnotherRollOff = true
		}
	}

	// If we need another roll-off, create it
	if needsAnotherRollOff {
		// Create the roll-off game with the repository
		rollOffGameOutput, err := s.gameRepo.CreateRollOffGame(ctx, &gameRepo.CreateRollOffGameInput{
			ChannelID:    rollOffGame.ChannelID,
			CreatorID:    rollOffGame.CreatorID,
			ParentGameID: input.ParentGameID, // Keep the original parent
			PlayerIDs:    winners,
			PlayerNames:  getPlayerNames(rollOffGame.Participants, winners),
		})

		if err != nil {
			return nil, fmt.Errorf("failed to create roll-off game: %w", err)
		}

		nextRollOffGameID = rollOffGameOutput.Game.ID
	} else {
		// No more roll-offs needed, update the parent game status if needed
		if input.Type == RollOffTypeLowest {
			// For lowest roll-off, the losers take drinks
			// Assign drinks to the losers
			for _, loserID := range winners {
				// Create a new drink record using the repository
				_, drinkErr := s.drinkLedgerRepo.CreateDrinkRecord(ctx, &ledgerRepo.CreateDrinkRecordInput{
					GameID:     input.ParentGameID,
					ToPlayerID: loserID,
					Reason:     models.DrinkReasonLowestRoll,
				})

				if drinkErr != nil {
					return nil, fmt.Errorf("failed to create drink record: %w", drinkErr)
				}
			}
		}

		// Update the roll-off game status to completed
		rollOffGame.Status = models.GameStatusCompleted
		rollOffGame.UpdatedAt = s.clock.Now()

		// Save the updated roll-off game
		err = s.gameRepo.SaveGame(ctx, &gameRepo.SaveGameInput{
			Game: rollOffGame,
		})
		if err != nil {
			return nil, err
		}
	}

	return &HandleRollOffOutput{
		Success:             true,
		WinnerPlayerIDs:     winners,
		NeedsAnotherRollOff: needsAnotherRollOff,
		NextRollOffGameID:   nextRollOffGameID,
	}, nil
}

func getPlayerNames(participants []*models.Participant, playerIDs []string) map[string]string {
	playerNames := make(map[string]string)
	for _, participant := range participants {
		for _, playerID := range playerIDs {
			if participant.PlayerID == playerID {
				playerNames[playerID] = participant.PlayerName
				break
			}
		}
	}
	return playerNames
}

// FindActiveRollOffGame finds an active roll-off game for a player in a main game's chain
// Returns the roll-off game if found, nil if not found, and an error if something went wrong
func (s *service) FindActiveRollOffGame(ctx context.Context, playerID string, mainGameID string) (*models.Game, error) {
	// First, get all roll-off games with the main game as parent
	rollOffGames, err := s.gameRepo.GetGamesByParent(ctx, &gameRepo.GetGamesByParentInput{
		ParentGameID: mainGameID,
	})
	if err != nil {
		return nil, err
	}

	// Filter for active roll-off games that include the player
	for _, game := range rollOffGames {
		// Only consider active roll-off games
		if game.Status != models.GameStatusRollOff {
			continue
		}

		// Check if the player is a participant in this roll-off
		participant := game.GetParticipant(playerID)
		if participant != nil {
			// Found an active roll-off game for this player
			return game, nil
		}
	}

	// Check for nested roll-offs (roll-offs of roll-offs)
	for _, game := range rollOffGames {
		// Recursively check for nested roll-offs
		nestedGame, err := s.FindActiveRollOffGame(ctx, playerID, game.ID)
		if err != nil {
			// If it's just a "roll-off game not found" error, continue searching
			if errors.Is(err, ErrRollOffGameNotFound) {
				continue
			}
			return nil, err
		}
		if nestedGame != nil {
			return nestedGame, nil
		}
	}

	// No active roll-off game found for this player
	return nil, nil
}

// GetGameByChannel retrieves a game by its Discord channel ID
func (s *service) GetGameByChannel(ctx context.Context, input *GetGameByChannelInput) (*GetGameByChannelOutput, error) {
	if input == nil || input.ChannelID == "" {
		return nil, errors.New("channel ID is required")
	}

	// Get the game from the repository
	game, err := s.gameRepo.GetGameByChannel(ctx, &gameRepo.GetGameByChannelInput{
		ChannelID: input.ChannelID,
	})
	if err != nil {
		// If it's a "game not found" error, return our service-level error
		if errors.Is(err, gameRepo.ErrGameNotFound) {
			return nil, ErrGameNotFound
		}
		// For any other error, wrap it and return
		return nil, fmt.Errorf("failed to get game by channel: %w", err)
	}

	return &GetGameByChannelOutput{
		Game: game,
	}, nil
}

// GetLeaderboard retrieves the leaderboard for a game
func (s *service) GetLeaderboard(ctx context.Context, input *GetLeaderboardInput) (*GetLeaderboardOutput, error) {
	if input == nil || input.GameID == "" {
		return nil, errors.New("game ID is required")
	}

	// Get the game to access participant information
	game, err := s.gameRepo.GetGame(ctx, &gameRepo.GetGameInput{
		GameID: input.GameID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get game: %w", err)
	}

	// Get the drink ledger for this game
	drinkRecords, err := s.drinkLedgerRepo.GetDrinkRecordsForGame(ctx, &ledgerRepo.GetDrinkRecordsForGameInput{
		GameID: input.GameID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get drink records: %w", err)
	}

	// Build maps to track drinks and payment status
	drinkCounts := make(map[string]int) // Total drinks owed
	paidCounts := make(map[string]int)  // Drinks paid

	// Process all drink records
	for _, record := range drinkRecords.Records {
		drinkCounts[record.ToPlayerID]++
		if record.Paid {
			paidCounts[record.ToPlayerID]++
		}
	}

	// Create a map of player IDs to their information
	playerMap := make(map[string]*LeaderboardEntry)

	// First, add all participants from the game
	for _, participant := range game.Participants {
		totalDrinks := drinkCounts[participant.PlayerID]
		paidDrinks := paidCounts[participant.PlayerID]

		playerMap[participant.PlayerID] = &LeaderboardEntry{
			PlayerID:   participant.PlayerID,
			PlayerName: participant.PlayerName,
			DrinkCount: totalDrinks,
			PaidCount:  paidDrinks,
		}
	}

	// Then add any players who have drinks but aren't in the game anymore
	for playerID := range drinkCounts {
		if _, exists := playerMap[playerID]; !exists {
			// Get player name
			player, err := s.playerRepo.GetPlayer(ctx, &playerRepo.GetPlayerInput{
				PlayerID: playerID,
			})

			totalDrinks := drinkCounts[playerID]
			paidDrinks := paidCounts[playerID]

			playerName := "Unknown Player"
			if err == nil {
				playerName = player.Name
			}

			playerMap[playerID] = &LeaderboardEntry{
				PlayerID:   playerID,
				PlayerName: playerName,
				DrinkCount: totalDrinks,
				PaidCount:  paidDrinks,
			}
		}
	}

	// Convert the map to a slice
	var entries []LeaderboardEntry
	for _, entry := range playerMap {
		entries = append(entries, *entry)
	}

	return &GetLeaderboardOutput{
		GameID:  input.GameID,
		Entries: entries,
	}, nil
}

// AbandonGame forcefully abandons a game regardless of its state
func (s *service) AbandonGame(ctx context.Context, input *AbandonGameInput) (*AbandonGameOutput, error) {
	// Get the game
	game, err := s.gameRepo.GetGame(ctx, &gameRepo.GetGameInput{
		GameID: input.GameID,
	})
	if err != nil {
		return nil, ErrGameNotFound
	}

	// Update game status to completed regardless of current state
	game.Status = models.GameStatusCompleted
	game.UpdatedAt = s.clock.Now()

	// Save the updated game
	err = s.gameRepo.SaveGame(ctx, &gameRepo.SaveGameInput{
		Game: game,
	})
	if err != nil {
		return nil, err
	}

	// Clear the CurrentGameID for all players in this game
	for _, participant := range game.Participants {
		// Get the player
		player, err := s.playerRepo.GetPlayer(ctx, &playerRepo.GetPlayerInput{
			PlayerID: participant.PlayerID,
		})
		if err != nil {
			// Log the error but continue with other players
			log.Printf("Error getting player %s: %v", participant.PlayerID, err)
			continue
		}

		// Only update if this is the player's current game
		if player.CurrentGameID == input.GameID {
			// Clear the current game ID
			player.CurrentGameID = ""

			// Save the updated player
			err = s.playerRepo.SavePlayer(ctx, &playerRepo.SavePlayerInput{
				Player: player,
			})
			if err != nil {
				// Log the error but continue with other players
				log.Printf("Error updating player %s: %v", participant.PlayerID, err)
			}
		}
	}

	// Delete the game to clean up all Redis keys including channel mapping
	// This is more reliable than just updating the status
	err = s.gameRepo.DeleteGame(ctx, &gameRepo.DeleteGameInput{
		GameID: input.GameID,
	})
	if err != nil {
		log.Printf("Warning: Failed to delete game %s: %v", input.GameID, err)
		// Continue anyway since we've already marked the game as completed
	}

	return &AbandonGameOutput{
		Success: true,
	}, nil
}

// UpdateGameMessage updates the Discord message ID associated with a game
func (s *service) UpdateGameMessage(ctx context.Context, input *UpdateGameMessageInput) (*UpdateGameMessageOutput, error) {
	// Get the game
	game, err := s.gameRepo.GetGame(ctx, &gameRepo.GetGameInput{
		GameID: input.GameID,
	})
	if err != nil {
		if errors.Is(err, gameRepo.ErrGameNotFound) {
			return nil, ErrGameNotFound
		}
		return nil, fmt.Errorf("failed to get game: %w", err)
	}

	// Update the message ID
	game.MessageID = input.MessageID
	game.UpdatedAt = s.clock.Now()

	// Save the updated game
	err = s.gameRepo.SaveGame(ctx, &gameRepo.SaveGameInput{
		Game: game,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to update game: %w", err)
	}

	return &UpdateGameMessageOutput{
		Success: true,
	}, nil
}

// GetGame retrieves a game by its ID
func (s *service) GetGame(ctx context.Context, input *GetGameInput) (*GetGameOutput, error) {
	// Get the game from the repository
	game, err := s.gameRepo.GetGame(ctx, &gameRepo.GetGameInput{
		GameID: input.GameID,
	})
	if err != nil {
		if errors.Is(err, gameRepo.ErrGameNotFound) {
			return nil, ErrGameNotFound
		}
		return nil, fmt.Errorf("failed to get game: %w", err)
	}

	return &GetGameOutput{
		Game: game,
	}, nil
}

// GetDrinkRecords retrieves all drink records for a game
func (s *service) GetDrinkRecords(ctx context.Context, input *GetDrinkRecordsInput) (*GetDrinkRecordsOutput, error) {
	// Get the drink records from the repository
	records, err := s.drinkLedgerRepo.GetDrinkRecordsForGame(ctx, &ledgerRepo.GetDrinkRecordsForGameInput{
		GameID: input.GameID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get drink records: %w", err)
	}

	return &GetDrinkRecordsOutput{
		Records: records.Records,
	}, nil
}

// GetPlayerTab retrieves a player's current tab (drinks owed and received)
func (s *service) GetPlayerTab(ctx context.Context, input *GetPlayerTabInput) (*GetPlayerTabOutput, error) {
	if input == nil || input.GameID == "" {
		return nil, errors.New("game ID is required")
	}

	if input.PlayerID == "" {
		return nil, errors.New("player ID is required")
	}

	// Get the game
	game, err := s.gameRepo.GetGame(ctx, &gameRepo.GetGameInput{
		GameID: input.GameID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get game: %w", err)
	}

	// Get the player
	player, err := s.playerRepo.GetPlayer(ctx, &playerRepo.GetPlayerInput{
		PlayerID: input.PlayerID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get player: %w", err)
	}

	// Get all drink records for the game
	drinkRecords, err := s.drinkLedgerRepo.GetDrinkRecordsForGame(ctx, &ledgerRepo.GetDrinkRecordsForGameInput{
		GameID: input.GameID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get drink records: %w", err)
	}

	// Create the player tab
	tab := &PlayerTab{
		PlayerID:       player.ID,
		PlayerName:     player.Name,
		DrinksOwed:     []*PlayerTabEntry{},
		DrinksAssigned: []*PlayerTabEntry{},
	}

	// Process all drink records
	for _, record := range drinkRecords.Records {
		// Get the from player name
		var fromPlayerName string
		if record.FromPlayerID == player.ID {
			fromPlayerName = player.Name
		} else {
			// Find the player in the game participants
			for _, participant := range game.Participants {
				if participant.PlayerID == record.FromPlayerID {
					fromPlayerName = participant.PlayerName
					break
				}
			}

			// If not found in participants, try to get from repository
			if fromPlayerName == "" {
				fromPlayer, err := s.playerRepo.GetPlayer(ctx, &playerRepo.GetPlayerInput{
					PlayerID: record.FromPlayerID,
				})
				if err != nil {
					fromPlayerName = "Unknown Player"
				} else {
					fromPlayerName = fromPlayer.Name
				}
			}
		}

		// Get the to player name
		var toPlayerName string
		if record.ToPlayerID == player.ID {
			toPlayerName = player.Name
		} else {
			// Find the player in the game participants
			for _, participant := range game.Participants {
				if participant.PlayerID == record.ToPlayerID {
					toPlayerName = participant.PlayerName
					break
				}
			}

			// If not found in participants, try to get from repository
			if toPlayerName == "" {
				toPlayer, err := s.playerRepo.GetPlayer(ctx, &playerRepo.GetPlayerInput{
					PlayerID: record.ToPlayerID,
				})
				if err != nil {
					toPlayerName = "Unknown Player"
				} else {
					toPlayerName = toPlayer.Name
				}
			}
		}

		// Create a tab entry for this drink record
		entry := &PlayerTabEntry{
			FromPlayerID:   record.FromPlayerID,
			FromPlayerName: fromPlayerName,
			ToPlayerID:     record.ToPlayerID,
			ToPlayerName:   toPlayerName,
			Reason:         record.Reason,
			Timestamp:      record.Timestamp,
			Paid:           record.Paid,
		}

		// Add to the appropriate list
		if record.ToPlayerID == player.ID {
			tab.DrinksOwed = append(tab.DrinksOwed, entry)
			if !record.Paid {
				tab.TotalOwed++
			}
		}

		if record.FromPlayerID == player.ID {
			tab.DrinksAssigned = append(tab.DrinksAssigned, entry)
			if !record.Paid {
				tab.TotalAssigned++
			}
		}
	}

	// Calculate net drinks
	tab.NetDrinks = tab.TotalOwed - tab.TotalAssigned

	return &GetPlayerTabOutput{
		Tab:  tab,
		Game: game,
	}, nil
}

// ResetGameTab resets the drink ledger for a game and returns the previous leaderboard
func (s *service) ResetGameTab(ctx context.Context, input *ResetGameTabInput) (*ResetGameTabOutput, error) {
	if input == nil || input.GameID == "" {
		return nil, errors.New("game ID is required")
	}

	if input.ResetterID == "" {
		return nil, errors.New("resetter ID is required")
	}

	// Get the game
	game, err := s.gameRepo.GetGame(ctx, &gameRepo.GetGameInput{
		GameID: input.GameID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get game: %w", err)
	}

	// Get the resetter's name
	resetter, err := s.playerRepo.GetPlayer(ctx, &playerRepo.GetPlayerInput{
		PlayerID: input.ResetterID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get resetter: %w", err)
	}

	// Get the current leaderboard before resetting
	leaderboardOutput, err := s.GetLeaderboard(ctx, &GetLeaderboardInput{
		GameID: input.GameID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get leaderboard: %w", err)
	}

	// Get all drink records for the game
	drinkRecords, err := s.drinkLedgerRepo.GetDrinkRecordsForGame(ctx, &ledgerRepo.GetDrinkRecordsForGameInput{
		GameID: input.GameID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get drink records: %w", err)
	}

	// Count total drinks
	totalDrinks := len(drinkRecords.Records)

	// Create a summary of the game's drink ledger before reset
	tabSummary := &GameTabSummary{
		GameID:       input.GameID,
		ResetTime:    s.clock.Now(),
		ResetterID:   input.ResetterID,
		ResetterName: resetter.Name,
		Leaderboard:  leaderboardOutput.Entries,
		TotalDrinks:  totalDrinks,
	}

	// Reset the drink ledger
	if input.ArchiveRecords {
		// Archive the records
		err = s.drinkLedgerRepo.ArchiveDrinkRecords(ctx, &ledgerRepo.ArchiveDrinkRecordsInput{
			GameID: input.GameID,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to archive drink records: %w", err)
		}
	} else {
		// Delete the records
		err = s.drinkLedgerRepo.DeleteDrinkRecords(ctx, &ledgerRepo.DeleteDrinkRecordsInput{
			GameID: input.GameID,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to delete drink records: %w", err)
		}
	}

	return &ResetGameTabOutput{
		Success:     true,
		PreviousTab: tabSummary,
		Game:        game,
	}, nil
}

// PayDrink marks a drink as paid
func (s *service) PayDrink(ctx context.Context, input *PayDrinkInput) (*PayDrinkOutput, error) {
	if input == nil || input.GameID == "" {
		return nil, errors.New("game ID is required")
	}

	if input.PlayerID == "" {
		return nil, errors.New("player ID is required")
	}

	if input.DrinkID == "" {
		return nil, errors.New("drink ID is required")
	}

	// Get the game
	game, err := s.gameRepo.GetGame(ctx, &gameRepo.GetGameInput{
		GameID: input.GameID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get game: %w", err)
	}

	// Get the drink record
	drinkRecords, err := s.drinkLedgerRepo.GetDrinkRecordsForGame(ctx, &ledgerRepo.GetDrinkRecordsForGameInput{
		GameID: input.GameID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get drink records: %w", err)
	}

	// Find the specific drink record
	var drinkRecord *models.DrinkLedger
	for _, record := range drinkRecords.Records {
		if record.ID == input.DrinkID {
			drinkRecord = record
			break
		}
	}

	if drinkRecord == nil {
		return nil, fmt.Errorf("drink record with ID %s not found", input.DrinkID)
	}

	// Verify the player is the one who owes the drink
	if drinkRecord.ToPlayerID != input.PlayerID {
		return nil, fmt.Errorf("player %s is not the one who owes this drink", input.PlayerID)
	}

	// Check if the drink is already paid
	if drinkRecord.Paid {
		return nil, fmt.Errorf("drink is already paid")
	}

	// Mark the drink as paid
	err = s.drinkLedgerRepo.MarkDrinkPaid(ctx, &ledgerRepo.MarkDrinkPaidInput{
		DrinkID: input.DrinkID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to mark drink as paid: %w", err)
	}

	// Update the drink record with the paid status
	drinkRecord.Paid = true
	drinkRecord.PaidTimestamp = s.clock.Now()

	return &PayDrinkOutput{
		Success:     true,
		Game:        game,
		DrinkRecord: drinkRecord,
	}, nil
}
