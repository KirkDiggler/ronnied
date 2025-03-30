package game

import (
	"context"
	"errors"
	"time"
	"github.com/google/uuid"

	"github.com/KirkDiggler/ronnied/internal/dice"
	"github.com/KirkDiggler/ronnied/internal/models"
	gameRepo "github.com/KirkDiggler/ronnied/internal/repositories/game"
	playerRepo "github.com/KirkDiggler/ronnied/internal/repositories/player"
	ledgerRepo "github.com/KirkDiggler/ronnied/internal/repositories/drink_ledger"
)

// Define errors
var (
	ErrGameNotFound      = errors.New("game not found")
	ErrPlayerNotFound    = errors.New("player not found")
	ErrPlayerAlreadyInGame = errors.New("player already in game")
	ErrGameAlreadyExists = errors.New("game already exists for this channel")
	ErrInvalidGameState  = errors.New("invalid game state")
	ErrPlayerNotInGame   = errors.New("player not in game")
	ErrGameFull          = errors.New("game is at maximum capacity")
)

// service implements the Service interface
type service struct {
	config              *Config
	gameRepo            gameRepo.Repository
	playerRepo          playerRepo.Repository
	drinkLedgerRepo     ledgerRepo.Repository
	diceRoller          *dice.Roller
}

// NewService creates a new game service
func NewService(ctx context.Context, cfg *Config, gameRepository gameRepo.Repository, playerRepository playerRepo.Repository, drinkLedgerRepository ledgerRepo.Repository, diceRoller *dice.Roller) (*service, error) {
	// Set default values if not provided
	if cfg == nil {
		cfg = &Config{
			MaxPlayers:        10,
			DiceSides:         6,
			CriticalHitValue:  6,
			CriticalFailValue: 1,
			MaxConcurrentGames: 100,
		}
	}
	
	return &service{
		config:          cfg,
		gameRepo:        gameRepository,
		playerRepo:      playerRepository,
		drinkLedgerRepo: drinkLedgerRepository,
		diceRoller:      diceRoller,
	}, nil
}

