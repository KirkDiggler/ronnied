package game

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sort"

	"github.com/KirkDiggler/ronnied/internal/models"
	ledgerRepo "github.com/KirkDiggler/ronnied/internal/repositories/drink_ledger"
	playerRepo "github.com/KirkDiggler/ronnied/internal/repositories/player"
)

// extractGuildIDFromChannel extracts the guild ID from a Discord channel ID
// In Discord, channel IDs are unique, but we can use a simple mapping for now
// In a real implementation, this would use the Discord API to get the guild ID for a channel
func (s *service) extractGuildIDFromChannel(ctx context.Context, channelID string) string {
	// For now, we'll use a simple approach - in a real implementation,
	// this would query Discord API or use a cached mapping
	
	// If we have a mapping service or Discord client, we'd use it here
	// For now, we'll just use the channel ID as the guild ID
	// This is a placeholder until proper Discord API integration
	return channelID
}

// getSessionIDForChannel gets the current session ID for a channel
// If no session exists, it creates a new one
func (s *service) getSessionIDForChannel(ctx context.Context, channelID string) string {
	if channelID == "" {
		return ""
	}

	// Extract the guild ID from the channel ID
	guildID := s.extractGuildIDFromChannel(ctx, channelID)
	if guildID == "" {
		return ""
	}

	// Try to get the current session for the guild
	currentSessionOutput, err := s.drinkLedgerRepo.GetCurrentSession(ctx, &ledgerRepo.GetCurrentSessionInput{
		GuildID: guildID,
	})
	
	// If there's an error or no session exists, create a new one
	if err != nil || currentSessionOutput.Session == nil {
		// Create a new session
		sessionOutput, err := s.drinkLedgerRepo.CreateSession(ctx, &ledgerRepo.CreateSessionInput{
			GuildID:   guildID,
			CreatedBy: "system", // Default to system since we don't have a user ID here
		})
		
		if err != nil {
			// If we can't create a session, just return empty string
			return ""
		}
		
		return sessionOutput.Session.ID
	}
	
	return currentSessionOutput.Session.ID
}

// CreateSession creates a new drinking session for a channel
func (s *service) CreateSession(ctx context.Context, input *CreateSessionInput) (*CreateSessionOutput, error) {
	if input == nil {
		return nil, errors.New("input cannot be nil")
	}

	if input.ChannelID == "" {
		return nil, errors.New("channel ID is required")
	}

	// Extract the guild ID from the channel ID
	guildID := s.extractGuildIDFromChannel(ctx, input.ChannelID)
	if guildID == "" {
		return nil, errors.New("failed to extract guild ID from channel")
	}

	// Create a new session using the repository
	sessionOutput, err := s.drinkLedgerRepo.CreateSession(ctx, &ledgerRepo.CreateSessionInput{
		GuildID:   guildID,
		CreatedBy: input.CreatedBy,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	return &CreateSessionOutput{
		Success: true,
		Session: sessionOutput.Session,
	}, nil
}

// GetSessionLeaderboard retrieves the leaderboard for the current session
func (s *service) GetSessionLeaderboard(ctx context.Context, input *GetSessionLeaderboardInput) (*GetSessionLeaderboardOutput, error) {
	if input == nil {
		return nil, errors.New("input cannot be nil")
	}

	var sessionID string
	var session *models.Session

	// If a specific session ID is provided, use that
	if input.SessionID != "" {
		sessionID = input.SessionID
		// We don't have a direct method to get a session by ID, so we'll need to use what we have
		// This is a limitation in the current API
	} else if input.ChannelID != "" {
		// Extract the guild ID from the channel ID
		guildID := s.extractGuildIDFromChannel(ctx, input.ChannelID)
		if guildID == "" {
			return nil, errors.New("failed to extract guild ID from channel")
		}

		// Get the current session for the guild
		currentSessionOutput, err := s.drinkLedgerRepo.GetCurrentSession(ctx, &ledgerRepo.GetCurrentSessionInput{
			GuildID: guildID,
		})
		
		if err != nil || currentSessionOutput.Session == nil {
			// No active session for this guild
			return &GetSessionLeaderboardOutput{
				Success: true,
				Session: nil,
				Entries: []LeaderboardEntry{},
			}, nil
		}
		
		sessionID = currentSessionOutput.Session.ID
		session = currentSessionOutput.Session
		
		// Log the session details for debugging
		log.Printf("GetSessionLeaderboard: Found session %s with CreatedAt %v", 
			session.ID, session.CreatedAt)
	} else {
		return nil, errors.New("either channel ID or session ID must be provided")
	}

	// Get all drink records for this session
	drinkRecords, err := s.drinkLedgerRepo.GetDrinkRecordsForSession(ctx, &ledgerRepo.GetDrinkRecordsForSessionInput{
		SessionID: sessionID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get drink records: %w", err)
	}

	// Build maps to track drinks and payment status
	drinkCounts := make(map[string]int)    // Total drinks owed
	paidCounts := make(map[string]int)     // Drinks paid
	playerNames := make(map[string]string) // Player names cache

	// Process all drink records
	for _, record := range drinkRecords.Records {
		drinkCounts[record.ToPlayerID]++
		if record.Paid {
			paidCounts[record.ToPlayerID]++
		}
	}

	// Create leaderboard entries
	var entries []LeaderboardEntry
	for playerID, drinkCount := range drinkCounts {
		// Try to get player name from cache
		playerName, ok := playerNames[playerID]
		if !ok {
			// If not in cache, try to get from repository
			player, err := s.playerRepo.GetPlayer(ctx, &playerRepo.GetPlayerInput{
				PlayerID: playerID,
			})
			if err == nil && player != nil {
				playerName = player.Name
				playerNames[playerID] = playerName
			} else {
				playerName = "Unknown Player"
			}
		}

		entries = append(entries, LeaderboardEntry{
			PlayerID:   playerID,
			PlayerName: playerName,
			DrinkCount: drinkCount,
			PaidCount:  paidCounts[playerID],
		})
	}

	// Sort entries by drink count (most drinks first)
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].DrinkCount > entries[j].DrinkCount
	})

	// If we have a session ID but no session object (from direct ID lookup), create a minimal session
	if session == nil && sessionID != "" {
		session = &models.Session{ID: sessionID}
	}

	return &GetSessionLeaderboardOutput{
		Success: true,
		Session: session,
		Entries: entries,
	}, nil
}

// StartNewSession creates a new drinking session for a channel (alias for CreateSession with a clearer name)
func (s *service) StartNewSession(ctx context.Context, input *StartNewSessionInput) (*StartNewSessionOutput, error) {
	if input == nil {
		return nil, errors.New("input cannot be nil")
	}

	if input.ChannelID == "" {
		return nil, errors.New("channel ID cannot be empty")
	}

	// Extract the guild ID from the channel ID
	guildID := s.extractGuildIDFromChannel(ctx, input.ChannelID)
	if guildID == "" {
		return nil, errors.New("failed to extract guild ID from channel")
	}

	// Create the session using CreateSession
	sessionOutput, err := s.CreateSession(ctx, &CreateSessionInput{
		ChannelID: input.ChannelID,
		CreatedBy: input.CreatorID,
	})
	if err != nil {
		return nil, err
	}

	return &StartNewSessionOutput{
		Success:   true,
		Session:   sessionOutput.Session,
		SessionID: sessionOutput.Session.ID,
	}, nil
}
