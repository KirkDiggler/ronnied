package game_test

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
	"github.com/KirkDiggler/ronnied/internal/services/game"
	"go.uber.org/mock/gomock"
	"github.com/stretchr/testify/suite"
)

type GameServiceTestSuite struct {
	suite.Suite
	mockCtrl          *gomock.Controller
	mockGameRepo      *gameMocks.MockRepository
	mockPlayerRepo    *playerMocks.MockRepository
	mockDrinkRepo     *ledgerMocks.MockRepository
	mockDiceRoller    *diceMocks.MockRoller
	mockClock         *mocks.MockClock
	mockUUID          *uuidMocks.MockUUID
	gameService       game.Service
	ctx               context.Context
	
	// Test data
	testTime          time.Time
	testGameID        string
	testChannelID     string
	testCreatorID     string
	testCreatorName   string
	testParticipantID string
	testPlayerID      string
	testPlayerName    string
	
	// Reusable test fixtures
	expectedGame           *models.Game
	expectedParticipant    *models.Participant
	expectedActiveGame     *models.Game
	expectedGameWithPlayer *models.Game
	expectedPlayer         *models.Player
	
	// Reusable test inputs
	createGameInput *game.CreateGameInput
	startGameInput  *game.StartGameInput
	joinGameInput   *game.JoinGameInput
	rollDiceInput   *game.RollDiceInput
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
	
	// Initialize reusable test inputs
	s.createGameInput = &game.CreateGameInput{
		ChannelID:   s.testChannelID,
		CreatorID:   s.testCreatorID,
		CreatorName: s.testCreatorName,
	}
	
	s.startGameInput = &game.StartGameInput{
		GameID:   s.testGameID,
		PlayerID: s.testCreatorID,
	}
	
	s.joinGameInput = &game.JoinGameInput{
		GameID:     s.testGameID,
		PlayerID:   s.testPlayerID,
		PlayerName: s.testPlayerName,
	}
	
	s.rollDiceInput = &game.RollDiceInput{
		GameID:   s.testGameID,
		PlayerID: s.testCreatorID,
	}

	// Create the service with mocked dependencies
	cfg := &game.Config{
		GameRepo:        s.mockGameRepo,
		PlayerRepo:      s.mockPlayerRepo,
		DrinkLedgerRepo: s.mockDrinkRepo,
		DiceRoller:      s.mockDiceRoller,
		Clock:           s.mockClock,
		UUIDGenerator:   s.mockUUID,
		MaxPlayers:      10, // Set a max players value for testing
		DiceSides:       6,  // Standard dice
		CriticalHitValue: 6, // Critical hit on 6
		CriticalFailValue: 1, // Critical fail on 1
	}

	var err error
	svc, err := game.New(cfg)
	s.Require().NoError(err)
	s.gameService = svc
}

func (s *GameServiceTestSuite) TearDownTest() {
	s.mockCtrl.Finish()
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
	s.True(errors.Is(err, game.ErrGameNotFound))
	s.Nil(output)
}

