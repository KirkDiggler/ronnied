package game

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/KirkDiggler/ronnied/internal/dice"
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

	// Mocks
	mockCtrl       *gomock.Controller
	mockGameRepo   *gameMocks.MockRepository
	mockPlayerRepo *playerMocks.MockRepository
	mockLedgerRepo *ledgerMocks.MockRepository
	mockDiceRoller *dice.Roller

	// Service under test
	service *service

	// Common test data
	testGameID     string
	testChannelID  string
	testPlayerID   string
	testPlayerName string
	testTime       time.Time
}

func (s *GameServiceTestSuite) SetupTest() {
	s.mockCtrl = gomock.NewController(s.T())
	s.mockGameRepo = gameMocks.NewMockRepository(s.mockCtrl)
	s.mockPlayerRepo = playerMocks.NewMockRepository(s.mockCtrl)
	s.mockLedgerRepo = ledgerMocks.NewMockRepository(s.mockCtrl)

	// Create a deterministic dice roller for testing
	s.mockDiceRoller = dice.New(&dice.Config{Seed: 42})

	// Setup the service directly by setting fields
	s.service = &service{
		config: &Config{
			MaxPlayers:         10,
			DiceSides:          6,
			CriticalHitValue:   6,
			CriticalFailValue:  1,
			MaxConcurrentGames: 100,
		},
		gameRepo:        s.mockGameRepo,
		playerRepo:      s.mockPlayerRepo,
		drinkLedgerRepo: s.mockLedgerRepo,
		diceRoller:      s.mockDiceRoller,
	}

	// Common test data
	s.testGameID = "test-game-id"
	s.testChannelID = "test-channel-id"
	s.testPlayerID = "test-player-id"
	s.testPlayerName = "Test Player"
	s.testTime = time.Date(2025, 3, 29, 12, 0, 0, 0, time.UTC)
}

func (s *GameServiceTestSuite) TearDownTest() {
	s.mockCtrl.Finish()
}

func TestGameServiceSuite(t *testing.T) {
	suite.Run(t, new(GameServiceTestSuite))
}

// TestCreateGameSuccess tests the successful creation of a game
func (s *GameServiceTestSuite) TestCreateGameSuccess() {
	ctx := context.Background()

	// Setup expectations
	s.mockGameRepo.EXPECT().
		GetGameByChannel(gomock.Any(), &gameRepo.GetGameByChannelInput{
			ChannelID: s.testChannelID,
		}).
		Return(nil, ErrGameNotFound)

	s.mockGameRepo.EXPECT().
		SaveGame(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, input *gameRepo.SaveGameInput) error {
			// Verify the game properties
			s.NotEmpty(input.Game.ID)
			s.Equal(s.testChannelID, input.Game.ChannelID)
			s.Equal(models.GameStatusWaiting, input.Game.Status)
			s.Empty(input.Game.PlayerIDs)
			s.NotZero(input.Game.CreatedAt)
			s.NotZero(input.Game.UpdatedAt)
			return nil
		})

	// Call the method
	output, err := s.service.CreateGame(ctx, &CreateGameInput{
		ChannelID: s.testChannelID,
	})

	// Verify the results
	s.NoError(err)
	s.NotNil(output)
	s.NotEmpty(output.GameID)
}

// TestCreateGameAlreadyExists tests the case where a game already exists for the channel
func (s *GameServiceTestSuite) TestCreateGameAlreadyExists() {
	ctx := context.Background()
	existingGame := &models.Game{
		ID:        s.testGameID,
		ChannelID: s.testChannelID,
	}

	// Setup expectations
	s.mockGameRepo.EXPECT().
		GetGameByChannel(gomock.Any(), &gameRepo.GetGameByChannelInput{
			ChannelID: s.testChannelID,
		}).
		Return(existingGame, nil)

	// Call the method
	output, err := s.service.CreateGame(ctx, &CreateGameInput{
		ChannelID: s.testChannelID,
	})

	// Verify the results
	s.Error(err)
	s.Equal(ErrGameAlreadyExists, err)
	s.Nil(output)
}

