package models

import (
	"time"
)

// ParticipantStatus represents the current state of a participant in a game
type ParticipantStatus string

const (
	// ParticipantStatusActive indicates a player is active in the game
	ParticipantStatusActive ParticipantStatus = "active"
	
	// ParticipantStatusNeedsToAssign indicates a player needs to assign a drink
	ParticipantStatusNeedsToAssign ParticipantStatus = "needs_to_assign"
	
	// ParticipantStatusWaitingToRoll indicates a player still needs to roll
	ParticipantStatusWaitingToRoll ParticipantStatus = "waiting_to_roll"
	
	// ParticipantStatusInRollOff indicates a player is participating in a roll-off
	ParticipantStatusInRollOff ParticipantStatus = "in_roll_off"
	
	// ParticipantStatusRolledInRollOff indicates a player has rolled in a roll-off
	ParticipantStatusRolledInRollOff ParticipantStatus = "rolled_in_roll_off"
)

// Participant represents a player's participation in a specific game
type Participant struct {
	// ID is a unique identifier for this participation
	ID string

	// GameID is the ID of the game the player is participating in
	GameID string

	// PlayerID is the ID of the player
	PlayerID string
	
	// PlayerName is the display name of the player
	PlayerName string

	// Status is the current state of the participant
	Status ParticipantStatus

	// RollValue is the value of the player's roll in this game
	RollValue int

	// RollTime is when the player rolled in this game
	RollTime *time.Time
}