func (s *GameServiceTestSuite) TestStartGame_NotCreator() {
	// Create a different player ID for this test
	notCreatorInput := &game.StartGameInput{
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
	s.Equal(game.ErrNotCreator, err)
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
	s.True(errors.Is(err, game.ErrInvalidGameState))
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
	s.Equal(game.ErrNotEnoughPlayers, err)
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
		ID:           s.testGameID,
		ChannelID:    s.testChannelID,
		CreatorID:    s.testCreatorID,
		Status:       models.GameStatusWaiting,
		CreatedAt:    s.testTime,
		UpdatedAt:    s.testTime,
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
	s.True(errors.Is(err, game.ErrGameNotFound))
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
	s.True(errors.Is(err, game.ErrGameActive))
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
	s.True(errors.Is(err, game.ErrGameFull))
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
		ID:           s.testGameID,
		ChannelID:    s.testChannelID,
		CreatorID:    s.testCreatorID,
		Status:       models.GameStatusActive,
		CreatedAt:    s.testTime,
		UpdatedAt:    s.testTime,
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
		SaveGame(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, input *gameRepo.SaveGameInput) error {
			// Verify the game has been updated correctly
			s.Equal(s.testTime, input.Game.UpdatedAt)
			
			// Verify the participant has been updated correctly
			participant := input.Game.GetParticipant(s.testCreatorID)
			s.Require().NotNil(participant)
			s.Equal(3, participant.RollValue)
			s.Equal(s.testTime, *participant.RollTime)
			s.Equal(models.ParticipantStatusActive, participant.Status)
			
			return nil
		})

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
		ID:           s.testGameID,
		ChannelID:    s.testChannelID,
		CreatorID:    s.testCreatorID,
		Status:       models.GameStatusActive,
		CreatedAt:    s.testTime,
		UpdatedAt:    s.testTime,
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
		SaveGame(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, input *gameRepo.SaveGameInput) error {
			// Verify the game has been updated correctly
			s.Equal(s.testTime, input.Game.UpdatedAt)
			
			// Verify the participant has been updated correctly
			participant := input.Game.GetParticipant(s.testCreatorID)
			s.Require().NotNil(participant)
			s.Equal(6, participant.RollValue)
			s.Equal(s.testTime, *participant.RollTime)
			s.Equal(models.ParticipantStatusNeedsToAssign, participant.Status)
			
			return nil
		})

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
		ID:           s.testGameID,
		ChannelID:    s.testChannelID,
		CreatorID:    s.testCreatorID,
		Status:       models.GameStatusActive,
		CreatedAt:    s.testTime,
		UpdatedAt:    s.testTime,
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
		}).
		Return(&ledgerRepo.CreateDrinkRecordOutput{}, nil)

	// Expect SaveGame to be called with the updated game
	s.mockGameRepo.EXPECT().
		SaveGame(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, input *gameRepo.SaveGameInput) error {
			// Verify the game has been updated correctly
			s.Equal(s.testTime, input.Game.UpdatedAt)
			
			// Verify the participant has been updated correctly
			participant := input.Game.GetParticipant(s.testCreatorID)
			s.Require().NotNil(participant)
			s.Equal(1, participant.RollValue)
			s.Equal(s.testTime, *participant.RollTime)
			s.Equal(models.ParticipantStatusActive, participant.Status)
			
			return nil
		})

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
		Return(nil, game.ErrGameNotFound)

	// Act
	output, err := s.gameService.RollDice(s.ctx, s.rollDiceInput)

	// Assert
	s.Require().Error(err)
	s.True(errors.Is(err, game.ErrGameNotFound))
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
	s.True(errors.Is(err, game.ErrInvalidGameState))
	s.Nil(output)
}

func (s *GameServiceTestSuite) TestRollDice_PlayerNotInGame() {
	// Create an active game with a different player
	activeGame := &models.Game{
		ID:           s.testGameID,
		ChannelID:    s.testChannelID,
		CreatorID:    s.testCreatorID,
		Status:       models.GameStatusActive,
		CreatedAt:    s.testTime,
		UpdatedAt:    s.testTime,
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
	s.True(errors.Is(err, game.ErrPlayerNotInGame))
	s.Nil(output)
}

func (s *GameServiceTestSuite) TestRollDice_PlayerAlreadyRolled() {
	// Create a game with a participant who has already rolled
	rollTime := s.testTime.Add(-time.Hour) // Rolled an hour ago
	activeGame := &models.Game{
		ID:           s.testGameID,
		ChannelID:    s.testChannelID,
		CreatorID:    s.testCreatorID,
		Status:       models.GameStatusActive,
		CreatedAt:    s.testTime,
		UpdatedAt:    s.testTime,
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
		ID:           s.testGameID,
		ChannelID:    s.testChannelID,
		CreatorID:    s.testCreatorID,
		Status:       models.GameStatusActive,
		CreatedAt:    s.testTime,
		UpdatedAt:    s.testTime,
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
		SaveGame(gomock.Any(), gomock.Any()).
		Return(expectedError)

	// Act
	output, err := s.gameService.RollDice(s.ctx, s.rollDiceInput)

	// Assert
	s.Require().Error(err)
	s.Contains(err.Error(), expectedError.Error(), "Expected error to contain the original error message")
	s.Nil(output)
}

func TestGameServiceSuite(t *testing.T) {
	suite.Run(t, new(GameServiceTestSuite))
}
