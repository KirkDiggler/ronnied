# Redis Repository Design

This document outlines the design decisions for implementing the repository layer using Redis as the storage backend for the Ronnied Discord bot.

## Overview

Redis is chosen for its:
- Speed and in-memory performance
- Rich data structure support
- Pub/Sub capabilities for potential future features
- Simplicity of setup and operation
- TTL support for automatic cleanup

## Data Structure Choices

### Game Repository

#### Game Objects
**Structure:** Redis String (JSON)
**Key Pattern:** `game:{gameID}`
**Rationale:** 
- Games are complex objects with nested participants
- JSON provides flexibility for schema evolution
- Single atomic read/write operations
- Simplifies serialization/deserialization

#### Channel-to-Game Mapping
**Structure:** Redis String
**Key Pattern:** `channel:{channelID}`
**Value:** GameID
**Rationale:**
- Simple 1:1 mapping
- Fast lookups by channel ID
- Atomic updates

#### Active Games Index
**Structure:** Redis Set
**Key:** `active_games`
**Members:** GameIDs
**Rationale:**
- Efficient membership checks
- Fast retrieval of all active games
- Set operations for filtering

### Player Repository

#### Player Objects
**Structure:** Redis String (JSON)
**Key Pattern:** `player:{playerID}`
**Rationale:**
- Similar to games, players have complex attributes
- JSON allows for flexible schema
- Single atomic operations

#### Player-Game Participation
**Structure:** Redis Set
**Key Pattern:** `player_games:{playerID}`
**Members:** GameIDs
**Rationale:**
- Quick lookup of all games a player is in
- Set operations for filtering

### Drink Ledger Repository

#### Drink Records
**Structure:** Redis Sorted Set
**Key Pattern:** `drinks:{gameID}`
**Score:** Timestamp
**Member:** JSON string of drink record
**Rationale:**
- Time-ordered drink records
- Range queries for time-based filtering
- Automatic sorting by when drinks were assigned

#### Player Drink Counts
**Structure:** Redis Hash
**Key Pattern:** `player_drinks:{playerID}`
**Fields:** 
  - `assigned`: Number of drinks assigned to others
  - `received`: Number of drinks received
**Rationale:**
- Atomic increment/decrement operations
- Efficient storage for numeric counters
- Fast retrieval of player statistics

## Performance Considerations

### Batching
- Use Redis pipelines for multiple operations
- Batch retrieval of multiple games/players
- Minimize network round-trips

### Caching
- Redis already serves as a cache
- Consider client-side caching for frequently accessed data

### Data Expiration
- Set TTL on completed games (e.g., 24 hours)
- Automatic cleanup of old data
- Configurable retention policy

## Error Handling

### Connection Failures
- Implement retry mechanism
- Log connection errors
- Graceful degradation

### Data Validation
- Validate data before storage
- Handle deserialization errors
- Return specific error types

## Testing Strategy

- Use miniredis for unit tests
- Avoid external dependencies in tests
- Test edge cases and error conditions
- Verify data persistence and retrieval

## Future Considerations

### Scaling
- Redis Cluster for horizontal scaling
- Sharding by game ID or channel ID
- Read replicas for high-read scenarios

### Backup and Recovery
- Redis persistence options (RDB/AOF)
- Backup strategy
- Disaster recovery plan

### Monitoring
- Key metrics to monitor
- Performance indicators
- Alerting thresholds
