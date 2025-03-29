# Repository Layer

The repository layer is responsible for data persistence and retrieval. This layer abstracts the storage mechanism from the rest of the application.

## Repositories

### Player Repository

The player repository manages player data, including drink tallies and game participation.

**Responsibilities:**
- Store player information
- Track drinks assigned to and by players
- Persist game participation data
- Retrieve player statistics

**Key Interfaces:**
```go
type PlayerRepository interface {
    SavePlayer(ctx context.Context, player *models.Player) error
    GetPlayer(ctx context.Context, userID string) (*models.Player, error)
    UpdateDrinkCount(ctx context.Context, userID string, delta int) error
    GetPlayersInGame(ctx context.Context, gameID string) ([]*models.Player, error)
    RemovePlayerFromGame(ctx context.Context, gameID, userID string) error
}
```

### Game Repository

The game repository manages game session data.

**Responsibilities:**
- Create and store game sessions
- Track active games
- Manage game state
- Archive completed games

**Key Interfaces:**
```go
type GameRepository interface {
    CreateGame(ctx context.Context, game *models.Game) error
    GetGame(ctx context.Context, gameID string) (*models.Game, error)
    UpdateGame(ctx context.Context, game *models.Game) error
    DeleteGame(ctx context.Context, gameID string) error
    GetActiveGames(ctx context.Context) ([]*models.Game, error)
}
```

## Implementation Notes

- Start with in-memory implementations for rapid development
- Design with future database implementation in mind
- Use interfaces to allow for different storage backends
- Consider using a simple key-value store initially
- Implement proper error handling and logging
- Use contexts for cancellation and timeouts
