package game

import (
	"context"
	"errors"

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
func New(ctx context.Context, cfg *Config) (*service, error) {
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

	if cfg.UUID == nil {
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
		uuid:       cfg.UUID,
	}, nil
}

// CreateGame creates a new game session in a Discord channel
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
				ID:       s.uuid.NewUUID(),
				GameID:   gameID,
				PlayerID: input.CreatorID,
				Status:   models.ParticipantStatusWaitingToRoll,
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

	// Ensure there are at least 2 players
	if len(game.Participants) < 2 {
		return nil, errors.New("at least 2 players are required to start a game")
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
			ID:       s.uuid.NewUUID(),
			GameID:   input.GameID,
			PlayerID: input.PlayerID,
			Status:   models.ParticipantStatusWaitingToRoll,
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

	// Find the participant in the game
	var participant *models.Participant
	for _, p := range game.Participants {
		if p.PlayerID == input.PlayerID {
			participant = p
			break
		}
	}

	if participant == nil {
		return nil, ErrPlayerNotInGame
	}

	// Check if the participant has already rolled
	if participant.RollTime != nil {
		return nil, errors.New("player has already rolled in this game")
	}

	// If game is in waiting state, set it to active
	if game.Status == models.GameStatusWaiting {
		game.Status = models.GameStatusActive
		game.UpdatedAt = s.clock.Now()
	}

	// Roll the dice
	rollValue := s.diceRoller.Roll(s.diceSides)
	now := s.clock.Now()

	// Determine if it's a critical hit or fail
	isCriticalHit := rollValue == s.criticalHitValue
	isCriticalFail := rollValue == s.criticalFailValue

	// Update participant's roll information
	participant.RollValue = rollValue
	participant.RollTime = &now
	
	// Update participant status based on roll
	if isCriticalHit {
		participant.Status = models.ParticipantStatusNeedsToAssign
	} else {
		participant.Status = models.ParticipantStatusActive
	}

	// Create a roll record - not saving it for now since we don't have a repository for it
	// TODO: Add a roll repository or add SaveRoll method to game repository
	_ = &models.Roll{
		ID:             s.uuid.NewUUID(),
		Value:          rollValue,
		PlayerID:       input.PlayerID,
		GameID:         input.GameID,
		Timestamp:      now,
		IsCriticalHit:  isCriticalHit,
		IsCriticalFail: isCriticalFail,
		IsLowestRoll:   false, // Will be determined later
	}

	// If it's a critical fail, automatically assign a drink to the player
	if isCriticalFail {
		// Create a drink record for the player (they drink their own)
		drinkRecord := &models.DrinkLedger{
			ID:           s.uuid.NewUUID(),
			GameID:       input.GameID,
			FromPlayerID: input.PlayerID, // Self-assigned on critical fail
			ToPlayerID:   input.PlayerID,
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
	allPlayersRolled := true
	for _, p := range game.Participants {
		if p.RollTime == nil {
			allPlayersRolled = false
			break
		}
	}

	return &RollDiceOutput{
		Value:          rollValue,
		IsCriticalHit:  isCriticalHit,
		IsCriticalFail: isCriticalFail,
		IsLowestRoll:   false, // Will be determined after all players roll
		NeedsRollOff:   false, // Will be determined after all players roll
		AllPlayersRolled: allPlayersRolled,
	}, nil
}

// AssignDrink records that one player has assigned a drink to another
func (s *service) AssignDrink(ctx context.Context, input *AssignDrinkInput) (*AssignDrinkOutput, error) {
	// Get the game
	game, err := s.gameRepo.GetGame(ctx, &gameRepo.GetGameInput{
		GameID: input.GameID,
	})
	if err != nil {
		return nil, ErrGameNotFound
	}

	// Check if game is active
	if game.Status != models.GameStatusActive {
		return nil, ErrInvalidGameState
	}

	// Find the assigning participant in the game
	var assigningParticipant *models.Participant
	for _, p := range game.Participants {
		if p.PlayerID == input.FromPlayerID {
			assigningParticipant = p
			break
		}
	}

	if assigningParticipant == nil {
		return nil, ErrPlayerNotInGame
	}

	// Check if the assigning participant is allowed to assign a drink
	if assigningParticipant.Status != models.ParticipantStatusNeedsToAssign {
		return nil, errors.New("player is not eligible to assign a drink")
	}

	// Find the target participant in the game
	var targetParticipant *models.Participant
	for _, p := range game.Participants {
		if p.PlayerID == input.ToPlayerID {
			targetParticipant = p
			break
		}
	}

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

	return &AssignDrinkOutput{
		Success: true,
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

	// Get all drink records for the game
	drinkRecords, err := s.drinkLedgerRepo.GetDrinkRecordsForGame(ctx, &ledgerRepo.GetDrinkRecordsForGameInput{
		GameID: input.GameID,
	})
	if err != nil {
		return nil, err
	}

	// Create a map to track player stats
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
		Success:         true,
		FinalLeaderboard: playerStats,
	}, nil
}

// HandleRollOff manages roll-offs for tied players
func (s *service) HandleRollOff(ctx context.Context, input *HandleRollOffInput) (*HandleRollOffOutput, error) {
	// TODO: Implement roll-off handling
	return nil, errors.New("not implemented")
}
