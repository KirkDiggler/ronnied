package game

import (
	"context"
	"testing"
	"time"

	"github.com/KirkDiggler/ronnied/internal/models"
	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/suite"
)

type RedisRepositoryTestSuite struct {
	suite.Suite
	mr      *miniredis.Miniredis
	client  *redis.Client
	repo    Repository
	testNow time.Time
}

func (s *RedisRepositoryTestSuite) SetupTest() {
	// Create a new miniredis server for each test
	mr, err := miniredis.Run()
	s.Require().NoError(err)
	s.mr = mr

	// Create a Redis client connected to the miniredis server
	s.client = redis.NewClient(&redis.Options{
		Addr: s.mr.Addr(),
	})

	// Create the repository
	repo, err := NewRedis(&Config{
		RedisClient: s.client,
	})
	s.Require().NoError(err)
	s.repo = repo

	// Set up test time
	s.testNow = time.Date(2025, 4, 5, 10, 0, 0, 0, time.UTC)
}

func (s *RedisRepositoryTestSuite) TearDownTest() {
	s.client.Close()
	s.mr.Close()
}

func TestRedisRepositoryTestSuite(t *testing.T) {
	suite.Run(t, new(RedisRepositoryTestSuite))
}

func (s *RedisRepositoryTestSuite) TestSaveAndGetGame() {
	// Create a test game
	game := &models.Game{
		ID:        "test-game-id",
		ChannelID: "test-channel-id",
		CreatorID: "test-creator-id",
		Status:    models.GameStatusWaiting,
		Participants: []*models.Participant{
			{
				ID:       "test-participant-id",
				GameID:   "test-game-id",
				PlayerID: "test-player-id",
				Status:   models.ParticipantStatusWaitingToRoll,
			},
		},
		CreatedAt: s.testNow,
		UpdatedAt: s.testNow,
	}

	// Save the game
	err := s.repo.SaveGame(context.Background(), &SaveGameInput{
		Game: game,
	})
	s.Require().NoError(err)

	// Get the game by ID
	retrievedGame, err := s.repo.GetGame(context.Background(), &GetGameInput{
		GameID: "test-game-id",
	})
	s.Require().NoError(err)
	s.Require().NotNil(retrievedGame)

	// Verify the game properties
	s.Equal("test-game-id", retrievedGame.ID)
	s.Equal("test-channel-id", retrievedGame.ChannelID)
	s.Equal("test-creator-id", retrievedGame.CreatorID)
	s.Equal(models.GameStatusWaiting, retrievedGame.Status)
	s.Len(retrievedGame.Participants, 1)
	s.Equal("test-participant-id", retrievedGame.Participants[0].ID)
	s.Equal("test-player-id", retrievedGame.Participants[0].PlayerID)
	s.Equal(models.ParticipantStatusWaitingToRoll, retrievedGame.Participants[0].Status)
	s.Equal(s.testNow.Unix(), retrievedGame.CreatedAt.Unix())
	s.Equal(s.testNow.Unix(), retrievedGame.UpdatedAt.Unix())
}

func (s *RedisRepositoryTestSuite) TestGetGameByChannel() {
	// Create a test game
	game := &models.Game{
		ID:        "test-game-id",
		ChannelID: "test-channel-id",
		Status:    models.GameStatusWaiting,
		CreatedAt: s.testNow,
		UpdatedAt: s.testNow,
	}

	// Save the game
	err := s.repo.SaveGame(context.Background(), &SaveGameInput{
		Game: game,
	})
	s.Require().NoError(err)

	// Get the game by channel ID
	retrievedGame, err := s.repo.GetGameByChannel(context.Background(), &GetGameByChannelInput{
		ChannelID: "test-channel-id",
	})
	s.Require().NoError(err)
	s.Require().NotNil(retrievedGame)

	// Verify the game properties
	s.Equal("test-game-id", retrievedGame.ID)
	s.Equal("test-channel-id", retrievedGame.ChannelID)
}

