package game

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/KirkDiggler/ronnied/internal/common/clock/mocks"
	uuidMocks "github.com/KirkDiggler/ronnied/internal/common/uuid/mocks"
	diceMocks "github.com/KirkDiggler/ronnied/internal/dice/mocks"
	"github.com/KirkDiggler/ronnied/internal/models"
	ledgerRepo "github.com/KirkDiggler/ronnied/internal/repositories/drink_ledger"
	ledgerMocks "github.com/KirkDiggler/ronnied/internal/repositories/drink_ledger/mocks"
	gameRepo "github.com/KirkDiggler/ronnied/internal/repositories/game"
	gameMocks "github.com/KirkDiggler/ronnied/internal/repositories/game/mocks"
	playerRepo "github.com/KirkDiggler/ronnied/internal/repositories/player"
	playerMocks "github.com/KirkDiggler/ronnied/internal/repositories/player/mocks"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

type GameServiceTestSuite struct {
	suite.Suite
	mockCtrl       *gomock.Controller
	mockGameRepo   *gameMocks.MockRepository
	mockPlayerRepo *playerMocks.MockRepository
	mockDrinkRepo  *ledgerMocks.MockRepository
	mockDiceRoller *diceMocks.MockRoller
	mockClock      *mocks.MockClock
	mockUUID       *uuidMocks.MockUUID
	gameService    Service
	ctx            context.Context

	// Test data
	testTime          time.Time
	testGameID        string
	testChannelID     string
	testCreatorID     string
	testCreatorName   string
	testParticipantID string
	testPlayerID      string
	testPlayerName    string
	testSessionID     string

	// Reusable test fixtures
	expectedGame           *models.Game
	expectedParticipant    *models.Participant
	expectedActiveGame     *models.Game
	expectedGameWithPlayer *models.Game
	expectedPlayer         *models.Player
	expectedSession        *models.Session

	// Reusable test inputs
	createGameInput *CreateGameInput
	startGameInput  *StartGameInput
	joinGameInput   *JoinGameInput
	rollDiceInput   *RollDiceInput
}

func (s *GameServiceTestSuite) SetupTest() {
	s.mockCtrl = gomock.NewController(s.T())
	s.mockGameRepo = gameMocks.NewMockRepository(s.mockCtrl)
	s.mockPlayerRepo = playerMocks.NewMockRepository(s.mockCtrl)
	s.mockDrinkRepo = ledgerMocks.NewMockRepository(s.mockCtrl)
	s.mockDiceRoller = diceMocks.NewMockRoller(s.mockCtrl)
	s.mockClock = mocks.NewMockClock(s.mockCtrl)
	s.mockUUID = uuidMocks.NewMockUUID(s.mockCtrl)

	s.ctx = context.Background()

	// Initialize test data
	s.testTime = time.Date(2025, 4, 19, 12, 0, 0, 0, time.UTC)
	s.testGameID = "test-game-id"
	s.testChannelID = "test-channel-id"
	s.testCreatorID = "test-creator-id"
	s.testCreatorName = "Test Creator"
	s.testParticipantID = "test-participant-id"
	s.testPlayerID = "test-player-id"
	s.testPlayerName = "Test Player"
	s.testSessionID = "test-session-id"

	// Set up the clock mock to return our test time
	s.mockClock.EXPECT().Now().Return(s.testTime).AnyTimes()

	// Initialize reusable test fixtures
	s.expectedParticipant = &models.Participant{
		ID:         s.testParticipantID,
		GameID:     s.testGameID,
		PlayerID:   s.testCreatorID,
		PlayerName: s.testCreatorName,
		Status:     models.ParticipantStatusWaitingToRoll,
	}

	// Basic game with no participants
	s.expectedGame = &models.Game{
		ID:           s.testGameID,
		ChannelID:    s.testChannelID,
		CreatorID:    s.testCreatorID,
		Status:       models.GameStatusWaiting,
		CreatedAt:    s.testTime,
		UpdatedAt:    s.testTime,
		Participants: []*models.Participant{},
	}

	// Game with creator as participant
	s.expectedGameWithPlayer = &models.Game{
		ID:           s.testGameID,
		ChannelID:    s.testChannelID,
		CreatorID:    s.testCreatorID,
		Status:       models.GameStatusWaiting,
		CreatedAt:    s.testTime,
		UpdatedAt:    s.testTime,
		Participants: []*models.Participant{s.expectedParticipant},
	}

	// Game in active state
	s.expectedActiveGame = &models.Game{
		ID:           s.testGameID,
		ChannelID:    s.testChannelID,
		CreatorID:    s.testCreatorID,
		Status:       models.GameStatusActive,
		CreatedAt:    s.testTime,
		UpdatedAt:    s.testTime,
		Participants: []*models.Participant{s.expectedParticipant},
	}

	// Player model
	s.expectedPlayer = &models.Player{
		ID:            s.testPlayerID,
		Name:          s.testPlayerName,
		CurrentGameID: "",
		LastRoll:      0,
		LastRollTime:  s.testTime,
	}

	// Session model
	s.expectedSession = &models.Session{
		ID:        s.testSessionID,
		ChannelID: s.testChannelID,
		CreatedAt: s.testTime,
		CreatedBy: "system",
		Active:    true,
	}

	// Initialize reusable test inputs
	s.createGameInput = &CreateGameInput{
		ChannelID:   s.testChannelID,
		CreatorID:   s.testCreatorID,
		CreatorName: s.testCreatorName,
	}

	s.startGameInput = &StartGameInput{
		GameID:   s.testGameID,
		PlayerID: s.testCreatorID,
	}

	s.joinGameInput = &JoinGameInput{
		GameID:     s.testGameID,
		PlayerID:   s.testPlayerID,
		PlayerName: s.testPlayerName,
	}

	s.rollDiceInput = &RollDiceInput{
		GameID:   s.testGameID,
		PlayerID: s.testCreatorID,
	}

	// Create the service with mocked dependencies
	cfg := &Config{
		GameRepo:          s.mockGameRepo,
		PlayerRepo:        s.mockPlayerRepo,
		DrinkLedgerRepo:   s.mockDrinkRepo,
		DiceRoller:        s.mockDiceRoller,
		Clock:             s.mockClock,
		UUIDGenerator:     s.mockUUID,
		MaxPlayers:        10, // Set a max players value for testing
		DiceSides:         6,  // Standard dice
		CriticalHitValue:  6,  // Critical hit on 6
		CriticalFailValue: 1,  // Critical fail on 1
	}

	var err error
	svc, err := New(cfg)
	s.Require().NoError(err)
	s.gameService = svc
}

func (s *GameServiceTestSuite) TearDownTest() {
	s.mockCtrl.Finish()
}

// setupSessionExpectations sets up the expectations for session-related calls
func (s *GameServiceTestSuite) setupSessionExpectations() {
	// Expect GetCurrentSession to be called for the channel
	s.mockDrinkRepo.EXPECT().
		GetCurrentSession(gomock.Any(), &ledgerRepo.GetCurrentSessionInput{
			ChannelID: s.testChannelID,
		}).
		Return(&ledgerRepo.GetCurrentSessionOutput{
			Session: s.expectedSession,
		}, nil).
		AnyTimes() // Use AnyTimes since multiple methods might call this
}

func (s *GameServiceTestSuite) TestCreateGame_HappyPath() {
	// Expect CreateGame to be called on the game repository
	s.mockGameRepo.EXPECT().
		CreateGame(gomock.Any(), &gameRepo.CreateGameInput{
			ChannelID: s.testChannelID,
			CreatorID: s.testCreatorID,
			Status:    models.GameStatusWaiting,
		}).
		Return(&gameRepo.CreateGameOutput{Game: s.expectedGame}, nil)

	// Expect CreateParticipant to be called on the game repository
	s.mockGameRepo.EXPECT().
		CreateParticipant(gomock.Any(), &gameRepo.CreateParticipantInput{
			GameID:     s.testGameID,
			PlayerID:   s.testCreatorID,
			PlayerName: s.testCreatorName,
			Status:     models.ParticipantStatusWaitingToRoll,
		}).
		Return(&gameRepo.CreateParticipantOutput{Participant: s.expectedParticipant}, nil)

	// Act
	output, err := s.gameService.CreateGame(s.ctx, s.createGameInput)

	// Assert
	s.Require().NoError(err)
	s.Require().NotNil(output)
	s.Equal(s.testGameID, output.GameID)
}

func (s *GameServiceTestSuite) TestCreateGame_CreateGameError() {
	expectedError := errors.New("failed to create game")

	// Expect CreateGame to be called on the game repository and return an error
	s.mockGameRepo.EXPECT().
		CreateGame(gomock.Any(), &gameRepo.CreateGameInput{
			ChannelID: s.testChannelID,
			CreatorID: s.testCreatorID,
			Status:    models.GameStatusWaiting,
		}).
		Return(nil, expectedError)

	// Act
	output, err := s.gameService.CreateGame(s.ctx, s.createGameInput)

	// Assert
	s.Require().Error(err)
	s.Equal(expectedError, err)
	s.Nil(output)
}

func (s *GameServiceTestSuite) TestCreateGame_CreateParticipantError() {
	expectedError := errors.New("failed to create participant")

	// Expect CreateGame to be called on the game repository
	s.mockGameRepo.EXPECT().
		CreateGame(gomock.Any(), &gameRepo.CreateGameInput{
			ChannelID: s.testChannelID,
			CreatorID: s.testCreatorID,
			Status:    models.GameStatusWaiting,
		}).
		Return(&gameRepo.CreateGameOutput{Game: s.expectedGame}, nil)

	// Expect CreateParticipant to be called on the game repository and return an error
	s.mockGameRepo.EXPECT().
		CreateParticipant(gomock.Any(), &gameRepo.CreateParticipantInput{
			GameID:     s.testGameID,
			PlayerID:   s.testCreatorID,
			PlayerName: s.testCreatorName,
			Status:     models.ParticipantStatusWaitingToRoll,
		}).
		Return(nil, expectedError)

	// Act
	output, err := s.gameService.CreateGame(s.ctx, s.createGameInput)

	// Assert
	s.Require().Error(err)
	s.Equal(expectedError, err)
	s.Nil(output)
}

// StartGame Tests

func (s *GameServiceTestSuite) TestStartGame_HappyPath() {
	// Expect GetGame to be called on the game repository
	s.mockGameRepo.EXPECT().
		GetGame(gomock.Any(), &gameRepo.GetGameInput{
			GameID: s.testGameID,
		}).
		Return(s.expectedGameWithPlayer, nil)

	// Expect SaveGame to be called with the updated game
	s.mockGameRepo.EXPECT().
		SaveGame(gomock.Any(), &gameRepo.SaveGameInput{
			Game: &models.Game{
				ID:           s.testGameID,
				ChannelID:    s.testChannelID,
				CreatorID:    s.testCreatorID,
				Status:       models.GameStatusActive,
				CreatedAt:    s.testTime,
				UpdatedAt:    s.testTime,
				Participants: []*models.Participant{s.expectedParticipant},
			},
		}).
		Return(nil)

	// Act
	output, err := s.gameService.StartGame(s.ctx, s.startGameInput)

	// Assert
	s.Require().NoError(err)
	s.Require().NotNil(output)
	s.True(output.Success)
}

func (s *GameServiceTestSuite) TestStartGame_GameNotFound() {
	// Expect GetGame to be called and return an error
	s.mockGameRepo.EXPECT().
		GetGame(gomock.Any(), &gameRepo.GetGameInput{
			GameID: s.testGameID,
		}).
		Return(nil, errors.New("game not found"))

	// Act
	output, err := s.gameService.StartGame(s.ctx, s.startGameInput)

	// Assert
	s.Require().Error(err)
	s.True(errors.Is(err, ErrGameNotFound))
	s.Nil(output)
}

