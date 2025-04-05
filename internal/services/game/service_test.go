package game

import (
	"context"
	"testing"
	"time"

	clockMocks "github.com/KirkDiggler/ronnied/internal/common/clock/mocks"
	uuidMocks "github.com/KirkDiggler/ronnied/internal/common/uuid/mocks"
	"github.com/KirkDiggler/ronnied/internal/dice"
	"github.com/KirkDiggler/ronnied/internal/models"
	ledgerRepo "github.com/KirkDiggler/ronnied/internal/repositories/drink_ledger"
	ledgerMocks "github.com/KirkDiggler/ronnied/internal/repositories/drink_ledger/mocks"
	gameRepo "github.com/KirkDiggler/ronnied/internal/repositories/game"
	gameMocks "github.com/KirkDiggler/ronnied/internal/repositories/game/mocks"
	playerMocks "github.com/KirkDiggler/ronnied/internal/repositories/player/mocks"

	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

type GameServiceTestSuite struct {
	suite.Suite

	// Mocks
	mockCtrl       *gomock.Controller
	mockGameRepo   *gameMocks.MockRepository
	mockPlayerRepo *playerMocks.MockRepository
	mockLedgerRepo *ledgerMocks.MockRepository
	mockDiceRoller *dice.Roller
	mockClock      *clockMocks.MockClock
	mockUUID       *uuidMocks.MockUUID

	// Service under test
	service *service

	// Common test data
	testGameID     string
	testChannelID  string
	testPlayerID1  string
	testPlayerID2  string
	testPlayerID3  string
	testTime       time.Time
}

func (s *GameServiceTestSuite) SetupTest() {
	s.mockCtrl = gomock.NewController(s.T())
	s.mockGameRepo = gameMocks.NewMockRepository(s.mockCtrl)
	s.mockPlayerRepo = playerMocks.NewMockRepository(s.mockCtrl)
	s.mockLedgerRepo = ledgerMocks.NewMockRepository(s.mockCtrl)

	s.mockClock = clockMocks.NewMockClock(s.mockCtrl)
	s.mockUUID = uuidMocks.NewMockUUID(s.mockCtrl)

	// Create a deterministic dice roller for testing
	s.mockDiceRoller = dice.New(&dice.Config{Seed: 42})

	// Setup the service directly by setting fields
	s.service = &service{
		// Configuration parameters
		maxPlayers:         10,
		diceSides:          6,
		criticalHitValue:   6,
		criticalFailValue:  1,
		maxConcurrentGames: 100,

		// Repository dependencies
		gameRepo:        s.mockGameRepo,
		playerRepo:      s.mockPlayerRepo,
		drinkLedgerRepo: s.mockLedgerRepo,

		// Service dependencies
		diceRoller: s.mockDiceRoller,
		clock:      s.mockClock,
		uuid:       s.mockUUID,
	}

	// Common test data
	s.testGameID = "test-game-id"
	s.testChannelID = "test-channel-id"
	s.testPlayerID1 = "player-id-1"
	s.testPlayerID2 = "player-id-2"
	s.testPlayerID3 = "player-id-3"
	s.testTime = time.Date(2025, 3, 29, 12, 0, 0, 0, time.UTC)

	// Setup common mock behaviors
	s.mockClock.EXPECT().Now().Return(s.testTime).AnyTimes()
}

func (s *GameServiceTestSuite) TearDownTest() {
	s.mockCtrl.Finish()
}

func TestGameServiceSuite(t *testing.T) {
	suite.Run(t, new(GameServiceTestSuite))
}

// TestHandleRollOffHighestTie tests the happy path for a roll-off with players tied for highest roll
func (s *GameServiceTestSuite) TestHandleRollOffHighestTie() {
	ctx := context.Background()
	
	// Test data
	parentGameID := "parent-game-id"
	rollOffGameID := "roll-off-game-id"
	newRollOffGameID := "new-roll-off-game-id"
	rollTime := s.testTime
	
	// Create a roll-off game where both players have rolled the same value
	rollOffGame := &models.Game{
		ID:           rollOffGameID,
		ParentGameID: parentGameID,
		Status:       models.GameStatusRollOff,
		ChannelID:    s.testChannelID,
		CreatorID:    "creator-id",
		Participants: []*models.Participant{
			{
				ID:        "participant1-id",
				GameID:    rollOffGameID,
				PlayerID:  s.testPlayerID1,
				RollValue: 4, // Tied roll
				RollTime:  &rollTime,
				Status:    models.ParticipantStatusActive,
			},
			{
				ID:        "participant2-id",
				GameID:    rollOffGameID,
				PlayerID:  s.testPlayerID2,
				RollValue: 4, // Tied roll
				RollTime:  &rollTime,
				Status:    models.ParticipantStatusActive,
			},
		},
	}

	// Setup expectations
	s.mockGameRepo.EXPECT().
		GetGame(gomock.Any(), &gameRepo.GetGameInput{
			GameID: rollOffGameID,
		}).
		Return(rollOffGame, nil)

	// Expect UUIDs to be generated for the new roll-off game and participants
	s.mockUUID.EXPECT().
		NewUUID().
		Return(newRollOffGameID)

	s.mockUUID.EXPECT().
		NewUUID().
		Return("new-participant1-id")

	s.mockUUID.EXPECT().
		NewUUID().
		Return("new-participant2-id")

	// Expect the new roll-off game to be saved
	s.mockGameRepo.EXPECT().
		SaveGame(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, input *gameRepo.SaveGameInput) error {
			// Verify the new roll-off game
			s.Equal(newRollOffGameID, input.Game.ID)
			s.Equal(parentGameID, input.Game.ParentGameID) // Keep original parent
			s.Equal(models.GameStatusRollOff, input.Game.Status)
			s.Equal(s.testChannelID, input.Game.ChannelID)
			s.Equal("creator-id", input.Game.CreatorID)
			
			// Verify participants
			s.Equal(2, len(input.Game.Participants))
			s.Equal(s.testPlayerID1, input.Game.Participants[0].PlayerID)
			s.Equal(s.testPlayerID2, input.Game.Participants[1].PlayerID)
			s.Equal(models.ParticipantStatusWaitingToRoll, input.Game.Participants[0].Status)
			s.Equal(models.ParticipantStatusWaitingToRoll, input.Game.Participants[1].Status)
			
			return nil
		})

	// Call the method
	output, err := s.service.HandleRollOff(ctx, &HandleRollOffInput{
		ParentGameID:  parentGameID,
		RollOffGameID: rollOffGameID,
		PlayerIDs:     []string{s.testPlayerID1, s.testPlayerID2},
		Type:          RollOffTypeHighest,
	})

	// Verify the results
	s.NoError(err)
	s.NotNil(output)
	s.True(output.Success)
	s.True(output.NeedsAnotherRollOff)
	s.Equal([]string{s.testPlayerID1, s.testPlayerID2}, output.WinnerPlayerIDs) // Both tied
	s.Equal(newRollOffGameID, output.NextRollOffGameID)
}

