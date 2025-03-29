package player

import (
	"context"

	"github.com/KirkDiggler/ronnied/internal/models"
)

// Repository defines the interface for player data persistence
type Repository interface {
	// SavePlayer persists a player
	SavePlayer(ctx context.Context, input *SavePlayerInput) error
	
	// GetPlayer retrieves a player by ID
	GetPlayer(ctx context.Context, input *GetPlayerInput) (*models.Player, error)
	
	// GetPlayersInGame retrieves all players in a game
	GetPlayersInGame(ctx context.Context, input *GetPlayersInGameInput) (*GetPlayersInGameOutput, error)
	
	// UpdatePlayerGame updates a player's current game
	UpdatePlayerGame(ctx context.Context, input *UpdatePlayerGameInput) error
}
