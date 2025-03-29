package game

import "context"

// Service defines the interface for game operations
type Service interface {
	// CreateGame creates a new game session in a Discord channel
	CreateGame(ctx context.Context, input *CreateGameInput) (*CreateGameOutput, error)
	
	// JoinGame adds a player to an existing game
	JoinGame(ctx context.Context, input *JoinGameInput) (*JoinGameOutput, error)
	
	// LeaveGame removes a player from a game
	LeaveGame(ctx context.Context, input *LeaveGameInput) (*LeaveGameOutput, error)
	
	// RollDice performs a dice roll for a player
	RollDice(ctx context.Context, input *RollDiceInput) (*RollDiceOutput, error)
	
	// AssignDrink records that one player has assigned a drink to another
	AssignDrink(ctx context.Context, input *AssignDrinkInput) (*AssignDrinkOutput, error)
	
	// GetLeaderboard returns the current standings for a game
	GetLeaderboard(ctx context.Context, input *GetLeaderboardInput) (*GetLeaderboardOutput, error)
	
	// ResetGame clears all drink assignments in a game
	ResetGame(ctx context.Context, input *ResetGameInput) (*ResetGameOutput, error)
	
	// EndGame concludes a game session
	EndGame(ctx context.Context, input *EndGameInput) (*EndGameOutput, error)
	
	// HandleRollOff manages roll-offs for tied players
	HandleRollOff(ctx context.Context, input *HandleRollOffInput) (*HandleRollOffOutput, error)
}