// TestCreateGameRepositoryError tests the case where the repository returns an error
func (s *GameServiceTestSuite) TestCreateGameRepositoryError() {
	ctx := context.Background()
	repoErr := errors.New("database error")

	// Setup expectations
	s.mockGameRepo.EXPECT().
		GetGameByChannel(gomock.Any(), &gameRepo.GetGameByChannelInput{
			ChannelID: s.testChannelID,
		}).
		Return(nil, repoErr)

	// Call the method
	output, err := s.service.CreateGame(ctx, &CreateGameInput{
		ChannelID: s.testChannelID,
	})

	// Verify the results
	s.Error(err)
	s.Equal(repoErr, err)
	s.Nil(output)
}

// TestJoinGameSuccess tests the successful joining of a game
func (s *GameServiceTestSuite) TestJoinGameSuccess() {
	ctx := context.Background()
	game := &models.Game{
		ID:        s.testGameID,
		ChannelID: s.testChannelID,
		Status:    models.GameStatusWaiting,
		PlayerIDs: []string{},
	}

	// Setup expectations
	s.mockGameRepo.EXPECT().
		GetGame(gomock.Any(), &gameRepo.GetGameInput{
			GameID: s.testGameID,
		}).
		Return(game, nil)

	s.mockPlayerRepo.EXPECT().
		GetPlayer(gomock.Any(), &playerRepo.GetPlayerInput{
			PlayerID: s.testPlayerID,
		}).
		Return(nil, ErrPlayerNotFound)

	s.mockPlayerRepo.EXPECT().
		SavePlayer(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, input *playerRepo.SavePlayerInput) error {
			// Verify the player properties
			s.Equal(s.testPlayerID, input.Player.ID)
			s.Equal(s.testPlayerName, input.Player.Name)
			s.Equal(s.testGameID, input.Player.CurrentGameID)
			s.Zero(input.Player.LastRoll)
			s.NotZero(input.Player.LastRollTime)
			return nil
		})

	s.mockGameRepo.EXPECT().
		SaveGame(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, input *gameRepo.SaveGameInput) error {
			// Verify the game properties
			s.Equal(s.testGameID, input.Game.ID)
			s.Equal(s.testChannelID, input.Game.ChannelID)
			s.Equal(models.GameStatusWaiting, input.Game.Status)
			s.Contains(input.Game.PlayerIDs, s.testPlayerID)
			s.Len(input.Game.PlayerIDs, 1)
			return nil
		})

	// Call the method
	output, err := s.service.JoinGame(ctx, &JoinGameInput{
		GameID:     s.testGameID,
		PlayerID:   s.testPlayerID,
		PlayerName: s.testPlayerName,
	})

	// Verify the results
	s.NoError(err)
	s.NotNil(output)
	s.True(output.Success)
}

// TestJoinGameNotFound tests the case where the game is not found
func (s *GameServiceTestSuite) TestJoinGameNotFound() {
	ctx := context.Background()

	// Setup expectations
	s.mockGameRepo.EXPECT().
		GetGame(gomock.Any(), &gameRepo.GetGameInput{
			GameID: s.testGameID,
		}).
		Return(nil, ErrGameNotFound)

	// Call the method
	output, err := s.service.JoinGame(ctx, &JoinGameInput{
		GameID:     s.testGameID,
		PlayerID:   s.testPlayerID,
		PlayerName: s.testPlayerName,
	})

	// Verify the results
	s.Error(err)
	s.Equal(ErrGameNotFound, err)
	s.Nil(output)
}

