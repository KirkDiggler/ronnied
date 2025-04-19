package models

import (
	"time"
)

// DrinkReason represents why a drink was assigned
type DrinkReason string

const (
	// DrinkReasonCriticalHit indicates a drink assigned due to rolling a critical hit (6)
	DrinkReasonCriticalHit DrinkReason = "critical_hit"
	
	// DrinkReasonCriticalFail indicates a drink assigned due to rolling a critical fail (1)
	DrinkReasonCriticalFail DrinkReason = "critical_fail"
	
	// DrinkReasonLowestRoll indicates a drink assigned due to having the lowest roll
	DrinkReasonLowestRoll DrinkReason = "lowest_roll"
)

// DrinkLedger records a drink assignment between players
type DrinkLedger struct {
	// ID is the unique identifier for the drink record
	ID string
	
	// FromPlayerID is the ID of the player assigning the drink
	FromPlayerID string
	
	// ToPlayerID is the ID of the player receiving the drink
	ToPlayerID string
	
	// GameID is the ID of the game where the drink was assigned
	GameID string
	
	// Reason is why the drink was assigned
	Reason DrinkReason
	
	// Timestamp is when the drink was assigned
	Timestamp time.Time
	
	// Paid indicates if the drink has been paid
	Paid bool
	
	// PaidTimestamp is when the drink was paid
	PaidTimestamp time.Time
	
	// Archived indicates if the drink record has been archived
	Archived bool
	
	// ArchivedTimestamp is when the drink was archived
	ArchivedTimestamp time.Time
}