func (s *GameServiceTestSuite) TestStartGame_NotCreator() {
	// Create a different player ID for this test
	notCreatorInput := &StartGameInput{
		GameID:   s.testGameID,
		PlayerID: "not-creator-id",
	}

	// Expect GetGame to be called on the game repository
	s.mockGameRepo.EXPECT().
		GetGame(gomock.Any(), &gameRepo.GetGameInput{
			GameID: s.testGameID,
		}).
		Return(s.expectedGameWithPlayer, nil)

	// Act
	output, err := s.gameService.StartGame(s.ctx, notCreatorInput)

	// Assert
	s.Require().Error(err)
	s.Equal(ErrNotCreator, err)
	s.Nil(output)
}

func (s *GameServiceTestSuite) TestStartGame_InvalidGameState() {
	// Create a game that's already active
	activeGame := &models.Game{
		ID:           s.testGameID,
		ChannelID:    s.testChannelID,
		CreatorID:    s.testCreatorID,
		Status:       models.GameStatusActive,
		CreatedAt:    s.testTime,
		UpdatedAt:    s.testTime,
		Participants: []*models.Participant{s.expectedParticipant},
	}

	// Expect GetGame to be called on the game repository
	s.mockGameRepo.EXPECT().
		GetGame(gomock.Any(), &gameRepo.GetGameInput{
			GameID: s.testGameID,
		}).
		Return(activeGame, nil)

	// Act
	output, err := s.gameService.StartGame(s.ctx, s.startGameInput)

	// Assert
	s.Require().Error(err)
	s.True(errors.Is(err, ErrInvalidGameState))
	s.Nil(output)
}

func (s *GameServiceTestSuite) TestStartGame_NoPlayers() {
	// Create a game with no participants
	emptyGame := &models.Game{
		ID:           s.testGameID,
		ChannelID:    s.testChannelID,
		CreatorID:    s.testCreatorID,
		Status:       models.GameStatusWaiting,
		CreatedAt:    s.testTime,
		UpdatedAt:    s.testTime,
		Participants: []*models.Participant{},
	}

	// Expect GetGame to be called on the game repository
	s.mockGameRepo.EXPECT().
		GetGame(gomock.Any(), &gameRepo.GetGameInput{
			GameID: s.testGameID,
		}).
		Return(emptyGame, nil)

	// Act
	output, err := s.gameService.StartGame(s.ctx, s.startGameInput)

	// Assert
	s.Require().Error(err)
	s.Equal(ErrNotEnoughPlayers, err)
	s.Nil(output)
}

func (s *GameServiceTestSuite) TestStartGame_SaveGameError() {
	expectedError := errors.New("failed to save game")

	// Expect GetGame to be called on the game repository
	s.mockGameRepo.EXPECT().
		GetGame(gomock.Any(), &gameRepo.GetGameInput{
			GameID: s.testGameID,
		}).
		Return(s.expectedGameWithPlayer, nil)

	// Expect SaveGame to be called and return an error
	s.mockGameRepo.EXPECT().
		SaveGame(gomock.Any(), &gameRepo.SaveGameInput{
			Game: &models.Game{
				ID:           s.testGameID,
				ChannelID:    s.testChannelID,
				CreatorID:    s.testCreatorID,
				Status:       models.GameStatusActive,
				CreatedAt:    s.testTime,
				UpdatedAt:    s.testTime,
				Participants: []*models.Participant{s.expectedParticipant},
			},
		}).
		DoAndReturn(func(_ context.Context, input *gameRepo.SaveGameInput) error {
			// Verify the game is marked as completed
			s.Equal(models.GameStatusActive, input.Game.Status)
			return expectedError
		}).
		Return(expectedError)

	// Act
	output, err := s.gameService.StartGame(s.ctx, s.startGameInput)

	// Assert
	s.Require().Error(err)
	s.Equal(expectedError, err)
	s.Nil(output)
}

// JoinGame Tests

func (s *GameServiceTestSuite) TestJoinGame_HappyPath() {
	// Expect GetGame to be called on the game repository
	s.mockGameRepo.EXPECT().
		GetGame(gomock.Any(), &gameRepo.GetGameInput{
			GameID: s.testGameID,
		}).
		Return(s.expectedGame, nil)

	// Expect GetPlayer to be called on the player repository
	s.mockPlayerRepo.EXPECT().
		GetPlayer(gomock.Any(), &playerRepo.GetPlayerInput{
			PlayerID: s.testPlayerID,
		}).
		Return(nil, errors.New("player not found"))

	// Expect SavePlayer to be called on the player repository
	s.mockPlayerRepo.EXPECT().
		SavePlayer(gomock.Any(), &playerRepo.SavePlayerInput{
			Player: &models.Player{
				ID:            s.testPlayerID,
				Name:          s.testPlayerName,
				CurrentGameID: s.testGameID,
				LastRoll:      0,
				LastRollTime:  s.testTime,
			},
		}).
		Return(nil)

	// Expect CreateParticipant to be called on the game repository
	s.mockGameRepo.EXPECT().
		CreateParticipant(gomock.Any(), &gameRepo.CreateParticipantInput{
			GameID:     s.testGameID,
			PlayerID:   s.testPlayerID,
			PlayerName: s.testPlayerName,
			Status:     models.ParticipantStatusWaitingToRoll,
		}).
		Return(&gameRepo.CreateParticipantOutput{
			Participant: &models.Participant{
				ID:         "new-participant-id",
				GameID:     s.testGameID,
				PlayerID:   s.testPlayerID,
				PlayerName: s.testPlayerName,
				Status:     models.ParticipantStatusWaitingToRoll,
			},
		}, nil)

	// Act
	output, err := s.gameService.JoinGame(s.ctx, s.joinGameInput)

	// Assert
	s.Require().NoError(err)
	s.Require().NotNil(output)
	s.True(output.Success)
	s.False(output.AlreadyJoined)
}

func (s *GameServiceTestSuite) TestJoinGame_PlayerAlreadyInGame() {
	// Create a game with the test player already in it
	gameWithPlayer := &models.Game{
		ID:        s.testGameID,
		ChannelID: s.testChannelID,
		CreatorID: s.testCreatorID,
		Status:    models.GameStatusWaiting,
		CreatedAt: s.testTime,
		UpdatedAt: s.testTime,
		Participants: []*models.Participant{
			{
				ID:         "existing-participant-id",
				GameID:     s.testGameID,
				PlayerID:   s.testPlayerID,
				PlayerName: s.testPlayerName,
				Status:     models.ParticipantStatusWaitingToRoll,
			},
		},
	}

	// Expect GetGame to be called on the game repository
	s.mockGameRepo.EXPECT().
		GetGame(gomock.Any(), &gameRepo.GetGameInput{
			GameID: s.testGameID,
		}).
		Return(gameWithPlayer, nil)

	// Act
	output, err := s.gameService.JoinGame(s.ctx, s.joinGameInput)

	// Assert
	s.Require().NoError(err)
	s.Require().NotNil(output)
	s.True(output.Success)
	s.True(output.AlreadyJoined)
}

func (s *GameServiceTestSuite) TestJoinGame_GameNotFound() {
	// Expect GetGame to be called on the game repository and return an error
	s.mockGameRepo.EXPECT().
		GetGame(gomock.Any(), &gameRepo.GetGameInput{
			GameID: s.testGameID,
		}).
		Return(nil, errors.New("game not found"))

	// Act
	output, err := s.gameService.JoinGame(s.ctx, s.joinGameInput)

	// Assert
	s.Require().Error(err)
	s.True(errors.Is(err, ErrGameNotFound))
	s.Nil(output)
}

func (s *GameServiceTestSuite) TestJoinGame_GameActive() {
	// Expect GetGame to be called on the game repository
	s.mockGameRepo.EXPECT().
		GetGame(gomock.Any(), &gameRepo.GetGameInput{
			GameID: s.testGameID,
		}).
		Return(s.expectedActiveGame, nil)

	// Act
	output, err := s.gameService.JoinGame(s.ctx, s.joinGameInput)

	// Assert
	s.Require().Error(err)
	s.True(errors.Is(err, ErrGameActive))
	s.Nil(output)
}

func (s *GameServiceTestSuite) TestJoinGame_GameFull() {
	// Create a game with max players
	participants := make([]*models.Participant, 10) // Matches MaxPlayers in config
	for i := 0; i < 10; i++ {
		participants[i] = &models.Participant{
			ID:         fmt.Sprintf("participant-%d", i),
			GameID:     s.testGameID,
			PlayerID:   fmt.Sprintf("player-%d", i),
			PlayerName: fmt.Sprintf("Player %d", i),
			Status:     models.ParticipantStatusWaitingToRoll,
		}
	}

	fullGame := &models.Game{
		ID:           s.testGameID,
		ChannelID:    s.testChannelID,
		CreatorID:    s.testCreatorID,
		Status:       models.GameStatusWaiting,
		CreatedAt:    s.testTime,
		UpdatedAt:    s.testTime,
		Participants: participants,
	}

	// Expect GetGame to be called on the game repository
	s.mockGameRepo.EXPECT().
		GetGame(gomock.Any(), &gameRepo.GetGameInput{
			GameID: s.testGameID,
		}).
		Return(fullGame, nil)

	// Act
	output, err := s.gameService.JoinGame(s.ctx, s.joinGameInput)

	// Assert
	s.Require().Error(err)
	s.True(errors.Is(err, ErrGameFull))
	s.Nil(output)
}

func (s *GameServiceTestSuite) TestJoinGame_ExistingPlayerWithNoGame() {
	// Expect GetGame to be called on the game repository
	s.mockGameRepo.EXPECT().
		GetGame(gomock.Any(), &gameRepo.GetGameInput{
			GameID: s.testGameID,
		}).
		Return(s.expectedGame, nil)

	// Create a player with no current game
	playerWithNoGame := &models.Player{
		ID:            s.testPlayerID,
		Name:          s.testPlayerName,
		CurrentGameID: "",
		LastRoll:      0,
		LastRollTime:  s.testTime,
	}

	// Expect GetPlayer to be called on the player repository
	s.mockPlayerRepo.EXPECT().
		GetPlayer(gomock.Any(), &playerRepo.GetPlayerInput{
			PlayerID: s.testPlayerID,
		}).
		Return(playerWithNoGame, nil)

	// Expect SavePlayer to be called on the player repository
	s.mockPlayerRepo.EXPECT().
		SavePlayer(gomock.Any(), &playerRepo.SavePlayerInput{
			Player: &models.Player{
				ID:            s.testPlayerID,
				Name:          s.testPlayerName,
				CurrentGameID: s.testGameID,
				LastRoll:      0,
				LastRollTime:  s.testTime,
			},
		}).
		Return(nil)

	// Expect CreateParticipant to be called on the game repository
	s.mockGameRepo.EXPECT().
		CreateParticipant(gomock.Any(), &gameRepo.CreateParticipantInput{
			GameID:     s.testGameID,
			PlayerID:   s.testPlayerID,
			PlayerName: s.testPlayerName,
			Status:     models.ParticipantStatusWaitingToRoll,
		}).
		Return(&gameRepo.CreateParticipantOutput{
			Participant: &models.Participant{
				ID:         "new-participant-id",
				GameID:     s.testGameID,
				PlayerID:   s.testPlayerID,
				PlayerName: s.testPlayerName,
				Status:     models.ParticipantStatusWaitingToRoll,
			},
		}, nil)

	// Act
	output, err := s.gameService.JoinGame(s.ctx, s.joinGameInput)

	// Assert
	s.Require().NoError(err)
	s.Require().NotNil(output)
	s.True(output.Success)
	s.False(output.AlreadyJoined)
}

