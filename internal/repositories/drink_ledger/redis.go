package drink_ledger

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"
	"github.com/google/uuid"

	"github.com/KirkDiggler/ronnied/internal/models"
	"github.com/redis/go-redis/v9"
)

const (
	// Key prefixes for Redis
	drinkKeyPrefix        = "drink:"
	gameDrinksKeyPrefix   = "game_drinks:"
	playerDrinksKeyPrefix = "player_drinks:"
	playerStatsKeyPrefix  = "player_stats:"
)

// ErrDrinkNotFound is returned when a drink record is not found
var ErrDrinkNotFound = errors.New("drink record not found")

// Config holds configuration for the Redis drink ledger repository
type Config struct {
	// Redis client
	RedisClient *redis.Client
}

// redisRepository implements the Repository interface using Redis
type redisRepository struct {
	client *redis.Client
}

// NewRedis creates a new Redis-backed drink ledger repository
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

// AddDrinkRecord adds a drink record to the ledger
func (r *redisRepository) AddDrinkRecord(ctx context.Context, input *AddDrinkRecordInput) error {
	if input == nil || input.Record == nil {
		return errors.New("input and record cannot be nil")
	}

	record := input.Record

	// Ensure the record has an ID and timestamp
	if record.ID == "" {
		return errors.New("drink record ID cannot be empty")
	}

	if record.Timestamp.IsZero() {
		record.Timestamp = time.Now()
	}

	// Marshal the record to JSON
	recordJSON, err := json.Marshal(record)
	if err != nil {
		return fmt.Errorf("failed to marshal drink record: %w", err)
	}

	// Create a Redis transaction
	pipe := r.client.Pipeline()

	// Store the drink record
	drinkKey := fmt.Sprintf("%s%s", drinkKeyPrefix, record.ID)
	pipe.Set(ctx, drinkKey, recordJSON, 0) // No expiration for now

	// Add to the game's drink records sorted set
	gameKey := fmt.Sprintf("%s%s", gameDrinksKeyPrefix, record.GameID)
	pipe.ZAdd(ctx, gameKey, redis.Z{
		Score:  float64(record.Timestamp.Unix()),
		Member: record.ID,
	})

	// Add to the "from player" drink records sorted set
	fromPlayerKey := fmt.Sprintf("%s%s:from", playerDrinksKeyPrefix, record.FromPlayerID)
	pipe.ZAdd(ctx, fromPlayerKey, redis.Z{
		Score:  float64(record.Timestamp.Unix()),
		Member: record.ID,
	})

	// Add to the "to player" drink records sorted set
	toPlayerKey := fmt.Sprintf("%s%s:to", playerDrinksKeyPrefix, record.ToPlayerID)
	pipe.ZAdd(ctx, toPlayerKey, redis.Z{
		Score:  float64(record.Timestamp.Unix()),
		Member: record.ID,
	})

	// Update player stats
	fromPlayerStatsKey := fmt.Sprintf("%s%s", playerStatsKeyPrefix, record.FromPlayerID)
	pipe.HIncrBy(ctx, fromPlayerStatsKey, "assigned", 1)

	toPlayerStatsKey := fmt.Sprintf("%s%s", playerStatsKeyPrefix, record.ToPlayerID)
	pipe.HIncrBy(ctx, toPlayerStatsKey, "received", 1)

	// Execute the transaction
	_, err = pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to add drink record: %w", err)
	}

	return nil
}

// GetDrinkRecordsForGame retrieves all drink records for a game
func (r *redisRepository) GetDrinkRecordsForGame(ctx context.Context, input *GetDrinkRecordsForGameInput) (*GetDrinkRecordsForGameOutput, error) {
	if input == nil || input.GameID == "" {
		return nil, errors.New("input and game ID cannot be empty")
	}

	// Get all drink IDs for the game
	gameKey := fmt.Sprintf("%s%s", gameDrinksKeyPrefix, input.GameID)
	drinkIDs, err := r.client.ZRange(ctx, gameKey, 0, -1).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get drink IDs for game: %w", err)
	}

	// If there are no drinks, return an empty slice
	if len(drinkIDs) == 0 {
		return &GetDrinkRecordsForGameOutput{
			Records: []*models.DrinkLedger{},
		}, nil
	}

	// Get all drink records in parallel using a pipeline
	pipe := r.client.Pipeline()
	drinkCommands := make(map[string]*redis.StringCmd)

	for _, drinkID := range drinkIDs {
		drinkKey := fmt.Sprintf("%s%s", drinkKeyPrefix, drinkID)
		drinkCommands[drinkID] = pipe.Get(ctx, drinkKey)
	}

	// Execute the pipeline
	_, err = pipe.Exec(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get drink records: %w", err)
	}

	// Process the results
	records := make([]*models.DrinkLedger, 0, len(drinkIDs))
	for drinkID, cmd := range drinkCommands {
		recordJSON, err := cmd.Result()
		if err != nil {
			if err == redis.Nil {
				// Drink record was deleted between getting the IDs and fetching the record
				continue
			}
			return nil, fmt.Errorf("failed to get drink record %s: %w", drinkID, err)
		}

		var record models.DrinkLedger
		if err := json.Unmarshal([]byte(recordJSON), &record); err != nil {
			return nil, fmt.Errorf("failed to unmarshal drink record %s: %w", drinkID, err)
		}

		records = append(records, &record)
	}

	return &GetDrinkRecordsForGameOutput{
		Records: records,
	}, nil
}

