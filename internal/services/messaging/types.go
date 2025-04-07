package messaging

import "github.com/KirkDiggler/ronnied/internal/models"

// MessageType represents different categories of messages
type MessageType string

const (
	// MessageTypeJoinGame represents messages when a player joins a game
	MessageTypeJoinGame MessageType = "join_game"
	
	// MessageTypeGameStatus represents messages about the game status
	MessageTypeGameStatus MessageType = "game_status"
	
	// MessageTypeRollResult represents messages about dice roll results
	MessageTypeRollResult MessageType = "roll_result"
	
	// MessageTypeError represents error messages
	MessageTypeError MessageType = "error"
)

// MessageTone represents the tone of a message
type MessageTone string

const (
	// ToneNeutral is a neutral tone
	ToneNeutral MessageTone = "neutral"
	
	// ToneFunny is a humorous tone
	ToneFunny MessageTone = "funny"
	
	// ToneSarcastic is a sarcastic tone
	ToneSarcastic MessageTone = "sarcastic"
	
	// ToneEncouraging is an encouraging tone
	ToneEncouraging MessageTone = "encouraging"
	
	// ToneCelebration is a celebratory tone
	ToneCelebration MessageTone = "celebration"
)

// GetJoinGameMessageInput contains parameters for getting a join game message
type GetJoinGameMessageInput struct {
	// PlayerName is the name of the player joining
	PlayerName string
	
	// GameStatus is the current status of the game
	GameStatus models.GameStatus
	
	// AlreadyJoined indicates if the player was already in the game
	AlreadyJoined bool
	
	// PreferredTone is the preferred tone for the message (optional)
	PreferredTone MessageTone
}

// GetJoinGameMessageOutput contains the result of getting a join game message
type GetJoinGameMessageOutput struct {
	// Message is the generated message
	Message string
	
	// Tone is the tone of the message
	Tone MessageTone
}

// GetGameStatusMessageInput contains parameters for getting a game status message
type GetGameStatusMessageInput struct {
	// GameStatus is the current status of the game
	GameStatus models.GameStatus
	
	// ParticipantCount is the number of participants in the game
	ParticipantCount int
	
	// PreferredTone is the preferred tone for the message (optional)
	PreferredTone MessageTone
}

// GetGameStatusMessageOutput contains the result of getting a game status message
type GetGameStatusMessageOutput struct {
	// Message is the generated message
	Message string
	
	// Tone is the tone of the message
	Tone MessageTone
}

// GetRollResultMessageInput contains parameters for getting a roll result message
type GetRollResultMessageInput struct {
	// PlayerName is the name of the player who rolled
	PlayerName string
	
	// RollValue is the value of the roll
	RollValue int
	
	// IsCriticalHit indicates if the roll was a critical hit (6)
	IsCriticalHit bool
	
	// IsCriticalFail indicates if the roll was a critical fail (1)
	IsCriticalFail bool
	
	// PreferredTone is the preferred tone for the message (optional)
	PreferredTone MessageTone
}

// GetRollResultMessageOutput contains the result of getting a roll result message
type GetRollResultMessageOutput struct {
	// Message is the generated message
	Message string
	
	// Tone is the tone of the message
	Tone MessageTone
}

// GetErrorMessageInput contains parameters for getting an error message
type GetErrorMessageInput struct {
	// ErrorType is the type of error
	ErrorType string
	
	// PreferredTone is the preferred tone for the message (optional)
	PreferredTone MessageTone
}

// GetErrorMessageOutput contains the result of getting an error message
type GetErrorMessageOutput struct {
	// Message is the generated message
	Message string
	
	// Tone is the tone of the message
	Tone MessageTone
}

// ServiceConfig contains configuration for the messaging service
type ServiceConfig struct {
	// Repository is the repository for storing and retrieving messages
	// This is commented out for now, but can be uncommented when we add a repository
	// Repository Repository
}
