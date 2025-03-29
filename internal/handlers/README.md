# Handler Layer

The handler layer is responsible for interfacing with Discord and processing user commands. This layer translates Discord events into calls to the service layer.

## Handlers

### Discord Handler

The Discord handler manages the connection to Discord and registers event listeners.

**Responsibilities:**
- Establish and maintain Discord connection
- Register command handlers
- Process Discord events
- Route commands to appropriate handlers

**Key Components:**
```go
// Bot represents the Discord bot instance
type Bot struct {
    Session *discordgo.Session
    // Dependencies
    GameService game.Service
    DiceService dice.Service
}

// New creates a new Discord bot
func New(token string, gameService game.Service, diceService dice.Service) (*Bot, error)

// Start initializes the Discord connection
func (b *Bot) Start() error

// Stop gracefully shuts down the Discord connection
func (b *Bot) Stop() error
```

### Command Handler

The command handler processes specific Discord commands and translates them into service calls.

**Responsibilities:**
- Parse command arguments
- Validate user input
- Call appropriate service methods
- Format and send responses

**Commands to Implement:**
- `/join` - Join the current game session
- `/roll` - Roll dice
- `/drinks` - View drink tally
- `/leaderboard` - Display leaderboard
- `/reset` - Reset game session

## Implementation Notes

- Use DiscordGo for Discord API integration
- Implement slash commands for better user experience
- Handle errors gracefully with user-friendly messages
- Log all commands and important events
- Use dependency injection for services
