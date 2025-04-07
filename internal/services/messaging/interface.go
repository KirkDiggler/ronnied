package messaging

import "context"

// Service defines the interface for the messaging service
type Service interface {
	// GetJoinGameMessage returns a message for when a player joins a game
	GetJoinGameMessage(ctx context.Context, input *GetJoinGameMessageInput) (*GetJoinGameMessageOutput, error)
	
	// GetGameStatusMessage returns a message describing the current game status
	GetGameStatusMessage(ctx context.Context, input *GetGameStatusMessageInput) (*GetGameStatusMessageOutput, error)
	
	// GetRollResultMessage returns a message for a dice roll result
	GetRollResultMessage(ctx context.Context, input *GetRollResultMessageInput) (*GetRollResultMessageOutput, error)
	
	// GetErrorMessage returns a user-friendly error message
	GetErrorMessage(ctx context.Context, input *GetErrorMessageInput) (*GetErrorMessageOutput, error)
}
