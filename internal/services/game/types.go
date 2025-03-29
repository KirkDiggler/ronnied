package game

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

// RollOffType represents the type of roll-off
type RollOffType string

const (
	// RollOffTypeHighest indicates a roll-off for players with the highest roll
	RollOffTypeHighest RollOffType = "highest"
	
	// RollOffTypeLowest indicates a roll-off for players with the lowest roll
	RollOffTypeLowest RollOffType = "lowest"
)

// Config holds configuration for the game service
type Config struct {
	// Maximum number of players per game
	MaxPlayers int
	
	// Number of sides on the dice
	DiceSides int
	
	// Value that counts as a critical hit
	CriticalHitValue int
	
	// Value that counts as a critical fail
	CriticalFailValue int
	
	// Maximum number of concurrent games
	MaxConcurrentGames int
}

// CreateGameInput contains parameters for creating a new game
type CreateGameInput struct {
	// ChannelID is the Discord channel ID where the game is being played
	ChannelID string
}

// CreateGameOutput contains the result of creating a new game
type CreateGameOutput struct {
	// GameID is the unique identifier for the created game
	GameID string
}

// JoinGameInput contains parameters for joining a game
type JoinGameInput struct {
	// GameID is the unique identifier for the game
	GameID string
	
	// PlayerID is the Discord user ID of the player
	PlayerID string
	
	// PlayerName is the display name of the player
	PlayerName string
}

// JoinGameOutput contains the result of joining a game
type JoinGameOutput struct {
	// Success indicates if the player successfully joined the game
	Success bool
}

// LeaveGameInput contains parameters for leaving a game
type LeaveGameInput struct {
	// GameID is the unique identifier for the game
	GameID string
	
	// PlayerID is the Discord user ID of the player
	PlayerID string
}

// LeaveGameOutput contains the result of leaving a game
type LeaveGameOutput struct {
	// Success indicates if the player successfully left the game
	Success bool
}

// RollDiceInput contains parameters for rolling dice
type RollDiceInput struct {
	// GameID is the unique identifier for the game
	GameID string
	
	// PlayerID is the Discord user ID of the player
	PlayerID string
}

// RollDiceOutput contains the result of rolling dice
type RollDiceOutput struct {
	// Value is the result of the dice roll
	Value int
	
	// IsCriticalHit indicates if the roll was a critical hit
	IsCriticalHit bool
	
	// IsCriticalFail indicates if the roll was a critical fail
	IsCriticalFail bool
	
	// IsLowestRoll indicates if the roll was the lowest in the game
	// This will be false initially and may be updated after all players roll
	IsLowestRoll bool
	
	// NeedsRollOff indicates if a roll-off is needed
	NeedsRollOff bool
	
	// RollOffType indicates the type of roll-off needed (if any)
	RollOffType RollOffType
	
	// RollOffGameID is the ID of the roll-off game (if created)
	RollOffGameID string
}

// AssignDrinkInput contains parameters for assigning a drink
type AssignDrinkInput struct {
	// GameID is the unique identifier for the game
	GameID string
	
	// FromPlayerID is the Discord user ID of the player assigning the drink
	FromPlayerID string
	
	// ToPlayerID is the Discord user ID of the player receiving the drink
	ToPlayerID string
	
	// Reason is why the drink is being assigned
	Reason DrinkReason
}

// AssignDrinkOutput contains the result of assigning a drink
type AssignDrinkOutput struct {
	// Success indicates if the drink was successfully assigned
	Success bool
}

// GetLeaderboardInput contains parameters for retrieving a leaderboard
type GetLeaderboardInput struct {
	// GameID is the unique identifier for the game
	GameID string
}

// GetLeaderboardOutput contains the result of retrieving a leaderboard
type GetLeaderboardOutput struct {
	// PlayerStats contains statistics for each player
	PlayerStats []*PlayerStats
	
	// GameID is the unique identifier for the game
	GameID string
}

// PlayerStats represents a player's statistics in a game
type PlayerStats struct {
	// PlayerID is the Discord user ID of the player
	PlayerID string
	
	// PlayerName is the display name of the player
	PlayerName string
	
	// DrinksAssigned is the number of drinks assigned to others
	DrinksAssigned int
	
	// DrinksReceived is the number of drinks received from others
	DrinksReceived int
	
	// LastRoll is the value of the player's last roll
	LastRoll int
	
	// LastRollTime is when the player last rolled
	LastRollTime time.Time
}

// ResetGameInput contains parameters for resetting a game
type ResetGameInput struct {
	// GameID is the unique identifier for the game
	GameID string
}

// ResetGameOutput contains the result of resetting a game
type ResetGameOutput struct {
	// Success indicates if the game was successfully reset
	Success bool
}

// EndGameInput contains parameters for ending a game
type EndGameInput struct {
	// GameID is the unique identifier for the game
	GameID string
}

// EndGameOutput contains the result of ending a game
type EndGameOutput struct {
	// Success indicates if the game was successfully ended
	Success bool
	
	// FinalLeaderboard contains the final game statistics
	FinalLeaderboard *GetLeaderboardOutput
}

// HandleRollOffInput contains parameters for handling a roll-off
type HandleRollOffInput struct {
	// ParentGameID is the ID of the original game
	ParentGameID string
	
	// RollOffGameID is the ID of the roll-off game
	RollOffGameID string
	
	// PlayerIDs are the IDs of players participating in the roll-off
	PlayerIDs []string
	
	// Type is the type of roll-off (highest or lowest)
	Type RollOffType
}

// HandleRollOffOutput contains the result of handling a roll-off
type HandleRollOffOutput struct {
	// Success indicates if the roll-off was successfully handled
	Success bool
	
	// WinnerPlayerIDs contains the IDs of players who won the roll-off
	WinnerPlayerIDs []string
	
	// NeedsAnotherRollOff indicates if another roll-off is needed
	NeedsAnotherRollOff bool
	
	// NextRollOffGameID is the ID of the next roll-off game (if needed)
	NextRollOffGameID string
}
