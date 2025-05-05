package messaging

import (
	"github.com/KirkDiggler/ronnied/internal/models"
)

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

// GetJoinGameErrorMessageInput is the input for GetJoinGameErrorMessage
type GetJoinGameErrorMessageInput struct {
	PlayerName string
	ErrorType  string
	Tone       MessageTone
}

// GetJoinGameErrorMessageOutput is the output for GetJoinGameErrorMessage
type GetJoinGameErrorMessageOutput struct {
	Title   string
	Message string
}

// GetGameStatusMessageInput is the input for GetGameStatusMessage
type GetGameStatusMessageInput struct {
	GameStatus       models.GameStatus
	ParticipantCount int
	Tone             MessageTone
}

// GetGameStatusMessageOutput is the output for GetGameStatusMessage
type GetGameStatusMessageOutput struct {
	Message string
}

// GetRollResultMessageInput contains the input for GetRollResultMessage
type GetRollResultMessageInput struct {
	PlayerName       string
	RollValue        int
	IsCriticalHit    bool
	IsCriticalFail   bool
	IsPersonalMessage bool // Indicates if this is a personal/ephemeral message to the player
}

// GetRollResultMessageOutput contains the output for GetRollResultMessage
type GetRollResultMessageOutput struct {
	Title   string
	Message string
}

// GetGameStartedMessageInput contains the input for GetGameStartedMessage
type GetGameStartedMessageInput struct {
	CreatorName string
	PlayerCount int
}

// GetGameStartedMessageOutput contains the output for GetGameStartedMessage
type GetGameStartedMessageOutput struct {
	Message string
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

// GetRollWhisperMessageInput contains parameters for getting a whisper message after a roll
type GetRollWhisperMessageInput struct {
	// PlayerName is the name of the player who rolled
	PlayerName string
	
	// RollValue is the value that was rolled
	RollValue int
	
	// IsCriticalHit indicates if the roll was a critical hit (e.g., 6)
	IsCriticalHit bool
	
	// IsCriticalFail indicates if the roll was a critical fail (e.g., 1)
	IsCriticalFail bool
	
	// PreferredTone is the preferred tone for the message (optional)
	PreferredTone MessageTone
}

// GetRollWhisperMessageOutput contains the result of getting a whisper message
type GetRollWhisperMessageOutput struct {
	// Message is the generated whisper message
	Message string
	
	// Tone is the tone of the message
	Tone MessageTone
}

// GetLeaderboardMessageInput is the input for GetLeaderboardMessage
type GetLeaderboardMessageInput struct {
	PlayerName  string
	DrinkCount  int
	Rank        int
	TotalPlayers int
}

// GetLeaderboardMessageOutput is the output for GetLeaderboardMessage
type GetLeaderboardMessageOutput struct {
	Message string
}

// GetPayDrinkMessageInput contains parameters for getting a pay drink message
type GetPayDrinkMessageInput struct {
	// PlayerName is the name of the player paying the drink
	PlayerName string
	
	// DrinkCount is the number of drinks being paid
	DrinkCount int
}

// GetPayDrinkMessageOutput contains the output for a pay drink message
type GetPayDrinkMessageOutput struct {
	// Title is the title of the message
	Title string
	
	// Message is the body of the message
	Message string
}

// GetRollCommentInput contains parameters for getting a roll comment for the shared message
type GetRollCommentInput struct {
	// PlayerName is the name of the player who rolled
	PlayerName string
	
	// RollValue is the value of the roll
	RollValue int
	
	// IsCriticalHit indicates if the roll was a critical hit (6)
	IsCriticalHit bool
	
	// IsCriticalFail indicates if the roll was a critical fail (1)
	IsCriticalFail bool
}

// GetRollCommentOutput contains the result of getting a roll comment
type GetRollCommentOutput struct {
	// Comment is the generated comment
	Comment string
}

// GetDrinkAssignmentMessageInput contains parameters for getting a drink assignment message
type GetDrinkAssignmentMessageInput struct {
	// FromPlayerName is the name of the player assigning the drink
	FromPlayerName string
	
	// ToPlayerName is the name of the player receiving the drink
	ToPlayerName string
	
	// Reason is why the drink was assigned
	Reason models.DrinkReason
}

// GetDrinkAssignmentMessageOutput contains the result of getting a drink assignment message
type GetDrinkAssignmentMessageOutput struct {
	// Message is the generated message
	Message string
}

// ServiceConfig contains configuration for the messaging service
type ServiceConfig struct {
	// Repository is the repository for storing and retrieving messages
	// This is commented out for now, but can be uncommented when we add a repository
	// Repository Repository
}
