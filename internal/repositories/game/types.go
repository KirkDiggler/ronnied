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
