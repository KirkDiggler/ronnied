# Discord Handler Layer Design

This document outlines the design options and architectural approaches for implementing the Discord handler layer in the Ronnied bot.

## Architecture Options

### 1. Monolithic Handler

**Description:**
A single handler class that manages all Discord interactions and directly calls service methods.

**Pros:**
- Simple to implement initially
- Straightforward flow from Discord events to service calls
- Less boilerplate code

**Cons:**
- Becomes unwieldy as commands grow
- Difficult to test individual commands
- Tightly couples Discord-specific code with business logic

### 2. Command Pattern

**Description:**
Individual command handlers that implement a common interface, registered with a command router.

**Pros:**
- Modular and extensible
- Each command is isolated and testable
- Easy to add new commands without modifying existing code
- Follows single responsibility principle

**Cons:**
- More initial setup required
- Requires command registration mechanism
- Slightly more complex architecture

### 3. Event-Driven Architecture

**Description:**
Discord events trigger internal events that are processed by dedicated handlers.

**Pros:**
- Highly decoupled
- Supports complex workflows that span multiple interactions
- Can handle asynchronous processing
- Scales well for complex bots

**Cons:**
- More complex to implement
- Requires event bus or message broker
- May be overkill for simpler bots

## Recommended Approach: Command Pattern

For Ronnied, the **Command Pattern** offers the best balance of modularity, testability, and simplicity. Here's how it would work:

### Core Components

#### 1. Bot

The main entry point that:
- Establishes connection to Discord
- Registers command handlers
- Routes incoming interactions to appropriate handlers

```go
type Bot struct {
    session      *discordgo.Session
    commands     map[string]CommandHandler
    gameService  game.Service
    // Other dependencies
}
```

#### 2. Command Handler Interface

```go
type CommandHandler interface {
    // Handle processes a Discord interaction
    Handle(s *discordgo.Session, i *discordgo.InteractionCreate) error
    
    // Register registers the command with Discord
    Register(s *discordgo.Session, guildID string) error
    
    // GetName returns the command name
    GetName() string
}
```

#### 3. Base Command

A base struct that implements common functionality for all commands:

```go
type BaseCommand struct {
    Name        string
    Description string
    Options     []*discordgo.ApplicationCommandOption
}
```

#### 4. Individual Command Implementations

```go
type RollCommand struct {
    BaseCommand
    gameService game.Service
    diceService dice.Service
}

func (c *RollCommand) Handle(s *discordgo.Session, i *discordgo.InteractionCreate) error {
    // Extract parameters
    // Call service methods
    // Format and send response
}
```

### Command Registration Flow

1. Bot initializes and creates command handlers
2. Each handler registers its command with Discord
3. Bot stores handlers in a map keyed by command name
4. When an interaction is received, the bot looks up the appropriate handler and calls its Handle method

### Response Handling

For consistent user experience, we should implement a response helper:

```go
type ResponseBuilder struct {
    // Fields for building a response
}

func NewResponse() *ResponseBuilder {
    // Initialize a new response
}

func (r *ResponseBuilder) WithEmbed(title, description string) *ResponseBuilder {
    // Add an embed to the response
}

func (r *ResponseBuilder) WithError(message string) *ResponseBuilder {
    // Format an error response
}

func (r *ResponseBuilder) Send(s *discordgo.Session, i *discordgo.Interaction) error {
    // Send the response to Discord
}
```

## Implementation Plan

1. Create the command handler interface and base command struct
2. Implement the bot struct with command registration
3. Create individual command handlers:
   - JoinCommand
   - RollCommand
   - LeaderboardCommand
   - DrinksCommand
   - ResetCommand
4. Implement response builder for consistent UI
5. Add error handling and logging

## Testing Strategy

1. **Unit Tests**: Test individual command handlers with mocked Discord session and interactions
2. **Integration Tests**: Test command flow from handler to service
3. **Mock Discord API**: Use a mock Discord API for testing without real Discord connections

## Discord API Considerations

### Slash Commands vs. Message Commands

Slash commands (`/roll`, `/join`, etc.) are recommended over message commands (`!roll`, `!join`) because they:
- Provide better user experience with parameter hints
- Are more discoverable
- Have built-in parameter validation
- Are the preferred approach by Discord

### Interaction Types

Handle different interaction types appropriately:
- `ApplicationCommand`: Initial command invocation
- `MessageComponent`: Button clicks, select menus, etc.
- `ApplicationCommandAutocomplete`: Autocomplete suggestions

### Permissions

Implement permission checks in command handlers:
- Check if user has required permissions
- Verify bot has necessary permissions
- Handle permission errors gracefully

## User Experience Considerations

1. **Responsive Feedback**: Acknowledge commands immediately, even if processing takes time
2. **Error Messages**: Provide clear, user-friendly error messages
3. **Help Text**: Include detailed help for each command
4. **Visual Elements**: Use embeds, buttons, and select menus for rich interactions
5. **Stateful Interactions**: Maintain context across multiple interactions when needed

## Future Extensions

1. **Component Interactions**: Support for buttons, select menus, modals
2. **Context Menus**: Right-click menu commands
3. **Autocomplete**: Parameter suggestions
4. **Localization**: Support for multiple languages
5. **Ephemeral Responses**: Private responses visible only to the command user