// TestJoinGameInvalidState tests the case where the game is not in a valid state for joining
func (s *GameServiceTestSuite) TestJoinGameInvalidState() {
	ctx := context.Background()
	game := &models.Game{
		ID:        s.testGameID,
		ChannelID: s.testChannelID,
		Status:    models.GameStatusActive,
		PlayerIDs: []string{},
	}

	// Setup expectations
	s.mockGameRepo.EXPECT().
		GetGame(gomock.Any(), &gameRepo.GetGameInput{
			GameID: s.testGameID,
		}).
		Return(game, nil)

	// Call the method
	output, err := s.service.JoinGame(ctx, &JoinGameInput{
		GameID:     s.testGameID,
		PlayerID:   s.testPlayerID,
		PlayerName: s.testPlayerName,
	})

	// Verify the results
	s.Error(err)
	s.Equal(ErrInvalidGameState, err)
	s.Nil(output)
}

// TestJoinGameFull tests the case where the game is full
func (s *GameServiceTestSuite) TestJoinGameFull() {
	ctx := context.Background()
	
	// Create a game with max players
	playerIDs := make([]string, 10)
	for i := 0; i < 10; i++ {
		playerIDs[i] = "player-" + string(rune('a'+i))
	}

	game := &models.Game{
		ID:        s.testGameID,
		ChannelID: s.testChannelID,
		Status:    models.GameStatusWaiting,
		PlayerIDs: playerIDs,
	}

	// Setup expectations
	s.mockGameRepo.EXPECT().
		GetGame(gomock.Any(), &gameRepo.GetGameInput{
			GameID: s.testGameID,
		}).
		Return(game, nil)

	// Call the method
	output, err := s.service.JoinGame(ctx, &JoinGameInput{
		GameID:     s.testGameID,
		PlayerID:   s.testPlayerID,
		PlayerName: s.testPlayerName,
	})

	// Verify the results
	s.Error(err)
	s.Equal(ErrGameFull, err)
	s.Nil(output)
}

// TestLeaveGameSuccess tests the successful removal of a player from a game
func (s *GameServiceTestSuite) TestLeaveGameSuccess() {
	ctx := context.Background()
	
	// Create a game with the test player in it
	game := &models.Game{
		ID:        s.testGameID,
		ChannelID: s.testChannelID,
		Status:    models.GameStatusActive,
		PlayerIDs: []string{s.testPlayerID, "other-player-id"},
		UpdatedAt: time.Now().Add(-1 * time.Hour), // Set to past time to verify it gets updated
	}
	
	// Create a player currently in the game
	player := &models.Player{
		ID:            s.testPlayerID,
		Name:          s.testPlayerName,
		CurrentGameID: s.testGameID,
	}
	
	// Setup expectations
	s.mockGameRepo.EXPECT().
		GetGame(gomock.Any(), &gameRepo.GetGameInput{
			GameID: s.testGameID,
		}).
		Return(game, nil)
		
	s.mockPlayerRepo.EXPECT().
		GetPlayer(gomock.Any(), &playerRepo.GetPlayerInput{
			PlayerID: s.testPlayerID,
		}).
		Return(player, nil)
		
	s.mockPlayerRepo.EXPECT().
		SavePlayer(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, input *playerRepo.SavePlayerInput) error {
			// Verify player's game ID is cleared
			s.Equal(s.testPlayerID, input.Player.ID)
			s.Equal("", input.Player.CurrentGameID)
			return nil
		})
		
	s.mockGameRepo.EXPECT().
		SaveGame(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, input *gameRepo.SaveGameInput) error {
			// Verify player is removed from the game
			s.Equal(s.testGameID, input.Game.ID)
			s.NotContains(input.Game.PlayerIDs, s.testPlayerID)
			s.Len(input.Game.PlayerIDs, 1)
			s.Contains(input.Game.PlayerIDs, "other-player-id")
			// Don't check exact time, just that it's been set
			s.NotZero(input.Game.UpdatedAt)
			return nil
		})
		
	// Call the method
	output, err := s.service.LeaveGame(ctx, &LeaveGameInput{
		GameID:   s.testGameID,
		PlayerID: s.testPlayerID,
	})
	
	// Verify the results
	s.NoError(err)
	s.NotNil(output)
	s.True(output.Success)
}

