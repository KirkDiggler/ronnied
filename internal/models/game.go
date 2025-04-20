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

// DisplayTitle returns a user-friendly title for the game status
func (s GameStatus) DisplayTitle() string {
	switch s {
	case GameStatusWaiting:
		return "Ronnied Game - Waiting for Players"
	case GameStatusActive:
		return "Ronnied Game - In Progress"
	case GameStatusRollOff:
		return "Ronnied Game - Roll-Off in Progress"
	case GameStatusCompleted:
		return "Ronnied Game - Completed"
	default:
		return "Ronnied Game"
	}
}

// Description returns a user-friendly description for the game status
func (s GameStatus) Description() string {
	switch s {
	case GameStatusWaiting:
		return "Waiting for players to join. Click the Join button to join the game!"
	case GameStatusActive:
		return "Game in progress. Players should check their DMs for a roll button."
	case GameStatusRollOff:
		return "A roll-off is in progress to determine who drinks!"
	case GameStatusCompleted:
		return "Game completed. Check the leaderboard to see who owes drinks!"
	default:
		return "Unknown game status."
	}
}

// IsWaiting returns true if the game status is waiting
func (s GameStatus) IsWaiting() bool {
	return s == GameStatusWaiting
}

// IsActive returns true if the game status is active or roll-off
func (s GameStatus) IsActive() bool {
	return s == GameStatusActive || s == GameStatusRollOff
}

// IsRollOff returns true if the game status is roll-off
func (s GameStatus) IsRollOff() bool {
	return s == GameStatusRollOff
}

// IsCompleted returns true if the game status is completed
func (s GameStatus) IsCompleted() bool {
	return s == GameStatusCompleted
}

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

	// HighestRollOffGameID is the ID of a roll-off game for highest rollers
	HighestRollOffGameID string

	// LowestRollOffGameID is the ID of a roll-off game for lowest rollers
	LowestRollOffGameID string

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

// IsReadyToComplete checks if all players have completed their actions
// and the game is ready to be completed
func (g *Game) IsReadyToComplete() bool {
	// If there are no participants, the game is not ready to complete
	if len(g.Participants) == 0 {
		return false
	}

	// Check if all participants have completed their actions
	for _, participant := range g.Participants {
		// Check if everyone has rolled
		if participant.RollTime == nil {
			return false
		}

		// Check if anyone still needs to assign a drink
		if participant.Status == ParticipantStatusNeedsToAssign {
			return false
		}
	}

	// All checks passed, the game is ready to complete
	return true
}
