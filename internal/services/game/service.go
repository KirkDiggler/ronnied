package game

import (
	"context"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"time"

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
	uuid       uuid.Generator
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

	// Ensure the game is in waiting status
	if game.Status != models.GameStatusWaiting {
		return nil, ErrInvalidGameState
	}

	// Ensure there is at least 1 player (the creator)
	if len(game.Participants) < 1 {
		return nil, ErrNotEnoughPlayers
	}

	// Get the creator's name
	creatorName := "Unknown Creator"
	for _, p := range game.Participants {
		if p.PlayerID == game.CreatorID {
			creatorName = p.PlayerName
			break
		}
	}

	// Check if the player is the game creator
	isCreator := game.CreatorID == input.PlayerID

	// If not the creator, check if force start is allowed
	forceStarted := false
	if !isCreator {
		// Only allow force start if explicitly requested and game is older than 5 minutes
		if !input.ForceStart {
			return nil, ErrNotCreator
		}

		// Calculate game age
		gameAge := s.clock.Now().Sub(game.CreatedAt)
		fiveMinutes := 5 * time.Minute

		// If game is less than 5 minutes old, don't allow force start
		if gameAge < fiveMinutes {
			return nil, fmt.Errorf("%w: game must be at least 5 minutes old for non-creator to start (current age: %v)",
				ErrNotCreator, gameAge.Round(time.Second))
		}

		// Game is old enough, allow force start
		forceStarted = true

		// Assign a drink to the creator for delaying
		_, err = s.drinkLedgerRepo.CreateDrinkRecord(ctx, &ledgerRepo.CreateDrinkRecordInput{
			GameID:       input.GameID,
			FromPlayerID: input.PlayerID,
			ToPlayerID:   game.CreatorID,
			Reason:       models.DrinkReasonDelayedStart,
			Timestamp:    s.clock.Now(),
			SessionID:    s.getSessionIDForChannel(ctx, game.ChannelID),
		})

		if err != nil {
			// Log the error but don't fail the operation
			log.Printf("Error assigning drink to creator for delayed start: %v", err)
		}
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
		Success:      true,
		ForceStarted: forceStarted,
		CreatorID:    game.CreatorID,
		CreatorName:  creatorName,
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
		return nil, fmt.Errorf("failed to get game: %w", err)
	}

	// Process the roll based on the game state
	if game.Status.IsRollOff() {
		// This is a roll-off
		return s.processRollOffInGame(ctx, input, game)
	} else if game.Status == models.GameStatusActive {
		// This is a normal active game
		return s.processMainGameRoll(ctx, input, game)
	} else {
		// Invalid game state for rolling
		return nil, ErrInvalidGameState
	}
}

