package game

import "github.com/KirkDiggler/ronnied/internal/models"

type SaveGameInput struct {
	Game *models.Game
}

type GetGameInput struct {
	GameID string
}

type GetGameByChannelInput struct {
	ChannelID string
}

type DeleteGameInput struct {
	GameID string
}

type GetActiveGamesInput struct {
}

type GetActiveGamesOutput struct {
	Games []*models.Game
}

type GetGamesByParentInput struct {
	ParentGameID string
}

// CreateGameInput contains parameters for creating a new game
type CreateGameInput struct {
	ChannelID string
	CreatorID string
	Status    models.GameStatus
}

// CreateGameOutput contains the result of creating a new game
type CreateGameOutput struct {
	Game *models.Game
}

// CreateRollOffGameInput contains parameters for creating a new roll-off game
type CreateRollOffGameInput struct {
	ChannelID    string
	CreatorID    string
	ParentGameID string
	PlayerIDs    []string
	PlayerNames  map[string]string // Map of player ID to player name
}

// CreateRollOffGameOutput contains the result of creating a new roll-off game
type CreateRollOffGameOutput struct {
	Game *models.Game
}

// CreateParticipantInput contains parameters for creating a new participant
type CreateParticipantInput struct {
	GameID     string
	PlayerID   string
	PlayerName string
	Status     models.ParticipantStatus
}

// CreateParticipantOutput contains the result of creating a new participant
type CreateParticipantOutput struct {
	Participant *models.Participant
}
