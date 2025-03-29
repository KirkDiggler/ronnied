package models

import (
	"time"
)

// Roll represents a dice roll in a game
type Roll struct {
	// ID is the unique identifier for the roll
	ID string
	
	// Value is the result of the dice roll
	Value int
	
	// PlayerID is the ID of the player who made the roll
	PlayerID string
	
	// GameID is the ID of the game the roll belongs to
	GameID string
	
	// Timestamp is when the roll was made
	Timestamp time.Time
	
	// IsCriticalHit indicates if the roll was a critical hit
	IsCriticalHit bool
	
	// IsCriticalFail indicates if the roll was a critical fail
	IsCriticalFail bool
	
	// IsLowestRoll indicates if the roll was the lowest in the game
	IsLowestRoll bool
}
