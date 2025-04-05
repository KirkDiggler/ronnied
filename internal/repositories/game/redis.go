package game

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/KirkDiggler/ronnied/internal/models"
	"github.com/redis/go-redis/v9"
)

const (
	// Key prefixes for Redis
	gameKeyPrefix     = "game:"
	channelKeyPrefix  = "channel:"
	activeGamesKey    = "active_games"
)

// ErrGameNotFound is returned when a game is not found
var ErrGameNotFound = errors.New("game not found")

// Config holds configuration for the Redis game repository
type Config struct {
	// Redis client
	RedisClient *redis.Client
}

// redisRepository implements the Repository interface using Redis
type redisRepository struct {
	client *redis.Client
}

// NewRedis creates a new Redis-backed game repository
func NewRedis(cfg *Config) (*redisRepository, error) {
	// Validate config
	if cfg == nil {
		return nil, errors.New("config cannot be nil")
	}

	if cfg.RedisClient == nil {
		return nil, errors.New("redis client cannot be nil")
	}

	// Test connection
	if err := cfg.RedisClient.Ping(context.Background()).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	return &redisRepository{
		client: cfg.RedisClient,
	}, nil
}

// SaveGame persists a game to Redis
func (r *redisRepository) SaveGame(ctx context.Context, input *SaveGameInput) error {
	if input == nil || input.Game == nil {
		return errors.New("input and game cannot be nil")
	}

	// Marshal the game to JSON
	gameJSON, err := json.Marshal(input.Game)
	if err != nil {
		return fmt.Errorf("failed to marshal game: %w", err)
	}

	// Create a Redis transaction
	pipe := r.client.Pipeline()

	// Save the game
	gameKey := fmt.Sprintf("%s%s", gameKeyPrefix, input.Game.ID)
	pipe.Set(ctx, gameKey, gameJSON, 0) // No expiration for now

	// If the game has a channel ID, update the channel-to-game mapping
	if input.Game.ChannelID != "" {
		channelKey := fmt.Sprintf("%s%s", channelKeyPrefix, input.Game.ChannelID)
		pipe.Set(ctx, channelKey, input.Game.ID, 0)
	}

	// If the game is active, add it to the active games set
	if input.Game.Status == models.GameStatusActive || input.Game.Status == models.GameStatusRollOff {
		pipe.SAdd(ctx, activeGamesKey, input.Game.ID)
	} else {
		// If the game is not active, remove it from the active games set
		pipe.SRem(ctx, activeGamesKey, input.Game.ID)
	}

	// Execute the transaction
	_, err = pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to save game: %w", err)
	}

	return nil
}

// GetGame retrieves a game by ID from Redis
func (r *redisRepository) GetGame(ctx context.Context, input *GetGameInput) (*models.Game, error) {
	if input == nil || input.GameID == "" {
		return nil, errors.New("input and game ID cannot be empty")
	}

	// Get the game from Redis
	gameKey := fmt.Sprintf("%s%s", gameKeyPrefix, input.GameID)
	gameJSON, err := r.client.Get(ctx, gameKey).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, ErrGameNotFound
		}
		return nil, fmt.Errorf("failed to get game: %w", err)
	}

	// Unmarshal the game from JSON
	var game models.Game
	if err := json.Unmarshal([]byte(gameJSON), &game); err != nil {
		return nil, fmt.Errorf("failed to unmarshal game: %w", err)
	}

	return &game, nil
}

// GetGameByChannel retrieves a game by channel ID from Redis
func (r *redisRepository) GetGameByChannel(ctx context.Context, input *GetGameByChannelInput) (*models.Game, error) {
	if input == nil || input.ChannelID == "" {
		return nil, errors.New("input and channel ID cannot be empty")
	}

	// Get the game ID from the channel-to-game mapping
	channelKey := fmt.Sprintf("%s%s", channelKeyPrefix, input.ChannelID)
	gameID, err := r.client.Get(ctx, channelKey).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, ErrGameNotFound
		}
		return nil, fmt.Errorf("failed to get game ID for channel: %w", err)
	}

	// Get the game using the game ID
	return r.GetGame(ctx, &GetGameInput{
		GameID: gameID,
	})
}

// DeleteGame removes a game from Redis
func (r *redisRepository) DeleteGame(ctx context.Context, input *DeleteGameInput) error {
	if input == nil || input.GameID == "" {
		return errors.New("input and game ID cannot be empty")
	}

	// Get the game first to get its channel ID
	game, err := r.GetGame(ctx, &GetGameInput{
		GameID: input.GameID,
	})
	if err != nil {
		return err
	}

	// Create a Redis transaction
	pipe := r.client.Pipeline()

	// Delete the game
	gameKey := fmt.Sprintf("%s%s", gameKeyPrefix, input.GameID)
	pipe.Del(ctx, gameKey)

	// If the game has a channel ID, delete the channel-to-game mapping
	if game.ChannelID != "" {
		channelKey := fmt.Sprintf("%s%s", channelKeyPrefix, game.ChannelID)
		pipe.Del(ctx, channelKey)
	}

	// Remove the game from the active games set
	pipe.SRem(ctx, activeGamesKey, input.GameID)

	// Execute the transaction
	_, err = pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to delete game: %w", err)
	}

	return nil
}

// GetActiveGames retrieves all active games from Redis
func (r *redisRepository) GetActiveGames(ctx context.Context, input *GetActiveGamesInput) (*GetActiveGamesOutput, error) {
	// Get all active game IDs from the set
	gameIDs, err := r.client.SMembers(ctx, activeGamesKey).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get active game IDs: %w", err)
	}

	// If there are no active games, return an empty slice
	if len(gameIDs) == 0 {
		return &GetActiveGamesOutput{
			Games: []*models.Game{},
		}, nil
	}

	// Get all games in parallel using a pipeline
	pipe := r.client.Pipeline()
	gameCommands := make(map[string]*redis.StringCmd)

	for _, gameID := range gameIDs {
		gameKey := fmt.Sprintf("%s%s", gameKeyPrefix, gameID)
		gameCommands[gameID] = pipe.Get(ctx, gameKey)
	}

	// Execute the pipeline
	_, err = pipe.Exec(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get active games: %w", err)
	}

	// Process the results
	games := make([]*models.Game, 0, len(gameIDs))
	for gameID, cmd := range gameCommands {
		gameJSON, err := cmd.Result()
		if err != nil {
			if err == redis.Nil {
				// Game was deleted between getting the IDs and fetching the game
				continue
			}
			return nil, fmt.Errorf("failed to get game %s: %w", gameID, err)
		}

		var game models.Game
		if err := json.Unmarshal([]byte(gameJSON), &game); err != nil {
			return nil, fmt.Errorf("failed to unmarshal game %s: %w", gameID, err)
		}

		games = append(games, &game)
	}

	return &GetActiveGamesOutput{
		Games: games,
	}, nil
}