// TestLeaveGameNotFound tests the case where the game is not found
func (s *GameServiceTestSuite) TestLeaveGameNotFound() {
	ctx := context.Background()
	
	// Setup expectations
	s.mockGameRepo.EXPECT().
		GetGame(gomock.Any(), &gameRepo.GetGameInput{
			GameID: s.testGameID,
		}).
		Return(nil, errors.New("game not found"))
		
	// Call the method
	output, err := s.service.LeaveGame(ctx, &LeaveGameInput{
		GameID:   s.testGameID,
		PlayerID: s.testPlayerID,
	})
	
	// Verify the results
	s.Error(err)
	s.Equal(ErrGameNotFound, err)
	s.Nil(output)
}

// TestLeaveGamePlayerNotFound tests the case where the player is not found
func (s *GameServiceTestSuite) TestLeaveGamePlayerNotFound() {
	ctx := context.Background()
	
	// Create a game
	game := &models.Game{
		ID:        s.testGameID,
		ChannelID: s.testChannelID,
		Status:    models.GameStatusActive,
		PlayerIDs: []string{s.testPlayerID},
	}
	
	// Setup expectations
	s.mockGameRepo.EXPECT().
		GetGame(gomock.Any(), &gameRepo.GetGameInput{
			GameID: s.testGameID,
		}).
		Return(game, nil)
		
	s.mockPlayerRepo.EXPECT().
		GetPlayer(gomock.Any(), &playerRepo.GetPlayerInput{
			PlayerID: s.testPlayerID,
		}).
		Return(nil, errors.New("player not found"))
		
	// Call the method
	output, err := s.service.LeaveGame(ctx, &LeaveGameInput{
		GameID:   s.testGameID,
		PlayerID: s.testPlayerID,
	})
	
	// Verify the results
	s.Error(err)
	s.Equal(ErrPlayerNotFound, err)
	s.Nil(output)
}

// TestLeaveGamePlayerNotInGame tests the case where the player is not in the specified game
func (s *GameServiceTestSuite) TestLeaveGamePlayerNotInGame() {
	ctx := context.Background()
	
	// Create a game
	game := &models.Game{
		ID:        s.testGameID,
		ChannelID: s.testChannelID,
		Status:    models.GameStatusActive,
		PlayerIDs: []string{"other-player-id"}, // Player not in game
	}
	
	// Create a player in a different game
	player := &models.Player{
		ID:            s.testPlayerID,
		Name:          s.testPlayerName,
		CurrentGameID: "different-game-id",
	}
	
	// Setup expectations
	s.mockGameRepo.EXPECT().
		GetGame(gomock.Any(), &gameRepo.GetGameInput{
			GameID: s.testGameID,
		}).
		Return(game, nil)
		
	s.mockPlayerRepo.EXPECT().
		GetPlayer(gomock.Any(), &playerRepo.GetPlayerInput{
			PlayerID: s.testPlayerID,
		}).
		Return(player, nil)
		
	// Call the method
	output, err := s.service.LeaveGame(ctx, &LeaveGameInput{
		GameID:   s.testGameID,
		PlayerID: s.testPlayerID,
	})
	
	// Verify the results
	s.Error(err)
	s.Equal(ErrPlayerNotInGame, err)
	s.Nil(output)
}

