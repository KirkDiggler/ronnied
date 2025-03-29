# Models

This package contains the core data structures used throughout the application. These models are shared across all layers of the application.

## Core Models

### Game

The Game model represents a game session.

**Key Properties:**
- Game ID
- Channel ID
- Players
- Status (active, completed)
- Creation time
- Last activity time

### Player

The Player model represents a user participating in a game.

**Key Properties:**
- User ID
- Username
- Games participated in
- Drinks owed
- Drinks assigned to others
- Roll history

### RollResult

The RollResult model represents the outcome of a dice roll.

**Key Properties:**
- Roll value
- Is critical hit
- Is critical fail
- Player who rolled
- Timestamp

### Leaderboard

The Leaderboard model represents the current standings in a game.

**Key Properties:**
- Game ID
- Player rankings
- Drink tallies
- Most drinks assigned
- Most drinks received

## Implementation Notes

- Keep models simple and focused
- Use proper Go idioms (e.g., pointer receivers when appropriate)
- Consider using custom JSON marshaling/unmarshaling for complex types
- Document fields with comments
- Use appropriate types (e.g., time.Time for timestamps)
- Consider validation methods for models
