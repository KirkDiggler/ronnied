package drink_ledger

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

func (s *RedisRepositoryTestSuite) TestAddAndGetDrinkRecord() {
	// Create a test drink record
	record := &models.DrinkLedger{
		ID:           "test-drink-id",
		FromPlayerID: "from-player-id",
		ToPlayerID:   "to-player-id",
		GameID:       "test-game-id",
		Reason:       models.DrinkReasonCriticalHit,
		Timestamp:    s.testNow,
		Paid:         false,
	}

	// Add the drink record
	err := s.repo.AddDrinkRecord(context.Background(), &AddDrinkRecordInput{
		Record: record,
	})
	s.Require().NoError(err)

	// Get the drink records for the game
	gameOutput, err := s.repo.GetDrinkRecordsForGame(context.Background(), &GetDrinkRecordsForGameInput{
		GameID: "test-game-id",
	})
	s.Require().NoError(err)
	s.Require().Len(gameOutput.Records, 1)

	// Verify the record properties
	s.Equal("test-drink-id", gameOutput.Records[0].ID)
	s.Equal("from-player-id", gameOutput.Records[0].FromPlayerID)
	s.Equal("to-player-id", gameOutput.Records[0].ToPlayerID)
	s.Equal("test-game-id", gameOutput.Records[0].GameID)
	s.Equal(models.DrinkReasonCriticalHit, gameOutput.Records[0].Reason)
	s.Equal(s.testNow.Unix(), gameOutput.Records[0].Timestamp.Unix())
	s.False(gameOutput.Records[0].Paid)
}

func (s *RedisRepositoryTestSuite) TestGetDrinkRecordsForPlayer() {
	// Create test drink records
	records := []*models.DrinkLedger{
		{
			ID:           "drink-1",
			FromPlayerID: "player-1",
			ToPlayerID:   "player-2",
			GameID:       "game-1",
			Reason:       models.DrinkReasonCriticalHit,
			Timestamp:    s.testNow,
		},
		{
			ID:           "drink-2",
			FromPlayerID: "player-2",
			ToPlayerID:   "player-1",
			GameID:       "game-1",
			Reason:       models.DrinkReasonCriticalFail,
			Timestamp:    s.testNow.Add(time.Minute),
		},
		{
			ID:           "drink-3",
			FromPlayerID: "player-1",
			ToPlayerID:   "player-3",
			GameID:       "game-2",
			Reason:       models.DrinkReasonLowestRoll,
			Timestamp:    s.testNow.Add(time.Minute * 2),
		},
	}

	// Add all records
	for _, record := range records {
		err := s.repo.AddDrinkRecord(context.Background(), &AddDrinkRecordInput{
			Record: record,
		})
		s.Require().NoError(err)
	}

	// Get drinks for player-1 (should have 3 records - 2 from and 1 to)
	player1Output, err := s.repo.GetDrinkRecordsForPlayer(context.Background(), &GetDrinkRecordsForPlayerInput{
		PlayerID: "player-1",
	})
	s.Require().NoError(err)
	s.Require().Len(player1Output.Records, 3)

	// Create a map for easier verification
	recordMap := make(map[string]*models.DrinkLedger)
	for _, record := range player1Output.Records {
		recordMap[record.ID] = record
	}

	// Verify all expected records are present
	s.Contains(recordMap, "drink-1")
	s.Contains(recordMap, "drink-2")
	s.Contains(recordMap, "drink-3")

	// Get drinks for player-3 (should have 1 record - received from player-1)
	player3Output, err := s.repo.GetDrinkRecordsForPlayer(context.Background(), &GetDrinkRecordsForPlayerInput{
		PlayerID: "player-3",
	})
	s.Require().NoError(err)
	s.Require().Len(player3Output.Records, 1)
	s.Equal("drink-3", player3Output.Records[0].ID)
}

func (s *RedisRepositoryTestSuite) TestMarkDrinkPaid() {
	// Create a test drink record
	record := &models.DrinkLedger{
		ID:           "test-drink-id",
		FromPlayerID: "from-player-id",
		ToPlayerID:   "to-player-id",
		GameID:       "test-game-id",
		Reason:       models.DrinkReasonCriticalHit,
		Timestamp:    s.testNow,
		Paid:         false,
	}

	// Add the drink record
	err := s.repo.AddDrinkRecord(context.Background(), &AddDrinkRecordInput{
		Record: record,
	})
	s.Require().NoError(err)

	// Mark the drink as paid
	err = s.repo.MarkDrinkPaid(context.Background(), &MarkDrinkPaidInput{
		DrinkID: "test-drink-id",
	})
	s.Require().NoError(err)

	// Get the drink records for the game to verify it's marked as paid
	gameOutput, err := s.repo.GetDrinkRecordsForGame(context.Background(), &GetDrinkRecordsForGameInput{
		GameID: "test-game-id",
	})
	s.Require().NoError(err)
	s.Require().Len(gameOutput.Records, 1)
	s.True(gameOutput.Records[0].Paid)
	s.NotZero(gameOutput.Records[0].PaidTimestamp)
}

func (s *RedisRepositoryTestSuite) TestGetEmptyResults() {
	// Get drinks for a game with no records
	gameOutput, err := s.repo.GetDrinkRecordsForGame(context.Background(), &GetDrinkRecordsForGameInput{
		GameID: "non-existent-game",
	})
	s.Require().NoError(err)
	s.Require().Empty(gameOutput.Records)

	// Get drinks for a player with no records
	playerOutput, err := s.repo.GetDrinkRecordsForPlayer(context.Background(), &GetDrinkRecordsForPlayerInput{
		PlayerID: "non-existent-player",
	})
	s.Require().NoError(err)
	s.Require().Empty(playerOutput.Records)
}

func (s *RedisRepositoryTestSuite) TestMarkNonExistentDrink() {
	// Try to mark a non-existent drink as paid
	err := s.repo.MarkDrinkPaid(context.Background(), &MarkDrinkPaidInput{
		DrinkID: "non-existent-drink",
	})
	s.Require().Error(err)
	s.Equal(ErrDrinkNotFound, err)
}
