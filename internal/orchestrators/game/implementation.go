package game

import (
	"context"
	"errors"
	"fmt"

	"github.com/KirkDiggler/ronnied/internal/common/clock"
	"github.com/KirkDiggler/ronnied/internal/common/uuid"
	"github.com/KirkDiggler/ronnied/internal/dice"
	"github.com/KirkDiggler/ronnied/internal/models"
	ledgerRepo "github.com/KirkDiggler/ronnied/internal/repositories/drink_ledger"
	gameRepo "github.com/KirkDiggler/ronnied/internal/repositories/game"
	playerRepo "github.com/KirkDiggler/ronnied/internal/repositories/player"
)

// Common errors
var (
	ErrNilConfig          = errors.New("config cannot be nil")
	ErrNilGameRepo        = errors.New("game repository cannot be nil")
	ErrNilPlayerRepo      = errors.New("player repository cannot be nil")
	ErrNilDrinkLedgerRepo = errors.New("drink ledger repository cannot be nil")
	ErrNilDiceRoller      = errors.New("dice roller cannot be nil")
	ErrNilClock           = errors.New("clock cannot be nil")
	ErrNilUUIDGenerator   = errors.New("UUID generator cannot be nil")
	ErrInvalidGameState   = errors.New("invalid game state for this operation")
	ErrPlayerNotInGame    = errors.New("player is not in this game")
	ErrPlayerNotInRollOff = errors.New("player is not in the roll-off")
)

const (
	defaultMaxPlayers         = 10
	defaultDiceSides          = 6
	defaultCriticalHitValue   = 6
	defaultCriticalFailValue  = 1
	defaultMaxConcurrentGames = 5
)

// orchestrator implements the GameOrchestrator interface
type orchestrator struct {
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
	diceRoller      dice.Roller
	clock           clock.Clock
	uuid            uuid.Generator
}

// Config holds the configuration for the game orchestrator
type Config struct {
	// Configuration parameters
	MaxPlayers         int
	DiceSides          int
	CriticalHitValue   int
	CriticalFailValue  int
	MaxConcurrentGames int

	// Repository dependencies
	GameRepo        gameRepo.Repository
	PlayerRepo      playerRepo.Repository
	DrinkLedgerRepo ledgerRepo.Repository
	DiceRoller      dice.Roller
	Clock           clock.Clock
	UUIDGenerator   uuid.Generator
}

// NewOrchestrator creates a new game orchestrator
func NewOrchestrator(config *Config) (*orchestrator, error) {
	// Validate config
	if config == nil {
		return nil, ErrNilConfig
	}

	if config.GameRepo == nil {
		return nil, ErrNilGameRepo
	}

	// Set default values
	if config.MaxPlayers <= 0 {
		config.MaxPlayers = defaultMaxPlayers
	}

	if config.DiceSides <= 0 {
		config.DiceSides = defaultDiceSides
	}

	if config.CriticalHitValue <= 0 {
		config.CriticalHitValue = config.DiceSides // Default to highest possible roll
	}

	if config.CriticalFailValue <= 0 {
		config.CriticalFailValue = 1 // Default to lowest possible roll
	}

	if config.MaxConcurrentGames <= 0 {
		config.MaxConcurrentGames = defaultMaxConcurrentGames
	}

	// Create and return the orchestrator
	return &orchestrator{
		maxPlayers:         config.MaxPlayers,
		diceSides:          config.DiceSides,
		criticalHitValue:   config.CriticalHitValue,
		criticalFailValue:  config.CriticalFailValue,
		maxConcurrentGames: config.MaxConcurrentGames,
		gameRepo:           config.GameRepo,
		playerRepo:         config.PlayerRepo,
		drinkLedgerRepo:    config.DrinkLedgerRepo,
		diceRoller:         config.DiceRoller,
		clock:              config.Clock,
		uuid:               config.UUIDGenerator,
	}, nil
}

