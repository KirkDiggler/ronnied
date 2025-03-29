# Service Layer

The service layer contains the core game logic for Ronnied. Each service is encapsulated in its own package to maintain separation of concerns and allow for easier testing and maintenance.

## Services

### Game Service (`game`)

The game service manages game sessions, player participation, and overall game flow.

**Responsibilities:**
- Create and manage game sessions
- Track players in each session
- Coordinate dice rolling rounds
- Maintain drink assignments
- Generate leaderboards

**Key Interfaces:**
```go
type GameService interface {
    CreateGame(channelID string) (*models.Game, error)
    JoinGame(gameID, userID, username string) error
    LeaveGame(gameID, userID string) error
    RollDice(gameID, userID string) (*models.RollResult, error)
    AssignDrink(gameID, fromUserID, toUserID string) error
    GetLeaderboard(gameID string) (*models.Leaderboard, error)
    ResetGame(gameID string) error
}
```

### Dice Service (`dice`)

The dice service handles the mechanics of dice rolling and determining outcomes.

**Responsibilities:**
- Generate random dice rolls
- Determine special outcomes (critical hit/fail)
- Track roll statistics

**Key Interfaces:**
```go
type DiceService interface {
    Roll() int
    IsCriticalHit(roll int) bool
    IsCriticalFail(roll int) bool
}
```

## Implementation Notes

- Each service should be in its own package
- Services should depend on repository interfaces, not implementations
- Use dependency injection for repositories
- Services should be stateless where possible
- Use context for operations that might need cancellation
- Log important events and errors