func (s *GameServiceTestSuite) TestJoinGame_ExistingPlayerWithDifferentGame() {
	// Expect GetGame to be called on the game repository
	s.mockGameRepo.EXPECT().
		GetGame(gomock.Any(), &gameRepo.GetGameInput{
			GameID: s.testGameID,
		}).
		Return(s.expectedGame, nil)

	// Create a player with a different current game
	playerWithDifferentGame := &models.Player{
		ID:            s.testPlayerID,
		Name:          s.testPlayerName,
		CurrentGameID: "different-game-id",
		LastRoll:      0,
		LastRollTime:  s.testTime,
	}

	// Expect GetPlayer to be called on the player repository
	s.mockPlayerRepo.EXPECT().
		GetPlayer(gomock.Any(), &playerRepo.GetPlayerInput{
			PlayerID: s.testPlayerID,
		}).
		Return(playerWithDifferentGame, nil)

	// Expect UpdatePlayerGame to be called on the player repository
	s.mockPlayerRepo.EXPECT().
		UpdatePlayerGame(gomock.Any(), &playerRepo.UpdatePlayerGameInput{
			PlayerID: s.testPlayerID,
			GameID:   s.testGameID,
		}).
		Return(nil)

	// Expect CreateParticipant to be called on the game repository
	s.mockGameRepo.EXPECT().
		CreateParticipant(gomock.Any(), &gameRepo.CreateParticipantInput{
			GameID:     s.testGameID,
			PlayerID:   s.testPlayerID,
			PlayerName: s.testPlayerName,
			Status:     models.ParticipantStatusWaitingToRoll,
		}).
		Return(&gameRepo.CreateParticipantOutput{
			Participant: &models.Participant{
				ID:         "new-participant-id",
				GameID:     s.testGameID,
				PlayerID:   s.testPlayerID,
				PlayerName: s.testPlayerName,
				Status:     models.ParticipantStatusWaitingToRoll,
			},
		}, nil)

	// Act
	output, err := s.gameService.JoinGame(s.ctx, s.joinGameInput)

	// Assert
	s.Require().NoError(err)
	s.Require().NotNil(output)
	s.True(output.Success)
	s.False(output.AlreadyJoined)
}

func (s *GameServiceTestSuite) TestJoinGame_SavePlayerError() {
	// Expect GetGame to be called on the game repository
	s.mockGameRepo.EXPECT().
		GetGame(gomock.Any(), &gameRepo.GetGameInput{
			GameID: s.testGameID,
		}).
		Return(s.expectedGame, nil)

	// Expect GetPlayer to be called on the player repository
	s.mockPlayerRepo.EXPECT().
		GetPlayer(gomock.Any(), &playerRepo.GetPlayerInput{
			PlayerID: s.testPlayerID,
		}).
		Return(nil, errors.New("player not found"))

	// Expect SavePlayer to be called on the player repository and return an error
	expectedError := errors.New("failed to save player")
	s.mockPlayerRepo.EXPECT().
		SavePlayer(gomock.Any(), gomock.Any()).
		Return(expectedError)

	// Act
	output, err := s.gameService.JoinGame(s.ctx, s.joinGameInput)

	// Assert
	s.Require().Error(err)
	s.Equal(expectedError, err)
	s.Nil(output)
}

func (s *GameServiceTestSuite) TestJoinGame_CreateParticipantError() {
	// Expect GetGame to be called on the game repository
	s.mockGameRepo.EXPECT().
		GetGame(gomock.Any(), &gameRepo.GetGameInput{
			GameID: s.testGameID,
		}).
		Return(s.expectedGame, nil)

	// Expect GetPlayer to be called on the player repository
	s.mockPlayerRepo.EXPECT().
		GetPlayer(gomock.Any(), &playerRepo.GetPlayerInput{
			PlayerID: s.testPlayerID,
		}).
		Return(nil, errors.New("player not found"))

	// Expect SavePlayer to be called on the player repository
	s.mockPlayerRepo.EXPECT().
		SavePlayer(gomock.Any(), gomock.Any()).
		Return(nil)

	// Expect CreateParticipant to be called on the game repository and return an error
	expectedError := errors.New("failed to create participant")
	s.mockGameRepo.EXPECT().
		CreateParticipant(gomock.Any(), gomock.Any()).
		Return(nil, expectedError)

	// Act
	output, err := s.gameService.JoinGame(s.ctx, s.joinGameInput)

	// Assert
	s.Require().Error(err)
	s.Equal(expectedError, err)
	s.Nil(output)
}

// RollDice Tests

func (s *GameServiceTestSuite) TestRollDice_RegularRoll() {
	// Create an active game with multiple participants, one who hasn't rolled yet
	activeGame := &models.Game{
		ID:        s.testGameID,
		ChannelID: s.testChannelID,
		CreatorID: s.testCreatorID,
		Status:    models.GameStatusActive,
		CreatedAt: s.testTime,
		UpdatedAt: s.testTime,
		Participants: []*models.Participant{
			{
				ID:         s.testParticipantID,
				GameID:     s.testGameID,
				PlayerID:   s.testCreatorID,
				PlayerName: s.testCreatorName,
				Status:     models.ParticipantStatusWaitingToRoll,
				RollValue:  0,
				RollTime:   nil, // Hasn't rolled yet
			},
			{
				ID:         "another-participant-id",
				GameID:     s.testGameID,
				PlayerID:   s.testPlayerID,
				PlayerName: s.testPlayerName,
				Status:     models.ParticipantStatusWaitingToRoll,
				RollValue:  0,
				RollTime:   nil, // Hasn't rolled yet
			},
		},
	}

	// Expect GetGame to be called on the game repository
	s.mockGameRepo.EXPECT().
		GetGame(gomock.Any(), &gameRepo.GetGameInput{
			GameID: s.testGameID,
		}).
		Return(activeGame, nil)

	// Expect Roll to be called on the dice roller
	s.mockDiceRoller.EXPECT().
		Roll(6). // 6-sided dice
		Return(3)

	// Expect SaveGame to be called with the updated game
	s.mockGameRepo.EXPECT().
		SaveGame(gomock.Any(), &gameRepo.SaveGameInput{
			Game: &models.Game{
				ID:        s.testGameID,
				ChannelID: s.testChannelID,
				CreatorID: s.testCreatorID,
				Status:    models.GameStatusActive,
				CreatedAt: s.testTime,
				UpdatedAt: s.testTime,
				Participants: []*models.Participant{
					{
						ID:         s.testParticipantID,
						GameID:     s.testGameID,
						PlayerID:   s.testCreatorID,
						PlayerName: s.testCreatorName,
						Status:     models.ParticipantStatusActive,
						RollValue:  3,
						RollTime:   &s.testTime,
					},
					{
						ID:         "another-participant-id",
						GameID:     s.testGameID,
						PlayerID:   s.testPlayerID,
						PlayerName: s.testPlayerName,
						Status:     models.ParticipantStatusWaitingToRoll,
						RollValue:  0,
						RollTime:   nil,
					},
				},
			},
		}).
		DoAndReturn(func(_ context.Context, input *gameRepo.SaveGameInput) error {
			// Verify the game is marked as completed
			s.Equal(models.GameStatusActive, input.Game.Status)
			return nil
		}).
		Return(nil)

	// Act
	output, err := s.gameService.RollDice(s.ctx, s.rollDiceInput)

	// Assert
	s.Require().NoError(err)
	s.Require().NotNil(output)
	s.Equal(3, output.Value)
	s.Equal(3, output.RollValue)
	s.Equal(s.testCreatorID, output.PlayerID)
	s.Equal(s.testCreatorName, output.PlayerName)
	s.False(output.IsCriticalHit)
	s.False(output.IsCriticalFail)
	s.False(output.IsLowestRoll)
	s.False(output.NeedsRollOff)
	s.False(output.AllPlayersRolled) // Now this should be false since not all players have rolled
}

func (s *GameServiceTestSuite) TestRollDice_CriticalHit() {
	// Create an active game with multiple participants, one who hasn't rolled yet
	activeGame := &models.Game{
		ID:        s.testGameID,
		ChannelID: s.testChannelID,
		CreatorID: s.testCreatorID,
		Status:    models.GameStatusActive,
		CreatedAt: s.testTime,
		UpdatedAt: s.testTime,
		Participants: []*models.Participant{
			{
				ID:         s.testParticipantID,
				GameID:     s.testGameID,
				PlayerID:   s.testCreatorID,
				PlayerName: s.testCreatorName,
				Status:     models.ParticipantStatusWaitingToRoll,
				RollValue:  0,
				RollTime:   nil, // Hasn't rolled yet
			},
			{
				ID:         "another-participant-id",
				GameID:     s.testGameID,
				PlayerID:   s.testPlayerID,
				PlayerName: s.testPlayerName,
				Status:     models.ParticipantStatusWaitingToRoll,
				RollValue:  0,
				RollTime:   nil, // Hasn't rolled yet
			},
			{
				ID:         "third-participant-id",
				GameID:     s.testGameID,
				PlayerID:   "third-player-id",
				PlayerName: "Third Player",
				Status:     models.ParticipantStatusWaitingToRoll,
				RollValue:  0,
				RollTime:   nil, // Hasn't rolled yet
			},
		},
	}

	// Expect GetGame to be called on the game repository
	s.mockGameRepo.EXPECT().
		GetGame(gomock.Any(), &gameRepo.GetGameInput{
			GameID: s.testGameID,
		}).
		Return(activeGame, nil)

	// Expect Roll to be called on the dice roller and return a critical hit
	s.mockDiceRoller.EXPECT().
		Roll(6). // 6-sided dice
		Return(6)

	// Expect SaveGame to be called with the updated game
	s.mockGameRepo.EXPECT().
		SaveGame(gomock.Any(), &gameRepo.SaveGameInput{
			Game: &models.Game{
				ID:        s.testGameID,
				ChannelID: s.testChannelID,
				CreatorID: s.testCreatorID,
				Status:    models.GameStatusActive,
				CreatedAt: s.testTime,
				UpdatedAt: s.testTime,
				Participants: []*models.Participant{
					{
						ID:         s.testParticipantID,
						GameID:     s.testGameID,
						PlayerID:   s.testCreatorID,
						PlayerName: s.testCreatorName,
						Status:     models.ParticipantStatusNeedsToAssign,
						RollValue:  6,
						RollTime:   &s.testTime,
					},
					{
						ID:         "another-participant-id",
						GameID:     s.testGameID,
						PlayerID:   s.testPlayerID,
						PlayerName: s.testPlayerName,
						Status:     models.ParticipantStatusWaitingToRoll,
						RollValue:  0,
						RollTime:   nil,
					},
					{
						ID:         "third-participant-id",
						GameID:     s.testGameID,
						PlayerID:   "third-player-id",
						PlayerName: "Third Player",
						Status:     models.ParticipantStatusWaitingToRoll,
						RollValue:  0,
						RollTime:   nil,
					},
				},
			},
		}).
		DoAndReturn(func(_ context.Context, input *gameRepo.SaveGameInput) error {
			// Verify the game is marked as completed
			s.Equal(models.GameStatusActive, input.Game.Status)
			return nil
		}).
		Return(nil)

	// Act
	output, err := s.gameService.RollDice(s.ctx, s.rollDiceInput)

	// Assert
	s.Require().NoError(err)
	s.Require().NotNil(output)
	s.Equal(6, output.Value)
	s.Equal(6, output.RollValue)
	s.Equal(s.testCreatorID, output.PlayerID)
	s.Equal(s.testCreatorName, output.PlayerName)
	s.True(output.IsCriticalHit)
	s.False(output.IsCriticalFail)
	s.False(output.IsLowestRoll)
	s.False(output.NeedsRollOff)
	s.False(output.AllPlayersRolled) // Not all players have rolled

	// Verify eligible players for drink assignment
	s.Require().Len(output.EligiblePlayers, 2)

	// Check that both other players are eligible (not the current player)
	eligiblePlayerIDs := []string{output.EligiblePlayers[0].PlayerID, output.EligiblePlayers[1].PlayerID}
	s.Contains(eligiblePlayerIDs, s.testPlayerID)
	s.Contains(eligiblePlayerIDs, "third-player-id")

	// Verify none of the eligible players is the current player
	for _, player := range output.EligiblePlayers {
		s.False(player.IsCurrentPlayer)
	}
}

