package game

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/KirkDiggler/ronnied/internal/models"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

const (
	// Key prefixes for Redis
	gameKeyPrefix    = "game:"
	channelKeyPrefix = "channel:"
	activeGamesKey   = "active_games"
	parentChildIndex = "parent:child:index:" // Index for parent-child relationships
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

	// Add the game to the parent-child index
	if input.Game.ParentGameID != "" {
		parentChildIndexKey := fmt.Sprintf("%s%s", parentChildIndex, input.Game.ParentGameID)
		pipe.ZAdd(ctx, parentChildIndexKey, redis.Z{
			Score:  float64(input.Game.CreatedAt.UnixNano()),
			Member: input.Game.ID,
		})
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

	// Remove the game from the parent-child index
	if game.ParentGameID != "" {
		parentChildIndexKey := fmt.Sprintf("%s%s", parentChildIndex, game.ParentGameID)
		pipe.ZRem(ctx, parentChildIndexKey, input.GameID)
	}

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

// GetGamesByParent retrieves all games with a specific parent game ID from Redis
func (r *redisRepository) GetGamesByParent(ctx context.Context, input *GetGamesByParentInput) ([]*models.Game, error) {
	// Get the list of child game IDs for this parent
	childGameIDs, err := r.client.ZRange(ctx, fmt.Sprintf("%s%s", parentChildIndex, input.ParentGameID), 0, -1).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get child games: %w", err)
	}

	// If there are no child games, return an empty slice
	if len(childGameIDs) == 0 {
		return []*models.Game{}, nil
	}

	// Get each child game
	games := make([]*models.Game, 0, len(childGameIDs))
	for _, gameID := range childGameIDs {
		game, err := r.GetGame(ctx, &GetGameInput{GameID: gameID})
		if err != nil {
			// Skip games that can't be found
			if errors.Is(err, ErrGameNotFound) {
				continue
			}
			return nil, err
		}
		games = append(games, game)
	}

	return games, nil
}

// CreateGame creates a new game with a generated UUID
func (r *redisRepository) CreateGame(ctx context.Context, input *CreateGameInput) (*CreateGameOutput, error) {
	// Validate input
	if input == nil {
		return nil, errors.New("input cannot be nil")
	}

	if input.ChannelID == "" {
		return nil, errors.New("channel ID cannot be empty")
	}

	if input.CreatorID == "" {
		return nil, errors.New("creator ID cannot be empty")
	}

	// Generate a new UUID for the game
	gameID := uuid.New().String()

	// Create the game
	now := time.Now()
	game := &models.Game{
		ID:           gameID,
		ChannelID:    input.ChannelID,
		CreatorID:    input.CreatorID,
		Status:       input.Status,
		Participants: []*models.Participant{},
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	// Save the game
	err := r.SaveGame(ctx, &SaveGameInput{Game: game})
	if err != nil {
		return nil, fmt.Errorf("failed to save game: %w", err)
	}

	return &CreateGameOutput{Game: game}, nil
}

// CreateRollOffGame creates a new roll-off game with a generated UUID
func (r *redisRepository) CreateRollOffGame(ctx context.Context, input *CreateRollOffGameInput) (*CreateRollOffGameOutput, error) {
	// Validate input
	if input == nil {
		return nil, errors.New("input cannot be nil")
	}

	if input.ChannelID == "" {
		return nil, errors.New("channel ID cannot be empty")
	}

	if input.CreatorID == "" {
		return nil, errors.New("creator ID cannot be empty")
	}

	if input.ParentGameID == "" {
		return nil, errors.New("parent game ID cannot be empty")
	}

	if len(input.PlayerIDs) == 0 {
		return nil, errors.New("player IDs cannot be empty")
	}

	// Generate a new UUID for the game
	gameID := uuid.New().String()

	// Create the game
	now := time.Now()
	game := &models.Game{
		ID:           gameID,
		ChannelID:    input.ChannelID,
		CreatorID:    input.CreatorID,
		Status:       models.GameStatusRollOff,
		ParentGameID: input.ParentGameID,
		Participants: []*models.Participant{},
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	// Create participants for each player
	for _, playerID := range input.PlayerIDs {
		participantID := uuid.New().String()
		playerName := ""

		// Get the player name if available
		if input.PlayerNames != nil {
			if name, ok := input.PlayerNames[playerID]; ok {
				playerName = name
			}
		}

		participant := &models.Participant{
			ID:         participantID,
			GameID:     gameID,
			PlayerID:   playerID,
			PlayerName: playerName,
			Status:     models.ParticipantStatusWaitingToRoll,
		}

		game.Participants = append(game.Participants, participant)
	}

	// Save the game
	err := r.SaveGame(ctx, &SaveGameInput{Game: game})
	if err != nil {
		return nil, fmt.Errorf("failed to save roll-off game: %w", err)
	}

	return &CreateRollOffGameOutput{Game: game}, nil
}

// CreateParticipant creates a new participant with a generated UUID
func (r *redisRepository) CreateParticipant(ctx context.Context, input *CreateParticipantInput) (*CreateParticipantOutput, error) {
	// Validate input
	if input == nil {
		return nil, errors.New("input cannot be nil")
	}

	if input.GameID == "" {
		return nil, errors.New("game ID cannot be empty")
	}

	if input.PlayerID == "" {
		return nil, errors.New("player ID cannot be empty")
	}

	// Get the game
	game, err := r.GetGame(ctx, &GetGameInput{GameID: input.GameID})
	if err != nil {
		return nil, fmt.Errorf("failed to get game: %w", err)
	}

	// Check if the player is already a participant
	for _, p := range game.Participants {
		if p.PlayerID == input.PlayerID {
			return nil, errors.New("player is already a participant")
		}
	}

	// Generate a new UUID for the participant
	participantID := uuid.New().String()

	// Create the participant
	participant := &models.Participant{
		ID:         participantID,
		GameID:     input.GameID,
		PlayerID:   input.PlayerID,
		PlayerName: input.PlayerName,
		Status:     input.Status,
	}

	// Add the participant to the game
	game.Participants = append(game.Participants, participant)
	game.UpdatedAt = time.Now()

	// Save the updated game
	err = r.SaveGame(ctx, &SaveGameInput{Game: game})
	if err != nil {
		return nil, fmt.Errorf("failed to save game with new participant: %w", err)
	}

	return &CreateParticipantOutput{Participant: participant}, nil
}
