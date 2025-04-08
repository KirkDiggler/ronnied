package messaging

import "context"

// Service is the interface for the messaging service
type Service interface {
	// GetJoinGameMessage returns a message for when a player joins a game
	GetJoinGameMessage(ctx context.Context, input *GetJoinGameMessageInput) (*GetJoinGameMessageOutput, error)
	
	// GetJoinGameErrorMessage returns a message for a specific error when joining a game
	GetJoinGameErrorMessage(ctx context.Context, input *GetJoinGameErrorMessageInput) (*GetJoinGameErrorMessageOutput, error)
	
	// GetGameStatusMessage returns a dynamic message based on the game status
	GetGameStatusMessage(ctx context.Context, input *GetGameStatusMessageInput) (*GetGameStatusMessageOutput, error)
	
	// GetRollResultMessage returns a dynamic message for a dice roll result
	GetRollResultMessage(ctx context.Context, input *GetRollResultMessageInput) (*GetRollResultMessageOutput, error)
	
	// GetErrorMessage returns a user-friendly error message
	GetErrorMessage(ctx context.Context, input *GetErrorMessageInput) (*GetErrorMessageOutput, error)
}