func (s *GameServiceTestSuite) TestRollDice_CriticalFail() {
	// Create an active game with multiple participants, one who hasn't rolled yet
	activeGame := &models.Game{
		ID:        s.testGameID,
		ChannelID: s.testChannelID,
		CreatorID: s.testCreatorID,
		Status:    models.GameStatusActive,
		CreatedAt: s.testTime,
		UpdatedAt: s.testTime,
		Participants: []*models.Participant{
			{
				ID:         s.testParticipantID,
				GameID:     s.testGameID,
				PlayerID:   s.testCreatorID,
				PlayerName: s.testCreatorName,
				Status:     models.ParticipantStatusWaitingToRoll,
				RollValue:  0,
				RollTime:   nil, // Hasn't rolled yet
			},
			{
				ID:         "another-participant-id",
				GameID:     s.testGameID,
				PlayerID:   s.testPlayerID,
				PlayerName: s.testPlayerName,
				Status:     models.ParticipantStatusWaitingToRoll,
				RollValue:  0,
				RollTime:   nil, // Hasn't rolled yet
			},
		},
	}

	// Set up session expectations
	s.setupSessionExpectations()

	// Expect GetGame to be called and return the active game
	s.mockGameRepo.EXPECT().
		GetGame(gomock.Any(), &gameRepo.GetGameInput{
			GameID: s.testGameID,
		}).
		Return(activeGame, nil)

	// Expect Roll to be called on the dice roller and return a critical fail
	s.mockDiceRoller.EXPECT().
		Roll(6). // 6-sided dice
		Return(1)

	// Expect CreateDrinkRecord to be called for the critical fail
	s.mockDrinkRepo.EXPECT().
		CreateDrinkRecord(gomock.Any(), &ledgerRepo.CreateDrinkRecordInput{
			GameID:       s.testGameID,
			FromPlayerID: s.testCreatorID,
			ToPlayerID:   s.testCreatorID,
			Reason:       models.DrinkReasonCriticalFail,
			SessionID:    s.testSessionID,
			Timestamp:    s.testTime,
		}).
		Return(&ledgerRepo.CreateDrinkRecordOutput{}, nil)

	// Expect SaveGame to be called with the updated game
	s.mockGameRepo.EXPECT().
		SaveGame(gomock.Any(), &gameRepo.SaveGameInput{
			Game: &models.Game{
				ID:        s.testGameID,
				ChannelID: s.testChannelID,
				CreatorID: s.testCreatorID,
				Status:    models.GameStatusActive,
				CreatedAt: s.testTime,
				UpdatedAt: s.testTime,
				Participants: []*models.Participant{
					{
						ID:         s.testParticipantID,
						GameID:     s.testGameID,
						PlayerID:   s.testCreatorID,
						PlayerName: s.testCreatorName,
						Status:     models.ParticipantStatusActive,
						RollValue:  1,
						RollTime:   &s.testTime,
					},
					{
						ID:         "another-participant-id",
						GameID:     s.testGameID,
						PlayerID:   s.testPlayerID,
						PlayerName: s.testPlayerName,
						Status:     models.ParticipantStatusWaitingToRoll,
						RollValue:  0,
						RollTime:   nil,
					},
				},
			},
		}).
		DoAndReturn(func(_ context.Context, input *gameRepo.SaveGameInput) error {
			// Verify the game is marked as completed
			s.Equal(models.GameStatusActive, input.Game.Status)
			return nil
		}).
		Return(nil)

	// Act
	output, err := s.gameService.RollDice(s.ctx, s.rollDiceInput)

	// Assert
	s.Require().NoError(err)
	s.Require().NotNil(output)
	s.Equal(1, output.Value)
	s.Equal(1, output.RollValue)
	s.Equal(s.testCreatorID, output.PlayerID)
	s.Equal(s.testCreatorName, output.PlayerName)
	s.False(output.IsCriticalHit)
	s.True(output.IsCriticalFail)
	s.False(output.IsLowestRoll)
	s.False(output.NeedsRollOff)
	s.False(output.AllPlayersRolled) // Not all players have rolled
}

func (s *GameServiceTestSuite) TestRollDice_GameNotFound() {
	// Expect GetGame to be called on the game repository and return an error
	s.mockGameRepo.EXPECT().
		GetGame(gomock.Any(), &gameRepo.GetGameInput{
			GameID: s.testGameID,
		}).
		Return(nil, ErrGameNotFound)

	// Act
	output, err := s.gameService.RollDice(s.ctx, s.rollDiceInput)

	// Assert
	s.Require().Error(err)
	s.True(errors.Is(err, ErrGameNotFound))
	s.Nil(output)
}

func (s *GameServiceTestSuite) TestRollDice_InvalidGameState() {
	// Create a completed game
	completedGame := &models.Game{
		ID:           s.testGameID,
		ChannelID:    s.testChannelID,
		CreatorID:    s.testCreatorID,
		Status:       models.GameStatusCompleted,
		CreatedAt:    s.testTime,
		UpdatedAt:    s.testTime,
		Participants: []*models.Participant{s.expectedParticipant},
	}

	// Expect GetGame to be called on the game repository
	s.mockGameRepo.EXPECT().
		GetGame(gomock.Any(), &gameRepo.GetGameInput{
			GameID: s.testGameID,
		}).
		Return(completedGame, nil)

	// Act
	output, err := s.gameService.RollDice(s.ctx, s.rollDiceInput)

	// Assert
	s.Require().Error(err)
	s.True(errors.Is(err, ErrInvalidGameState))
	s.Nil(output)
}

func (s *GameServiceTestSuite) TestRollDice_PlayerNotInGame() {
	// Create an active game with a different player
	activeGame := &models.Game{
		ID:        s.testGameID,
		ChannelID: s.testChannelID,
		CreatorID: s.testCreatorID,
		Status:    models.GameStatusActive,
		CreatedAt: s.testTime,
		UpdatedAt: s.testTime,
		Participants: []*models.Participant{
			{
				ID:         "different-participant-id",
				GameID:     s.testGameID,
				PlayerID:   "different-player-id",
				PlayerName: "Different Player",
				Status:     models.ParticipantStatusWaitingToRoll,
			},
		},
	}

	// Expect GetGame to be called on the game repository
	s.mockGameRepo.EXPECT().
		GetGame(gomock.Any(), &gameRepo.GetGameInput{
			GameID: s.testGameID,
		}).
		Return(activeGame, nil)

	// Act
	output, err := s.gameService.RollDice(s.ctx, s.rollDiceInput)

	// Assert
	s.Require().Error(err)
	s.True(errors.Is(err, ErrPlayerNotInGame))
	s.Nil(output)
}

func (s *GameServiceTestSuite) TestRollDice_PlayerAlreadyRolled() {
	// Create a game with a participant who has already rolled
	rollTime := s.testTime.Add(-time.Hour) // Rolled an hour ago
	activeGame := &models.Game{
		ID:        s.testGameID,
		ChannelID: s.testChannelID,
		CreatorID: s.testCreatorID,
		Status:    models.GameStatusActive,
		CreatedAt: s.testTime,
		UpdatedAt: s.testTime,
		Participants: []*models.Participant{
			{
				ID:         s.testParticipantID,
				GameID:     s.testGameID,
				PlayerID:   s.testCreatorID,
				PlayerName: s.testCreatorName,
				Status:     models.ParticipantStatusActive,
				RollValue:  4,
				RollTime:   &rollTime, // Already rolled
			},
		},
	}

	// Expect GetGame to be called on the game repository
	s.mockGameRepo.EXPECT().
		GetGame(gomock.Any(), &gameRepo.GetGameInput{
			GameID: s.testGameID,
		}).
		Return(activeGame, nil)

	// Act
	output, err := s.gameService.RollDice(s.ctx, s.rollDiceInput)

	// Assert
	s.Require().Error(err)
	s.Contains(err.Error(), "has already rolled in this game")
	s.Nil(output)
}

func (s *GameServiceTestSuite) TestRollDice_SaveGameError() {
	// Create an active game with a participant who hasn't rolled yet
	activeGame := &models.Game{
		ID:        s.testGameID,
		ChannelID: s.testChannelID,
		CreatorID: s.testCreatorID,
		Status:    models.GameStatusActive,
		CreatedAt: s.testTime,
		UpdatedAt: s.testTime,
		Participants: []*models.Participant{
			{
				ID:         s.testParticipantID,
				GameID:     s.testGameID,
				PlayerID:   s.testCreatorID,
				PlayerName: s.testCreatorName,
				Status:     models.ParticipantStatusWaitingToRoll,
				RollValue:  0,
				RollTime:   nil, // Hasn't rolled yet
			},
		},
	}

	// Expect GetGame to be called on the game repository
	s.mockGameRepo.EXPECT().
		GetGame(gomock.Any(), &gameRepo.GetGameInput{
			GameID: s.testGameID,
		}).
		Return(activeGame, nil)

	// Expect Roll to be called on the dice roller
	s.mockDiceRoller.EXPECT().
		Roll(6). // 6-sided dice
		Return(3)

	// Expect SaveGame to be called and return an error
	expectedError := errors.New("failed to save game")
	s.mockGameRepo.EXPECT().
		SaveGame(gomock.Any(), &gameRepo.SaveGameInput{
			Game: &models.Game{
				ID:        s.testGameID,
				ChannelID: s.testChannelID,
				CreatorID: s.testCreatorID,
				Status:    models.GameStatusActive,
				CreatedAt: s.testTime,
				UpdatedAt: s.testTime,
				Participants: []*models.Participant{
					{
						ID:         s.testParticipantID,
						GameID:     s.testGameID,
						PlayerID:   s.testCreatorID,
						PlayerName: s.testCreatorName,
						Status:     models.ParticipantStatusActive,
						RollValue:  3,
						RollTime:   &s.testTime,
					},
				},
			},
		}).
		DoAndReturn(func(_ context.Context, input *gameRepo.SaveGameInput) error {
			// Verify the game is marked as completed
			s.Equal(models.GameStatusActive, input.Game.Status)
			return expectedError
		}).
		Return(expectedError)

	// Act
	output, err := s.gameService.RollDice(s.ctx, s.rollDiceInput)

	// Assert
	s.Require().Error(err)
	s.Contains(err.Error(), expectedError.Error(), "Expected error to contain the original error message")
	s.Nil(output)
}

// RollOff Tests

