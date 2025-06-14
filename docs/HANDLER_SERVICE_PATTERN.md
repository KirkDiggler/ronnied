# Handler-Service Pattern for Ronnied Discord Bot

## Core Principles

1. **Separation of Concerns**
   - Handlers should only handle Discord interactions and rendering
   - Services should contain all game logic and business rules
   - Repositories should handle data persistence

2. **Clean Handler Pattern**
   - Handlers should be thin and focused on rendering
   - All logic should be in the service layer where it can be unit tested
   - Handlers should not parse or interpret data beyond what's needed for rendering

3. **Rich Service Responses**
   - Service methods should return structured responses with all rendering information
   - Avoid returning errors that need to be parsed by handlers
   - Use typed errors for specific error conditions

## Ideal Flow Pattern

### 1. Handler Receives Interaction
```go
func (b *Bot) handleSomeInteraction(s *discordgo.Session, i *discordgo.InteractionCreate, channelID, userID string) error {
    ctx := context.Background()
    
    // Call service with minimal processing
    output, err := b.someService.SomeAction(ctx, &service.SomeActionInput{
        ChannelID: channelID,
        UserID: userID,
        // Other relevant data from the interaction
    })
    
    // Handle errors with specific responses based on error type
    if err != nil {
        if errors.Is(err, service.ErrSpecificError) {
            return RespondWithEphemeralMessage(s, i, "Specific error message")
        }
        // Default error handling
        return RespondWithEphemeralMessage(s, i, fmt.Sprintf("%v", err))
    }
    
    // Use the service output to build the response
    // No business logic here, just rendering
    return renderGameState(s, i, output)
}
```

### 2. Service Method Structure
```go
func (s *service) SomeAction(ctx context.Context, input *SomeActionInput) (*SomeActionOutput, error) {
    // Validate input
    if input == nil {
        return nil, ErrNilInput
    }
    
    // Perform business logic
    // ...
    
    // Return a rich output with all rendering information
    return &SomeActionOutput{
        // All data needed for rendering
        Title: "Some Title",
        Description: "Some description",
        Components: []ComponentInfo{...},
        // Any state information needed
        GameState: someState,
        // Information about related entities
        RelatedEntities: []EntityInfo{...},
    }, nil
}
```

### 3. Dedicated Rendering Functions
```go
// renderGameState handles rendering a game state to Discord
func renderGameState(s *discordgo.Session, i *discordgo.InteractionCreate, output *service.SomeActionOutput) error {
    var components []discordgo.MessageComponent
    
    // Build components based on output.Components
    for _, componentInfo := range output.Components {
        // Create appropriate Discord component
        // (button, select menu, etc.)
        components = append(components, createComponent(componentInfo))
    }
    
    // Create embeds based on output information
    embeds := []*discordgo.MessageEmbed{
        {
            Title:       output.Title,
            Description: output.Description,
            Color:       getColorForState(output.GameState),
            // Other embed fields based on output
        },
    }
    
    // Respond with the built components and embeds
    return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
        Type: discordgo.InteractionResponseUpdateMessage,
        Data: &discordgo.InteractionResponseData{
            Embeds:     embeds,
            Components: components,
        },
    })
}
```

## Example: Roll Dice Flow

### Current Issues
- Handler parses error messages to extract roll-off game IDs
- Handler contains logic for creating UI components
- Handler needs to make additional service calls to get more information

### Improved Approach

1. **Enhanced RollDice Service Method**
```go
func (s *service) RollDice(ctx context.Context, input *RollDiceInput) (*RollDiceOutput, error) {
    // Game logic...
    
    return &RollDiceOutput{
        // Basic roll information
        Value: rollValue,
        IsCriticalHit: isCriticalHit,
        IsCriticalFail: isCriticalFail,
        
        // UI rendering information
        Title: fmt.Sprintf("You Rolled a %d", rollValue),
        Description: getDescriptionForRoll(rollValue, isCriticalHit, isCriticalFail),
        
        // For critical hits, include eligible players for drink assignment
        EligiblePlayers: getEligiblePlayersForDrink(game, userID),
        
        // Include information about the main game and any roll-offs
        MainGame: gameInfo,
        RollOffGames: rollOffGamesInfo,
        
        // Flag if player should be redirected to a roll-off
        ShouldRedirectToRollOff: shouldRedirect,
        RollOffGameID: rollOffGameID,
    }, nil
}
```

2. **Simplified Handler**
```go
func (b *Bot) handleRollDiceButton(s *discordgo.Session, i *discordgo.InteractionCreate, channelID, userID string) error {
    ctx := context.Background()
    
    output, err := b.gameService.RollDice(ctx, &game.RollDiceInput{
        ChannelID: channelID,
        UserID: userID,
    })
    
    if err != nil {
        // Handle specific error types with appropriate messages
        return handleRollDiceError(s, i, err)
    }
    
    // Check if player should be redirected to a roll-off
    if output.ShouldRedirectToRollOff {
        return RespondWithEphemeralMessage(s, i, 
            "You need to roll in the roll-off game. Check the game message for details.")
    }
    
    // Render the response using the output
    return renderRollDiceResponse(s, i, output)
}

// Dedicated rendering function for roll dice responses
func renderRollDiceResponse(s *discordgo.Session, i *discordgo.InteractionCreate, output *game.RollDiceOutput) error {
    var components []discordgo.MessageComponent
    
    // Build components based on the roll result
    if output.IsCriticalHit {
        // Create player selection dropdown
        playerSelect := createPlayerSelectMenu(output.EligiblePlayers)
        components = append(components, playerSelect)
    } else {
        // Create roll again button
        rollButton := createRollButton()
        components = append(components, rollButton)
    }
    
    // Create embeds
    embeds := []*discordgo.MessageEmbed{
        {
            Title:       output.Title,
            Description: output.Description,
            Color:       0x00ff00, // Green color
        },
    }
    
    // Respond with the built components and embeds
    return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
        Type: discordgo.InteractionResponseUpdateMessage,
        Data: &discordgo.InteractionResponseData{
            Embeds:     embeds,
            Components: createActionRows(components),
        },
    })
}
```

## Benefits of This Pattern

1. **Testability**: Service logic can be unit tested without Discord dependencies
2. **Maintainability**: Clear separation of concerns makes code easier to understand and modify
3. **Consistency**: Standardized pattern for all interactions
4. **Reduced Duplication**: UI rendering logic can be centralized
5. **Error Handling**: Typed errors provide better context and avoid string parsing

## Implementation Steps

1. Define rich output structures for service methods
2. Create rendering helpers in the handler package
3. Update service methods to return complete rendering information
4. Simplify handlers to focus on rendering
5. Use typed errors instead of string-based errors

By following this pattern, we'll create a more maintainable and testable codebase with clear separation between Discord interaction handling and game logic.