// CreateGame creates a new game session in a Discord channel
func (s *service) CreateGame(ctx context.Context, input *CreateGameInput) (*CreateGameOutput, error) {
	// Check if a game already exists for this channel
	existingGame, err := s.gameRepo.GetGameByChannel(ctx, &gameRepo.GetGameByChannelInput{
		ChannelID: input.ChannelID,
	})
	
	// If there's no error, a game exists
	if err == nil && existingGame != nil {
		return nil, ErrGameAlreadyExists
	}
	
	// Only proceed if the error is "not found"
	if err != nil && !errors.Is(err, ErrGameNotFound) {
		return nil, err
	}
	
	// Create a new game
	gameID := uuid.New().String()
	now := time.Now()
	
	game := &models.Game{
		ID:        gameID,
		ChannelID: input.ChannelID,
		Status:    models.GameStatusWaiting,
		PlayerIDs: []string{},
		CreatedAt: now,
		UpdatedAt: now,
	}
	
	// Save the game
	err = s.gameRepo.SaveGame(ctx, &gameRepo.SaveGameInput{
		Game: game,
	})
	if err != nil {
		return nil, err
	}
	
	return &CreateGameOutput{
		GameID: gameID,
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
	if len(game.PlayerIDs) >= s.config.MaxPlayers {
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
		now := time.Now()
		player := &models.Player{
			ID:           input.PlayerID,
			Name:         input.PlayerName,
			CurrentGameID: input.GameID,
			LastRoll:     0,
			LastRollTime: now,
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
	for _, pid := range game.PlayerIDs {
		if pid == input.PlayerID {
			playerAlreadyInGame = true
			break
		}
	}
	
	if !playerAlreadyInGame {
		game.PlayerIDs = append(game.PlayerIDs, input.PlayerID)
		game.UpdatedAt = time.Now()
		
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

// LeaveGame removes a player from a game
func (s *service) LeaveGame(ctx context.Context, input *LeaveGameInput) (*LeaveGameOutput, error) {
	// Get the game
	game, err := s.gameRepo.GetGame(ctx, &gameRepo.GetGameInput{
		GameID: input.GameID,
	})
	if err != nil {
		return nil, ErrGameNotFound
	}
	
	// Get the player
	player, err := s.playerRepo.GetPlayer(ctx, &playerRepo.GetPlayerInput{
		PlayerID: input.PlayerID,
	})
	if err != nil {
		return nil, ErrPlayerNotFound
	}
	
	// Check if player is in the game
	if player.CurrentGameID != input.GameID {
		return nil, ErrPlayerNotInGame
	}
	
	// Remove player from game
	updatedPlayerIDs := make([]string, 0, len(game.PlayerIDs))
	playerFound := false
	
	for _, pid := range game.PlayerIDs {
		if pid != input.PlayerID {
			updatedPlayerIDs = append(updatedPlayerIDs, pid)
		} else {
			playerFound = true
		}
	}
	
	if !playerFound {
		return nil, ErrPlayerNotInGame
	}
	
	// Update player's current game
	player.CurrentGameID = ""
	err = s.playerRepo.SavePlayer(ctx, &playerRepo.SavePlayerInput{
		Player: player,
	})
	if err != nil {
		return nil, err
	}
	
	// Update game
	game.PlayerIDs = updatedPlayerIDs
	game.UpdatedAt = time.Now()
	
	err = s.gameRepo.SaveGame(ctx, &gameRepo.SaveGameInput{
		Game: game,
	})
	if err != nil {
		return nil, err
	}
	
	return &LeaveGameOutput{
		Success: true,
	}, nil
}

// RollDice performs a dice roll for a player
func (s *service) RollDice(ctx context.Context, input *RollDiceInput) (*RollDiceOutput, error) {
	// Get the game
	game, err := s.gameRepo.GetGame(ctx, &gameRepo.GetGameInput{
		GameID: input.GameID,
	})
	if err != nil {
		return nil, ErrGameNotFound
	}
	
	// Check if game is active or waiting
	if game.Status != models.GameStatusActive && game.Status != models.GameStatusWaiting {
		return nil, ErrInvalidGameState
	}
	
	// Get the player
	player, err := s.playerRepo.GetPlayer(ctx, &playerRepo.GetPlayerInput{
		PlayerID: input.PlayerID,
	})
	if err != nil {
		return nil, ErrPlayerNotFound
	}
	
	// Check if player is in the game
	if player.CurrentGameID != input.GameID {
		return nil, ErrPlayerNotInGame
	}
	
	// If game is in waiting state, set it to active
	if game.Status == models.GameStatusWaiting {
		game.Status = models.GameStatusActive
	}
	
	// Roll the dice
	rollValue := s.diceRoller.Roll(s.config.DiceSides)
	now := time.Now()
	
	// Determine if it's a critical hit or fail
	isCriticalHit := rollValue == s.config.CriticalHitValue
	isCriticalFail := rollValue == s.config.CriticalFailValue
	
	// Update player's last roll
	player.LastRoll = rollValue
	player.LastRollTime = now
	
	err = s.playerRepo.SavePlayer(ctx, &playerRepo.SavePlayerInput{
		Player: player,
	})
	if err != nil {
		return nil, err
	}
	
	// Create a roll record - not saving it for now since we don't have a repository for it
	// TODO: Add a roll repository or add SaveRoll method to game repository
	_ = &models.Roll{
		ID:            uuid.New().String(),
		Value:         rollValue,
		PlayerID:      input.PlayerID,
		GameID:        input.GameID,
		Timestamp:     now,
		IsCriticalHit: isCriticalHit,
		IsCriticalFail: isCriticalFail,
		IsLowestRoll:  false, // Will be determined later
	}
	
	// If it's a critical fail, automatically assign a drink to the player
	if isCriticalFail {
		// Create a drink record for the player (they drink their own)
		drinkID := uuid.New().String()
		drinkRecord := &models.DrinkLedger{
			ID:           drinkID,
			FromPlayerID: input.PlayerID, // Self-assigned
			ToPlayerID:   input.PlayerID,
			GameID:       input.GameID,
			Reason:       models.DrinkReasonCriticalFail,
			Timestamp:    now,
			Paid:         false,
		}
		
		err = s.drinkLedgerRepo.AddDrinkRecord(ctx, &ledgerRepo.AddDrinkRecordInput{
			Record: drinkRecord,
		})
		if err != nil {
			return nil, err
		}
	}
	
	// Update the game's timestamp
	game.UpdatedAt = now
	err = s.gameRepo.SaveGame(ctx, &gameRepo.SaveGameInput{
		Game: game,
	})
	if err != nil {
		return nil, err
	}
	
	// Check if all players have rolled
	// This would involve getting all players and checking their last roll times
	// For now, we'll just return the roll result
	
	return &RollDiceOutput{
		Value:         rollValue,
		IsCriticalHit: isCriticalHit,
		IsCriticalFail: isCriticalFail,
		IsLowestRoll:  false, // Will be determined after all players roll
		NeedsRollOff:  false, // Will be determined after all players roll
	}, nil
}

// AssignDrink records that one player has assigned a drink to another
func (s *service) AssignDrink(ctx context.Context, input *AssignDrinkInput) (*AssignDrinkOutput, error) {
	// TODO: Implement drink assignment
	return nil, errors.New("not implemented")
}

// GetLeaderboard returns the current standings for a game
func (s *service) GetLeaderboard(ctx context.Context, input *GetLeaderboardInput) (*GetLeaderboardOutput, error) {
	// TODO: Implement leaderboard generation
	return nil, errors.New("not implemented")
}

// ResetGame clears all drink assignments in a game
func (s *service) ResetGame(ctx context.Context, input *ResetGameInput) (*ResetGameOutput, error) {
	// TODO: Implement game reset
	return nil, errors.New("not implemented")
}

// EndGame concludes a game session
func (s *service) EndGame(ctx context.Context, input *EndGameInput) (*EndGameOutput, error) {
	// TODO: Implement game ending
	return nil, errors.New("not implemented")
}

// HandleRollOff manages roll-offs for tied players
func (s *service) HandleRollOff(ctx context.Context, input *HandleRollOffInput) (*HandleRollOffOutput, error) {
	// TODO: Implement roll-off handling
	return nil, errors.New("not implemented")
}