func (s *GameServiceTestSuite) TestRollDice_RollOffGame() {
	// Create a parent roll-off game
	parentRollOffGame := &models.Game{
		ID:           "parent-roll-off-id",
		ChannelID:    s.testChannelID,
		CreatorID:    s.testCreatorID,
		ParentGameID: s.testGameID,
		Status:       models.GameStatusRollOff,
		CreatedAt:    s.testTime,
		UpdatedAt:    s.testTime,
		Participants: []*models.Participant{
			{
				ID:         s.testParticipantID,
				GameID:     "parent-roll-off-id",
				PlayerID:   s.testCreatorID,
				PlayerName: s.testCreatorName,
				Status:     models.ParticipantStatusActive,
				RollValue:  6, // Already rolled
				RollTime:   &s.testTime,
			},
			{
				ID:         "another-participant-id",
				GameID:     "parent-roll-off-id",
				PlayerID:   s.testPlayerID,
				PlayerName: s.testPlayerName,
				Status:     models.ParticipantStatusActive,
				RollValue:  6, // Already rolled, tied with creator
				RollTime:   &s.testTime,
			},
			{
				ID:         "third-participant-id",
				GameID:     "parent-roll-off-id",
				PlayerID:   "third-player-id",
				PlayerName: "Third Player",
				Status:     models.ParticipantStatusActive,
				RollValue:  3, // Lower roll
				RollTime:   &s.testTime,
			},
		},
	}

	// Create a nested roll-off game for the tied players
	nestedRollOffGame := &models.Game{
		ID:           "nested-roll-off-id",
		ChannelID:    s.testChannelID,
		CreatorID:    s.testCreatorID,
		ParentGameID: "parent-roll-off-id", // This is a nested roll-off
		Status:       models.GameStatusRollOff,
		CreatedAt:    s.testTime,
		UpdatedAt:    s.testTime,
		Participants: []*models.Participant{
			{
				ID:         "nested-participant-1",
				GameID:     "nested-roll-off-id",
				PlayerID:   s.testCreatorID,
				PlayerName: s.testCreatorName,
				Status:     models.ParticipantStatusWaitingToRoll,
				// No roll value or time yet
			},
			{
				ID:         "nested-participant-2",
				GameID:     "nested-roll-off-id",
				PlayerID:   s.testPlayerID,
				PlayerName: s.testPlayerName,
				Status:     models.ParticipantStatusWaitingToRoll,
				// No roll value or time yet
			},
		},
	}

	// Set up input for RollDice - note we're targeting the parent roll-off
	rollDiceInput := &RollDiceInput{
		GameID:   "parent-roll-off-id",
		PlayerID: s.testCreatorID,
	}

	// Expect GetGame to be called for the parent roll-off
	s.mockGameRepo.EXPECT().
		GetGame(gomock.Any(), &gameRepo.GetGameInput{
			GameID: "parent-roll-off-id",
		}).
		Return(parentRollOffGame, nil)

	// Expect GetGamesByParent to be called to find nested roll-offs
	s.mockGameRepo.EXPECT().
		GetGamesByParent(gomock.Any(), &gameRepo.GetGamesByParentInput{
			ParentGameID: "parent-roll-off-id",
		}).
		Return([]*models.Game{nestedRollOffGame}, nil)

	// Expect the dice to be rolled (use 6 as the default sides for testing)
	s.mockDiceRoller.EXPECT().
		Roll(6).
		Return(5) // Regular roll, not critical

	// Expect SaveGame to be called with updated participant roll in the NESTED game
	s.mockGameRepo.EXPECT().
		SaveGame(gomock.Any(), &gameRepo.SaveGameInput{
			Game: &models.Game{
				ID:           "nested-roll-off-id",
				ChannelID:    s.testChannelID,
				CreatorID:    s.testCreatorID,
				ParentGameID: "parent-roll-off-id",
				Status:       models.GameStatusRollOff,
				CreatedAt:    s.testTime,
				UpdatedAt:    s.testTime,
				Participants: []*models.Participant{
					{
						ID:         "nested-participant-1",
						GameID:     "nested-roll-off-id",
						PlayerID:   s.testCreatorID,
						PlayerName: s.testCreatorName,
						Status:     models.ParticipantStatusActive,
						RollValue:  5,
						RollTime:   &s.testTime,
					},
					{
						ID:         "nested-participant-2",
						GameID:     "nested-roll-off-id",
						PlayerID:   s.testPlayerID,
						PlayerName: s.testPlayerName,
						Status:     models.ParticipantStatusWaitingToRoll,
						RollValue:  0,
						RollTime:   nil,
					},
				},
			},
		}).
		DoAndReturn(func(_ context.Context, input *gameRepo.SaveGameInput) error {
			// Verify the NESTED game has been updated with the participant's roll
			s.Equal("nested-roll-off-id", input.Game.ID)
			s.Equal(models.GameStatusRollOff, input.Game.Status)

			// Find the participant who rolled
			var rolledParticipant *models.Participant
			for _, p := range input.Game.Participants {
				if p.PlayerID == s.testCreatorID {
					rolledParticipant = p
					break
				}
			}

			// Verify participant roll was updated in the nested game
			s.NotNil(rolledParticipant)
			s.Equal(5, rolledParticipant.RollValue)
			s.NotNil(rolledParticipant.RollTime)
			s.Equal(models.ParticipantStatusActive, rolledParticipant.Status)

			return nil
		}).
		Return(nil)

	// Act
	output, err := s.gameService.RollDice(s.ctx, rollDiceInput)

	// Assert
	s.Require().NoError(err)
	s.Require().NotNil(output)
	s.Equal(5, output.RollValue)
	s.False(output.IsCriticalHit)
	s.False(output.IsCriticalFail)
}

func (s *GameServiceTestSuite) TestEndGame_HighestRollTie() {
	// Create a game where multiple players have tied for the highest roll
	gameWithRolls := &models.Game{
		ID:        s.testGameID,
		ChannelID: s.testChannelID,
		CreatorID: s.testCreatorID,
		Status:    models.GameStatusActive,
		CreatedAt: s.testTime,
		UpdatedAt: s.testTime,
		Participants: []*models.Participant{
			{
				ID:         s.testParticipantID,
				GameID:     s.testGameID,
				PlayerID:   s.testCreatorID,
				PlayerName: s.testCreatorName,
				Status:     models.ParticipantStatusActive,
				RollValue:  6, // Tied for highest roll (creator)
				RollTime:   &s.testTime,
			},
			{
				ID:         "another-participant-id",
				GameID:     s.testGameID,
				PlayerID:   s.testPlayerID,
				PlayerName: s.testPlayerName,
				Status:     models.ParticipantStatusActive,
				RollValue:  6, // Tied for highest roll
				RollTime:   &s.testTime,
			},
			{
				ID:         "third-participant-id",
				GameID:     s.testGameID,
				PlayerID:   "third-player-id",
				PlayerName: "Third Player",
				Status:     models.ParticipantStatusActive,
				RollValue:  3, // Lowest roll
				RollTime:   &s.testTime,
			},
		},
	}

	// Set up session expectations
	s.setupSessionExpectations()

	// Expect GetDrinkRecordsForGame to be called
	s.mockDrinkRepo.EXPECT().
		GetDrinkRecordsForGame(gomock.Any(), &ledgerRepo.GetDrinkRecordsForGameInput{
			GameID: s.testGameID,
		}).
		Return(&ledgerRepo.GetDrinkRecordsForGameOutput{
			Records: []*models.DrinkLedger{},
		}, nil)

	// Set up mock for creating a roll-off game
	rollOffGame := &models.Game{
		ID:           "roll-off-game-id",
		ChannelID:    s.testChannelID,
		CreatorID:    s.testCreatorID,
		ParentGameID: s.testGameID,
		Status:       models.GameStatusRollOff,
		CreatedAt:    s.testTime,
		UpdatedAt:    s.testTime,
	}

	// Expect CreateRollOffGame to be called with both tied players, including the creator
	s.mockGameRepo.EXPECT().
		CreateRollOffGame(gomock.Any(), &gameRepo.CreateRollOffGameInput{
			ChannelID:    s.testChannelID,
			CreatorID:    s.testCreatorID,
			ParentGameID: s.testGameID,
			PlayerIDs:    []string{s.testCreatorID, s.testPlayerID},
			PlayerNames:  map[string]string{s.testCreatorID: s.testCreatorName, s.testPlayerID: s.testPlayerName},
		}).
		Return(&gameRepo.CreateRollOffGameOutput{
			Game: rollOffGame,
		}, nil)

	// Expect SaveGame to be called to update the parent game
	s.mockGameRepo.EXPECT().
		SaveGame(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, input *gameRepo.SaveGameInput) error {
			// Verify the game is marked as roll-off
			s.Equal(models.GameStatusRollOff, input.Game.Status)
			s.Equal(rollOffGame.ID, input.Game.HighestRollOffGameID)
			return nil
		}).
		Return(nil)

	// Expect GetPlayer to be called for ALL participants
	// First participant (creator) - may be called multiple times
	s.mockPlayerRepo.EXPECT().
		GetPlayer(gomock.Any(), &playerRepo.GetPlayerInput{
			PlayerID: s.testCreatorID,
		}).
		Return(&models.Player{
			ID:            s.testCreatorID,
			Name:          s.testCreatorName,
			CurrentGameID: s.testGameID,
		}, nil)

	// Second participant - may be called multiple times
	s.mockPlayerRepo.EXPECT().
		GetPlayer(gomock.Any(), &playerRepo.GetPlayerInput{
			PlayerID: s.testPlayerID,
		}).
		Return(&models.Player{
			ID:            s.testPlayerID,
			Name:          s.testPlayerName,
			CurrentGameID: s.testGameID,
		}, nil)

	// Expect SavePlayer to be called for each tied player (only the highest rollers)
	s.mockPlayerRepo.EXPECT().
		SavePlayer(gomock.Any(), &playerRepo.SavePlayerInput{
			Player: &models.Player{
				ID:            s.testCreatorID,
				Name:          s.testCreatorName,
				CurrentGameID: rollOffGame.ID,
			},
		}).
		Return(nil)

	s.mockPlayerRepo.EXPECT().
		SavePlayer(gomock.Any(), &playerRepo.SavePlayerInput{
			Player: &models.Player{
				ID:            s.testPlayerID,
				Name:          s.testPlayerName,
				CurrentGameID: rollOffGame.ID,
			},
		}).
		Return(nil)

	// Act
	output, err := s.gameService.EndGame(s.ctx, &EndGameInput{
		Game: gameWithRolls,
	})

	// Assert
	s.Require().NoError(err)
	s.Require().NotNil(output)
	s.True(output.NeedsRollOff)
	s.Equal(RollOffTypeHighest, output.RollOffType)
	s.Equal(rollOffGame.ID, output.RollOffGameID)
	s.Equal(2, len(output.RollOffPlayerIDs))
	s.Contains(output.RollOffPlayerIDs, s.testCreatorID) // Creator should be included
	s.Contains(output.RollOffPlayerIDs, s.testPlayerID)
}

func (s *GameServiceTestSuite) TestEndGame_LowestRollTie() {
	// Create a game where multiple players have tied for the lowest roll
	game := &models.Game{
		ID:        s.testGameID,
		ChannelID: s.testChannelID,
		CreatorID: s.testCreatorID,
		Status:    models.GameStatusActive,
		CreatedAt: s.testTime,
		UpdatedAt: s.testTime,
		Participants: []*models.Participant{
			{
				ID:         s.testParticipantID,
				GameID:     s.testGameID,
				PlayerID:   s.testCreatorID,
				PlayerName: s.testCreatorName,
				Status:     models.ParticipantStatusActive,
				RollValue:  6, // Highest roll
				RollTime:   &s.testTime,
			},
			{
				ID:         "another-participant-id",
				GameID:     s.testGameID,
				PlayerID:   s.testPlayerID,
				PlayerName: s.testPlayerName,
				Status:     models.ParticipantStatusActive,
				RollValue:  1, // Tied for lowest roll
				RollTime:   &s.testTime,
			},
			{
				ID:         "third-participant-id",
				GameID:     s.testGameID,
				PlayerID:   "third-player-id",
				PlayerName: "Third Player",
				Status:     models.ParticipantStatusActive,
				RollValue:  1, // Tied for lowest roll
				RollTime:   &s.testTime,
			},
		},
	}

	// Set up session expectations
	s.setupSessionExpectations()

	// Expect GetDrinkRecordsForGame to be called
	s.mockDrinkRepo.EXPECT().
		GetDrinkRecordsForGame(gomock.Any(), &ledgerRepo.GetDrinkRecordsForGameInput{
			GameID: s.testGameID,
		}).
		Return(&ledgerRepo.GetDrinkRecordsForGameOutput{
			Records: []*models.DrinkLedger{},
		}, nil)

	// Set up mock for creating a roll-off game
	rollOffGame := &models.Game{
		ID:           "roll-off-game-id",
		ChannelID:    s.testChannelID,
		CreatorID:    s.testCreatorID,
		ParentGameID: s.testGameID,
		Status:       models.GameStatusRollOff,
		CreatedAt:    s.testTime,
		UpdatedAt:    s.testTime,
	}

	// Expect CreateRollOffGame to be called for lowest rollers
	s.mockGameRepo.EXPECT().
		CreateRollOffGame(gomock.Any(), &gameRepo.CreateRollOffGameInput{
			ChannelID:    s.testChannelID,
			CreatorID:    s.testCreatorID,
			ParentGameID: s.testGameID,
			PlayerIDs:    []string{s.testPlayerID, "third-player-id"},
			PlayerNames:  map[string]string{s.testPlayerID: s.testPlayerName, "third-player-id": "Third Player"},
		}).
		Return(&gameRepo.CreateRollOffGameOutput{
			Game: rollOffGame,
		}, nil)

	// Expect SaveGame to be called to update the parent game
	s.mockGameRepo.EXPECT().
		SaveGame(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, input *gameRepo.SaveGameInput) error {
			// Verify the game is marked as roll-off
			s.Equal(models.GameStatusRollOff, input.Game.Status)
			s.Equal(rollOffGame.ID, input.Game.LowestRollOffGameID)
			return nil
		}).
		Return(nil)

	// Second participant - may be called multiple times
	s.mockPlayerRepo.EXPECT().
		GetPlayer(gomock.Any(), &playerRepo.GetPlayerInput{
			PlayerID: s.testPlayerID,
		}).
		Return(&models.Player{
			ID:            s.testPlayerID,
			Name:          s.testPlayerName,
			CurrentGameID: s.testGameID,
		}, nil)

	// Third participant - may be called multiple times
	s.mockPlayerRepo.EXPECT().
		GetPlayer(gomock.Any(), &playerRepo.GetPlayerInput{
			PlayerID: "third-player-id",
		}).
		Return(&models.Player{
			ID:            "third-player-id",
			Name:          "Third Player",
			CurrentGameID: s.testGameID,
		}, nil).
		MinTimes(0)

	// Expect SavePlayer to be called for each tied player (only the lowest rollers)
	s.mockPlayerRepo.EXPECT().
		SavePlayer(gomock.Any(), &playerRepo.SavePlayerInput{
			Player: &models.Player{
				ID:            s.testPlayerID,
				Name:          s.testPlayerName,
				CurrentGameID: rollOffGame.ID,
			},
		}).
		Return(nil)

	s.mockPlayerRepo.EXPECT().
		SavePlayer(gomock.Any(), &playerRepo.SavePlayerInput{
			Player: &models.Player{
				ID:            "third-player-id",
				Name:          "Third Player",
				CurrentGameID: rollOffGame.ID,
			},
		}).
		Return(nil)

	// Act
	output, err := s.gameService.EndGame(s.ctx, &EndGameInput{
		Game: game,
	})

	// Assert
	s.Require().NoError(err)
	s.Require().NotNil(output)
	s.True(output.NeedsRollOff)
	s.Equal(RollOffTypeLowest, output.RollOffType)
	s.Equal(rollOffGame.ID, output.RollOffGameID)
	s.Equal(2, len(output.RollOffPlayerIDs))
	s.Contains(output.RollOffPlayerIDs, s.testPlayerID)
	s.Contains(output.RollOffPlayerIDs, "third-player-id")
}

