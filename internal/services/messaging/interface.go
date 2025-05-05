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
	
	// GetGameStartedMessage returns a dynamic message for when a game is started
	GetGameStartedMessage(ctx context.Context, input *GetGameStartedMessageInput) (*GetGameStartedMessageOutput, error)
	
	// GetErrorMessage returns a user-friendly error message
	GetErrorMessage(ctx context.Context, input *GetErrorMessageInput) (*GetErrorMessageOutput, error)
	
	// GetRollWhisperMessage returns a supportive whisper message after a roll
	GetRollWhisperMessage(ctx context.Context, input *GetRollWhisperMessageInput) (*GetRollWhisperMessageOutput, error)

	// GetLeaderboardMessage returns a funny message for a player in the leaderboard
	GetLeaderboardMessage(ctx context.Context, input *GetLeaderboardMessageInput) (*GetLeaderboardMessageOutput, error)

	// GetPayDrinkMessage returns a fun message when a player pays a drink
	GetPayDrinkMessage(ctx context.Context, input *GetPayDrinkMessageInput) (*GetPayDrinkMessageOutput, error)
	
	// GetRollComment returns a comment for a roll in the shared game message
	GetRollComment(ctx context.Context, input *GetRollCommentInput) (*GetRollCommentOutput, error)
	
	// GetDrinkAssignmentMessage returns a message for a drink assignment in the shared game message
	GetDrinkAssignmentMessage(ctx context.Context, input *GetDrinkAssignmentMessageInput) (*GetDrinkAssignmentMessageOutput, error)
}