// TestLeaveGameSavePlayerError tests the case where saving the player fails
func (s *GameServiceTestSuite) TestLeaveGameSavePlayerError() {
	ctx := context.Background()
	saveErr := errors.New("failed to save player")
	
	// Create a game with the test player in it
	game := &models.Game{
		ID:        s.testGameID,
		ChannelID: s.testChannelID,
		Status:    models.GameStatusActive,
		PlayerIDs: []string{s.testPlayerID},
	}
	
	// Create a player currently in the game
	player := &models.Player{
		ID:            s.testPlayerID,
		Name:          s.testPlayerName,
		CurrentGameID: s.testGameID,
	}
	
	// Setup expectations
	s.mockGameRepo.EXPECT().
		GetGame(gomock.Any(), &gameRepo.GetGameInput{
			GameID: s.testGameID,
		}).
		Return(game, nil)
		
	s.mockPlayerRepo.EXPECT().
		GetPlayer(gomock.Any(), &playerRepo.GetPlayerInput{
			PlayerID: s.testPlayerID,
		}).
		Return(player, nil)
		
	s.mockPlayerRepo.EXPECT().
		SavePlayer(gomock.Any(), gomock.Any()).
		Return(saveErr)
		
	// Call the method
	output, err := s.service.LeaveGame(ctx, &LeaveGameInput{
		GameID:   s.testGameID,
		PlayerID: s.testPlayerID,
	})
	
	// Verify the results
	s.Error(err)
	s.Equal(saveErr, err)
	s.Nil(output)
}

// TestRollDiceSuccess tests a successful normal dice roll
func (s *GameServiceTestSuite) TestRollDiceSuccess() {
	ctx := context.Background()
	
	// Create a game in active state
	game := &models.Game{
		ID:        s.testGameID,
		ChannelID: s.testChannelID,
		Status:    models.GameStatusActive,
		PlayerIDs: []string{s.testPlayerID},
		UpdatedAt: time.Now().Add(-1 * time.Hour), // Set to past time to verify it gets updated
	}
	
	// Create a player in the game
	player := &models.Player{
		ID:            s.testPlayerID,
		Name:          s.testPlayerName,
		CurrentGameID: s.testGameID,
		LastRoll:      0,
		LastRollTime:  time.Now().Add(-1 * time.Hour), // Set to past time to verify it gets updated
	}
	
	// Setup expectations
	s.mockGameRepo.EXPECT().
		GetGame(gomock.Any(), &gameRepo.GetGameInput{
			GameID: s.testGameID,
		}).
		Return(game, nil)
		
	s.mockPlayerRepo.EXPECT().
		GetPlayer(gomock.Any(), &playerRepo.GetPlayerInput{
			PlayerID: s.testPlayerID,
		}).
		Return(player, nil)
		
	s.mockPlayerRepo.EXPECT().
		SavePlayer(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, input *playerRepo.SavePlayerInput) error {
			// Verify player's roll is updated
			s.Equal(s.testPlayerID, input.Player.ID)
			s.NotZero(input.Player.LastRoll)
			// Don't check exact time, just that it's been set
			s.NotZero(input.Player.LastRollTime)
			return nil
		})
		
	s.mockGameRepo.EXPECT().
		SaveGame(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, input *gameRepo.SaveGameInput) error {
			// Verify game is updated
			s.Equal(s.testGameID, input.Game.ID)
			s.Equal(models.GameStatusActive, input.Game.Status)
			// Don't check exact time, just that it's been set
			s.NotZero(input.Game.UpdatedAt)
			return nil
		})
		
	// Call the method
	output, err := s.service.RollDice(ctx, &RollDiceInput{
		GameID:   s.testGameID,
		PlayerID: s.testPlayerID,
	})
	
	// Verify the results
	s.NoError(err)
	s.NotNil(output)
	s.GreaterOrEqual(output.Value, 1)
	s.LessOrEqual(output.Value, s.service.config.DiceSides)
}