func (s *GameServiceTestSuite) TestEndGame_BothHighestAndLowestRollTies() {
	// Set up session expectations
	s.setupSessionExpectations()

	// Create a game where there are ties for both highest and lowest rolls
	game := &models.Game{
		ID:        s.testGameID,
		ChannelID: s.testChannelID,
		CreatorID: s.testCreatorID,
		Status:    models.GameStatusActive,
		CreatedAt: s.testTime,
		UpdatedAt: s.testTime,
		Participants: []*models.Participant{
			{
				ID:         s.testParticipantID,
				GameID:     s.testGameID,
				PlayerID:   s.testCreatorID,
				PlayerName: s.testCreatorName,
				Status:     models.ParticipantStatusActive,
				RollValue:  6, // Tied for highest roll (creator)
				RollTime:   &s.testTime,
			},
			{
				ID:         "another-participant-id",
				GameID:     s.testGameID,
				PlayerID:   s.testPlayerID,
				PlayerName: s.testPlayerName,
				Status:     models.ParticipantStatusActive,
				RollValue:  6, // Tied for highest roll
				RollTime:   &s.testTime,
			},
			{
				ID:         "third-participant-id",
				GameID:     s.testGameID,
				PlayerID:   "third-player-id",
				PlayerName: "Third Player",
				Status:     models.ParticipantStatusActive,
				RollValue:  1, // Tied for lowest roll
				RollTime:   &s.testTime,
			},
			{
				ID:         "fourth-participant-id",
				GameID:     s.testGameID,
				PlayerID:   "fourth-player-id",
				PlayerName: "Fourth Player",
				Status:     models.ParticipantStatusActive,
				RollValue:  1, // Tied for lowest roll
				RollTime:   &s.testTime,
			},
		},
	}

	// Define roll-off games
	highestRollOffGame := &models.Game{
		ID:           "highest-roll-off-game-id",
		ChannelID:    s.testChannelID,
		CreatorID:    s.testCreatorID,
		ParentGameID: s.testGameID, // This is a roll-off for the parent game
		Status:       models.GameStatusRollOff,
		CreatedAt:    s.testTime,
		UpdatedAt:    s.testTime,
	}

	lowestRollOffGame := &models.Game{
		ID:           "lowest-roll-off-game-id",
		ChannelID:    s.testChannelID,
		CreatorID:    s.testCreatorID,
		ParentGameID: s.testGameID, // This is a roll-off for the parent game
		Status:       models.GameStatusRollOff,
		CreatedAt:    s.testTime,
		UpdatedAt:    s.testTime,
	}

	// Expect GetDrinkRecordsForGame to be called
	s.mockDrinkRepo.EXPECT().
		GetDrinkRecordsForGame(gomock.Any(), &ledgerRepo.GetDrinkRecordsForGameInput{
			GameID: s.testGameID,
		}).
		Return(&ledgerRepo.GetDrinkRecordsForGameOutput{
			Records: []*models.DrinkLedger{},
		}, nil)

	// Expect CreateRollOffGame to be called for highest rollers
	s.mockGameRepo.EXPECT().
		CreateRollOffGame(gomock.Any(), &gameRepo.CreateRollOffGameInput{
			ChannelID:    s.testChannelID,
			CreatorID:    s.testCreatorID,
			ParentGameID: s.testGameID,
			PlayerIDs:    []string{s.testCreatorID, s.testPlayerID},
			PlayerNames:  map[string]string{s.testCreatorID: s.testCreatorName, s.testPlayerID: s.testPlayerName},
		}).
		Return(&gameRepo.CreateRollOffGameOutput{
			Game: highestRollOffGame,
		}, nil)

	// Expect CreateRollOffGame to be called for lowest rollers
	s.mockGameRepo.EXPECT().
		CreateRollOffGame(gomock.Any(), &gameRepo.CreateRollOffGameInput{
			ChannelID:    s.testChannelID,
			CreatorID:    s.testCreatorID,
			ParentGameID: s.testGameID,
			PlayerIDs:    []string{"third-player-id", "fourth-player-id"},
			PlayerNames:  map[string]string{"third-player-id": "Third Player", "fourth-player-id": "Fourth Player"},
		}).
		Return(&gameRepo.CreateRollOffGameOutput{
			Game: lowestRollOffGame,
		}, nil)

	// Expect GetPlayer to be called for all players in roll-offs
	// First participant (creator)
	s.mockPlayerRepo.EXPECT().
		GetPlayer(gomock.Any(), &playerRepo.GetPlayerInput{
			PlayerID: s.testCreatorID,
		}).
		Return(&models.Player{
			ID:            s.testCreatorID,
			Name:          s.testCreatorName,
			CurrentGameID: s.testGameID,
		}, nil)

	// Second participant
	s.mockPlayerRepo.EXPECT().
		GetPlayer(gomock.Any(), &playerRepo.GetPlayerInput{
			PlayerID: s.testPlayerID,
		}).
		Return(&models.Player{
			ID:            s.testPlayerID,
			Name:          s.testPlayerName,
			CurrentGameID: s.testGameID,
		}, nil)

	// Third participant
	s.mockPlayerRepo.EXPECT().
		GetPlayer(gomock.Any(), &playerRepo.GetPlayerInput{
			PlayerID: "third-player-id",
		}).
		Return(&models.Player{
			ID:            "third-player-id",
			Name:          "Third Player",
			CurrentGameID: s.testGameID,
		}, nil)

	// Fourth participant
	s.mockPlayerRepo.EXPECT().
		GetPlayer(gomock.Any(), &playerRepo.GetPlayerInput{
			PlayerID: "fourth-player-id",
		}).
		Return(&models.Player{
			ID:            "fourth-player-id",
			Name:          "Fourth Player",
			CurrentGameID: s.testGameID,
		}, nil)

	// Expect SavePlayer to be called for each player in roll-offs
	// First participant (creator) - highest roll-off
	s.mockPlayerRepo.EXPECT().
		SavePlayer(gomock.Any(), &playerRepo.SavePlayerInput{
			Player: &models.Player{
				ID:            s.testCreatorID,
				Name:          s.testCreatorName,
				CurrentGameID: highestRollOffGame.ID,
			},
		}).
		Return(nil)

	// Second participant - highest roll-off
	s.mockPlayerRepo.EXPECT().
		SavePlayer(gomock.Any(), &playerRepo.SavePlayerInput{
			Player: &models.Player{
				ID:            s.testPlayerID,
				Name:          s.testPlayerName,
				CurrentGameID: highestRollOffGame.ID,
			},
		}).
		Return(nil)

	// Third participant - lowest roll-off
	s.mockPlayerRepo.EXPECT().
		SavePlayer(gomock.Any(), &playerRepo.SavePlayerInput{
			Player: &models.Player{
				ID:            "third-player-id",
				Name:          "Third Player",
				CurrentGameID: lowestRollOffGame.ID,
			},
		}).
		Return(nil)

	// Fourth participant - lowest roll-off
	s.mockPlayerRepo.EXPECT().
		SavePlayer(gomock.Any(), &playerRepo.SavePlayerInput{
			Player: &models.Player{
				ID:            "fourth-player-id",
				Name:          "Fourth Player",
				CurrentGameID: lowestRollOffGame.ID,
			},
		}).
		Return(nil)

	// Expect SaveGame to be called once with both roll-off game IDs
	s.mockGameRepo.EXPECT().
		SaveGame(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, input *gameRepo.SaveGameInput) error {
			// Verify the game has been updated with the roll-off game IDs
			s.Equal(models.GameStatusRollOff, input.Game.Status)
			s.Equal(highestRollOffGame.ID, input.Game.HighestRollOffGameID)
			s.Equal(lowestRollOffGame.ID, input.Game.LowestRollOffGameID)
			return nil
		}).
		Return(nil)

	// Act
	output, err := s.gameService.EndGame(s.ctx, &EndGameInput{
		Game: game,
	})

	// Assert
	s.Require().NoError(err)
	s.Require().NotNil(output)

	// Check that both roll-off flags are set
	s.True(output.NeedsHighestRollOff)
	s.Equal(highestRollOffGame.ID, output.HighestRollOffGameID)
	s.Equal(2, len(output.HighestRollOffPlayerIDs))
	s.Contains(output.HighestRollOffPlayerIDs, s.testCreatorID) // Creator should be included
	s.Contains(output.HighestRollOffPlayerIDs, s.testPlayerID)

	s.True(output.NeedsLowestRollOff)
	s.Equal(lowestRollOffGame.ID, output.LowestRollOffGameID)
	s.Equal(2, len(output.LowestRollOffPlayerIDs))
	s.Contains(output.LowestRollOffPlayerIDs, "third-player-id")
	s.Contains(output.LowestRollOffPlayerIDs, "fourth-player-id")

	// Check backward compatibility fields
	s.True(output.NeedsRollOff)
	s.Equal(RollOffTypeHighest, output.RollOffType) // Highest takes precedence
	s.Equal(highestRollOffGame.ID, output.RollOffGameID)
	s.Equal(2, len(output.RollOffPlayerIDs))
	s.Contains(output.RollOffPlayerIDs, s.testCreatorID)
	s.Contains(output.RollOffPlayerIDs, s.testPlayerID)
}

