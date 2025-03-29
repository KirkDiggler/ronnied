package models

import (
	"time"
)

// GameStatus represents the current state of a game
type GameStatus string

const (
	// GameStatusWaiting indicates a game is waiting for players to join
	GameStatusWaiting GameStatus = "waiting"
	
	// GameStatusActive indicates a game is in progress
	GameStatusActive GameStatus = "active"
	
	// GameStatusRollOff indicates a game is in a roll-off state
	GameStatusRollOff GameStatus = "roll_off"
	
	// GameStatusCompleted indicates a game has been completed
	GameStatusCompleted GameStatus = "completed"
)

// Game represents a dice rolling game session
type Game struct {
	// ID is the unique identifier for the game
	ID string
	
	// ChannelID is the Discord channel where the game is being played
	ChannelID string
	
	// Status is the current state of the game
	Status GameStatus
	
	// ParentGameID is the ID of the parent game (for roll-offs)
	ParentGameID string
	
	// PlayerIDs contains the IDs of players in the game
	PlayerIDs []string
	
	// CreatedAt is when the game was created
	CreatedAt time.Time
	
	// UpdatedAt is when the game was last updated
	UpdatedAt time.Time
	
	// MessageID is the ID of the main game message in Discord
	MessageID string
}