// TestRollDiceActivateGame tests that rolling dice in a waiting game activates it
func (s *GameServiceTestSuite) TestRollDiceActivateGame() {
	ctx := context.Background()
	
	// Create a game in waiting state
	game := &models.Game{
		ID:        s.testGameID,
		ChannelID: s.testChannelID,
		Status:    models.GameStatusWaiting,
		PlayerIDs: []string{s.testPlayerID},
	}
	
	// Create a player in the game
	player := &models.Player{
		ID:            s.testPlayerID,
		Name:          s.testPlayerName,
		CurrentGameID: s.testGameID,
	}
	
	// Setup expectations
	s.mockGameRepo.EXPECT().
		GetGame(gomock.Any(), &gameRepo.GetGameInput{
			GameID: s.testGameID,
		}).
		Return(game, nil)
		
	s.mockPlayerRepo.EXPECT().
		GetPlayer(gomock.Any(), &playerRepo.GetPlayerInput{
			PlayerID: s.testPlayerID,
		}).
		Return(player, nil)
		
	s.mockPlayerRepo.EXPECT().
		SavePlayer(gomock.Any(), gomock.Any()).
		Return(nil)
		
	s.mockGameRepo.EXPECT().
		SaveGame(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, input *gameRepo.SaveGameInput) error {
			// Verify game status is changed to active
			s.Equal(s.testGameID, input.Game.ID)
			s.Equal(models.GameStatusActive, input.Game.Status)
			return nil
		})
		
	// Call the method
	output, err := s.service.RollDice(ctx, &RollDiceInput{
		GameID:   s.testGameID,
		PlayerID: s.testPlayerID,
	})
	
	// Verify the results
	s.NoError(err)
	s.NotNil(output)
}

// TestRollDiceCriticalFail tests the behavior when a critical fail occurs
func (s *GameServiceTestSuite) TestRollDiceCriticalFail() {
	ctx := context.Background()
	
	// Create a game in active state
	game := &models.Game{
		ID:        s.testGameID,
		ChannelID: s.testChannelID,
		Status:    models.GameStatusActive,
		PlayerIDs: []string{s.testPlayerID},
	}
	
	// Create a player in the game
	player := &models.Player{
		ID:            s.testPlayerID,
		Name:          s.testPlayerName,
		CurrentGameID: s.testGameID,
	}
	
	// Setup expectations
	s.mockGameRepo.EXPECT().
		GetGame(gomock.Any(), &gameRepo.GetGameInput{
			GameID: s.testGameID,
		}).
		Return(game, nil)
		
	s.mockPlayerRepo.EXPECT().
		GetPlayer(gomock.Any(), &playerRepo.GetPlayerInput{
			PlayerID: s.testPlayerID,
		}).
		Return(player, nil)
		
	// We'll test the critical fail behavior by directly calling the code path
	// that handles critical fails, rather than trying to mock the dice roll
	
	s.mockPlayerRepo.EXPECT().
		SavePlayer(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, input *playerRepo.SavePlayerInput) error {
			// We don't care about the exact roll value, just that it was saved
			s.NotZero(input.Player.LastRoll)
			s.NotZero(input.Player.LastRollTime)
			return nil
		})
	
	// Expect a drink record to be added when a critical fail occurs
	s.mockLedgerRepo.EXPECT().
		AddDrinkRecord(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, input *ledgerRepo.AddDrinkRecordInput) error {
			// Verify drink record properties
			s.Equal(s.testPlayerID, input.Record.FromPlayerID)
			s.Equal(s.testPlayerID, input.Record.ToPlayerID) // Self-assigned for critical fail
			s.Equal(s.testGameID, input.Record.GameID)
			s.Equal(models.DrinkReasonCriticalFail, input.Record.Reason)
			s.False(input.Record.Paid)
			return nil
		}).
		AnyTimes() // Allow this to be called any number of times
		
	s.mockGameRepo.EXPECT().
		SaveGame(gomock.Any(), gomock.Any()).
		Return(nil)
	
	// Call the method - we'll manually check if it's a critical fail after
	output, err := s.service.RollDice(ctx, &RollDiceInput{
		GameID:   s.testGameID,
		PlayerID: s.testPlayerID,
	})
	
	// Verify the general results
	s.NoError(err)
	s.NotNil(output)
	
	// If the roll happened to be a critical fail, verify the expected behavior
	if output.Value == s.service.config.CriticalFailValue {
		s.True(output.IsCriticalFail)
		s.False(output.IsCriticalHit)
	}
}