func (s *RedisRepositoryTestSuite) TestGetActiveGames() {
	// Create test games with different statuses
	activeGame := &models.Game{
		ID:        "active-game-id",
		ChannelID: "active-channel-id",
		Status:    models.GameStatusActive,
		CreatedAt: s.testNow,
		UpdatedAt: s.testNow,
	}

	rollOffGame := &models.Game{
		ID:        "rolloff-game-id",
		ChannelID: "rolloff-channel-id",
		Status:    models.GameStatusRollOff,
		CreatedAt: s.testNow,
		UpdatedAt: s.testNow,
	}

	waitingGame := &models.Game{
		ID:        "waiting-game-id",
		ChannelID: "waiting-channel-id",
		Status:    models.GameStatusWaiting,
		CreatedAt: s.testNow,
		UpdatedAt: s.testNow,
	}

	completedGame := &models.Game{
		ID:        "completed-game-id",
		ChannelID: "completed-channel-id",
		Status:    models.GameStatusCompleted,
		CreatedAt: s.testNow,
		UpdatedAt: s.testNow,
	}

	// Save all games
	s.Require().NoError(s.repo.SaveGame(context.Background(), &SaveGameInput{Game: activeGame}))
	s.Require().NoError(s.repo.SaveGame(context.Background(), &SaveGameInput{Game: rollOffGame}))
	s.Require().NoError(s.repo.SaveGame(context.Background(), &SaveGameInput{Game: waitingGame}))
	s.Require().NoError(s.repo.SaveGame(context.Background(), &SaveGameInput{Game: completedGame}))

	// Get active games
	result, err := s.repo.GetActiveGames(context.Background(), &GetActiveGamesInput{})
	s.Require().NoError(err)
	s.Require().NotNil(result)

	// Verify that only active and roll-off games are returned
	s.Len(result.Games, 2)

	// Create a map for easier verification
	gameMap := make(map[string]*models.Game)
	for _, game := range result.Games {
		gameMap[game.ID] = game
	}

	// Verify the active game is in the results
	activeResult, ok := gameMap["active-game-id"]
	s.True(ok)
	s.Equal(models.GameStatusActive, activeResult.Status)

	// Verify the roll-off game is in the results
	rollOffResult, ok := gameMap["rolloff-game-id"]
	s.True(ok)
	s.Equal(models.GameStatusRollOff, rollOffResult.Status)

	// Verify waiting and completed games are not in the results
	_, ok = gameMap["waiting-game-id"]
	s.False(ok)
	_, ok = gameMap["completed-game-id"]
	s.False(ok)
}

func (s *RedisRepositoryTestSuite) TestDeleteGame() {
	// Create a test game
	game := &models.Game{
		ID:        "test-game-id",
		ChannelID: "test-channel-id",
		Status:    models.GameStatusActive,
		CreatedAt: s.testNow,
		UpdatedAt: s.testNow,
	}

	// Save the game
	err := s.repo.SaveGame(context.Background(), &SaveGameInput{
		Game: game,
	})
	s.Require().NoError(err)

	// Verify the game exists
	_, err = s.repo.GetGame(context.Background(), &GetGameInput{
		GameID: "test-game-id",
	})
	s.Require().NoError(err)

	// Delete the game
	err = s.repo.DeleteGame(context.Background(), &DeleteGameInput{
		GameID: "test-game-id",
	})
	s.Require().NoError(err)

	// Verify the game no longer exists
	_, err = s.repo.GetGame(context.Background(), &GetGameInput{
		GameID: "test-game-id",
	})
	s.Require().Error(err)
	s.Equal(ErrGameNotFound, err)

	// Verify the channel mapping is also removed
	_, err = s.repo.GetGameByChannel(context.Background(), &GetGameByChannelInput{
		ChannelID: "test-channel-id",
	})
	s.Require().Error(err)
	s.Equal(ErrGameNotFound, err)

	// Verify the game is removed from active games
	result, err := s.repo.GetActiveGames(context.Background(), &GetActiveGamesInput{})
	s.Require().NoError(err)
	s.Len(result.Games, 0)
}

func (s *RedisRepositoryTestSuite) TestGameStatusTransition() {
	// Create a test game in waiting status
	game := &models.Game{
		ID:        "test-game-id",
		ChannelID: "test-channel-id",
		Status:    models.GameStatusWaiting,
		CreatedAt: s.testNow,
		UpdatedAt: s.testNow,
	}

	// Save the game
	err := s.repo.SaveGame(context.Background(), &SaveGameInput{
		Game: game,
	})
	s.Require().NoError(err)

	// Verify it's not in active games
	result, err := s.repo.GetActiveGames(context.Background(), &GetActiveGamesInput{})
	s.Require().NoError(err)
	s.Len(result.Games, 0)

	// Update the game to active status
	game.Status = models.GameStatusActive
	game.UpdatedAt = s.testNow.Add(time.Minute)

	// Save the updated game
	err = s.repo.SaveGame(context.Background(), &SaveGameInput{
		Game: game,
	})
	s.Require().NoError(err)

	// Verify it's now in active games
	result, err = s.repo.GetActiveGames(context.Background(), &GetActiveGamesInput{})
	s.Require().NoError(err)
	s.Len(result.Games, 1)
	s.Equal("test-game-id", result.Games[0].ID)
	s.Equal(models.GameStatusActive, result.Games[0].Status)

	// Update the game to completed status
	game.Status = models.GameStatusCompleted
	game.UpdatedAt = s.testNow.Add(time.Minute * 2)

	// Save the updated game
	err = s.repo.SaveGame(context.Background(), &SaveGameInput{
		Game: game,
	})
	s.Require().NoError(err)

	// Verify it's no longer in active games
	result, err = s.repo.GetActiveGames(context.Background(), &GetActiveGamesInput{})
	s.Require().NoError(err)
	s.Len(result.Games, 0)
}
