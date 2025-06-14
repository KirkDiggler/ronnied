package game

import (
	"context"
	"testing"
	"time"

	"github.com/KirkDiggler/ronnied/internal/common/clock/mocks"
	uuidMocks "github.com/KirkDiggler/ronnied/internal/common/uuid/mocks"
	diceMocks "github.com/KirkDiggler/ronnied/internal/dice/mocks"
	"github.com/KirkDiggler/ronnied/internal/models"
	ledgerRepoMocks "github.com/KirkDiggler/ronnied/internal/repositories/drink_ledger/mocks"
	gameRepo "github.com/KirkDiggler/ronnied/internal/repositories/game"
	gameRepoMocks "github.com/KirkDiggler/ronnied/internal/repositories/game/mocks"
	playerRepoMocks "github.com/KirkDiggler/ronnied/internal/repositories/player/mocks"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

// RollDiceTestSuite tests the RollDice functionality in the game orchestrator
type RollDiceTestSuite struct {
	suite.Suite

	// Test controller for mocks
	ctrl *gomock.Controller

	// Mocks
	mockGameRepo   *gameRepoMocks.MockRepository
	mockPlayerRepo *playerRepoMocks.MockRepository
	mockLedgerRepo *ledgerRepoMocks.MockRepository
	mockDiceRoller *diceMocks.MockRoller
	mockClock      *mocks.MockClock
	mockUUID       *uuidMocks.MockGenerator

	// Orchestrator instance
	orchestrator GameOrchestrator

	// Common test data
	testTime time.Time
	ctx      context.Context
}

// SetupTest initializes the test suite before each test
func (s *RollDiceTestSuite) SetupTest() {
	s.ctrl = gomock.NewController(s.T())
	s.mockGameRepo = gameRepoMocks.NewMockRepository(s.ctrl)
	s.mockPlayerRepo = playerRepoMocks.NewMockRepository(s.ctrl)
	s.mockLedgerRepo = ledgerRepoMocks.NewMockRepository(s.ctrl)
	s.mockDiceRoller = diceMocks.NewMockRoller(s.ctrl)
	s.mockClock = mocks.NewMockClock(s.ctrl)
	s.mockUUID = uuidMocks.NewMockGenerator(s.ctrl)
	s.testTime = time.Date(2025, 5, 17, 20, 0, 0, 0, time.UTC)
	s.ctx = context.Background()

	// Initialize orchestrator with mocks
	orch, err := NewOrchestrator(&Config{
		MaxPlayers:         10,
		DiceSides:          6,
		CriticalHitValue:   6,
		CriticalFailValue:  1,
		MaxConcurrentGames: 5,
		GameRepo:           s.mockGameRepo,
		PlayerRepo:         s.mockPlayerRepo,
		DrinkLedgerRepo:    s.mockLedgerRepo,
		DiceRoller:         s.mockDiceRoller,
		Clock:              s.mockClock,
		UUIDGenerator:      s.mockUUID,
	})
	s.NoError(err)
	s.orchestrator = orch
}

// TearDownTest cleans up after each test
func (s *RollDiceTestSuite) TearDownTest() {
	s.ctrl.Finish()
}

// TestRollDiceInMainGame tests rolling dice in a main game (not a roll-off)
func (s *RollDiceTestSuite) TestRollDiceInMainGame() {
	// Setup test data
	gameID := "game-1"
	playerID := "player-1"
	rollValue := 4

	// Create a game in active state
	game := &models.Game{
		ID:     gameID,
		Status: models.GameStatusActive,
		Participants: []*models.Participant{
			{
				PlayerID: playerID,
				Status:   models.ParticipantStatusActive,
			},
		},
	}

	// Mock getting the game
	s.mockGameRepo.EXPECT().GetGame(gomock.Any(), &gameRepo.GetGameInput{
		GameID: gameID,
	}).Return(game, nil)

	// Mock rolling the dice
	s.mockDiceRoller.EXPECT().Roll(6).Return(rollValue)

	// Mock the current time
	s.mockClock.EXPECT().Now().Return(s.testTime)

	// Mock saving the game
	s.mockGameRepo.EXPECT().SaveGame(gomock.Any(), gomock.Any()).Return(nil)

	// Call the function being tested
	output, err := s.orchestrator.RollDice(s.ctx, &RollDiceInput{
		GameID:   gameID,
		PlayerID: playerID,
	})

	// Verify results
	s.NoError(err)
	s.NotNil(output)
	s.Equal(playerID, output.PlayerID)
	s.Equal(rollValue, output.RollValue)
	s.False(output.IsRollOffRoll)
	s.Equal(game, output.Game)
}

// TestRollDiceInRollOff tests rolling dice in a roll-off game
func (s *RollDiceTestSuite) TestRollDiceInRollOff() {
	// Setup test data
	gameID := "game-1"
	playerID := "player-1"
	rollValue := 5

	// Create a game in roll-off state
	game := &models.Game{
		ID:              gameID,
		Status:          models.GameStatusRollOff,
		RollOffPlayerIDs: []string{playerID, "player-2"},
		Participants: []*models.Participant{
			{
				PlayerID: playerID,
				Status:   models.ParticipantStatusInRollOff,
			},
			{
				PlayerID: "player-2",
				Status:   models.ParticipantStatusInRollOff,
			},
		},
	}

	// Mock getting the game
	s.mockGameRepo.EXPECT().GetGame(gomock.Any(), &gameRepo.GetGameInput{
		GameID: gameID,
	}).Return(game, nil)

	// Mock rolling the dice
	s.mockDiceRoller.EXPECT().Roll(6).Return(rollValue)

	// Mock the current time
	s.mockClock.EXPECT().Now().Return(s.testTime)

	// Mock saving the game
	s.mockGameRepo.EXPECT().SaveGame(gomock.Any(), gomock.Any()).Return(nil)

	// Call the function being tested
	output, err := s.orchestrator.RollDice(s.ctx, &RollDiceInput{
		GameID:   gameID,
		PlayerID: playerID,
	})

	// Verify results
	s.NoError(err)
	s.NotNil(output)
	s.Equal(playerID, output.PlayerID)
	s.Equal(rollValue, output.RollValue)
	s.True(output.IsRollOffRoll)
	s.Equal(game, output.Game)
}

// TestPlayerNotInGame tests rolling dice when player is not in the game
func (s *RollDiceTestSuite) TestPlayerNotInGame() {
	// Setup test data
	gameID := "game-1"
	playerID := "player-not-in-game"

	// Create a game without the player
	game := &models.Game{
		ID:     gameID,
		Status: models.GameStatusActive,
		Participants: []*models.Participant{
			{
				PlayerID: "other-player",
				Status:   models.ParticipantStatusActive,
			},
		},
	}

	// Mock getting the game
	s.mockGameRepo.EXPECT().GetGame(gomock.Any(), &gameRepo.GetGameInput{
		GameID: gameID,
	}).Return(game, nil)

	// Call the function being tested
	output, err := s.orchestrator.RollDice(s.ctx, &RollDiceInput{
		GameID:   gameID,
		PlayerID: playerID,
	})

	// Verify results
	s.Error(err)
	s.Equal(ErrPlayerNotInGame, err)
	s.Nil(output)
}

// TestRunRollDiceTests runs the roll dice test suite
func TestRunRollDiceTests(t *testing.T) {
	suite.Run(t, new(RollDiceTestSuite))
}