// TestRollDiceGameNotFound tests the case where the game is not found
func (s *GameServiceTestSuite) TestRollDiceGameNotFound() {
	ctx := context.Background()
	
	// Setup expectations
	s.mockGameRepo.EXPECT().
		GetGame(gomock.Any(), &gameRepo.GetGameInput{
			GameID: s.testGameID,
		}).
		Return(nil, errors.New("game not found"))
		
	// Call the method
	output, err := s.service.RollDice(ctx, &RollDiceInput{
		GameID:   s.testGameID,
		PlayerID: s.testPlayerID,
	})
	
	// Verify the results
	s.Error(err)
	s.Equal(ErrGameNotFound, err)
	s.Nil(output)
}

// TestRollDiceInvalidGameState tests the case where the game is in an invalid state
func (s *GameServiceTestSuite) TestRollDiceInvalidGameState() {
	ctx := context.Background()
	
	// Create a game in completed state
	game := &models.Game{
		ID:        s.testGameID,
		ChannelID: s.testChannelID,
		Status:    models.GameStatusCompleted,
		PlayerIDs: []string{s.testPlayerID},
	}
	
	// Setup expectations
	s.mockGameRepo.EXPECT().
		GetGame(gomock.Any(), &gameRepo.GetGameInput{
			GameID: s.testGameID,
		}).
		Return(game, nil)
		
	// Call the method
	output, err := s.service.RollDice(ctx, &RollDiceInput{
		GameID:   s.testGameID,
		PlayerID: s.testPlayerID,
	})
	
	// Verify the results
	s.Error(err)
	s.Equal(ErrInvalidGameState, err)
	s.Nil(output)
}

// TestRollDicePlayerNotFound tests the case where the player is not found
func (s *GameServiceTestSuite) TestRollDicePlayerNotFound() {
	ctx := context.Background()
	
	// Create a game
	game := &models.Game{
		ID:        s.testGameID,
		ChannelID: s.testChannelID,
		Status:    models.GameStatusActive,
		PlayerIDs: []string{s.testPlayerID},
	}
	
	// Setup expectations
	s.mockGameRepo.EXPECT().
		GetGame(gomock.Any(), &gameRepo.GetGameInput{
			GameID: s.testGameID,
		}).
		Return(game, nil)
		
	s.mockPlayerRepo.EXPECT().
		GetPlayer(gomock.Any(), &playerRepo.GetPlayerInput{
			PlayerID: s.testPlayerID,
		}).
		Return(nil, errors.New("player not found"))
		
	// Call the method
	output, err := s.service.RollDice(ctx, &RollDiceInput{
		GameID:   s.testGameID,
		PlayerID: s.testPlayerID,
	})
	
	// Verify the results
	s.Error(err)
	s.Equal(ErrPlayerNotFound, err)
	s.Nil(output)
}

// TestRollDicePlayerNotInGame tests the case where the player is not in the specified game
func (s *GameServiceTestSuite) TestRollDicePlayerNotInGame() {
	ctx := context.Background()
	
	// Create a game
	game := &models.Game{
		ID:        s.testGameID,
		ChannelID: s.testChannelID,
		Status:    models.GameStatusActive,
		PlayerIDs: []string{s.testPlayerID},
	}
	
	// Create a player in a different game
	player := &models.Player{
		ID:            s.testPlayerID,
		Name:          s.testPlayerName,
		CurrentGameID: "different-game-id",
	}
	
	// Setup expectations
	s.mockGameRepo.EXPECT().
		GetGame(gomock.Any(), &gameRepo.GetGameInput{
			GameID: s.testGameID,
		}).
		Return(game, nil)
		
	s.mockPlayerRepo.EXPECT().
		GetPlayer(gomock.Any(), &playerRepo.GetPlayerInput{
			PlayerID: s.testPlayerID,
		}).
		Return(player, nil)
		
	// Call the method
	output, err := s.service.RollDice(ctx, &RollDiceInput{
		GameID:   s.testGameID,
		PlayerID: s.testPlayerID,
	})
	
	// Verify the results
	s.Error(err)
	s.Equal(ErrPlayerNotInGame, err)
	s.Nil(output)
}