// processRollOffInGame handles dice rolling for participants in a roll-off state within the main game
func (s *service) processRollOffInGame(ctx context.Context, input *RollDiceInput, game *models.Game) (*RollDiceOutput, error) {
	// Check if game is in a valid roll-off state
	if !game.Status.IsRollOff() {
		return nil, fmt.Errorf("%w: game status is %s, expected roll-off", ErrInvalidGameState, game.Status)
	}

	// Check if player is part of the roll-off
	isInRollOff := false
	for _, playerID := range game.RollOffPlayerIDs {
		if playerID == input.PlayerID {
			isInRollOff = true
			break
		}
	}

	if !isInRollOff {
		return nil, ErrPlayerNotInRollOff
	}

	// Find the participant in the game
	participant := game.GetParticipant(input.PlayerID)
	if participant == nil {
		return nil, ErrPlayerNotInGame
	}

	// Check if the participant has already rolled in this roll-off round
	if participant.Status == models.ParticipantStatusRolledInRollOff {
		return nil, fmt.Errorf("player %s has already rolled in this roll-off round", participant.PlayerName)
	}

	// Roll the dice
	rollValue := s.diceRoller.Roll(s.diceSides)
	now := s.clock.Now()

	// Update the participant's roll
	participant.RollValue = rollValue
	participant.RollTime = &now
	participant.Status = models.ParticipantStatusRolledInRollOff

	// Update the game
	game.UpdatedAt = now

	// Save the game
	err := s.gameRepo.SaveGame(ctx, &gameRepo.SaveGameInput{
		Game: game,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to save game: %w", err)
	}

	// Check if all players in the roll-off have rolled
	allPlayersRolled := true
	for _, playerID := range game.RollOffPlayerIDs {
		participant := game.GetParticipant(playerID)
		if participant == nil || participant.Status != models.ParticipantStatusRolledInRollOff {
			allPlayersRolled = false
			break
		}
	}

	// If all players have rolled, complete the roll-off
	if allPlayersRolled {
		_, err = s.CompleteRollOff(ctx, &CompleteRollOffInput{
			GameID: game.ID,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to complete roll-off: %w", err)
		}

		// Reload the game to get the updated state after completing the roll-off
		gameOutput, err := s.gameRepo.GetGame(ctx, &gameRepo.GetGameInput{
			GameID: game.ID,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to reload game after completing roll-off: %w", err)
		}
		game = gameOutput
	}

	// Prepare result information for roll-off
	result := fmt.Sprintf("You Rolled a %d in the Roll-Off!", rollValue)
	details := "Your roll has been recorded."

	// Add more detailed information about the roll-off type
	if game.Status == models.GameStatusRollOffHighest {
		details += "\n\nThis is a highest roll tie-breaker. The player with the highest roll wins!"
	} else if game.Status == models.GameStatusRollOffLowest {
		details += "\n\nThis is a lowest roll tie-breaker. The player with the lowest roll loses!"
	}

	if allPlayersRolled {
		details += "\n\nAll players have rolled in this roll-off."

		// Add information about the outcome if the roll-off is complete
		if game.Status == models.GameStatusActive || game.Status == models.GameStatusCompleted {
			// Roll-off has been completed, add outcome information
			if game.RollOffType == models.RollOffTypeHighest {
				// Find the player with the highest roll
				highestRoll := 0
				highestRollerName := ""
				for _, playerID := range game.RollOffPlayerIDs {
					participant := game.GetParticipant(playerID)
					if participant != nil && participant.RollValue > highestRoll {
						highestRoll = participant.RollValue
						highestRollerName = participant.PlayerName
					}
				}
				details += fmt.Sprintf("\n\n🏆 %s won the roll-off with a %d!", highestRollerName, highestRoll)
			} else if game.RollOffType == models.RollOffTypeLowest {
				// Find the player with the lowest roll
				lowestRoll := s.diceSides + 1
				lowestRollerName := ""
				for _, playerID := range game.RollOffPlayerIDs {
					participant := game.GetParticipant(playerID)
					if participant != nil && participant.RollValue < lowestRoll && participant.RollValue > 0 {
						lowestRoll = participant.RollValue
						lowestRollerName = participant.PlayerName
					}
				}
				details += fmt.Sprintf("\n\n💀 %s lost the roll-off with a %d!", lowestRollerName, lowestRoll)
			}
		}
	}

	return &RollDiceOutput{
		PlayerID:         input.PlayerID,
		RollValue:        rollValue,
		Result:           result,
		Details:          details,
		EligiblePlayers:  nil, // No player options in roll-offs
		Game:             game,
		IsRollOffRoll:    true,
		AllPlayersRolled: allPlayersRolled,
	}, nil
}

// processMainGameRoll handles dice rolling in a main game
func (s *service) processMainGameRoll(ctx context.Context, input *RollDiceInput, game *models.Game) (*RollDiceOutput, error) {
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
			_, err := s.drinkLedgerRepo.CreateDrinkRecord(ctx, &ledgerRepo.CreateDrinkRecordInput{
				GameID:       input.GameID,
				FromPlayerID: input.PlayerID,
				ToPlayerID:   input.PlayerID,
				Reason:       models.DrinkReasonCriticalFail,
				Timestamp:    now,
				SessionID:    s.getSessionIDForChannel(ctx, game.ChannelID),
			})
			if err != nil {
				return nil, fmt.Errorf("failed to create critical fail drink record: %w", err)
			}
		}
	}

	// Update the game
	game.UpdatedAt = now
	err := s.gameRepo.SaveGame(ctx, &gameRepo.SaveGameInput{
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
	var rollOffGame *models.Game
	var rollOffGames []*models.Game

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
			if err != nil {
				return nil, fmt.Errorf("failed to end game: %w", err)
			}

			if endGameOutput.NeedsRollOff {
				needsRollOff = true
				rollOffType = string(endGameOutput.RollOffType)
				rollOffGameID = endGameOutput.RollOffGameID

				// Get the roll-off game data
				if rollOffGameID != "" {
					rollOffGameOutput, err := s.gameRepo.GetGame(ctx, &gameRepo.GetGameInput{
						GameID: rollOffGameID,
					})
					if err == nil {
						rollOffGame = rollOffGameOutput
						rollOffGames = append(rollOffGames, rollOffGameOutput)
					}
				}
			}
		}
	}

	// Check for any existing roll-off games
	if game.RollOffGameID != "" && game.RollOffGameID != rollOffGameID {
		existingRollOffOutput, err := s.gameRepo.GetGame(ctx, &gameRepo.GetGameInput{
			GameID: game.RollOffGameID,
		})
		if err == nil {
			// Only add if we didn't already add this game
			if rollOffGame == nil || existingRollOffOutput.ID != rollOffGame.ID {
				rollOffGames = append(rollOffGames, existingRollOffOutput)
			}
		}
	}

	// Get all roll-off games for this main game
	rollOffGamesOutput, err := s.gameRepo.GetGamesByParent(ctx, &gameRepo.GetGamesByParentInput{
		ParentGameID: game.ID,
	})
	if err == nil && len(rollOffGamesOutput) > 0 {
		// Add any roll-off games we haven't already added
		for _, g := range rollOffGamesOutput {
			if g.Status == models.GameStatusRollOff {
				// Check if we already have this game
				alreadyHave := false
				for _, existing := range rollOffGames {
					if existing.ID == g.ID {
						alreadyHave = true
						break
					}
				}
				if !alreadyHave {
					rollOffGames = append(rollOffGames, g)
				}
			}
		}
	}

	// Get the player name
	playerName := ""
	for _, p := range game.Participants {
		if p.PlayerID == input.PlayerID {
			playerName = p.PlayerName
			break
		}
	}

	// Prepare result information
	result, details, eligiblePlayers := s.prepareRollResult(
		isCriticalHit,
		isCriticalFail,
		rollValue,
		input.PlayerID,
		game,
	)

	// Build the list of game IDs that need to be updated
	gameIDsToUpdate := []string{input.GameID}

	// Add roll-off game IDs to the update list
	for _, g := range rollOffGames {
		gameIDsToUpdate = append(gameIDsToUpdate, g.ID)
	}
	return &RollDiceOutput{
		RollValue:         rollValue,
		PlayerID:          input.PlayerID,
		PlayerName:        playerName,
		IsCriticalHit:     isCriticalHit,
		IsCriticalFail:    isCriticalFail,
		AllPlayersRolled:  allPlayersRolled,
		NeedsRollOff:      needsRollOff,
		RollOffType:       RollOffType(rollOffType),
		RollOffGameID:     rollOffGameID,
		ActiveRollOffGame: rollOffGame,
		RollOffGames:      rollOffGames,
		Result:            result,
		Details:           details,
		EligiblePlayers:   eligiblePlayers,
		Game:              game,
		IsRollOffRoll:     false,
		ParentGameID:      "",
		ParentGame:        nil,
		GameIDsToUpdate:   gameIDsToUpdate,
	}, nil
}

// processRollOffRoll handles dice rolling in a roll-off game
func (s *service) processRollOffRoll(ctx context.Context, input *RollDiceInput, rollOffGame *models.Game) (*RollDiceOutput, error) {
	// Check if game is in a valid state for rolling
	if !isValidGameStateForRolling(rollOffGame.Status) {
		return nil, fmt.Errorf("%w: roll-off game status is %s", ErrInvalidGameState, rollOffGame.Status)
	}

	// Find the participant in the game
	participant := rollOffGame.GetParticipant(input.PlayerID)
	if participant == nil {
		return nil, ErrPlayerNotInGame
	}

	// Check if the participant has already rolled
	if participant.RollTime != nil {
		return nil, fmt.Errorf("player %s has already rolled in this roll-off", participant.PlayerName)
	}

	// Roll the dice
	rollValue := s.diceRoller.Roll(s.diceSides)
	now := s.clock.Now()

	// Update the participant's roll
	participant.RollValue = rollValue
	participant.RollTime = &now
	participant.Status = models.ParticipantStatusActive // No critical hits/fails in roll-offs

	// Update the game
	rollOffGame.UpdatedAt = now
	err := s.gameRepo.SaveGame(ctx, &gameRepo.SaveGameInput{
		Game: rollOffGame,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to save roll-off game: %w", err)
	}

	// Get parent game if this is a roll-off
	var parentGame *models.Game
	var rollOffGames []*models.Game

	if rollOffGame.ParentGameID != "" {
		parentGameOutput, err := s.gameRepo.GetGame(ctx, &gameRepo.GetGameInput{
			GameID: rollOffGame.ParentGameID,
		})
		if err != nil {
			// Log but don't fail if we can't get the parent game
			log.Printf("Warning: Failed to get parent game: %v", err)
		} else {
			parentGame = parentGameOutput

			// Check if the parent game has other roll-off games
			if parentGame.HighestRollOffGameID != "" && parentGame.HighestRollOffGameID != rollOffGame.ID {
				highestRollOffOutput, err := s.gameRepo.GetGame(ctx, &gameRepo.GetGameInput{
					GameID: parentGame.HighestRollOffGameID,
				})
				if err == nil {
					rollOffGames = append(rollOffGames, highestRollOffOutput)
				}
			}

			if parentGame.LowestRollOffGameID != "" && parentGame.LowestRollOffGameID != rollOffGame.ID {
				lowestRollOffOutput, err := s.gameRepo.GetGame(ctx, &gameRepo.GetGameInput{
					GameID: parentGame.LowestRollOffGameID,
				})
				if err == nil {
					rollOffGames = append(rollOffGames, lowestRollOffOutput)
				}
			}
		}
	}

	// Add the current roll-off game to the list
	rollOffGames = append(rollOffGames, rollOffGame)

	// Check if all players have rolled
	allPlayersRolled := true
	for _, p := range rollOffGame.Participants {
		if p.RollTime == nil {
			allPlayersRolled = false
			break
		}
	}

	// If all players have rolled, try to end the roll-off game
	var endGameOutput *EndGameOutput
	needsRollOff := false
	rollOffType := ""
	rollOffGameID := ""
	var nestedRollOffGame *models.Game

	if allPlayersRolled {
		endGameOutput, err = s.EndGame(ctx, &EndGameInput{
			Game: rollOffGame,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to end roll-off game: %w", err)
		}

		if endGameOutput.NeedsRollOff {
			needsRollOff = true
			rollOffType = string(endGameOutput.RollOffType)
			rollOffGameID = endGameOutput.RollOffGameID

			// Get the nested roll-off game if one was created
			if rollOffGameID != "" {
				nestedRollOffOutput, err := s.gameRepo.GetGame(ctx, &gameRepo.GetGameInput{
					GameID: rollOffGameID,
				})
				if err == nil {
					nestedRollOffGame = nestedRollOffOutput
					rollOffGames = append(rollOffGames, nestedRollOffOutput)
				}
			}
		}
	}

	// Get the player name
	playerName := ""
	for _, p := range rollOffGame.Participants {
		if p.PlayerID == input.PlayerID {
			playerName = p.PlayerName
			break
		}
	}

	// Prepare result information for roll-off
	result := fmt.Sprintf("You Rolled a %d in the Roll-Off!", rollValue)
	details := "Your roll has been recorded."

	// Add more detailed information about the roll-off
	if rollOffGame.ParentGameID != "" && parentGame != nil {
		// Determine roll-off type based on the parent game's references
		if parentGame.HighestRollOffGameID == rollOffGame.ID {
			details += "\n\nThis is a highest roll tie-breaker. The player with the highest roll wins!"
		} else if parentGame.LowestRollOffGameID == rollOffGame.ID {
			details += "\n\nThis is a lowest roll tie-breaker. The player with the lowest roll loses!"
		}
	}

	if allPlayersRolled {
		details += "\n\nAll players have rolled in this roll-off."

		// Add information about the outcome if the roll-off is complete
		if rollOffGame.Status == models.GameStatusCompleted {
			// Find the winner/loser based on roll-off type
			if parentGame != nil {
				if parentGame.HighestRollOffGameID == rollOffGame.ID {
					// Find the player with the highest roll
					highestRoll := 0
					highestRollerName := ""
					for _, p := range rollOffGame.Participants {
						if p.RollValue > highestRoll {
							highestRoll = p.RollValue
							highestRollerName = p.PlayerName
						}
					}
					details += fmt.Sprintf("\n\n🏆 %s won the roll-off with a %d!", highestRollerName, highestRoll)
				} else if parentGame.LowestRollOffGameID == rollOffGame.ID {
					// Find the player with the lowest roll
					lowestRoll := s.diceSides + 1
					lowestRollerName := ""
					for _, p := range rollOffGame.Participants {
						if p.RollValue < lowestRoll && p.RollValue > 0 {
							lowestRoll = p.RollValue
							lowestRollerName = p.PlayerName
						}
					}
					details += fmt.Sprintf("\n\n💀 %s lost the roll-off with a %d!", lowestRollerName, lowestRoll)
				}
			}
		}
	}

	// Determine which game IDs need to be updated
	gameIDsToUpdate := []string{rollOffGame.ID}

	// Add parent game ID to the update list
	if rollOffGame.ParentGameID != "" {
		gameIDsToUpdate = append(gameIDsToUpdate, rollOffGame.ParentGameID)
	}

	// Add any nested roll-off game to the update list
	if rollOffGame.RollOffGameID != "" {
		gameIDsToUpdate = append(gameIDsToUpdate, rollOffGame.RollOffGameID)
	}

	// Add IDs of all roll-off games to the update list
	for _, g := range rollOffGames {
		// Check if we already have this ID
		alreadyHave := false
		for _, id := range gameIDsToUpdate {
			if id == g.ID {
				alreadyHave = true
				break
			}
		}
		if !alreadyHave {
			gameIDsToUpdate = append(gameIDsToUpdate, g.ID)
		}
	}

	return &RollDiceOutput{
		RollValue:         rollValue,
		PlayerID:          input.PlayerID,
		PlayerName:        playerName,
		IsCriticalHit:     false, // No critical hits in roll-offs
		IsCriticalFail:    false, // No critical fails in roll-offs
		AllPlayersRolled:  allPlayersRolled,
		NeedsRollOff:      needsRollOff,
		RollOffType:       RollOffType(rollOffType),
		RollOffGameID:     rollOffGameID,
		ActiveRollOffGame: nestedRollOffGame,
		RollOffGames:      rollOffGames,
		Result:            result,
		Details:           details,
		EligiblePlayers:   nil, // No drink assignments in roll-offs
		Game:              rollOffGame,
		IsRollOffRoll:     true,
		ParentGameID:      rollOffGame.ParentGameID,
		ParentGame:        parentGame,
		GameIDsToUpdate:   gameIDsToUpdate,
	}, nil
}

// prepareRollResult creates the user-facing result messages based on roll outcome
func (s *service) prepareRollResult(isCriticalHit, isCriticalFail bool, rollValue int, playerID string, game *models.Game) (string, string, []PlayerOption) {
	var result, details string
	var eligiblePlayers []PlayerOption

	// Set result and details based on roll result
	if isCriticalHit {
		result = fmt.Sprintf("You Rolled a %d! Critical Hit!", rollValue)
		details = "Select a player to assign a drink:"

		// Get eligible players for drink assignment
		for _, p := range game.Participants {
			isCurrentPlayer := p.PlayerID == playerID

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
				if p.PlayerID == playerID {
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

	return result, details, eligiblePlayers
}

// isValidGameStateForRolling checks if a game state allows dice rolling
func isValidGameStateForRolling(status models.GameStatus) bool {
	return status == models.GameStatusActive ||
		status == models.GameStatusRollOff ||
		status == models.GameStatusWaiting
}

// AssignDrink records that one player has assigned a drink to another
// IMPORTANT: DO NOT REMOVE THIS FUNCTIONALITY - It is a core game mechanic that allows players
// to assign drinks when they roll a critical hit (6). This is an essential part of the game flow
// and removing it would break a fundamental aspect of the drinking game.
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

	// ROLL-OFF FUNCTIONALITY TEMPORARILY DISABLED
	// Check for ties with the highest roll (critical hits)
	if len(highestRollPlayerIDs) > 1 {
		// Instead of creating a roll-off, we'll just log that there was a tie
		log.Printf("Highest roll tie detected between %d players. Roll-offs disabled.", len(highestRollPlayerIDs))
		
		// We're not setting needsHighestRollOff = true, so the game will complete normally
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
		// ROLL-OFF FUNCTIONALITY TEMPORARILY DISABLED
		// Instead of creating a roll-off for lowest rollers, we'll just log that there was a tie
		log.Printf("Lowest roll tie detected between %d players. Roll-offs disabled.", len(lowestRollPlayerIDs))
		
		// We're not setting needsLowestRollOff = true, so the game will complete normally
		
		// Since we have multiple lowest rollers, we'll randomly select one to assign a drink to
		if len(lowestRollPlayerIDs) > 0 {
			// Pick a random player from the tied lowest rollers
			randomIndex := rand.Intn(len(lowestRollPlayerIDs))
			lowestPlayerID := lowestRollPlayerIDs[randomIndex]
			
			// Determine which game ID to use for the drink record
			targetGameID := game.ID
			if isRollOffGame {
				// If this is a roll-off game, assign the drink to the parent game
				targetGameID = game.ParentGameID
			}
			
			// Create a drink record for the randomly selected lowest roller
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
		}
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

// StartRollOff initiates a roll-off within a game
func (s *service) StartRollOff(ctx context.Context, input *StartRollOffInput) (*StartRollOffOutput, error) {
	// Validate input
	if input == nil {
		return nil, errors.New("input cannot be nil")
	}

	if input.GameID == "" {
		return nil, errors.New("game ID cannot be empty")
	}

	if len(input.PlayerIDs) < 2 {
		return nil, errors.New("at least 2 players are required for a roll-off")
	}

	if input.Type != RollOffTypeHighest && input.Type != RollOffTypeLowest {
		return nil, errors.New("invalid roll-off type")
	}

	// Get the game
	game, err := s.gameRepo.GetGame(ctx, &gameRepo.GetGameInput{
		GameID: input.GameID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get game: %w", err)
	}

	// Ensure the game is in a valid state for starting a roll-off
	if game.Status != models.GameStatusActive {
		return nil, ErrInvalidGameState
	}

	// Set the game to roll-off state
	if input.Type == RollOffTypeHighest {
		game.Status = models.GameStatusRollOffHighest
	} else {
		game.Status = models.GameStatusRollOffLowest
	}

	// Set roll-off properties
	game.RollOffType = models.RollOffType(input.Type) // Convert from service type to model type
	game.RollOffPlayerIDs = input.PlayerIDs
	game.RollOffRound++
	game.UpdatedAt = s.clock.Now()

	// Update participant statuses for roll-off players
	for _, participant := range game.Participants {
		// Check if this participant is part of the roll-off
		isInRollOff := false
		for _, playerID := range input.PlayerIDs {
			if participant.PlayerID == playerID {
				isInRollOff = true
				break
			}
		}

		if isInRollOff {
			// Reset roll for roll-off participants
			participant.Status = models.ParticipantStatusInRollOff
			participant.RollTime = nil
			participant.RollValue = 0
		}
	}

	// Save the updated game
	err = s.gameRepo.SaveGame(ctx, &gameRepo.SaveGameInput{
		Game: game,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to save game: %w", err)
	}

	return &StartRollOffOutput{
		Success: true,
		Game:    game,
	}, nil
}

// CompleteRollOff finalizes a roll-off and processes the results
func (s *service) CompleteRollOff(ctx context.Context, input *CompleteRollOffInput) (*CompleteRollOffOutput, error) {
	// Validate input
	if input == nil {
		return nil, errors.New("input cannot be nil")
	}

	if input.GameID == "" {
		return nil, errors.New("game ID cannot be empty")
	}

	// Get the game
	game, err := s.gameRepo.GetGame(ctx, &gameRepo.GetGameInput{
		GameID: input.GameID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get game: %w", err)
	}

	// Ensure the game is in a roll-off state
	if !game.Status.IsRollOff() {
		return nil, ErrInvalidGameState
	}

	// Check if all roll-off participants have rolled
	allRolled := true
	var highestValue int
	var lowestValue int = s.diceSides + 1 // Initialize to a value higher than possible

	// Track players with highest/lowest rolls
	highestPlayers := []string{}
	lowestPlayers := []string{}
	rollOffParticipants := []*models.Participant{}

	// First pass: check if all have rolled and find highest/lowest values
	for _, participant := range game.Participants {
		// Check if this participant is part of the roll-off
		isInRollOff := false
		for _, playerID := range game.RollOffPlayerIDs {
			if participant.PlayerID == playerID {
				isInRollOff = true
				rollOffParticipants = append(rollOffParticipants, participant)
				break
			}
		}

		if !isInRollOff {
			continue
		}

		// Check if player has rolled
		if participant.RollTime == nil || participant.Status != models.ParticipantStatusRolledInRollOff {
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

	// If not all players have rolled, we can't complete the roll-off yet
	if !allRolled {
		return &CompleteRollOffOutput{
			Success:             true,
			Game:                game,
			NeedsAnotherRollOff: false,
		}, nil
	}

	// Second pass: identify players with highest/lowest rolls
	for _, participant := range rollOffParticipants {
		if participant.RollValue == highestValue {
			highestPlayers = append(highestPlayers, participant.PlayerID)
		}

		if participant.RollValue == lowestValue {
			lowestPlayers = append(lowestPlayers, participant.PlayerID)
		}
	}

	// Determine winners and if another roll-off is needed
	var winners []string
	var needsAnotherRollOff bool
	var drinkAssignments []*models.DrinkLedger

	if game.Status == models.GameStatusRollOffHighest || game.Status == models.GameStatusRollOff && string(game.RollOffType) == string(RollOffTypeHighest) {
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

	// If we need another roll-off, set up for the next round
	if needsAnotherRollOff {
		// Update roll-off player IDs to only include the tied players
		game.RollOffPlayerIDs = winners
		game.RollOffRound++

		// Reset the roll status for the tied players
		for _, participant := range game.Participants {
			isInNextRollOff := false
			for _, playerID := range winners {
				if participant.PlayerID == playerID {
					isInNextRollOff = true
					break
				}
			}

			if isInNextRollOff {
				participant.Status = models.ParticipantStatusInRollOff
				participant.RollTime = nil
				participant.RollValue = 0
			}
		}
	} else {
		// No more roll-offs needed, finalize the results
		if game.Status == models.GameStatusRollOffLowest || game.Status == models.GameStatusRollOff && string(game.RollOffType) == string(RollOffTypeLowest) {
			// For lowest roll-off, the losers take drinks
			for _, loserID := range winners {
				// Create a new drink record
				drinkOutput, drinkErr := s.drinkLedgerRepo.CreateDrinkRecord(ctx, &ledgerRepo.CreateDrinkRecordInput{
					GameID:     game.ID,
					ToPlayerID: loserID,
					Reason:     models.DrinkReasonLowestRoll,
				})

				if drinkErr != nil {
					return nil, fmt.Errorf("failed to create drink record: %w", drinkErr)
				}

				// Add the drink record to our assignments
				drinkAssignments = append(drinkAssignments, drinkOutput.Record)
			}
		}

		// Reset the game to active state
		game.Status = models.GameStatusActive
		game.RollOffType = "" // Empty string for no roll-off type
		game.RollOffPlayerIDs = nil

		// Reset all participants to active status
		for _, participant := range game.Participants {
			if participant.Status == models.ParticipantStatusInRollOff ||
				participant.Status == models.ParticipantStatusRolledInRollOff {
				participant.Status = models.ParticipantStatusActive
			}
		}
	}

	game.UpdatedAt = s.clock.Now()

	// Save the updated game
	err = s.gameRepo.SaveGame(ctx, &gameRepo.SaveGameInput{
		Game: game,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to save game: %w", err)
	}

	return &CompleteRollOffOutput{
		Success:             true,
		Game:                game,
		WinnerPlayerIDs:     winners,
		NeedsAnotherRollOff: needsAnotherRollOff,
		DrinkAssignments:    drinkAssignments,
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

	// Get active roll-off games if this is a main game
	var activeRollOffGames []*models.Game
	if !game.Status.IsRollOff() {
		// Get all roll-off games with this game as parent
		rollOffGames, err := s.gameRepo.GetGamesByParent(ctx, &gameRepo.GetGamesByParentInput{
			ParentGameID: game.ID,
		})
		if err == nil && len(rollOffGames) > 0 {
			// Filter for active roll-off games
			for _, rollOffGame := range rollOffGames {
				if rollOffGame.Status == models.GameStatusRollOff {
					activeRollOffGames = append(activeRollOffGames, rollOffGame)
				}
			}
		}
	}

	return &GetGameByChannelOutput{
		Game:               game,
		ActiveRollOffGames: activeRollOffGames,
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

	// Get active roll-off games if this is a main game
	var activeRollOffGames []*models.Game
	if !game.Status.IsRollOff() {
		// Get all roll-off games with this game as parent
		rollOffGames, err := s.gameRepo.GetGamesByParent(ctx, &gameRepo.GetGamesByParentInput{
			ParentGameID: game.ID,
		})
		if err == nil && len(rollOffGames) > 0 {
			// Filter for active roll-off games
			for _, rollOffGame := range rollOffGames {
				if rollOffGame.Status == models.GameStatusRollOff {
					activeRollOffGames = append(activeRollOffGames, rollOffGame)
				}
			}
		}
	}

	return &GetGameOutput{
		Game:               game,
		ActiveRollOffGames: activeRollOffGames,
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

	// Get the game
	game, err := s.gameRepo.GetGame(ctx, &gameRepo.GetGameInput{
		GameID: input.GameID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get game: %w", err)
	}

	// Get the session ID from the game's channel
	sessionID := s.getSessionIDForChannel(ctx, game.ChannelID)
	if sessionID == "" {
		return nil, fmt.Errorf("no active session found for channel")
	}

	// Get all drink records for this session
	sessionDrinkRecords, err := s.drinkLedgerRepo.GetDrinkRecordsForSession(ctx, &ledgerRepo.GetDrinkRecordsForSessionInput{
		SessionID: sessionID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get session drink records: %w", err)
	}

	// Find the first unpaid drink for this player
	var drinkRecord *models.DrinkLedger
	for _, record := range sessionDrinkRecords.Records {
		if record.ToPlayerID == input.PlayerID && !record.Paid {
			drinkRecord = record
			break
		}
	}

	// If no unpaid drink found, return an error
	if drinkRecord == nil {
		return nil, fmt.Errorf("no unpaid drinks found for player %s", input.PlayerID)
	}

	// Mark the drink as paid
	err = s.drinkLedgerRepo.MarkDrinkPaid(ctx, &ledgerRepo.MarkDrinkPaidInput{
		DrinkID: drinkRecord.ID,
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
