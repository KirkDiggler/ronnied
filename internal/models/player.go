package models

import (
	"time"
)

// Player represents a participant in a game
type Player struct {
	// ID is the Discord user ID of the player
	ID string
	
	// Name is the display name of the player
	Name string
	
	// CurrentGameID is the ID of the game the player is currently in
	CurrentGameID string
	
	// LastRoll is the value of the player's last roll
	LastRoll int
	
	// LastRollTime is when the player last rolled
	LastRollTime time.Time
}
