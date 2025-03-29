package player

import "github.com/KirkDiggler/ronnied/internal/models"

// SavePlayerInput contains parameters for saving a player
type SavePlayerInput struct {
	Player *models.Player
}

// GetPlayerInput contains parameters for retrieving a player
type GetPlayerInput struct {
	PlayerID string
}

// GetPlayersInGameInput contains parameters for retrieving players in a game
type GetPlayersInGameInput struct {
	GameID string
}

// GetPlayersInGameOutput contains the result of retrieving players in a game
type GetPlayersInGameOutput struct {
	Players []*models.Player
}

// UpdatePlayerGameInput contains parameters for updating a player's game
type UpdatePlayerGameInput struct {
	PlayerID string
	GameID   string
}
