# My thoughts

The game can have nested games that are the roll offs. keeping the rolling around a game will have a lot of resuse. we should have a helper function for a player in the game can have ShouldRoll() or FindActiveRollOffGame() that can return the roll off game that can also be used. if a player is in any roll off game they could be directed there. this is all ijnternal and the handler never needs to know. this should also be pretty easy to test. roll in standard, roll in opne roll off game, roll in a roll off of a roll off. the data can cust be saved as we go so the mocks would be vlear how to setup.

# Roll-Off Implementation Guide

This document provides a detailed blueprint of how roll-offs are implemented in the Ronnied Discord bot. It covers the flow of data and control through the system, from the model layer to the service layer to the handler layer, and explains the design considerations that went into the implementation.

## Table of Contents

1. [Overview](#overview)
2. [Model Layer](#model-layer)
3. [Service Layer](#service-layer)
4. [Handler Layer](#handler-layer)
5. [Data Flow](#data-flow)
6. [Edge Cases and Error Handling](#edge-cases-and-error-handling)
7. [Future Considerations](#future-considerations)

## Overview

Roll-offs are a critical part of the Ronnied drinking game, used to break ties when multiple players have the same roll value. The implementation integrates roll-offs directly into the main game as a state, rather than creating separate game entities. This simplifies state management and improves the user experience.

### Key Design Principles

1. **Integrated State Management**: Roll-offs are implemented as states within the main game, not as separate entities.
2. **Clear Participant Status**: Participants have explicit statuses to track their roll-off participation.
3. **Service Layer Logic**: All roll-off logic is encapsulated in the service layer, with the handler layer focusing on rendering.
4. **Automatic State Transitions**: The system automatically transitions between states when all players have rolled.

## Model Layer

### Game Status

The `GameStatus` type includes dedicated states for roll-offs:

```go
const (
    GameStatusWaiting       GameStatus = "waiting"
    GameStatusActive        GameStatus = "active"
    GameStatusRollOff       GameStatus = "roll_off"       // Legacy roll-off state
    GameStatusRollOffHighest GameStatus = "roll_off_highest" // New integrated highest roll-off state
    GameStatusRollOffLowest  GameStatus = "roll_off_lowest"  // New integrated lowest roll-off state
    GameStatusCompleted     GameStatus = "completed"
)
```

The `IsRollOff()` method on `GameStatus` checks if a status is any type of roll-off:

```go
func (s GameStatus) IsRollOff() bool {
    return s == GameStatusRollOff || s == GameStatusRollOffHighest || s == GameStatusRollOffLowest
}
```

### Roll-Off Type

The `RollOffType` enum defines the type of roll-off:

```go
type RollOffType string

const (
    RollOffTypeHighest RollOffType = "highest"
    RollOffTypeLowest  RollOffType = "lowest"
)
```

### Game Model

The `Game` model includes fields for roll-off properties:

```go
type Game struct {
    ID              string
    ChannelID       string
    CreatorID       string
    Status          GameStatus
    RollOffType     RollOffType      // Type of roll-off (highest or lowest)
    RollOffPlayerIDs []string        // IDs of players in the roll-off
    RollOffRound    int              // Current roll-off round
    Participants    []*Participant
    // Other fields...
}
```

### Participant Status

The `ParticipantStatus` type includes states for roll-off participation:

```go
const (
    ParticipantStatusActive         ParticipantStatus = "active"
    ParticipantStatusInRollOff      ParticipantStatus = "in_roll_off"      // Player is in a roll-off but hasn't rolled
    ParticipantStatusRolledInRollOff ParticipantStatus = "rolled_in_roll_off" // Player has rolled in the current roll-off
    // Other statuses...
)
```

## Service Layer

### Service Interface

The service interface defines methods for roll-off operations:

```go
type Service interface {
    // StartRollOff initiates a roll-off within a game
    StartRollOff(ctx context.Context, input *StartRollOffInput) (*StartRollOffOutput, error)
    
    // CompleteRollOff finalizes a roll-off and processes the results
    CompleteRollOff(ctx context.Context, input *CompleteRollOffInput) (*CompleteRollOffOutput, error)
    
    // RollDice performs a dice roll for a player
    RollDice(ctx context.Context, input *RollDiceInput) (*RollDiceOutput, error)
    
    // Other methods...
}
```

### Roll Dice Flow

The `RollDice` method routes to the appropriate processing method based on game state:

```go
func (s *service) RollDice(ctx context.Context, input *RollDiceInput) (*RollDiceOutput, error) {
    // Get the game
    game, err := s.gameRepo.GetGame(ctx, &gameRepo.GetGameInput{
        GameID: input.GameID,
    })
    if err != nil {
        return nil, fmt.Errorf("failed to get game: %w", err)
    }
    
    // Process the roll based on the game state
    if game.Status.IsRollOff() {
        // This is a roll-off
        return s.processRollOffInGame(ctx, input, game)
    } else if game.Status == models.GameStatusActive {
        // This is a normal active game
        return s.processMainGameRoll(ctx, input, game)
    } else {
        // Invalid game state for rolling
        return nil, ErrInvalidGameState
    }
}
```

### Process Roll-Off

The `processRollOffInGame` method handles dice rolling for participants in a roll-off state:

```go
func (s *service) processRollOffInGame(ctx context.Context, input *RollDiceInput, game *models.Game) (*RollDiceOutput, error) {
    // Check if game is in a valid roll-off state
    if !game.Status.IsRollOff() {
        return nil, fmt.Errorf("%w: game status is %s, expected roll-off", ErrInvalidGameState, game.Status)
    }
    
    // Check if player is part of the roll-off
    isInRollOff := false
    for _, playerID := range game.RollOffPlayerIDs {
        if playerID == input.PlayerID {
            isInRollOff = true
            break
        }
    }
    
    if !isInRollOff {
        return nil, ErrPlayerNotInRollOff
    }
    
    // Find the participant in the game
    participant := game.GetParticipant(input.PlayerID)
    if participant == nil {
        return nil, ErrPlayerNotInGame
    }
    
    // Check if the participant has already rolled in this roll-off round
    if participant.Status == models.ParticipantStatusRolledInRollOff {
        return nil, fmt.Errorf("player %s has already rolled in this roll-off round", participant.PlayerName)
    }
    
    // Roll the dice and update participant
    rollValue := s.diceRoller.Roll(s.diceSides)
    now := s.clock.Now()
    
    participant.RollValue = rollValue
    participant.RollTime = &now
    participant.Status = models.ParticipantStatusRolledInRollOff
    
    // Save the game
    err := s.gameRepo.SaveGame(ctx, &gameRepo.SaveGameInput{
        Game: game,
    })
    if err != nil {
        return nil, fmt.Errorf("failed to save game: %w", err)
    }
    
    // Check if all players in the roll-off have rolled
    allPlayersRolled := true
    for _, playerID := range game.RollOffPlayerIDs {
        participant := game.GetParticipant(playerID)
        if participant == nil || participant.Status != models.ParticipantStatusRolledInRollOff {
            allPlayersRolled = false
            break
        }
    }
    
    // If all players have rolled, complete the roll-off
    if allPlayersRolled {
        _, err = s.CompleteRollOff(ctx, &CompleteRollOffInput{
            GameID: game.ID,
        })
        if err != nil {
            return nil, fmt.Errorf("failed to complete roll-off: %w", err)
        }
        
        // Reload the game to get the updated state
        gameOutput, err := s.gameRepo.GetGame(ctx, &gameRepo.GetGameInput{
            GameID: game.ID,
        })
        if err != nil {
            return nil, fmt.Errorf("failed to reload game after completing roll-off: %w", err)
        }
        game = gameOutput
    }
    
    // Prepare result information
    result := fmt.Sprintf("You Rolled a %d in the Roll-Off!", rollValue)
    details := "Your roll has been recorded."
    
    // Add more detailed information based on roll-off type
    // ... (additional details logic)
    
    return &RollDiceOutput{
        PlayerID:         input.PlayerID,
        RollValue:        rollValue,
        Result:           result,
        Details:          details,
        EligiblePlayers:  nil, // No player options in roll-offs
        Game:             game,
        IsRollOffRoll:    true,
        AllPlayersRolled: allPlayersRolled,
    }, nil
}
```

### Start Roll-Off

The `StartRollOff` method initiates a roll-off within a game:

```go
func (s *service) StartRollOff(ctx context.Context, input *StartRollOffInput) (*StartRollOffOutput, error) {
    // Get the game
    game, err := s.gameRepo.GetGame(ctx, &gameRepo.GetGameInput{
        GameID: input.GameID,
    })
    if err != nil {
        return nil, fmt.Errorf("failed to get game: %w", err)
    }
    
    // Validate game state
    if !game.Status.IsActive() && !game.Status.IsRollOff() {
        return nil, fmt.Errorf("%w: game status is %s", ErrInvalidGameState, game.Status)
    }
    
    // Validate player IDs
    if len(input.PlayerIDs) < 2 {
        return nil, errors.New("at least two players are required for a roll-off")
    }
    
    // Validate that all players are in the game
    for _, playerID := range input.PlayerIDs {
        participant := game.GetParticipant(playerID)
        if participant == nil {
            return nil, fmt.Errorf("%w: player %s not found in game", ErrPlayerNotInGame, playerID)
        }
    }
    
    // Set up the roll-off
    now := s.clock.Now()
    
    // Update game status based on roll-off type
    if input.RollOffType == RollOffTypeHighest {
        game.Status = models.GameStatusRollOffHighest
    } else {
        game.Status = models.GameStatusRollOffLowest
    }
    
    // Update game roll-off properties
    game.RollOffType = input.RollOffType
    game.RollOffPlayerIDs = input.PlayerIDs
    game.RollOffRound++
    game.UpdatedAt = now
    
    // Update participant statuses
    for _, playerID := range input.PlayerIDs {
        participant := game.GetParticipant(playerID)
        if participant != nil {
            participant.Status = models.ParticipantStatusInRollOff
            participant.RollValue = 0
            participant.RollTime = nil
        }
    }
    
    // Save the game
    err = s.gameRepo.SaveGame(ctx, &gameRepo.SaveGameInput{
        Game: game,
    })
    if err != nil {
        return nil, fmt.Errorf("failed to save game: %w", err)
    }
    
    return &StartRollOffOutput{
        Game: game,
    }, nil
}
```

### Complete Roll-Off

The `CompleteRollOff` method finalizes a roll-off and processes the results:

```go
func (s *service) CompleteRollOff(ctx context.Context, input *CompleteRollOffInput) (*CompleteRollOffOutput, error) {
    // Get the game
    game, err := s.gameRepo.GetGame(ctx, &gameRepo.GetGameInput{
        GameID: input.GameID,
    })
    if err != nil {
        return nil, fmt.Errorf("failed to get game: %w", err)
    }
    
    // Validate game state
    if !game.Status.IsRollOff() {
        return nil, fmt.Errorf("%w: game status is %s, expected roll-off", ErrInvalidGameState, game.Status)
    }
    
    // Check if all players in the roll-off have rolled
    allPlayersRolled := true
    for _, playerID := range game.RollOffPlayerIDs {
        participant := game.GetParticipant(playerID)
        if participant == nil || participant.Status != models.ParticipantStatusRolledInRollOff {
            allPlayersRolled = false
            break
        }
    }
    
    if !allPlayersRolled {
        return nil, errors.New("not all players have rolled in the roll-off")
    }
    
    // Process the roll-off results
    now := s.clock.Now()
    
    // Determine the outcome based on roll-off type
    if game.RollOffType == models.RollOffTypeHighest {
        // Find players with the highest roll
        highestRoll := 0
        var highestRollers []string
        
        for _, playerID := range game.RollOffPlayerIDs {
            participant := game.GetParticipant(playerID)
            if participant == nil {
                continue
            }
            
            if participant.RollValue > highestRoll {
                highestRoll = participant.RollValue
                highestRollers = []string{participant.PlayerID}
            } else if participant.RollValue == highestRoll {
                highestRollers = append(highestRollers, participant.PlayerID)
            }
        }
        
        // If there's still a tie, start another roll-off round
        if len(highestRollers) > 1 {
            // Update for next roll-off round
            game.RollOffPlayerIDs = highestRollers
            game.RollOffRound++
            
            // Reset participant statuses for the next round
            for _, playerID := range highestRollers {
                participant := game.GetParticipant(playerID)
                if participant != nil {
                    participant.Status = models.ParticipantStatusInRollOff
                    participant.RollValue = 0
                    participant.RollTime = nil
                }
            }
        } else {
            // We have a winner, return to active state
            game.Status = models.GameStatusActive
            
            // Reset all participants to active
            for _, p := range game.Participants {
                p.Status = models.ParticipantStatusActive
            }
        }
    } else if game.RollOffType == models.RollOffTypeLowest {
        // Similar logic for lowest roll-off
        // ...
    }
    
    // Save the game
    game.UpdatedAt = now
    err = s.gameRepo.SaveGame(ctx, &gameRepo.SaveGameInput{
        Game: game,
    })
    if err != nil {
        return nil, fmt.Errorf("failed to save game: %w", err)
    }
    
    return &CompleteRollOffOutput{
        Game: game,
    }, nil
}
```

## Handler Layer

### Error Handling

The handler layer includes error handling for roll-off specific errors:

```go
// In handleRollDiceButton function
switch err {
case game.ErrPlayerNotInRollOff:
    // The player is not part of the current roll-off
    _, err = s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
        Content: "You are not part of the current roll-off. Please wait for your turn.",
        Flags:   discordgo.MessageFlagsEphemeral,
    })
    return err
}
```

### UI Rendering

The `renderGameMessage` function handles the visual representation of roll-offs:

```go
// In renderGameMessage function
case models.GameStatusRollOff, models.GameStatusRollOffHighest, models.GameStatusRollOffLowest:
    // Determine roll-off type description
    rollOffTypeDesc := "Roll-Off"
    rollOffEmoji := "⚔️"
    rollOffMessage := "Players in the roll-off need to roll again to break the tie."
    
    if game.Status == models.GameStatusRollOffHighest {
        rollOffTypeDesc = "Highest Roll-Off"
        rollOffEmoji = "🏆"
        rollOffMessage = "Players tied for the highest roll need to roll again to determine the winner!"
    } else if game.Status == models.GameStatusRollOffLowest {
        rollOffTypeDesc = "Lowest Roll-Off"
        rollOffEmoji = "💀"
        rollOffMessage = "Players tied for the lowest roll need to roll again to determine the loser!"
    }
    
    embed.Description = fmt.Sprintf("%s **ROLL-OFF IN PROGRESS!** %s\n*May the odds be ever in your favor!*", rollOffEmoji, rollOffMessage)
    
    // Add fields for roll-off status
    embed.Fields = []*discordgo.MessageEmbedField{
        {
            Name:   "📊 Status",
            Value:  fmt.Sprintf("%s %s", rollOffEmoji, rollOffTypeDesc),
            Inline: true,
        },
        {
            Name:   "👥 Players",
            Value:  fmt.Sprintf("%d", len(game.Participants)),
            Inline: true,
        },
    }
    
    // For integrated roll-offs, add a field showing which players are in the roll-off
    if game.Status == models.GameStatusRollOffHighest || game.Status == models.GameStatusRollOffLowest {
        // Build a list of players in the roll-off
        rollOffPlayers := ""  
        for _, playerID := range game.RollOffPlayerIDs {
            for _, p := range game.Participants {
                if p.PlayerID == playerID {
                    rollOffPlayers += fmt.Sprintf("• **%s**\n", p.PlayerName)
                    break
                }
            }
        }
        
        if rollOffPlayers != "" {
            embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
                Name:   "🎯 Roll-Off Participants",
                Value:  rollOffPlayers,
                Inline: false,
            })
        }
        
        // Add roll-off round information
        embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
            Name:   "🔄 Roll-Off Round",
            Value:  fmt.Sprintf("Round %d", game.RollOffRound),
            Inline: true,
        })
        
        // Add roll-off type information
        rollOffTypeField := &discordgo.MessageEmbedField{
            Name: "🎲 Roll-Off Type",
            Value: "Tie-breaker",
            Inline: true,
        }
        
        if game.RollOffType == models.RollOffTypeHighest {
            rollOffTypeField.Value = "🏆 Highest Roll Wins"
        } else if game.RollOffType == models.RollOffTypeLowest {
            rollOffTypeField.Value = "💀 Lowest Roll Loses"
        }
        
        embed.Fields = append(embed.Fields, rollOffTypeField)
    }
```

### Participant Status Rendering

The handler layer renders participant statuses differently based on roll-off participation:

```go
// Determine player status display based on roll-off state and participant status
if game.Status.IsRollOff() {
    if isInRollOff {
        // This player is part of the roll-off
        if p.Status == models.ParticipantStatusInRollOff {
            // Player needs to roll in the roll-off
            pendingRollers += fmt.Sprintf("• **%s**%s%s - 🎥 NEEDS TO ROLL IN ROLL-OFF! 🎲\n\n", p.PlayerName, rollInfo, rollComment)
        } else if p.Status == models.ParticipantStatusRolledInRollOff {
            // Player has already rolled in the roll-off
            pendingRollers += fmt.Sprintf("• %s%s%s - ✅ Rolled in roll-off\n\n", p.PlayerName, rollInfo, rollComment)
        } else {
            // Default case for roll-off participants
            pendingRollers += fmt.Sprintf("• **%s**%s%s - 🎥 In roll-off\n\n", p.PlayerName, rollInfo, rollComment)
        }
    } else {
        // This player is not part of the roll-off
        pendingRollers += fmt.Sprintf("• %s%s%s - ⏳ Waiting for roll-off to complete\n\n", p.PlayerName, rollInfo, rollComment)
    }
} else {
    // Normal game state
    if p.RollTime == nil {
        pendingRollers += fmt.Sprintf("• **%s**%s%s - 🎥 NEEDS TO ROLL! 🎲\n\n", p.PlayerName, rollInfo, rollComment)
    } else {
        pendingRollers += fmt.Sprintf("• %s%s%s - ✅ Already rolled\n\n", p.PlayerName, rollInfo, rollComment)
    }
}
```

## Identifying and Tracking Roll-Off Players

A critical aspect of the roll-off implementation is how we identify and track which players should participate in a roll-off. This is handled through the `RollOffPlayerIDs` field in the `Game` model.

### How Players Are Selected for Roll-Offs

When the game logic detects a tie that requires a roll-off (typically during `EndGame` or after processing rolls), it follows these steps:

```go
// In EndGame or after processing rolls
if needsRollOff {
    // Find players with tied rolls
    var tiedPlayerIDs []string
    for _, p := range game.Participants {
        if p.RollValue == tiedRollValue {
            tiedPlayerIDs = append(tiedPlayerIDs, p.PlayerID)
        }
    }
    
    // Start the roll-off
    _, err = s.StartRollOff(ctx, &StartRollOffInput{
        GameID: game.ID,
        PlayerIDs: tiedPlayerIDs,  // These are the players who will be in the roll-off
        RollOffType: rollOffType,
    })
}
```

### How Roll-Off Players Are Stored

The `StartRollOff` method stores the player IDs in the game model:

```go
func (s *service) StartRollOff(ctx context.Context, input *StartRollOffInput) (*StartRollOffOutput, error) {
    // ...validation code...
    
    // Update game roll-off properties
    game.RollOffType = input.RollOffType
    game.RollOffPlayerIDs = input.PlayerIDs  // This is where the IDs are stored
    game.RollOffRound++
    
    // Update participant statuses
    for _, playerID := range input.PlayerIDs {
        participant := game.GetParticipant(playerID)
        if participant != nil {
            participant.Status = models.ParticipantStatusInRollOff
            participant.RollValue = 0
            participant.RollTime = nil
        }
    }
    
    // Save the game
    // ...
}
```

### How Roll-Off Participation Is Checked

When a player attempts to roll in a roll-off, the `processRollOffInGame` method checks if they're part of the roll-off:

```go
// Check if player is part of the roll-off
isInRollOff := false
for _, playerID := range game.RollOffPlayerIDs {
    if playerID == input.PlayerID {
        isInRollOff = true
        break
    }
}

if !isInRollOff {
    return nil, ErrPlayerNotInRollOff
}
```

This approach ensures that only the players who were tied can participate in the roll-off, while other players must wait for the roll-off to complete.

## Data Flow

The roll-off functionality follows this data flow:

1. **Initiation**:
   - A tie is detected in the main game during `EndGame` or when processing rolls
   - The service layer calls `StartRollOff` to transition the game to a roll-off state
   - The game's status is updated to `GameStatusRollOffHighest` or `GameStatusRollOffLowest`
   - The tied players' IDs are stored in `game.RollOffPlayerIDs`
   - Participants involved in the tie have their status set to `ParticipantStatusInRollOff`

2. **Rolling**:
   - Players in the roll-off call the `RollDice` method
   - The `RollDice` method routes to `processRollOffInGame` based on game state
   - `processRollOffInGame` checks if the player is part of the roll-off
   - If valid, the player's roll is recorded and status updated to `ParticipantStatusRolledInRollOff`

3. **Completion**:
   - After each roll, the system checks if all players in the roll-off have rolled
   - If all have rolled, `CompleteRollOff` is called automatically
   - `CompleteRollOff` determines the outcome based on roll-off type
   - If there's still a tie, another roll-off round is initiated
   - If there's a winner/loser, the game returns to the active state

4. **UI Updates**:
   - After each state change, the handler layer updates the UI
   - The `renderGameMessage` function displays the current roll-off state
   - Participant statuses are rendered differently based on roll-off participation
   - Game components (buttons) are updated based on the current state

## Edge Cases and Error Handling

### Player Not in Roll-Off

If a player who is not part of the roll-off attempts to roll:

```go
if !isInRollOff {
    return nil, ErrPlayerNotInRollOff
}
```

The handler layer catches this error and displays an appropriate message:

```go
case game.ErrPlayerNotInRollOff:
    // The player is not part of the current roll-off
    _, err = s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
        Content: "You are not part of the current roll-off. Please wait for your turn.",
        Flags:   discordgo.MessageFlagsEphemeral,
    })
    return err
```

### Already Rolled in Roll-Off

If a player attempts to roll again in the same roll-off round:

```go
if participant.Status == models.ParticipantStatusRolledInRollOff {
    return nil, fmt.Errorf("player %s has already rolled in this roll-off round", participant.PlayerName)
}
```

### Continuous Ties

If players continue to tie in roll-off rounds:

```go
// If there's still a tie, start another roll-off round
if len(highestRollers) > 1 {
    // Update for next roll-off round
    game.RollOffPlayerIDs = highestRollers
    game.RollOffRound++
    
    // Reset participant statuses for the next round
    for _, playerID := range highestRollers {
        participant := game.GetParticipant(playerID)
        if participant != nil {
            participant.Status = models.ParticipantStatusInRollOff
            participant.RollValue = 0
            participant.RollTime = nil
        }
    }
}
```

## Future Considerations

1. **Performance Optimization**:
   - Consider caching roll-off state to reduce database calls
   - Batch update participant statuses to minimize database writes

2. **UI Enhancements**:
   - Add animations or special effects for roll-offs
   - Implement a countdown timer for roll-offs to encourage quick resolution

3. **Game Mechanics**:
   - Consider adding special rules for extended roll-offs (e.g., after 3 rounds)
   - Add options for different roll-off types beyond highest/lowest

4. **Analytics**:
   - Track roll-off statistics (frequency, duration, outcomes)
   - Use data to optimize game balance and player experience

5. **Error Recovery**:
   - Implement mechanisms to recover from interrupted roll-offs
   - Add admin commands to force-complete stuck roll-offs

---

This implementation guide provides a comprehensive blueprint of how roll-offs are integrated into the Ronnied Discord bot. By following the data flow and understanding the design considerations, developers can maintain and extend the roll-off functionality while preserving the core game mechanics.
