package game

import (
	"context"

	"github.com/KirkDiggler/ronnied/internal/models"
)

// GameOrchestrator defines the interface for all game-related operations
type GameOrchestrator interface {
	// Game Lifecycle - User-facing operations
	CreateGame(ctx context.Context, input *CreateGameInput) (*CreateGameOutput, error)
	StartGame(ctx context.Context, input *StartGameInput) (*StartGameOutput, error)
	JoinGame(ctx context.Context, input *JoinGameInput) (*JoinGameOutput, error)
	EndGame(ctx context.Context, input *EndGameInput) (*EndGameOutput, error)
	AbandonGame(ctx context.Context, input *AbandonGameInput) (*AbandonGameOutput, error)

	// Player Actions - What players can do in a game
	RollDice(ctx context.Context, input *RollDiceInput) (*RollDiceOutput, error)
	AssignDrink(ctx context.Context, input *AssignDrinkInput) (*AssignDrinkOutput, error)
	PayDrink(ctx context.Context, input *PayDrinkInput) (*PayDrinkOutput, error)

	// Game Queries - Getting information about games
	GetGame(ctx context.Context, input *GetGameInput) (*GetGameOutput, error)
	GetGameByChannel(ctx context.Context, input *GetGameByChannelInput) (*GetGameByChannelOutput, error)

	// Game Management - Administrative operations
	UpdateGameMessage(ctx context.Context, input *UpdateGameMessageInput) (*UpdateGameMessageOutput, error)
}

// CreateGameInput represents the input for creating a new game
type CreateGameInput struct {
	ChannelID  string
	PlayerID   string
	PlayerName string
}

// CreateGameOutput represents the output from creating a new game
type CreateGameOutput struct {
	Game *models.Game
}

// StartGameInput represents the input for starting a game
type StartGameInput struct {
	GameID string
}

// StartGameOutput represents the output from starting a game
type StartGameOutput struct {
	Game *models.Game
}

// JoinGameInput represents the input for joining a game
type JoinGameInput struct {
	GameID     string
	PlayerID   string
	PlayerName string
}

// JoinGameOutput represents the output from joining a game
type JoinGameOutput struct {
	Game *models.Game
}

// RollDiceInput represents the input for rolling dice
type RollDiceInput struct {
	GameID   string
	PlayerID string
}

// RollDiceOutput represents the output from rolling dice
type RollDiceOutput struct {
	Game            *models.Game
	PlayerID        string
	RollValue       int
	Message         string
	DetailedMessage string
	IsRollOffRoll   bool
	Options         []PlayerOption
}

// PlayerOption represents an option available to a player
type PlayerOption struct {
	PlayerID   string
	PlayerName string
}

// AssignDrinkInput represents the input for assigning a drink
type AssignDrinkInput struct {
	GameID         string
	AssignerID     string
	RecipientID    string
	AssignerRollID string
}

// AssignDrinkOutput represents the output from assigning a drink
type AssignDrinkOutput struct {
	Game *models.Game
}

// PayDrinkInput represents the input for paying a drink
type PayDrinkInput struct {
	GameID      string
	PlayerID    string
	RecipientID string
}

// PayDrinkOutput represents the output from paying a drink
type PayDrinkOutput struct {
	Game *models.Game
}

// EndGameInput represents the input for ending a game
type EndGameInput struct {
	GameID string
}

// EndGameOutput represents the output from ending a game
type EndGameOutput struct {
	Game *models.Game
}

// AbandonGameInput represents the input for abandoning a game
type AbandonGameInput struct {
	GameID string
}

// AbandonGameOutput represents the output from abandoning a game
type AbandonGameOutput struct {
	Game *models.Game
}

// GetGameInput represents the input for getting a game
type GetGameInput struct {
	GameID string
}

// GetGameOutput represents the output from getting a game
type GetGameOutput struct {
	Game *models.Game
}

// GetGameByChannelInput represents the input for getting a game by channel
type GetGameByChannelInput struct {
	ChannelID string
}

// GetGameByChannelOutput represents the output from getting a game by channel
type GetGameByChannelOutput struct {
	Game *models.Game
}

// UpdateGameMessageInput represents the input for updating a game message
type UpdateGameMessageInput struct {
	GameID    string
	MessageID string
}

// UpdateGameMessageOutput represents the output from updating a game message
type UpdateGameMessageOutput struct {
	Game *models.Game
}

