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

	// CreatorID is the ID of the user who initiated the game
	CreatorID string

	// Status is the current state of the game
	Status GameStatus

	// ParentGameID is the ID of the parent game (for roll-offs)
	ParentGameID string

	// RollOffGameID is the ID of a roll-off game created from this game
	RollOffGameID string

	// Participants contains information about players participating in the game
	Participants []*Participant

	// MessageID is the Discord message ID for the game
	MessageID string

	// CreatedAt is when the game was created
	CreatedAt time.Time

	// UpdatedAt is when the game was last updated
	UpdatedAt time.Time
}

func (g *Game) GetCreatorName() string {
	// loop through participants and return the name of the creator
	for _, participant := range g.Participants {
		if participant.PlayerID == g.CreatorID {
			return participant.PlayerName
		}
	}

	return "Unknown Player"
}

// GetParticipant returns the participant with the given player ID or nil if they do not exist
func (g *Game) GetParticipant(playerID string) *Participant {
	for _, participant := range g.Participants {
		if participant.PlayerID == playerID {
			return participant
		}
	}

	return nil
}
