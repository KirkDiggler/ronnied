package player

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
	playerKeyPrefix     = "player:"
	gamePlayersKeyPrefix = "game_players:"
)

// ErrPlayerNotFound is returned when a player is not found
var ErrPlayerNotFound = errors.New("player not found")

// Config holds configuration for the Redis player repository
type Config struct {
	// Redis client
	RedisClient *redis.Client
}

// redisRepository implements the Repository interface using Redis
type redisRepository struct {
	client *redis.Client
}

// NewRedis creates a new Redis-backed player repository
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

// SavePlayer persists a player to Redis
func (r *redisRepository) SavePlayer(ctx context.Context, input *SavePlayerInput) error {
	if input == nil || input.Player == nil {
		return errors.New("input and player cannot be nil")
	}

	player := input.Player

	// Ensure the player has an ID
	if player.ID == "" {
		return errors.New("player ID cannot be empty")
	}

	// Marshal the player to JSON
	playerJSON, err := json.Marshal(player)
	if err != nil {
		return fmt.Errorf("failed to marshal player: %w", err)
	}

	// Create a Redis transaction
	pipe := r.client.Pipeline()

	// Save the player
	playerKey := fmt.Sprintf("%s%s", playerKeyPrefix, player.ID)
	pipe.Set(ctx, playerKey, playerJSON, 0) // No expiration for now

	// If the player is in a game, add them to the game's player set
	if player.CurrentGameID != "" {
		gamePlayersKey := fmt.Sprintf("%s%s", gamePlayersKeyPrefix, player.CurrentGameID)
		pipe.SAdd(ctx, gamePlayersKey, player.ID)
	}

	// Execute the transaction
	_, err = pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to save player: %w", err)
	}

	return nil
}

// GetPlayer retrieves a player by ID from Redis
func (r *redisRepository) GetPlayer(ctx context.Context, input *GetPlayerInput) (*models.Player, error) {
	if input == nil || input.PlayerID == "" {
		return nil, errors.New("input and player ID cannot be empty")
	}

	// Get the player from Redis
	playerKey := fmt.Sprintf("%s%s", playerKeyPrefix, input.PlayerID)
	playerJSON, err := r.client.Get(ctx, playerKey).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, ErrPlayerNotFound
		}
		return nil, fmt.Errorf("failed to get player: %w", err)
	}

	// Unmarshal the player from JSON
	var player models.Player
	if err := json.Unmarshal([]byte(playerJSON), &player); err != nil {
		return nil, fmt.Errorf("failed to unmarshal player: %w", err)
	}

	return &player, nil
}

// GetPlayersInGame retrieves all players in a game from Redis
func (r *redisRepository) GetPlayersInGame(ctx context.Context, input *GetPlayersInGameInput) (*GetPlayersInGameOutput, error) {
	if input == nil || input.GameID == "" {
		return nil, errors.New("input and game ID cannot be empty")
	}

	// Get all player IDs in the game
	gamePlayersKey := fmt.Sprintf("%s%s", gamePlayersKeyPrefix, input.GameID)
	playerIDs, err := r.client.SMembers(ctx, gamePlayersKey).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get player IDs for game: %w", err)
	}

	// If there are no players, return an empty slice
	if len(playerIDs) == 0 {
		return &GetPlayersInGameOutput{
			Players: []*models.Player{},
		}, nil
	}

	// Get all player records in parallel using a pipeline
	pipe := r.client.Pipeline()
	playerCommands := make(map[string]*redis.StringCmd)

	for _, playerID := range playerIDs {
		playerKey := fmt.Sprintf("%s%s", playerKeyPrefix, playerID)
		playerCommands[playerID] = pipe.Get(ctx, playerKey)
	}

	// Execute the pipeline
	_, err = pipe.Exec(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get players: %w", err)
	}

	// Process the results
	players := make([]*models.Player, 0, len(playerIDs))
	for playerID, cmd := range playerCommands {
		playerJSON, err := cmd.Result()
		if err != nil {
			if err == redis.Nil {
				// Player was deleted between getting the IDs and fetching the player
				continue
			}
			return nil, fmt.Errorf("failed to get player %s: %w", playerID, err)
		}

		var player models.Player
		if err := json.Unmarshal([]byte(playerJSON), &player); err != nil {
			return nil, fmt.Errorf("failed to unmarshal player %s: %w", playerID, err)
		}

		players = append(players, &player)
	}

	return &GetPlayersInGameOutput{
		Players: players,
	}, nil
}

// UpdatePlayerGame updates a player's current game in Redis
func (r *redisRepository) UpdatePlayerGame(ctx context.Context, input *UpdatePlayerGameInput) error {
	if input == nil || input.PlayerID == "" {
		return errors.New("input and player ID cannot be empty")
	}

	// Get the player first
	player, err := r.GetPlayer(ctx, &GetPlayerInput{
		PlayerID: input.PlayerID,
	})
	if err != nil {
		return err
	}

	// Create a Redis transaction
	pipe := r.client.Pipeline()

	// If the player is currently in a game, remove them from that game's player set
	if player.CurrentGameID != "" && player.CurrentGameID != input.GameID {
		oldGamePlayersKey := fmt.Sprintf("%s%s", gamePlayersKeyPrefix, player.CurrentGameID)
		pipe.SRem(ctx, oldGamePlayersKey, player.ID)
	}

	// Update the player's current game
	player.CurrentGameID = input.GameID

	// Marshal the updated player
	playerJSON, err := json.Marshal(player)
	if err != nil {
		return fmt.Errorf("failed to marshal player: %w", err)
	}

	// Save the updated player
	playerKey := fmt.Sprintf("%s%s", playerKeyPrefix, player.ID)
	pipe.Set(ctx, playerKey, playerJSON, 0)

	// If the player is joining a new game, add them to that game's player set
	if input.GameID != "" {
		newGamePlayersKey := fmt.Sprintf("%s%s", gamePlayersKeyPrefix, input.GameID)
		pipe.SAdd(ctx, newGamePlayersKey, player.ID)
	}

	// Execute the transaction
	_, err = pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to update player game: %w", err)
	}

	return nil
}