// GetDrinkRecordsForPlayer retrieves all drink records for a player
func (r *redisRepository) GetDrinkRecordsForPlayer(ctx context.Context, input *GetDrinkRecordsForPlayerInput) (*GetDrinkRecordsForPlayerOutput, error) {
	if input == nil || input.PlayerID == "" {
		return nil, errors.New("input and player ID cannot be empty")
	}

	// Get all drink IDs for the player (both assigned and received)
	fromPlayerKey := fmt.Sprintf("%s%s:from", playerDrinksKeyPrefix, input.PlayerID)
	toPlayerKey := fmt.Sprintf("%s%s:to", playerDrinksKeyPrefix, input.PlayerID)

	// Use a pipeline to get both sets of IDs
	pipe := r.client.Pipeline()
	fromCmd := pipe.ZRange(ctx, fromPlayerKey, 0, -1)
	toCmd := pipe.ZRange(ctx, toPlayerKey, 0, -1)

	_, err := pipe.Exec(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get drink IDs for player: %w", err)
	}

	fromDrinkIDs, err := fromCmd.Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get assigned drink IDs: %w", err)
	}

	toDrinkIDs, err := toCmd.Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get received drink IDs: %w", err)
	}

	// Combine and deduplicate drink IDs
	drinkIDMap := make(map[string]struct{})
	for _, id := range fromDrinkIDs {
		drinkIDMap[id] = struct{}{}
	}
	for _, id := range toDrinkIDs {
		drinkIDMap[id] = struct{}{}
	}

	// If there are no drinks, return an empty slice
	if len(drinkIDMap) == 0 {
		return &GetDrinkRecordsForPlayerOutput{
			Records: []*models.DrinkLedger{},
		}, nil
	}

	// Get all drink records in parallel using a pipeline
	pipe = r.client.Pipeline()
	drinkCommands := make(map[string]*redis.StringCmd)

	for drinkID := range drinkIDMap {
		drinkKey := fmt.Sprintf("%s%s", drinkKeyPrefix, drinkID)
		drinkCommands[drinkID] = pipe.Get(ctx, drinkKey)
	}

	// Execute the pipeline
	_, err = pipe.Exec(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get drink records: %w", err)
	}

	// Process the results
	records := make([]*models.DrinkLedger, 0, len(drinkIDMap))
	for drinkID, cmd := range drinkCommands {
		recordJSON, err := cmd.Result()
		if err != nil {
			if err == redis.Nil {
				// Drink record was deleted between getting the IDs and fetching the record
				continue
			}
			return nil, fmt.Errorf("failed to get drink record %s: %w", drinkID, err)
		}

		var record models.DrinkLedger
		if err := json.Unmarshal([]byte(recordJSON), &record); err != nil {
			return nil, fmt.Errorf("failed to unmarshal drink record %s: %w", drinkID, err)
		}

		records = append(records, &record)
	}

	return &GetDrinkRecordsForPlayerOutput{
		Records: records,
	}, nil
}

// CreateDrinkRecord creates a new drink record with a generated UUID
func (r *redisRepository) CreateDrinkRecord(ctx context.Context, input *CreateDrinkRecordInput) (*CreateDrinkRecordOutput, error) {
	// Validate input
	if input == nil {
		return nil, errors.New("input cannot be nil")
	}

	if input.GameID == "" {
		return nil, errors.New("game ID cannot be empty")
	}

	if input.ToPlayerID == "" {
		return nil, errors.New("recipient player ID cannot be empty")
	}

	// Generate a new UUID for the drink record
	drinkID := uuid.New().String()

	// Create the drink record
	record := &models.DrinkLedger{
		ID:           drinkID,
		GameID:       input.GameID,
		FromPlayerID: input.FromPlayerID,
		ToPlayerID:   input.ToPlayerID,
		Reason:       input.Reason,
		Timestamp:    input.Timestamp,
		Paid:         false,
	}

	// Save the drink record
	err := r.AddDrinkRecord(ctx, &AddDrinkRecordInput{Record: record})
	if err != nil {
		return nil, fmt.Errorf("failed to save drink record: %w", err)
	}

	return &CreateDrinkRecordOutput{Record: record}, nil
}

// MarkDrinkPaid marks a drink as paid
func (r *redisRepository) MarkDrinkPaid(ctx context.Context, input *MarkDrinkPaidInput) error {
	if input == nil || input.DrinkID == "" {
		return errors.New("input and drink ID cannot be empty")
	}

	// Get the drink record
	drinkKey := fmt.Sprintf("%s%s", drinkKeyPrefix, input.DrinkID)
	recordJSON, err := r.client.Get(ctx, drinkKey).Result()
	if err != nil {
		if err == redis.Nil {
			return ErrDrinkNotFound
		}
		return fmt.Errorf("failed to get drink record: %w", err)
	}

	// Unmarshal the record
	var record models.DrinkLedger
	if err := json.Unmarshal([]byte(recordJSON), &record); err != nil {
		return fmt.Errorf("failed to unmarshal drink record: %w", err)
	}

	// Update the record
	record.Paid = true
	record.PaidTimestamp = time.Now()

	// Marshal the updated record
	updatedRecordJSON, err := json.Marshal(record)
	if err != nil {
		return fmt.Errorf("failed to marshal updated drink record: %w", err)
	}

	// Save the updated record
	if err := r.client.Set(ctx, drinkKey, updatedRecordJSON, 0).Err(); err != nil {
		return fmt.Errorf("failed to save updated drink record: %w", err)
	}

	return nil
}
