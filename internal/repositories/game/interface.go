package game

//go:generate mockgen -package=mocks -destination=mocks/mock_repository.go github.com/KirkDiggler/ronnied/internal/repositories/game Repository

import (
	"context"

	"github.com/KirkDiggler/ronnied/internal/models"
)

// Repository defines the interface for game data persistence
type Repository interface {
	// SaveGame persists a game
	SaveGame(ctx context.Context, input *SaveGameInput) error
	
	// GetGame retrieves a game by ID
	GetGame(ctx context.Context, input *GetGameInput) (*models.Game, error)
	
	// GetGameByChannel retrieves a game by channel ID
	GetGameByChannel(ctx context.Context, input *GetGameByChannelInput) (*models.Game, error)
	
	// DeleteGame removes a game
	DeleteGame(ctx context.Context, input *DeleteGameInput) error
	
	// GetActiveGames retrieves all active games
	GetActiveGames(ctx context.Context, input *GetActiveGamesInput) (*GetActiveGamesOutput, error)
	
	// GetGamesByParent retrieves all games with a specific parent game ID
	GetGamesByParent(ctx context.Context, input *GetGamesByParentInput) ([]*models.Game, error)
}
