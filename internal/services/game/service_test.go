package game

import (
	"context"
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
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestEndGame_TransitionToRollOff(t *testing.T) {
	// Setup controller
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Create mocks
	mockGameRepo := gameMocks.NewMockRepository(ctrl)
	mockPlayerRepo := playerMocks.NewMockRepository(ctrl)
	mockDrinkLedgerRepo := ledgerMocks.NewMockRepository(ctrl)
	mockClock := mocks.NewMockClock(ctrl)
	mockUUID := uuidMocks.NewMockUUID(ctrl)
	mockDiceRoller := diceMocks.NewMockRoller(ctrl)

	// Fixed time for testing
	now := time.Date(2025, 4, 6, 12, 0, 0, 0, time.UTC)
	mockClock.EXPECT().Now().Return(now).AnyTimes()

	// Game ID and player IDs
	gameID := "game123"
	player1ID := "player1"
	player2ID := "player2"
	player3ID := "player3"
	channelID := "channel123"
	creatorID := player1ID

	// Create roll time for all players
	rollTime := now.Add(-5 * time.Minute)

	// Create a game with 3 players, 2 of them tied for lowest roll
	game := &models.Game{
		ID:        gameID,
		ChannelID: channelID,
		CreatorID: creatorID,
		Status:    models.GameStatusActive,
		Participants: []*models.Participant{
			{
				ID:         "participant1",
				GameID:     gameID,
				PlayerID:   player1ID,
				PlayerName: "Player 1",
				Status:     models.ParticipantStatusActive,
				RollValue:  5,
				RollTime:   &rollTime,
			},
			{
				ID:         "participant2",
				GameID:     gameID,
				PlayerID:   player2ID,
				PlayerName: "Player 2",
				Status:     models.ParticipantStatusActive,
				RollValue:  2, // Tied for lowest
				RollTime:   &rollTime,
			},
			{
				ID:         "participant3",
				GameID:     gameID,
				PlayerID:   player3ID,
				PlayerName: "Player 3",
				Status:     models.ParticipantStatusActive,
				RollValue:  2, // Tied for lowest
				RollTime:   &rollTime,
			},
		},
		CreatedAt: now.Add(-30 * time.Minute),
		UpdatedAt: now.Add(-10 * time.Minute),
	}

	// Mock GetGame to return our test game
	mockGameRepo.EXPECT().
		GetGame(gomock.Any(), &gameRepo.GetGameInput{GameID: gameID}).
		Return(game, nil)

	// Mock GetDrinkRecordsForGame to return empty records
	mockDrinkLedgerRepo.EXPECT().
		GetDrinkRecordsForGame(gomock.Any(), &ledgerRepo.GetDrinkRecordsForGameInput{GameID: gameID}).
		Return(&ledgerRepo.GetDrinkRecordsForGameOutput{Records: []*models.DrinkLedger{}}, nil)

	// Mock GetPlayer for each player
	player1 := &models.Player{ID: player1ID, Name: "Player 1", CurrentGameID: gameID}
	player2 := &models.Player{ID: player2ID, Name: "Player 2", CurrentGameID: gameID}
	player3 := &models.Player{ID: player3ID, Name: "Player 3", CurrentGameID: gameID}

	mockPlayerRepo.EXPECT().
		GetPlayer(gomock.Any(), &playerRepo.GetPlayerInput{PlayerID: player1ID}).
		Return(player1, nil)
	mockPlayerRepo.EXPECT().
		GetPlayer(gomock.Any(), &playerRepo.GetPlayerInput{PlayerID: player2ID}).
		Return(player2, nil)
	mockPlayerRepo.EXPECT().
		GetPlayer(gomock.Any(), &playerRepo.GetPlayerInput{PlayerID: player3ID}).
		Return(player3, nil)

	// Roll-off game ID
	rollOffGameID := "rolloff123"

	// Mock CreateRollOffGame
	rollOffGame := &models.Game{
		ID:           rollOffGameID,
		ChannelID:    channelID,
		CreatorID:    creatorID,
		Status:       models.GameStatusRollOff,
		ParentGameID: gameID,
		Participants: []*models.Participant{
			{
				ID:         "rolloff-participant1",
				GameID:     rollOffGameID,
				PlayerID:   player2ID,
				PlayerName: "Player 2",
				Status:     models.ParticipantStatusWaitingToRoll,
			},
			{
				ID:         "rolloff-participant2",
				GameID:     rollOffGameID,
				PlayerID:   player3ID,
				PlayerName: "Player 3",
				Status:     models.ParticipantStatusWaitingToRoll,
			},
		},
		CreatedAt: now,
		UpdatedAt: now,
	}

	// Expected player names map for roll-off game creation
	expectedPlayerNames := map[string]string{
		player2ID: "Player 2",
		player3ID: "Player 3",
	}

	mockGameRepo.EXPECT().
		CreateRollOffGame(gomock.Any(), &gameRepo.CreateRollOffGameInput{
			ChannelID:    channelID,
			CreatorID:    creatorID,
			ParentGameID: gameID,
			PlayerIDs:    []string{player2ID, player3ID},
			PlayerNames:  expectedPlayerNames,
		}).
		Return(&gameRepo.CreateRollOffGameOutput{Game: rollOffGame}, nil)

	// Mock GetPlayer for each player in the roll-off again
	mockPlayerRepo.EXPECT().
		GetPlayer(gomock.Any(), &playerRepo.GetPlayerInput{PlayerID: player2ID}).
		Return(player2, nil)
	mockPlayerRepo.EXPECT().
		GetPlayer(gomock.Any(), &playerRepo.GetPlayerInput{PlayerID: player3ID}).
		Return(player3, nil)

	// Mock SavePlayer for each player in the roll-off
	player2.CurrentGameID = rollOffGameID
	player3.CurrentGameID = rollOffGameID

	mockPlayerRepo.EXPECT().
		SavePlayer(gomock.Any(), &playerRepo.SavePlayerInput{Player: player2}).
		Return(nil)
	mockPlayerRepo.EXPECT().
		SavePlayer(gomock.Any(), &playerRepo.SavePlayerInput{Player: player3}).
		Return(nil)

	// Create service
	service, err := New(&Config{
		GameRepo:          mockGameRepo,
		PlayerRepo:        mockPlayerRepo,
		DrinkLedgerRepo:   mockDrinkLedgerRepo,
		DiceRoller:        mockDiceRoller,
		Clock:             mockClock,
		UUIDGenerator:     mockUUID,
		DiceSides:         6,
		CriticalHitValue:  6,
		CriticalFailValue: 1,
		MaxConcurrentGames: 10,
	})
	assert.NoError(t, err)

	// Call EndGame
	result, err := service.EndGame(context.Background(), &EndGameInput{
		GameID: gameID,
	})

	// Assertions
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.False(t, result.Success)
	assert.True(t, result.NeedsRollOff)
	assert.Equal(t, RollOffTypeLowest, result.RollOffType)
	assert.Equal(t, rollOffGameID, result.RollOffGameID)
	assert.ElementsMatch(t, []string{player2ID, player3ID}, result.RollOffPlayerIDs)
}

func TestEndGame_TransitionToRollOff_HighestRoll(t *testing.T) {
	// Setup controller
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Create mocks
	mockGameRepo := gameMocks.NewMockRepository(ctrl)
	mockPlayerRepo := playerMocks.NewMockRepository(ctrl)
	mockDrinkLedgerRepo := ledgerMocks.NewMockRepository(ctrl)
	mockClock := mocks.NewMockClock(ctrl)
	mockUUID := uuidMocks.NewMockUUID(ctrl)
	mockDiceRoller := diceMocks.NewMockRoller(ctrl)

	// Fixed time for testing
	now := time.Date(2025, 4, 6, 12, 0, 0, 0, time.UTC)
	mockClock.EXPECT().Now().Return(now).AnyTimes()

	// Game ID and player IDs
	gameID := "game123"
	player1ID := "player1"
	player2ID := "player2"
	channelID := "channel123"
	creatorID := player1ID

	// Create roll time for all players
	rollTime := now.Add(-5 * time.Minute)

	// Create a game with 2 players, both tied for highest roll (critical hit)
	game := &models.Game{
		ID:        gameID,
		ChannelID: channelID,
		CreatorID: creatorID,
		Status:    models.GameStatusActive,
		Participants: []*models.Participant{
			{
				ID:         "participant1",
				GameID:     gameID,
				PlayerID:   player1ID,
				PlayerName: "Player 1",
				Status:     models.ParticipantStatusActive,
				RollValue:  6, // Critical hit (tied)
				RollTime:   &rollTime,
			},
			{
				ID:         "participant2",
				GameID:     gameID,
				PlayerID:   player2ID,
				PlayerName: "Player 2",
				Status:     models.ParticipantStatusActive,
				RollValue:  6, // Critical hit (tied)
				RollTime:   &rollTime,
			},
		},
		CreatedAt: now.Add(-30 * time.Minute),
		UpdatedAt: now.Add(-10 * time.Minute),
	}

	// Mock GetGame to return our test game
	mockGameRepo.EXPECT().
		GetGame(gomock.Any(), &gameRepo.GetGameInput{GameID: gameID}).
		Return(game, nil)

	// Mock GetDrinkRecordsForGame to return empty records
	mockDrinkLedgerRepo.EXPECT().
		GetDrinkRecordsForGame(gomock.Any(), &ledgerRepo.GetDrinkRecordsForGameInput{GameID: gameID}).
		Return(&ledgerRepo.GetDrinkRecordsForGameOutput{Records: []*models.DrinkLedger{}}, nil)

	// Mock GetPlayer for each player
	player1 := &models.Player{ID: player1ID, Name: "Player 1", CurrentGameID: gameID}
	player2 := &models.Player{ID: player2ID, Name: "Player 2", CurrentGameID: gameID}

	mockPlayerRepo.EXPECT().
		GetPlayer(gomock.Any(), &playerRepo.GetPlayerInput{PlayerID: player1ID}).
		Return(player1, nil)
	mockPlayerRepo.EXPECT().
		GetPlayer(gomock.Any(), &playerRepo.GetPlayerInput{PlayerID: player2ID}).
		Return(player2, nil)

	// Roll-off game ID
	rollOffGameID := "rolloff123"

	// Mock CreateRollOffGame
	rollOffGame := &models.Game{
		ID:           rollOffGameID,
		ChannelID:    channelID,
		CreatorID:    creatorID,
		Status:       models.GameStatusRollOff,
		ParentGameID: gameID,
		Participants: []*models.Participant{
			{
				ID:         "rolloff-participant1",
				GameID:     rollOffGameID,
				PlayerID:   player1ID,
				PlayerName: "Player 1",
				Status:     models.ParticipantStatusWaitingToRoll,
			},
			{
				ID:         "rolloff-participant2",
				GameID:     rollOffGameID,
				PlayerID:   player2ID,
				PlayerName: "Player 2",
				Status:     models.ParticipantStatusWaitingToRoll,
			},
		},
		CreatedAt: now,
		UpdatedAt: now,
	}

	// Expected player names map for roll-off game creation
	expectedPlayerNames := map[string]string{
		player1ID: "Player 1",
		player2ID: "Player 2",
	}

	mockGameRepo.EXPECT().
		CreateRollOffGame(gomock.Any(), &gameRepo.CreateRollOffGameInput{
			ChannelID:    channelID,
			CreatorID:    creatorID,
			ParentGameID: gameID,
			PlayerIDs:    []string{player1ID, player2ID},
			PlayerNames:  expectedPlayerNames,
		}).
		Return(&gameRepo.CreateRollOffGameOutput{Game: rollOffGame}, nil)

	// Mock GetPlayer for each player in the roll-off again
	mockPlayerRepo.EXPECT().
		GetPlayer(gomock.Any(), &playerRepo.GetPlayerInput{PlayerID: player1ID}).
		Return(player1, nil)
	mockPlayerRepo.EXPECT().
		GetPlayer(gomock.Any(), &playerRepo.GetPlayerInput{PlayerID: player2ID}).
		Return(player2, nil)

	// Mock SavePlayer for each player in the roll-off
	player1.CurrentGameID = rollOffGameID
	player2.CurrentGameID = rollOffGameID

	mockPlayerRepo.EXPECT().
		SavePlayer(gomock.Any(), &playerRepo.SavePlayerInput{Player: player1}).
		Return(nil)
	mockPlayerRepo.EXPECT().
		SavePlayer(gomock.Any(), &playerRepo.SavePlayerInput{Player: player2}).
		Return(nil)

	// Create service
	service, err := New(&Config{
		GameRepo:           mockGameRepo,
		PlayerRepo:         mockPlayerRepo,
		DrinkLedgerRepo:    mockDrinkLedgerRepo,
		DiceRoller:         mockDiceRoller,
		Clock:              mockClock,
		UUIDGenerator:      mockUUID,
		DiceSides:          6,
		CriticalHitValue:   6,
		CriticalFailValue:  1,
		MaxConcurrentGames: 10,
	})
	assert.NoError(t, err)

	// Call EndGame
	result, err := service.EndGame(context.Background(), &EndGameInput{
		GameID: gameID,
	})

	// Assertions
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.False(t, result.Success)
	assert.True(t, result.NeedsRollOff)
	assert.Equal(t, RollOffTypeHighest, result.RollOffType)
	assert.Equal(t, rollOffGameID, result.RollOffGameID)
	assert.ElementsMatch(t, []string{player1ID, player2ID}, result.RollOffPlayerIDs)
}

func TestRollDice_InRollOffGame(t *testing.T) {
	// Setup controller
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Create mocks
	mockGameRepo := gameMocks.NewMockRepository(ctrl)
	mockPlayerRepo := playerMocks.NewMockRepository(ctrl)
	mockDrinkLedgerRepo := ledgerMocks.NewMockRepository(ctrl)
	mockClock := mocks.NewMockClock(ctrl)
	mockUUID := uuidMocks.NewMockUUID(ctrl)
	mockDiceRoller := diceMocks.NewMockRoller(ctrl)

	// Set up the mock dice roller to return a fixed value
	mockDiceRoller.EXPECT().Roll(6).Return(4).AnyTimes() // Always roll a 4

	// Fixed time for testing
	now := time.Date(2025, 4, 6, 12, 0, 0, 0, time.UTC)
	mockClock.EXPECT().Now().Return(now).AnyTimes()

	// Game IDs and player IDs
	mainGameID := "game123"
	rollOffGameID := "rolloff123"
	player1ID := "player1"
	player2ID := "player2"
	channelID := "channel123"

	// Create a roll-off game with 2 players who haven't rolled yet
	rollOffGame := &models.Game{
		ID:           rollOffGameID,
		ChannelID:    channelID,
		CreatorID:    player1ID,
		Status:       models.GameStatusRollOff,
		ParentGameID: mainGameID,
		Participants: []*models.Participant{
			{
				ID:         "rolloff-participant1",
				GameID:     rollOffGameID,
				PlayerID:   player1ID,
				PlayerName: "Player 1",
				Status:     models.ParticipantStatusWaitingToRoll,
			},
			{
				ID:         "rolloff-participant2",
				GameID:     rollOffGameID,
				PlayerID:   player2ID,
				PlayerName: "Player 2",
				Status:     models.ParticipantStatusWaitingToRoll,
			},
		},
		CreatedAt: now.Add(-5 * time.Minute),
		UpdatedAt: now.Add(-5 * time.Minute),
	}

	// Mock FindActiveRollOffGame
	mockGameRepo.EXPECT().
		GetGamesByParent(gomock.Any(), &gameRepo.GetGamesByParentInput{
			ParentGameID: mainGameID,
		}).
		Return([]*models.Game{rollOffGame}, nil)

	// Mock GetGame to return our test roll-off game
	mockGameRepo.EXPECT().
		GetGame(gomock.Any(), &gameRepo.GetGameInput{GameID: rollOffGameID}).
		Return(rollOffGame, nil)

	// Mock SaveGame to update the roll-off game with the roll
	mockGameRepo.EXPECT().
		SaveGame(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, input *gameRepo.SaveGameInput) error {
			// Verify the game was updated correctly
			assert.Equal(t, rollOffGameID, input.Game.ID)
			
			// Verify the player's roll was updated
			participant := input.Game.GetParticipant(player1ID)
			assert.NotNil(t, participant)
			assert.Equal(t, 4, participant.RollValue) // Should be the value from our mock dice roller
			assert.NotNil(t, participant.RollTime)
			assert.Equal(t, now, *participant.RollTime)
			assert.Equal(t, models.ParticipantStatusActive, participant.Status)
			
			return nil
		})

	// Create service
	service, err := New(&Config{
		GameRepo:          mockGameRepo,
		PlayerRepo:        mockPlayerRepo,
		DrinkLedgerRepo:   mockDrinkLedgerRepo,
		DiceRoller:        mockDiceRoller,
		Clock:             mockClock,
		UUIDGenerator:     mockUUID,
		DiceSides:         6,
		CriticalHitValue:  6,
		CriticalFailValue: 1,
		MaxConcurrentGames: 10,
	})
	assert.NoError(t, err)

	// Call RollDice for the main game, but it should detect and use the roll-off game
	result, err := service.RollDice(context.Background(), &RollDiceInput{
		GameID:   mainGameID,
		PlayerID: player1ID,
	})

	// Assertions
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 4, result.Value)
	assert.Equal(t, "", result.RollOffGameID) // This is empty because we're not ending the game
	assert.False(t, result.IsCriticalHit)
	assert.False(t, result.IsCriticalFail)
}
