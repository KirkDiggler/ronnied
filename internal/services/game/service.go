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

// Define errors
var (
	ErrGameNotFound        = errors.New("game not found")
	ErrPlayerNotFound      = errors.New("player not found")
	ErrPlayerAlreadyInGame = errors.New("player already in game")
	ErrGameAlreadyExists   = errors.New("game already exists for this channel")
	ErrInvalidGameState    = errors.New("invalid game state")
	ErrPlayerNotInGame     = errors.New("player not in game")
	ErrGameFull            = errors.New("game is at maximum capacity")
	ErrRollOffGameNotFound = errors.New("no active roll-off game found")
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
	diceRoller *dice.Roller
	clock      clock.Clock
	uuid       uuid.UUID
}

// New creates a new game service
func New(cfg *Config) (*service, error) {
	// Validate config
	if cfg == nil {
		return nil, errors.New("config cannot be nil")
	}

	// Validate required dependencies
	if cfg.GameRepo == nil {
		return nil, errors.New("game repository cannot be nil")
	}

	if cfg.PlayerRepo == nil {
		return nil, errors.New("player repository cannot be nil")
	}

	if cfg.DrinkLedgerRepo == nil {
		return nil, errors.New("drink ledger repository cannot be nil")
	}

	if cfg.DiceRoller == nil {
		return nil, errors.New("dice roller cannot be nil")
	}

	if cfg.Clock == nil {
		return nil, errors.New("clock cannot be nil")
	}

	if cfg.UUIDGenerator == nil {
		return nil, errors.New("UUID generator cannot be nil")
	}

	// Set default values for configuration parameters if not provided
	maxPlayers := cfg.MaxPlayers
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
	// Create a new game
	gameID := s.uuid.NewUUID()
	now := s.clock.Now()

	// initialize the game with the creator as the first participant
	game := &models.Game{
		ID:        gameID,
		ChannelID: input.ChannelID,
		CreatorID: input.CreatorID,
		Status:    models.GameStatusWaiting,
		Participants: []*models.Participant{
			{
				ID:         s.uuid.NewUUID(),
				GameID:     gameID,
				PlayerID:   input.CreatorID,
				PlayerName: input.CreatorName,
				Status:     models.ParticipantStatusWaitingToRoll,
			},
		},
		CreatedAt: now,
		UpdatedAt: now,
	}

	// Save the game
	err := s.gameRepo.SaveGame(ctx, &gameRepo.SaveGameInput{
		Game: game,
	})
	if err != nil {
		return nil, err
	}

	return &CreateGameOutput{
		GameID: gameID,
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
		return nil, errors.New("only the game creator can start the game")
	}

	// Ensure the game is in waiting status
	if game.Status != models.GameStatusWaiting {
		return nil, ErrInvalidGameState
	}

	// Ensure there is at least 1 player (the creator)
	if len(game.Participants) < 1 {
		return nil, errors.New("at least 1 player is required to start a game")
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

	// Check if the game is in a valid state for joining
	if game.Status != models.GameStatusWaiting {
		return nil, ErrInvalidGameState
	}

	// Check if the game is full
	if len(game.Participants) >= s.maxPlayers {
		return nil, ErrGameFull
	}

	// Check if player already exists
	existingPlayer, err := s.playerRepo.GetPlayer(ctx, &playerRepo.GetPlayerInput{
		PlayerID: input.PlayerID,
	})

	// If player exists, check if they're already in a game
	if err == nil {
		if existingPlayer.CurrentGameID != "" {
			// Check if they're already in this game
			if existingPlayer.CurrentGameID == input.GameID {
				return nil, ErrPlayerAlreadyInGame
			}

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

	// Add player to the game if not already in it
	playerAlreadyInGame := false
	for _, participant := range game.Participants {
		if participant.PlayerID == input.PlayerID {
			playerAlreadyInGame = true
			break
		}
	}

	if !playerAlreadyInGame {
		game.Participants = append(game.Participants, &models.Participant{
			ID:         s.uuid.NewUUID(),
			GameID:     input.GameID,
			PlayerID:   input.PlayerID,
			PlayerName: input.PlayerName,
			Status:     models.ParticipantStatusWaitingToRoll,
		})
		game.UpdatedAt = s.clock.Now()

		err = s.gameRepo.SaveGame(ctx, &gameRepo.SaveGameInput{
			Game: game,
		})
		if err != nil {
			return nil, err
		}
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

	// Check if the player should roll in a roll-off game instead
	rollOffGame, err := s.FindActiveRollOffGame(ctx, input.PlayerID, input.GameID)
	if err != nil && !errors.Is(err, ErrRollOffGameNotFound) {
		return nil, err
	}

	// If a roll-off game was found, update the input to use that game instead
	if rollOffGame != nil {
		input.GameID = rollOffGame.ID
	}

	// Get the game
	game, err := s.gameRepo.GetGame(ctx, &gameRepo.GetGameInput{
		GameID: input.GameID,
	})
	if err != nil {
		return nil, ErrGameNotFound
	}

	// Check if game is active or in roll-off
	if game.Status != models.GameStatusActive && game.Status != models.GameStatusRollOff {
		return nil, ErrInvalidGameState
	}

	// Find the participant in the game
	participant := game.GetParticipant(input.PlayerID)
	if participant == nil {
		return nil, ErrPlayerNotInGame
	}

	// Check if the participant has already rolled
	if participant.RollTime != nil {
		return nil, errors.New("player has already rolled in this game")
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
	}

	// Update the game
	game.UpdatedAt = now
	err = s.gameRepo.SaveGame(ctx, &gameRepo.SaveGameInput{
		Game: game,
	})
	if err != nil {
		return nil, err
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
				GameID: input.GameID,
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

	return &RollDiceOutput{
		Value:           rollValue,
		IsCriticalHit:   isCriticalHit,
		IsCriticalFail:  isCriticalFail,
		AllPlayersRolled: allPlayersRolled,
		NeedsRollOff:    needsRollOff,
		RollOffType:     RollOffType(rollOffType),
		RollOffGameID:   rollOffGameID,
	}, nil
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

	if input.FromPlayerID == input.ToPlayerID {
		return nil, errors.New("cannot assign a drink to yourself")
	}

	// Get the game
	game, err := s.gameRepo.GetGame(ctx, &gameRepo.GetGameInput{
		GameID: input.GameID,
	})
	if err != nil {
		return nil, ErrGameNotFound
	}

	// Check if game is active
	if game.Status != models.GameStatusActive && game.Status != models.GameStatusRollOff {
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

	// Create a drink record
	now := s.clock.Now()
	drinkRecord := &models.DrinkLedger{
		ID:           s.uuid.NewUUID(),
		GameID:       input.GameID,
		FromPlayerID: input.FromPlayerID,
		ToPlayerID:   input.ToPlayerID,
		Reason:       models.DrinkReason(input.Reason),
		Timestamp:    now,
		Paid:         false,
	}

	// Save the drink record
	err = s.drinkLedgerRepo.AddDrinkRecord(ctx, &ledgerRepo.AddDrinkRecordInput{
		Record: drinkRecord,
	})
	if err != nil {
		return nil, err
	}

	// Update the assigning participant's status
	assigningParticipant.Status = models.ParticipantStatusActive

	// Update the game
	game.UpdatedAt = now
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
			GameID: input.GameID,
		})
		if err == nil {
		} else {
			// Log the error but don't return it to the caller
			log.Printf("Error ending game after drink assignment: %v", err)
		}
	}

	return &AssignDrinkOutput{
		Success:   true,
		GameEnded: allPlayersRolled && allDrinksAssigned,
		EndGameOutput: endGameOutput,
	}, nil
}

// EndGame concludes a game session
func (s *service) EndGame(ctx context.Context, input *EndGameInput) (*EndGameOutput, error) {
	// Get the game
	game, err := s.gameRepo.GetGame(ctx, &gameRepo.GetGameInput{
		GameID: input.GameID,
	})
	if err != nil {
		return nil, ErrGameNotFound
	}

	// Check if game is active
	if game.Status != models.GameStatusActive && game.Status != models.GameStatusRollOff {
		return nil, ErrInvalidGameState
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
		GameID: input.GameID,
	})
	if err != nil {
		return nil, err
	}

	// Build a map of player ID to player stats
	playerStatsMap := make(map[string]*PlayerStats)

	// Initialize stats for all participants
	for _, participant := range game.Participants {
		// Get player info
		player, err := s.playerRepo.GetPlayer(ctx, &playerRepo.GetPlayerInput{
			PlayerID: participant.PlayerID,
		})
		if err != nil {
			return nil, err
		}

		// Initialize player stats
		playerStatsMap[participant.PlayerID] = &PlayerStats{
			PlayerID:       participant.PlayerID,
			PlayerName:     player.Name,
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

	// Convert map to slice for output
	playerStats := make([]*PlayerStats, 0, len(playerStatsMap))
	for _, stats := range playerStatsMap {
		playerStats = append(playerStats, stats)
	}

	// Update game status to completed
	game.Status = models.GameStatusCompleted
	game.UpdatedAt = s.clock.Now()

	// Save the updated game
	err = s.gameRepo.SaveGame(ctx, &gameRepo.SaveGameInput{
		Game: game,
	})
	if err != nil {
		return nil, err
	}

	return &EndGameOutput{
		Success:          true,
		FinalLeaderboard: playerStats,
	}, nil
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

	// If we need another roll-off, create a new roll-off game
	if needsAnotherRollOff {
		// Create a new roll-off game for the tied players
		newRollOffGameID := s.uuid.NewUUID()
		now := s.clock.Now()

		// Create participants for the new roll-off
		participants := make([]*models.Participant, 0, len(winners))
		for _, playerID := range winners {
			participants = append(participants, &models.Participant{
				ID:       s.uuid.NewUUID(),
				GameID:   newRollOffGameID,
				PlayerID: playerID,
				Status:   models.ParticipantStatusWaitingToRoll,
			})
		}

		// Create the new roll-off game
		newRollOffGame := &models.Game{
			ID:           newRollOffGameID,
			ChannelID:    rollOffGame.ChannelID,
			CreatorID:    rollOffGame.CreatorID,
			Status:       models.GameStatusRollOff,
			ParentGameID: input.ParentGameID, // Keep the original parent
			Participants: participants,
			CreatedAt:    now,
			UpdatedAt:    now,
		}

		// Save the new roll-off game
		err = s.gameRepo.SaveGame(ctx, &gameRepo.SaveGameInput{
			Game: newRollOffGame,
		})
		if err != nil {
			return nil, err
		}

		nextRollOffGameID = newRollOffGameID
	} else {
		// No more roll-offs needed, update the parent game status if needed
		if input.Type == RollOffTypeLowest {
			// For lowest roll-off, the losers take drinks
			// Assign drinks to the losers
			for _, loserID := range winners {
				// Create a drink ledger entry for each loser
				ledgerEntry := &models.DrinkLedger{
					ID:           s.uuid.NewUUID(),
					GameID:       input.ParentGameID,
					ToPlayerID:   loserID,
					FromPlayerID: "", // System-assigned drink
					Reason:       models.DrinkReasonLowestRoll,
					Timestamp:    s.clock.Now(),
				}

				// Save the drink ledger entry
				err = s.drinkLedgerRepo.AddDrinkRecord(ctx, &ledgerRepo.AddDrinkRecordInput{
					Record: ledgerEntry,
				})
				if err != nil {
					return nil, err
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
			return nil, err
		}
		if nestedGame != nil {
			return nestedGame, nil
		}
	}

	// No active roll-off game found for this player
	return nil, ErrRollOffGameNotFound
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

	// Get the drink ledger for this game
	drinkRecords, err := s.drinkLedgerRepo.GetDrinkRecordsForGame(ctx, &ledgerRepo.GetDrinkRecordsForGameInput{
		GameID: input.GameID,
	})
	if err != nil {
		return nil, err
	}

	// Build a map of player ID to drink count
	drinkCounts := make(map[string]int)
	for _, record := range drinkRecords.Records {
		drinkCounts[record.ToPlayerID]++
	}

	// Create leaderboard entries
	var entries []LeaderboardEntry
	for playerID, count := range drinkCounts {
		// Get player name
		player, err := s.playerRepo.GetPlayer(ctx, &playerRepo.GetPlayerInput{
			PlayerID: playerID,
		})
		if err != nil {
			// Skip players we can't find
			continue
		}

		entries = append(entries, LeaderboardEntry{
			PlayerID:   playerID,
			PlayerName: player.Name,
			DrinkCount: count,
		})
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