// RollDice handles a player rolling dice in a game
// This is a key method that will internally handle roll-offs
func (o *orchestrator) RollDice(ctx context.Context, input *RollDiceInput) (*RollDiceOutput, error) {
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
	game, err := o.gameRepo.GetGame(ctx, &gameRepo.GetGameInput{
		GameID: input.GameID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get game: %w", err)
	}

	// Check if the player is in the game
	participant := o.findParticipant(game, input.PlayerID)
	if participant == nil {
		return nil, ErrPlayerNotInGame
	}

	// Check if the player is in a roll-off
	rollOffGame, isInRollOff := o.checkPlayerInRollOff(game, input.PlayerID)

	if isInRollOff {
		// Process roll in roll-off
		return o.processRollOffRoll(ctx, input, rollOffGame)
	}

	// Process roll in main game
	return o.processMainGameRoll(ctx, input, game)
}

// findParticipant finds a participant in a game by player ID
// Returns nil if the player is not in the game
func (o *orchestrator) findParticipant(game *models.Game, playerID string) *models.Participant {
	if game == nil || playerID == "" {
		return nil
	}

	for _, participant := range game.Participants {
		if participant.PlayerID == playerID {
			return participant
		}
	}

	return nil
}

// checkPlayerInRollOff determines if a player is in a roll-off
// This is an internal helper method, not exposed in the interface
func (o *orchestrator) checkPlayerInRollOff(game *models.Game, playerID string) (*models.Game, bool) {
	// note: we need to load all games by parentid, we can iterate through that list and call a helper func for playerInfo or should roll.
	// maybe we can return the paerticipant item and easy check the status

	// Check if the game is in a roll-off state
	if game.Status == models.GameStatusRollOff {
		// Check if the player is part of the roll-off
		for _, rollOffPlayerID := range game.RollOffPlayerIDs {
			if rollOffPlayerID == playerID {
				return game, true
			}
		}
	}

	// Player is not in a roll-off
	return nil, false
}

// processMainGameRoll handles a dice roll in the main game
func (o *orchestrator) processMainGameRoll(ctx context.Context, input *RollDiceInput, game *models.Game) (*RollDiceOutput, error) {
	// Find the participant
	participant := o.findParticipant(game, input.PlayerID)
	if participant == nil {
		return nil, ErrPlayerNotInGame
	}

	// Check if the game is in a valid state for rolling
	if game.Status != models.GameStatusActive {
		return nil, ErrInvalidGameState
	}

	// Roll the dice
	rollValue := o.diceRoller.Roll(o.diceSides)

	// Update the participant's roll information
	participant.RollValue = rollValue
	currentTime := o.clock.Now()
	participant.RollTime = &currentTime

	// Determine if this is a critical hit or fail
	isCriticalHit := rollValue == o.criticalHitValue
	isCriticalFail := rollValue == o.criticalFailValue

	// Create appropriate messages
	message := fmt.Sprintf("You rolled a %d!", rollValue)
	detailedMessage := message

	if isCriticalHit {
		detailedMessage += " Critical Hit! You can assign a drink to another player."
	} else if isCriticalFail {
		detailedMessage += " Critical Fail! You must drink."
	}

	// Save the updated game
	saveErr := o.gameRepo.SaveGame(ctx, &gameRepo.SaveGameInput{
		Game: game,
	})
	if saveErr != nil {
		return nil, fmt.Errorf("failed to save game: %w", saveErr)
	}

	// Return the result
	return &RollDiceOutput{
		Game:            game,
		PlayerID:        input.PlayerID,
		RollValue:       rollValue,
		Message:         message,
		DetailedMessage: detailedMessage,
		IsRollOffRoll:   false,
	}, nil
}

// processRollOffRoll handles a dice roll in a roll-off
func (o *orchestrator) processRollOffRoll(ctx context.Context, input *RollDiceInput, game *models.Game) (*RollDiceOutput, error) {
	// Find the participant
	participant := o.findParticipant(game, input.PlayerID)
	if participant == nil {
		return nil, ErrPlayerNotInGame
	}

	// Check if the game is in a valid state for a roll-off
	if game.Status != models.GameStatusRollOff {
		return nil, ErrInvalidGameState
	}

	// Check if the player is part of the roll-off
	isInRollOff := false
	for _, rollOffPlayerID := range game.RollOffPlayerIDs {
		if rollOffPlayerID == input.PlayerID {
			isInRollOff = true
			break
		}
	}

	if !isInRollOff {
		return nil, ErrPlayerNotInRollOff
	}

	// Check if the participant has already rolled
	if participant.Status == models.ParticipantStatusRolledInRollOff {
		return nil, fmt.Errorf("player has already rolled in this roll-off")
	}

	// Roll the dice
	rollValue := o.diceRoller.Roll(o.diceSides)

	// Update the participant's roll information
	participant.RollValue = rollValue
	currentTime := o.clock.Now()
	participant.RollTime = &currentTime
	participant.Status = models.ParticipantStatusRolledInRollOff

	// Create appropriate messages
	message := fmt.Sprintf("You rolled a %d in the roll-off!", rollValue)
	detailedMessage := message

	// Check if all players have rolled
	allPlayersRolled := true
	for _, p := range game.Participants {
		for _, rollOffPlayerID := range game.RollOffPlayerIDs {
			if p.PlayerID == rollOffPlayerID && p.Status != models.ParticipantStatusRolledInRollOff {
				allPlayersRolled = false
				break
			}
		}
	}

	if allPlayersRolled {
		detailedMessage += "\n\nAll players have rolled in this roll-off."
	}

	// Save the updated game
	saveErr := o.gameRepo.SaveGame(ctx, &gameRepo.SaveGameInput{
		Game: game,
	})
	if saveErr != nil {
		return nil, fmt.Errorf("failed to save game: %w", saveErr)
	}

	// Return the result
	return &RollDiceOutput{
		Game:            game,
		PlayerID:        input.PlayerID,
		RollValue:       rollValue,
		Message:         message,
		DetailedMessage: detailedMessage,
		IsRollOffRoll:   true,
	}, nil
}

// Placeholder implementations for other interface methods
// These will be filled in as we develop the orchestrator

func (o *orchestrator) CreateGame(ctx context.Context, input *CreateGameInput) (*CreateGameOutput, error) {
	return nil, errors.New("not implemented")
}

func (o *orchestrator) StartGame(ctx context.Context, input *StartGameInput) (*StartGameOutput, error) {
	return nil, errors.New("not implemented")
}

func (o *orchestrator) JoinGame(ctx context.Context, input *JoinGameInput) (*JoinGameOutput, error) {
	return nil, errors.New("not implemented")
}

func (o *orchestrator) EndGame(ctx context.Context, input *EndGameInput) (*EndGameOutput, error) {
	return nil, errors.New("not implemented")
}

func (o *orchestrator) AbandonGame(ctx context.Context, input *AbandonGameInput) (*AbandonGameOutput, error) {
	return nil, errors.New("not implemented")
}

func (o *orchestrator) AssignDrink(ctx context.Context, input *AssignDrinkInput) (*AssignDrinkOutput, error) {
	return nil, errors.New("not implemented")
}

func (o *orchestrator) PayDrink(ctx context.Context, input *PayDrinkInput) (*PayDrinkOutput, error) {
	return nil, errors.New("not implemented")
}

func (o *orchestrator) GetGame(ctx context.Context, input *GetGameInput) (*GetGameOutput, error) {
	return nil, errors.New("not implemented")
}

func (o *orchestrator) GetGameByChannel(ctx context.Context, input *GetGameByChannelInput) (*GetGameByChannelOutput, error) {
	return nil, errors.New("not implemented")
}

func (o *orchestrator) UpdateGameMessage(ctx context.Context, input *UpdateGameMessageInput) (*UpdateGameMessageOutput, error) {
	return nil, errors.New("not implemented")
}
