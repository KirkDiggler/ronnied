package models

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
}

// Leaderboard represents the current standings in a game
type Leaderboard struct {
	// GameID is the unique identifier for the game
	GameID string
	
	// PlayerStats contains statistics for each player
	PlayerStats []*PlayerStats
}
