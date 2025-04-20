package drink_ledger

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/KirkDiggler/ronnied/internal/models"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// CreateSession creates a new drinking session
func (r *redisRepository) CreateSession(ctx context.Context, input *CreateSessionInput) (*CreateSessionOutput, error) {
	if input == nil {
		return nil, fmt.Errorf("input cannot be nil")
	}

	if input.GuildID == "" {
		return nil, fmt.Errorf("guild ID is required")
	}

	// Generate a new session ID
	sessionID := uuid.New().String()

	// Create a new session
	session := &models.Session{
		ID:        sessionID,
		GuildID:   input.GuildID,
		CreatedAt: time.Now(),
		CreatedBy: input.CreatedBy,
		Active:    true,
	}

	// Serialize the session
	sessionJSON, err := json.Marshal(session)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal session: %w", err)
	}

	// Store the session
	sessionKey := sessionKeyPrefix + sessionID
	err = r.client.Set(ctx, sessionKey, sessionJSON, 0).Err()
	if err != nil {
		return nil, fmt.Errorf("failed to store session: %w", err)
	}

	// Get the current active session for this guild
	guildSessionKey := guildSessionPrefix + input.GuildID
	oldSessionID, err := r.client.Get(ctx, guildSessionKey).Result()

	// If there's an existing session, mark it as inactive
	if err == nil && oldSessionID != "" {
		oldSessionKey := sessionKeyPrefix + oldSessionID
		oldSessionJSON, err := r.client.Get(ctx, oldSessionKey).Result()
		if err == nil {
			var oldSession models.Session
			if err := json.Unmarshal([]byte(oldSessionJSON), &oldSession); err == nil {
				oldSession.Active = false
				updatedJSON, err := json.Marshal(oldSession)
				if err == nil {
					r.client.Set(ctx, oldSessionKey, updatedJSON, 0)
				}
			}
		}
	}

	// Set this as the current active session for the guild
	err = r.client.Set(ctx, guildSessionKey, sessionID, 0).Err()
	if err != nil {
		return nil, fmt.Errorf("failed to set current session: %w", err)
	}

	return &CreateSessionOutput{
		Session: session,
	}, nil
}

// GetCurrentSession retrieves the current active session for a guild
func (r *redisRepository) GetCurrentSession(ctx context.Context, input *GetCurrentSessionInput) (*GetCurrentSessionOutput, error) {
	if input == nil {
		return nil, fmt.Errorf("input cannot be nil")
	}

	if input.GuildID == "" {
		return nil, fmt.Errorf("guild ID is required")
	}

	// Get the current session ID for this guild
	guildSessionKey := guildSessionPrefix + input.GuildID
	sessionID, err := r.client.Get(ctx, guildSessionKey).Result()
	if err != nil {
		if err == redis.Nil {
			// No session exists for this guild
			return &GetCurrentSessionOutput{
				Session: nil,
			}, nil
		}
		return nil, fmt.Errorf("failed to get current session ID: %w", err)
	}

	// Get the session details
	sessionKey := sessionKeyPrefix + sessionID
	sessionJSON, err := r.client.Get(ctx, sessionKey).Result()
	if err != nil {
		if err == redis.Nil {
			// Session doesn't exist anymore, clear the guild session
			r.client.Del(ctx, guildSessionKey)
			return &GetCurrentSessionOutput{
				Session: nil,
			}, nil
		}
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	// Deserialize the session
	var session models.Session
	if err := json.Unmarshal([]byte(sessionJSON), &session); err != nil {
		return nil, fmt.Errorf("failed to unmarshal session: %w", err)
	}

	return &GetCurrentSessionOutput{
		Session: &session,
	}, nil
}

// GetDrinkRecordsForSession retrieves all drink records for a session
func (r *redisRepository) GetDrinkRecordsForSession(ctx context.Context, input *GetDrinkRecordsForSessionInput) (*GetDrinkRecordsForSessionOutput, error) {
	if input == nil {
		return nil, fmt.Errorf("input cannot be nil")
	}

	if input.SessionID == "" {
		return nil, fmt.Errorf("session ID is required")
	}

	// Get all drink IDs for this session
	sessionDrinksKey := sessionDrinksPrefix + input.SessionID
	drinkIDs, err := r.client.SMembers(ctx, sessionDrinksKey).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get drink IDs for session: %w", err)
	}

	// If no drinks, return empty list
	if len(drinkIDs) == 0 {
		return &GetDrinkRecordsForSessionOutput{
			Records: []*models.DrinkLedger{},
		}, nil
	}

	// Get all drink records
	var records []*models.DrinkLedger
	for _, drinkID := range drinkIDs {
		drinkKey := drinkKeyPrefix + drinkID
		drinkJSON, err := r.client.Get(ctx, drinkKey).Result()
		if err != nil {
			if err == redis.Nil {
				// Drink doesn't exist anymore, skip it
				continue
			}
			return nil, fmt.Errorf("failed to get drink record: %w", err)
		}

		// Deserialize the drink record
		var record models.DrinkLedger
		if err := json.Unmarshal([]byte(drinkJSON), &record); err != nil {
			return nil, fmt.Errorf("failed to unmarshal drink record: %w", err)
		}

		records = append(records, &record)
	}

	return &GetDrinkRecordsForSessionOutput{
		Records: records,
	}, nil
}