func (s *GameServiceTestSuite) TestRollDice_NestedRollOffGame() {
	// Create a parent roll-off game
	parentRollOffGame := &models.Game{
		ID:           "parent-roll-off-id",
		ChannelID:    s.testChannelID,
		CreatorID:    s.testCreatorID,
		ParentGameID: s.testGameID,
		Status:       models.GameStatusRollOff,
		CreatedAt:    s.testTime,
		UpdatedAt:    s.testTime,
		Participants: []*models.Participant{
			{
				ID:         s.testParticipantID,
				GameID:     "parent-roll-off-id",
				PlayerID:   s.testCreatorID,
				PlayerName: s.testCreatorName,
				Status:     models.ParticipantStatusActive,
				RollValue:  6, // Already rolled
				RollTime:   &s.testTime,
			},
			{
				ID:         "another-participant-id",
				GameID:     "parent-roll-off-id",
				PlayerID:   s.testPlayerID,
				PlayerName: s.testPlayerName,
				Status:     models.ParticipantStatusActive,
				RollValue:  6, // Already rolled, tied with creator
				RollTime:   &s.testTime,
			},
			{
				ID:         "third-participant-id",
				GameID:     "parent-roll-off-id",
				PlayerID:   "third-player-id",
				PlayerName: "Third Player",
				Status:     models.ParticipantStatusActive,
				RollValue:  3, // Lower roll
				RollTime:   &s.testTime,
			},
		},
	}

	// Create a nested roll-off game for the tied players
	nestedRollOffGame := &models.Game{
		ID:           "nested-roll-off-id",
		ChannelID:    s.testChannelID,
		CreatorID:    s.testCreatorID,
		ParentGameID: "parent-roll-off-id", // This is a nested roll-off
		Status:       models.GameStatusRollOff,
		CreatedAt:    s.testTime,
		UpdatedAt:    s.testTime,
		Participants: []*models.Participant{
			{
				ID:         "nested-participant-1",
				GameID:     "nested-roll-off-id",
				PlayerID:   s.testCreatorID,
				PlayerName: s.testCreatorName,
				Status:     models.ParticipantStatusWaitingToRoll,
				// No roll value or time yet
			},
			{
				ID:         "nested-participant-2",
				GameID:     "nested-roll-off-id",
				PlayerID:   s.testPlayerID,
				PlayerName: s.testPlayerName,
				Status:     models.ParticipantStatusWaitingToRoll,
				// No roll value or time yet
			},
		},
	}

	// Set up input for RollDice - note we're targeting the parent roll-off
	rollDiceInput := &RollDiceInput{
		GameID:   "parent-roll-off-id",
		PlayerID: s.testCreatorID,
	}

	// Expect GetGame to be called for the parent roll-off
	s.mockGameRepo.EXPECT().
		GetGame(gomock.Any(), &gameRepo.GetGameInput{
			GameID: "parent-roll-off-id",
		}).
		Return(parentRollOffGame, nil)

	// Expect GetGamesByParent to be called to find nested roll-offs
	s.mockGameRepo.EXPECT().
		GetGamesByParent(gomock.Any(), &gameRepo.GetGamesByParentInput{
			ParentGameID: "parent-roll-off-id",
		}).
		Return([]*models.Game{nestedRollOffGame}, nil)

	// Expect the dice to be rolled (use 6 as the default sides for testing)
	s.mockDiceRoller.EXPECT().
		Roll(6).
		Return(5) // Regular roll, not critical

	// Expect SaveGame to be called with updated participant roll in the NESTED game
	s.mockGameRepo.EXPECT().
		SaveGame(gomock.Any(), &gameRepo.SaveGameInput{
			Game: &models.Game{
				ID:           "nested-roll-off-id",
				ChannelID:    s.testChannelID,
				CreatorID:    s.testCreatorID,
				ParentGameID: "parent-roll-off-id",
				Status:       models.GameStatusRollOff,
				CreatedAt:    s.testTime,
				UpdatedAt:    s.testTime,
				Participants: []*models.Participant{
					{
						ID:         "nested-participant-1",
						GameID:     "nested-roll-off-id",
						PlayerID:   s.testCreatorID,
						PlayerName: s.testCreatorName,
						Status:     models.ParticipantStatusActive,
						RollValue:  5,
						RollTime:   &s.testTime,
					},
					{
						ID:         "nested-participant-2",
						GameID:     "nested-roll-off-id",
						PlayerID:   s.testPlayerID,
						PlayerName: s.testPlayerName,
						Status:     models.ParticipantStatusWaitingToRoll,
						RollValue:  0,
						RollTime:   nil,
					},
				},
			},
		}).
		DoAndReturn(func(_ context.Context, input *gameRepo.SaveGameInput) error {
			// Verify the NESTED game has been updated with the participant's roll
			s.Equal("nested-roll-off-id", input.Game.ID)
			s.Equal(models.GameStatusRollOff, input.Game.Status)

			// Find the participant who rolled
			var rolledParticipant *models.Participant
			for _, p := range input.Game.Participants {
				if p.PlayerID == s.testCreatorID {
					rolledParticipant = p
					break
				}
			}

			// Verify participant roll was updated in the nested game
			s.NotNil(rolledParticipant)
			s.Equal(5, rolledParticipant.RollValue)
			s.NotNil(rolledParticipant.RollTime)
			s.Equal(models.ParticipantStatusActive, rolledParticipant.Status)

			return nil
		}).
		Return(nil)

	// Act
	output, err := s.gameService.RollDice(s.ctx, rollDiceInput)

	// Assert
	s.Require().NoError(err)
	s.Require().NotNil(output)
	s.Equal(5, output.RollValue)
	s.False(output.IsCriticalHit)
	s.False(output.IsCriticalFail)
}

func (s *GameServiceTestSuite) TestEndGame_CompletedLowestRollOff() {
	// Create a roll-off game that has been completed
	rollOffGame := &models.Game{
		ID:           "roll-off-game-id",
		ChannelID:    s.testChannelID,
		CreatorID:    s.testCreatorID,
		ParentGameID: s.testGameID, // This is a roll-off for the parent game
		Status:       models.GameStatusActive,
		CreatedAt:    s.testTime,
		UpdatedAt:    s.testTime,
		Participants: []*models.Participant{
			{
				ID:         "roll-off-participant-1",
				GameID:     "roll-off-game-id",
				PlayerID:   s.testPlayerID,
				PlayerName: s.testPlayerName,
				Status:     models.ParticipantStatusActive,
				RollValue:  3, // Higher roll in the roll-off
				RollTime:   &s.testTime,
			},
			{
				ID:         "roll-off-participant-2",
				GameID:     "roll-off-game-id",
				PlayerID:   "third-player-id",
				PlayerName: "Third Player",
				Status:     models.ParticipantStatusActive,
				RollValue:  1, // Lowest roll in the roll-off
				RollTime:   &s.testTime,
			},
		},
	}

	// Create the parent game with the roll-off game ID
	parentGame := &models.Game{
		ID:                  s.testGameID,
		ChannelID:           s.testChannelID,
		CreatorID:           s.testCreatorID,
		Status:              models.GameStatusRollOff,
		CreatedAt:           s.testTime,
		UpdatedAt:           s.testTime,
		LowestRollOffGameID: "roll-off-game-id", // This is a lowest roll-off
		Participants: []*models.Participant{
			{
				ID:         s.testParticipantID,
				GameID:     s.testGameID,
				PlayerID:   s.testCreatorID,
				PlayerName: s.testCreatorName,
				Status:     models.ParticipantStatusActive,
				RollValue:  6, // Highest roll in the parent game
				RollTime:   &s.testTime,
			},
			{
				ID:         "another-participant-id",
				GameID:     s.testGameID,
				PlayerID:   s.testPlayerID,
				PlayerName: s.testPlayerName,
				Status:     models.ParticipantStatusActive,
				RollValue:  3, // Middle roll
				RollTime:   &s.testTime,
			},
			{
				ID:         "third-participant-id",
				GameID:     s.testGameID,
				PlayerID:   "third-player-id",
				PlayerName: "Third Player",
				Status:     models.ParticipantStatusActive,
				RollValue:  1, // Tied for lowest roll
				RollTime:   &s.testTime,
			},
			{
				ID:         "fourth-participant-id",
				GameID:     s.testGameID,
				PlayerID:   "fourth-player-id",
				PlayerName: "Fourth Player",
				Status:     models.ParticipantStatusActive,
				RollValue:  1, // Tied for lowest roll
				RollTime:   &s.testTime,
			},
		},
	}

	// Set up session expectations
	s.setupSessionExpectations()

	// Expect GetDrinkRecordsForSession to be called
	s.mockDrinkRepo.EXPECT().
		GetDrinkRecordsForSession(gomock.Any(), &ledgerRepo.GetDrinkRecordsForSessionInput{
			SessionID: "test-session-id",
		}).
		Return(&ledgerRepo.GetDrinkRecordsForSessionOutput{
			Records: []*models.DrinkLedger{
				{
					ID:           "drink-1",
					GameID:       "previous-game-id",
					FromPlayerID: s.testCreatorID,
					ToPlayerID:   s.testPlayerID,
					Reason:       models.DrinkReasonCriticalHit,
					Timestamp:    s.testTime,
					SessionID:    "test-session-id",
					Paid:         false,
				},
			},
		}, nil)

	// Expect GetGame to be called for the parent game
	s.mockGameRepo.EXPECT().
		GetGame(gomock.Any(), &gameRepo.GetGameInput{
			GameID: s.testGameID,
		}).
		Return(parentGame, nil)

	// Expect GetDrinkRecordsForGame to be called
	s.mockDrinkRepo.EXPECT().
		GetDrinkRecordsForGame(gomock.Any(), &ledgerRepo.GetDrinkRecordsForGameInput{
			GameID: rollOffGame.ID,
		}).
		Return(&ledgerRepo.GetDrinkRecordsForGameOutput{
			Records: []*models.DrinkLedger{},
		}, nil)

	// Expect CreateDrinkRecord to be called for the lowest roller in the roll-off
	s.mockDrinkRepo.EXPECT().
		CreateDrinkRecord(gomock.Any(), &ledgerRepo.CreateDrinkRecordInput{
			GameID:     parentGame.ID,     // Drink is assigned to the parent game
			ToPlayerID: "third-player-id", // The player with the lowest roll in the roll-off
			Reason:     models.DrinkReasonLowestRoll,
			Timestamp:  s.testTime,
			SessionID:  "test-session-id",
		}).
		Return(&ledgerRepo.CreateDrinkRecordOutput{}, nil)

	// Expect SaveGame to be called to update the roll-off game status
	s.mockGameRepo.EXPECT().
		SaveGame(gomock.Any(), &gameRepo.SaveGameInput{
			Game: &models.Game{
				ID:           "roll-off-game-id",
				ChannelID:    s.testChannelID,
				CreatorID:    s.testCreatorID,
				ParentGameID: s.testGameID,
				Status:       models.GameStatusCompleted,
				CreatedAt:    s.testTime,
				UpdatedAt:    s.testTime,
				Participants: rollOffGame.Participants,
			},
		}).
		DoAndReturn(func(_ context.Context, input *gameRepo.SaveGameInput) error {
			// Verify the roll-off game is marked as completed
			s.Equal(models.GameStatusCompleted, input.Game.Status)
			return nil
		}).
		Return(nil)

	// Expect SaveGame to be called to update the parent game
	s.mockGameRepo.EXPECT().
		SaveGame(gomock.Any(), &gameRepo.SaveGameInput{
			Game: &models.Game{
				ID:                  s.testGameID,
				ChannelID:           s.testChannelID,
				CreatorID:           s.testCreatorID,
				Status:              models.GameStatusCompleted,
				CreatedAt:           s.testTime,
				UpdatedAt:           s.testTime,
				Participants:        parentGame.Participants,
				LowestRollOffGameID: "roll-off-game-id",
			},
		}).
		DoAndReturn(func(_ context.Context, input *gameRepo.SaveGameInput) error {
			// Verify the parent game is marked as completed
			s.Equal(models.GameStatusCompleted, input.Game.Status)
			return nil
		}).
		Return(nil)

	// Expect GetPlayer to be called for all participants
	s.mockPlayerRepo.EXPECT().
		GetPlayer(gomock.Any(), gomock.Any()).
		Return(&models.Player{
			ID:            s.testPlayerID,
			Name:          s.testPlayerName,
			CurrentGameID: rollOffGame.ID,
		}, nil).AnyTimes()

	// Expect SavePlayer to be called for all participants
	s.mockPlayerRepo.EXPECT().
		SavePlayer(gomock.Any(), gomock.Any()).
		Return(nil).AnyTimes()

	// Act
	output, err := s.gameService.EndGame(s.ctx, &EndGameInput{
		Game: rollOffGame,
	})

	// Assert
	s.Require().NoError(err)
	s.Require().NotNil(output)
	s.False(output.NeedsRollOff)             // No further roll-offs needed
	s.Equal("", output.RollOffGameID)        // No roll-off game ID
	s.Equal(0, len(output.RollOffPlayerIDs)) // No roll-off players

	// Verify the explicit fields
	s.False(output.NeedsHighestRollOff)
	s.Equal("", output.HighestRollOffGameID)
	s.Equal(0, len(output.HighestRollOffPlayerIDs))
	s.False(output.NeedsLowestRollOff)
	s.Equal("", output.LowestRollOffGameID)
	s.Equal(0, len(output.LowestRollOffPlayerIDs))
}

