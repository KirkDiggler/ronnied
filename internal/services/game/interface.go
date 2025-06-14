package game

import (
	"context"
)

// Service defines the interface for game operations
type Service interface {
	// CreateGame creates a new game session in a Discord channel
	CreateGame(ctx context.Context, input *CreateGameInput) (*CreateGameOutput, error)

	// JoinGame adds a player to an existing game
	JoinGame(ctx context.Context, input *JoinGameInput) (*JoinGameOutput, error)

	// StartGame transitions a game from waiting to active state
	StartGame(ctx context.Context, input *StartGameInput) (*StartGameOutput, error)

	// RollDice performs a dice roll for a player
	RollDice(ctx context.Context, input *RollDiceInput) (*RollDiceOutput, error)

	// AssignDrink records that one player has assigned a drink to another
	AssignDrink(ctx context.Context, input *AssignDrinkInput) (*AssignDrinkOutput, error)

	// EndGame concludes a game session
	EndGame(ctx context.Context, input *EndGameInput) (*EndGameOutput, error)

	// StartRollOff initiates a roll-off within a game
	StartRollOff(ctx context.Context, input *StartRollOffInput) (*StartRollOffOutput, error)

	// CompleteRollOff finalizes a roll-off and processes the results
	CompleteRollOff(ctx context.Context, input *CompleteRollOffInput) (*CompleteRollOffOutput, error)

	// GetGameByChannel retrieves a game by its Discord channel ID
	GetGameByChannel(ctx context.Context, input *GetGameByChannelInput) (*GetGameByChannelOutput, error)

	// GetGame retrieves a game by its ID
	GetGame(ctx context.Context, input *GetGameInput) (*GetGameOutput, error)

	// GetLeaderboard retrieves the leaderboard for a game
	GetLeaderboard(ctx context.Context, input *GetLeaderboardInput) (*GetLeaderboardOutput, error)

	// AbandonGame forcefully abandons a game regardless of its state
	AbandonGame(ctx context.Context, input *AbandonGameInput) (*AbandonGameOutput, error)

	// UpdateGameMessage updates the Discord message ID associated with a game
	UpdateGameMessage(ctx context.Context, input *UpdateGameMessageInput) (*UpdateGameMessageOutput, error)

	// GetDrinkRecords retrieves all drink records for a game
	GetDrinkRecords(ctx context.Context, input *GetDrinkRecordsInput) (*GetDrinkRecordsOutput, error)

	// GetPlayerTab retrieves a player's current tab (drinks owed and received)
	GetPlayerTab(ctx context.Context, input *GetPlayerTabInput) (*GetPlayerTabOutput, error)

	// ResetGameTab resets the drink ledger for a game and returns the previous leaderboard
	ResetGameTab(ctx context.Context, input *ResetGameTabInput) (*ResetGameTabOutput, error)

	// PayDrink marks a drink as paid
	PayDrink(ctx context.Context, input *PayDrinkInput) (*PayDrinkOutput, error)

	// CreateSession creates a new drinking session for a channel
	CreateSession(ctx context.Context, input *CreateSessionInput) (*CreateSessionOutput, error)

	// GetSessionLeaderboard retrieves the leaderboard for the current session
	GetSessionLeaderboard(ctx context.Context, input *GetSessionLeaderboardInput) (*GetSessionLeaderboardOutput, error)

	// StartNewSession creates a new drinking session for a channel (alias for CreateSession with a clearer name)
	StartNewSession(ctx context.Context, input *StartNewSessionInput) (*StartNewSessionOutput, error)

	// CanPlayerRoll checks if a player is eligible to roll dice in the current game state
	CanPlayerRoll(ctx context.Context, input *CanPlayerRollInput) (*CanPlayerRollOutput, error)

	// CanPlayerJoinGame checks if a player can join a specific game
	CanPlayerJoinGame(ctx context.Context, input *CanPlayerJoinGameInput) (*CanPlayerJoinGameOutput, error)

	// CanPlayerStartGame checks if a player can start a specific game
	CanPlayerStartGame(ctx context.Context, input *CanPlayerStartGameInput) (*CanPlayerStartGameOutput, error)

	// GetPlayerNamesForGame gets a map of player IDs to names for a game
	GetPlayerNamesForGame(ctx context.Context, input *GetPlayerNamesForGameInput) (*GetPlayerNamesForGameOutput, error)

	// GetActiveRollOffForPlayer finds the active roll-off game ID for a player if any
	GetActiveRollOffForPlayer(ctx context.Context, input *GetActiveRollOffForPlayerInput) (*GetActiveRollOffForPlayerOutput, error)
}
