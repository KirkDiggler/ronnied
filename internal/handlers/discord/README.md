# Discord Handler Implementation

This package implements the Discord bot interface for the Ronnied dice rolling game using the Command Pattern architecture.

## Architecture

The Discord handler uses a modular command pattern where:

1. The `Bot` struct manages the Discord connection and routes commands
2. Each command implements the `CommandHandler` interface
3. The `/ronnied` command supports subcommands for different game actions

## Components

### Bot

The main entry point that:
- Establishes connection to Discord
- Registers command handlers
- Routes incoming interactions to appropriate handlers

### Command Handler Interface

Defines the contract for all command handlers:
```go
type CommandHandler interface {
	GetName() string
	GetCommand() *discordgo.ApplicationCommand
	Handle(s *discordgo.Session, i *discordgo.InteractionCreate) error
}
```

### BaseCommand

A base struct that implements common functionality for all commands.

### RonniedCommand

Implements the `/ronnied` command with subcommands:
- `start`: Create a new game
- `join`: Join an existing game
- `roll`: Roll the dice
- `leaderboard`: Show the current drink tally

## Usage

To use the Discord handler:

1. Initialize the repositories and services
2. Create a new bot instance with the required services
3. Start the bot to establish the Discord connection
4. Register the commands with Discord

Example:
```go
// Initialize services
gameService := game.New(...)

// Create bot
bot, err := discord.New(&discord.Config{
    Token:       "YOUR_DISCORD_TOKEN",
    GameService: gameService,
})
if err != nil {
    log.Fatal(err)
}

// Start the bot
if err := bot.Start(); err != nil {
    log.Fatal(err)
}
```

## Command Flow

1. User enters a slash command in Discord
2. Discord sends an interaction to the bot
3. The bot routes the interaction to the appropriate command handler
4. The command handler processes the interaction and calls the relevant service methods
5. The command handler formats and sends a response back to Discord

## Error Handling

The handler includes helper functions for consistent error responses:
- `RespondWithMessage`: Sends a simple text message
- `RespondWithEmbed`: Sends a rich embed message
- `RespondWithError`: Sends an error message with red formatting

## Future Extensions

This implementation can be extended with:
1. Additional commands (e.g., `/stats`, `/help`)
2. More subcommands for the `/ronnied` command
3. Interactive components (buttons, select menus)
4. Ephemeral responses for user-specific information