func (s *GameServiceTestSuite) TestEndGame_IncludesSessionLeaderboard() {
	// Create a completed game
	game := &models.Game{
		ID:        s.testGameID,
		ChannelID: s.testChannelID,
		CreatorID: s.testCreatorID,
		Status:    models.GameStatusActive,
		CreatedAt: s.testTime,
		UpdatedAt: s.testTime,
		Participants: []*models.Participant{
			{
				ID:         s.testParticipantID,
				GameID:     s.testGameID,
				PlayerID:   s.testCreatorID,
				PlayerName: s.testCreatorName,
				Status:     models.ParticipantStatusActive,
				RollValue:  6, // Highest roll
				RollTime:   &s.testTime,
			},
			{
				ID:         "another-participant-id",
				GameID:     s.testGameID,
				PlayerID:   s.testPlayerID,
				PlayerName: s.testPlayerName,
				Status:     models.ParticipantStatusActive,
				RollValue:  3, // Middle roll
				RollTime:   &s.testTime,
			},
			{
				ID:         "third-participant-id",
				GameID:     s.testGameID,
				PlayerID:   "third-player-id",
				PlayerName: "Third Player",
				Status:     models.ParticipantStatusActive,
				RollValue:  1, // Lowest roll
				RollTime:   &s.testTime,
			},
		},
	}

	// Set up session expectations
	s.setupSessionExpectations()

	// Expect GetDrinkRecordsForGame to be called
	s.mockDrinkRepo.EXPECT().
		GetDrinkRecordsForGame(gomock.Any(), &ledgerRepo.GetDrinkRecordsForGameInput{
			GameID: s.testGameID,
		}).
		Return(&ledgerRepo.GetDrinkRecordsForGameOutput{
			Records: []*models.DrinkLedger{},
		}, nil)

	// Expect CreateDrinkRecord to be called for the lowest roller
	s.mockDrinkRepo.EXPECT().
		CreateDrinkRecord(gomock.Any(), &ledgerRepo.CreateDrinkRecordInput{
			GameID:     s.testGameID,
			ToPlayerID: "third-player-id",
			Reason:     models.DrinkReasonLowestRoll,
			Timestamp:  s.testTime,
			SessionID:  "test-session-id",
		}).
		Return(&ledgerRepo.CreateDrinkRecordOutput{}, nil)

	// Expect SaveGame to be called to update the game status
	s.mockGameRepo.EXPECT().
		SaveGame(gomock.Any(), &gameRepo.SaveGameInput{
			Game: &models.Game{
				ID:           s.testGameID,
				ChannelID:    s.testChannelID,
				CreatorID:    s.testCreatorID,
				Status:       models.GameStatusCompleted,
				CreatedAt:    s.testTime,
				UpdatedAt:    s.testTime,
				Participants: game.Participants,
			},
		}).Return(nil)

	// Mock session leaderboard data
	// Expect GetDrinkRecordsForSession to be called
	s.mockDrinkRepo.EXPECT().
		GetDrinkRecordsForSession(gomock.Any(), &ledgerRepo.GetDrinkRecordsForSessionInput{
			SessionID: "test-session-id",
		}).
		Return(&ledgerRepo.GetDrinkRecordsForSessionOutput{
			Records: []*models.DrinkLedger{
				{
					ID:           "drink-1",
					GameID:       "previous-game-id",
					FromPlayerID: s.testCreatorID,
					ToPlayerID:   s.testPlayerID,
					Reason:       models.DrinkReasonCriticalHit,
					Timestamp:    s.testTime,
					SessionID:    "test-session-id",
					Paid:         false,
				},
				{
					ID:           "drink-2",
					GameID:       "previous-game-id",
					FromPlayerID: s.testPlayerID,
					ToPlayerID:   s.testCreatorID,
					Reason:       models.DrinkReasonCriticalFail,
					Timestamp:    s.testTime,
					SessionID:    "test-session-id",
					Paid:         true,
				},
				{
					ID:         "drink-3",
					GameID:     "previous-game-id",
					ToPlayerID: s.testPlayerID,
					Reason:     models.DrinkReasonLowestRoll,
					Timestamp:  s.testTime,
					SessionID:  "test-session-id",
					Paid:       false,
				},
				{
					ID:           "drink-4",
					GameID:       "previous-game-id",
					FromPlayerID: "third-player-id",
					ToPlayerID:   s.testPlayerID,
					Reason:       models.DrinkReasonCriticalHit,
					Timestamp:    s.testTime,
					SessionID:    "test-session-id",
					Paid:         false,
				},
				{
					ID:         "drink-5",
					GameID:     "previous-game-id",
					ToPlayerID: "third-player-id",
					Reason:     models.DrinkReasonLowestRoll,
					Timestamp:  s.testTime,
					SessionID:  "test-session-id",
					Paid:       false,
				},
				{
					ID:         "drink-6",
					GameID:     s.testGameID,
					ToPlayerID: s.testCreatorID,
					Reason:     models.DrinkReasonCriticalFail,
					Timestamp:  s.testTime,
					SessionID:  "test-session-id",
					Paid:       false,
				},
			},
		}, nil)

	// Expect GetPlayer to be called for all participants in the session
	s.mockPlayerRepo.EXPECT().
		GetPlayer(gomock.Any(), gomock.Any()).
		Return(&models.Player{
			ID:            s.testPlayerID,
			Name:          s.testPlayerName,
			CurrentGameID: s.testGameID,
		}, nil).AnyTimes()

	// Act
	output, err := s.gameService.EndGame(s.ctx, &EndGameInput{
		Game: game,
	})

	// Assert
	s.Require().NoError(err)
	s.Require().NotNil(output)
	s.False(output.NeedsRollOff)             // No roll-offs needed
	s.Equal("", output.RollOffGameID)        // No roll-off game ID
	s.Equal(0, len(output.RollOffPlayerIDs)) // No roll-off players

	// Verify session information
	s.Equal("test-session-id", output.SessionID)
	s.Require().NotEmpty(output.SessionLeaderboard)

	// Verify leaderboard entries
	s.Equal(3, len(output.SessionLeaderboard))

	// Check if the leaderboard is sorted by drink count (most drinks first)
	s.Equal(s.testPlayerID, output.SessionLeaderboard[0].PlayerID)
	s.Equal(3, output.SessionLeaderboard[0].DrinkCount)
	s.Equal(0, output.SessionLeaderboard[0].PaidCount)

	s.Equal(s.testCreatorID, output.SessionLeaderboard[1].PlayerID)
	s.Equal(2, output.SessionLeaderboard[1].DrinkCount)
	s.Equal(1, output.SessionLeaderboard[1].PaidCount)

	s.Equal("third-player-id", output.SessionLeaderboard[2].PlayerID)
	s.Equal(1, output.SessionLeaderboard[2].DrinkCount)
	s.Equal(0, output.SessionLeaderboard[2].PaidCount)
}

func TestGameServiceSuite(t *testing.T) {
	suite.Run(t, new(GameServiceTestSuite))
}

func (s *GameServiceTestSuite) TestPayDrink_HappyPath() {
	// Set up test data
	testDrinkID := "test-drink-id"
	testDrink := &models.DrinkLedger{
		ID:           testDrinkID,
		GameID:       s.testGameID,
		FromPlayerID: s.testCreatorID,
		ToPlayerID:   s.testPlayerID,
		Reason:       models.DrinkReasonCriticalHit,
		Timestamp:    s.testTime,
		Paid:         false,
	}
	
	// Set up expectations
	// Get the game
	s.mockGameRepo.EXPECT().GetGame(s.ctx, &gameRepo.GetGameInput{
		GameID: s.testGameID,
	}).Return(s.expectedGameWithPlayer, nil)
	
	// Get the session ID for the channel
	s.mockDrinkRepo.EXPECT().GetCurrentSession(s.ctx, &ledgerRepo.GetCurrentSessionInput{
		ChannelID: s.testChannelID,
	}).Return(&ledgerRepo.GetCurrentSessionOutput{
		Session: &models.Session{
			ID: s.testSessionID,
		},
	}, nil)
	
	// Get drink records for the session
	s.mockDrinkRepo.EXPECT().GetDrinkRecordsForSession(s.ctx, &ledgerRepo.GetDrinkRecordsForSessionInput{
		SessionID: s.testSessionID,
	}).Return(&ledgerRepo.GetDrinkRecordsForSessionOutput{
		Records: []*models.DrinkLedger{testDrink},
	}, nil)
	
	// Mark the drink as paid
	s.mockDrinkRepo.EXPECT().MarkDrinkPaid(s.ctx, &ledgerRepo.MarkDrinkPaidInput{
		DrinkID: testDrinkID,
	}).Return(nil)
	
	// Execute the method
	result, err := s.gameService.PayDrink(s.ctx, &PayDrinkInput{
		GameID:   s.testGameID,
		PlayerID: s.testPlayerID,
	})
	
	// Verify the result
	s.NoError(err)
	s.NotNil(result)
	s.True(result.Success)
	s.Equal(s.expectedGameWithPlayer, result.Game)
	s.NotNil(result.DrinkRecord)
	s.Equal(testDrinkID, result.DrinkRecord.ID)
	s.True(result.DrinkRecord.Paid)
}

func (s *GameServiceTestSuite) TestPayDrink_NoUnpaidDrinks() {
	// Set up test data
	testDrinkID := "test-drink-id"
	testDrink := &models.DrinkLedger{
		ID:           testDrinkID,
		GameID:       s.testGameID,
		FromPlayerID: s.testCreatorID,
		ToPlayerID:   "different-player-id", // Different player
		Reason:       models.DrinkReasonCriticalHit,
		Timestamp:    s.testTime,
		Paid:         false,
	}
	
	// Set up expectations
	// Get the game
	s.mockGameRepo.EXPECT().GetGame(s.ctx, &gameRepo.GetGameInput{
		GameID: s.testGameID,
	}).Return(s.expectedGameWithPlayer, nil)
	
	// Get the session ID for the channel
	s.mockDrinkRepo.EXPECT().GetCurrentSession(s.ctx, &ledgerRepo.GetCurrentSessionInput{
		ChannelID: s.testChannelID,
	}).Return(&ledgerRepo.GetCurrentSessionOutput{
		Session: &models.Session{
			ID: s.testSessionID,
		},
	}, nil)
	
	// Get drink records for the session
	s.mockDrinkRepo.EXPECT().GetDrinkRecordsForSession(s.ctx, &ledgerRepo.GetDrinkRecordsForSessionInput{
		SessionID: s.testSessionID,
	}).Return(&ledgerRepo.GetDrinkRecordsForSessionOutput{
		Records: []*models.DrinkLedger{testDrink},
	}, nil)
	
	// Execute the method
	result, err := s.gameService.PayDrink(s.ctx, &PayDrinkInput{
		GameID:   s.testGameID,
		PlayerID: s.testPlayerID,
	})
	
	// Verify the result
	s.Error(err)
	s.Nil(result)
	s.Contains(err.Error(), "no unpaid drinks found")
}
