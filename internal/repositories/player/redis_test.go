package player

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

func (s *RedisRepositoryTestSuite) TestSaveAndGetPlayer() {
	// Create a test player
	player := &models.Player{
		ID:            "test-player-id",
		Name:          "Test Player",
		CurrentGameID: "test-game-id",
		LastRoll:      5,
		LastRollTime:  s.testNow,
	}

	// Save the player
	err := s.repo.SavePlayer(context.Background(), &SavePlayerInput{
		Player: player,
	})
	s.Require().NoError(err)

	// Get the player
	retrievedPlayer, err := s.repo.GetPlayer(context.Background(), &GetPlayerInput{
		PlayerID: "test-player-id",
	})
	s.Require().NoError(err)
	s.Require().NotNil(retrievedPlayer)

	// Verify the player properties
	s.Equal("test-player-id", retrievedPlayer.ID)
	s.Equal("Test Player", retrievedPlayer.Name)
	s.Equal("test-game-id", retrievedPlayer.CurrentGameID)
	s.Equal(5, retrievedPlayer.LastRoll)
	s.Equal(s.testNow.Unix(), retrievedPlayer.LastRollTime.Unix())
}

func (s *RedisRepositoryTestSuite) TestGetPlayersInGame() {
	// Create test players
	players := []*models.Player{
		{
			ID:            "player-1",
			Name:          "Player One",
			CurrentGameID: "game-1",
			LastRoll:      4,
			LastRollTime:  s.testNow,
		},
		{
			ID:            "player-2",
			Name:          "Player Two",
			CurrentGameID: "game-1",
			LastRoll:      6,
			LastRollTime:  s.testNow,
		},
		{
			ID:            "player-3",
			Name:          "Player Three",
			CurrentGameID: "game-2",
			LastRoll:      1,
			LastRollTime:  s.testNow,
		},
	}

	// Save all players
	for _, player := range players {
		err := s.repo.SavePlayer(context.Background(), &SavePlayerInput{
			Player: player,
		})
		s.Require().NoError(err)
	}

	// Get players in game-1
	game1Output, err := s.repo.GetPlayersInGame(context.Background(), &GetPlayersInGameInput{
		GameID: "game-1",
	})
	s.Require().NoError(err)
	s.Require().Len(game1Output.Players, 2)

	// Create a map for easier verification
	playerMap := make(map[string]*models.Player)
	for _, player := range game1Output.Players {
		playerMap[player.ID] = player
	}

	// Verify the players in game-1
	s.Contains(playerMap, "player-1")
	s.Contains(playerMap, "player-2")
	s.Equal("Player One", playerMap["player-1"].Name)
	s.Equal("Player Two", playerMap["player-2"].Name)

	// Get players in game-2
	game2Output, err := s.repo.GetPlayersInGame(context.Background(), &GetPlayersInGameInput{
		GameID: "game-2",
	})
	s.Require().NoError(err)
	s.Require().Len(game2Output.Players, 1)
	s.Equal("player-3", game2Output.Players[0].ID)
	s.Equal("Player Three", game2Output.Players[0].Name)

	// Get players in a non-existent game
	emptyOutput, err := s.repo.GetPlayersInGame(context.Background(), &GetPlayersInGameInput{
		GameID: "non-existent-game",
	})
	s.Require().NoError(err)
	s.Require().Empty(emptyOutput.Players)
}

func (s *RedisRepositoryTestSuite) TestUpdatePlayerGame() {
	// Create a test player
	player := &models.Player{
		ID:            "test-player-id",
		Name:          "Test Player",
		CurrentGameID: "old-game-id",
		LastRoll:      3,
		LastRollTime:  s.testNow,
	}

	// Save the player
	err := s.repo.SavePlayer(context.Background(), &SavePlayerInput{
		Player: player,
	})
	s.Require().NoError(err)

	// Verify the player is in the old game
	oldGameOutput, err := s.repo.GetPlayersInGame(context.Background(), &GetPlayersInGameInput{
		GameID: "old-game-id",
	})
	s.Require().NoError(err)
	s.Require().Len(oldGameOutput.Players, 1)
	s.Equal("test-player-id", oldGameOutput.Players[0].ID)

	// Update the player's game
	err = s.repo.UpdatePlayerGame(context.Background(), &UpdatePlayerGameInput{
		PlayerID: "test-player-id",
		GameID:   "new-game-id",
	})
	s.Require().NoError(err)

	// Get the updated player
	updatedPlayer, err := s.repo.GetPlayer(context.Background(), &GetPlayerInput{
		PlayerID: "test-player-id",
	})
	s.Require().NoError(err)
	s.Equal("new-game-id", updatedPlayer.CurrentGameID)

	// Verify the player is no longer in the old game
	oldGameOutput, err = s.repo.GetPlayersInGame(context.Background(), &GetPlayersInGameInput{
		GameID: "old-game-id",
	})
	s.Require().NoError(err)
	s.Require().Empty(oldGameOutput.Players)

	// Verify the player is in the new game
	newGameOutput, err := s.repo.GetPlayersInGame(context.Background(), &GetPlayersInGameInput{
		GameID: "new-game-id",
	})
	s.Require().NoError(err)
	s.Require().Len(newGameOutput.Players, 1)
	s.Equal("test-player-id", newGameOutput.Players[0].ID)
}

func (s *RedisRepositoryTestSuite) TestUpdatePlayerGameToNone() {
	// Create a test player
	player := &models.Player{
		ID:            "test-player-id",
		Name:          "Test Player",
		CurrentGameID: "game-id",
		LastRoll:      3,
		LastRollTime:  s.testNow,
	}

	// Save the player
	err := s.repo.SavePlayer(context.Background(), &SavePlayerInput{
		Player: player,
	})
	s.Require().NoError(err)

	// Update the player to have no game
	err = s.repo.UpdatePlayerGame(context.Background(), &UpdatePlayerGameInput{
		PlayerID: "test-player-id",
		GameID:   "", // Empty game ID
	})
	s.Require().NoError(err)

	// Get the updated player
	updatedPlayer, err := s.repo.GetPlayer(context.Background(), &GetPlayerInput{
		PlayerID: "test-player-id",
	})
	s.Require().NoError(err)
	s.Equal("", updatedPlayer.CurrentGameID)

	// Verify the player is no longer in the game
	gameOutput, err := s.repo.GetPlayersInGame(context.Background(), &GetPlayersInGameInput{
		GameID: "game-id",
	})
	s.Require().NoError(err)
	s.Require().Empty(gameOutput.Players)
}

func (s *RedisRepositoryTestSuite) TestGetNonExistentPlayer() {
	// Try to get a non-existent player
	_, err := s.repo.GetPlayer(context.Background(), &GetPlayerInput{
		PlayerID: "non-existent-player",
	})
	s.Require().Error(err)
	s.Equal(ErrPlayerNotFound, err)
}