// TestHandleRollOffLowestSingleLoser tests the happy path for a roll-off with a clear lowest roller
func (s *GameServiceTestSuite) TestHandleRollOffLowestSingleLoser() {
	ctx := context.Background()
	
	// Test data
	parentGameID := "parent-game-id"
	rollOffGameID := "roll-off-game-id"
	drinkID := "drink-id"
	rollTime := s.testTime
	
	// Create a roll-off game where players have different rolls
	rollOffGame := &models.Game{
		ID:           rollOffGameID,
		ParentGameID: parentGameID,
		Status:       models.GameStatusRollOff,
		ChannelID:    s.testChannelID,
		CreatorID:    "creator-id",
		Participants: []*models.Participant{
			{
				ID:        "participant1-id",
				GameID:    rollOffGameID,
				PlayerID:  s.testPlayerID1,
				RollValue: 5, // Higher roll
				RollTime:  &rollTime,
				Status:    models.ParticipantStatusActive,
			},
			{
				ID:        "participant2-id",
				GameID:    rollOffGameID,
				PlayerID:  s.testPlayerID2,
				RollValue: 3, // Lower roll - this player loses
				RollTime:  &rollTime,
				Status:    models.ParticipantStatusActive,
			},
			{
				ID:        "participant3-id",
				GameID:    rollOffGameID,
				PlayerID:  s.testPlayerID3,
				RollValue: 4, // Middle roll
				RollTime:  &rollTime,
				Status:    models.ParticipantStatusActive,
			},
		},
	}

	// Setup expectations
	s.mockGameRepo.EXPECT().
		GetGame(gomock.Any(), &gameRepo.GetGameInput{
			GameID: rollOffGameID,
		}).
		Return(rollOffGame, nil)

	// Expect a drink to be assigned to the loser
	s.mockUUID.EXPECT().
		NewUUID().
		Return(drinkID)

	s.mockLedgerRepo.EXPECT().
		AddDrinkRecord(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, input *ledgerRepo.AddDrinkRecordInput) error {
			// Verify the drink record
			s.Equal(drinkID, input.Record.ID)
			s.Equal(parentGameID, input.Record.GameID)
			s.Equal(s.testPlayerID2, input.Record.ToPlayerID) // player2 had the lowest roll
			s.Equal("", input.Record.FromPlayerID) // System-assigned
			s.Equal(models.DrinkReasonLowestRoll, input.Record.Reason)
			return nil
		})

	// Expect the roll-off game to be updated to completed
	s.mockGameRepo.EXPECT().
		SaveGame(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, input *gameRepo.SaveGameInput) error {
			// Verify the game is marked as completed
			s.Equal(models.GameStatusCompleted, input.Game.Status)
			s.Equal(rollOffGameID, input.Game.ID)
			return nil
		})

	// Call the method
	output, err := s.service.HandleRollOff(ctx, &HandleRollOffInput{
		ParentGameID:  parentGameID,
		RollOffGameID: rollOffGameID,
		PlayerIDs:     []string{s.testPlayerID1, s.testPlayerID2, s.testPlayerID3},
		Type:          RollOffTypeLowest,
	})

	// Verify the results
	s.NoError(err)
	s.NotNil(output)
	s.True(output.Success)
	s.False(output.NeedsAnotherRollOff)
	s.Equal([]string{s.testPlayerID2}, output.WinnerPlayerIDs) // player2 had the lowest roll
	s.Empty(output.NextRollOffGameID)
}