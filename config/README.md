# Configuration

This package handles loading and managing configuration for the Ronnied Discord bot.

## Configuration Structure

The configuration includes:

- Discord bot token
- Command prefix
- Logging settings
- Development mode flag
- Optional database connection details

## Implementation Notes

- Use environment variables for sensitive information (e.g., Discord token)
- Support loading from `.env` file for local development
- Provide sensible defaults where appropriate
- Validate configuration on startup
- Consider using a structured configuration library like Viper
- Log configuration on startup (excluding sensitive information)
