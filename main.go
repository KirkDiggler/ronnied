package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/KirkDiggler/ronnied/internal/common/clock"
	"github.com/KirkDiggler/ronnied/internal/common/uuid"
	"github.com/KirkDiggler/ronnied/internal/dice"
	"github.com/KirkDiggler/ronnied/internal/handlers/discord"
	"github.com/KirkDiggler/ronnied/internal/repositories/drink_ledger"
	"github.com/KirkDiggler/ronnied/internal/repositories/game"
	"github.com/KirkDiggler/ronnied/internal/repositories/player"
	gameService "github.com/KirkDiggler/ronnied/internal/services/game"
	messagingService "github.com/KirkDiggler/ronnied/internal/services/messaging"
	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"
)

func main() {
	fmt.Println("Starting Ronnied - Discord Dice Drinking Game Bot")
	
	// Load .env file
	if err := godotenv.Load(); err != nil {
		log.Println("Warning: No .env file found. Using environment variables.")
	}
	
	// Get Discord token from environment
	discordToken := getEnv("DISCORD_TOKEN", "")
	if discordToken == "" {
		log.Fatal("DISCORD_TOKEN environment variable is required")
	}
	
	// Get application ID for the bot
	applicationID := getEnv("APPLICATION_ID", "")
	
	// Get optional guild ID for development
	guildID := getEnv("GUILD_ID", "")
	
	// Initialize Redis client
	redisAddr := getEnv("REDIS_ADDR", "localhost:6379")
	redisPassword := getEnv("REDIS_PASSWORD", "")
	
	fmt.Printf("Connecting to Redis at %s...\n", redisAddr)
	redisClient := redis.NewClient(&redis.Options{
		Addr:     redisAddr,
		Password: redisPassword,
		DB:       0,
	})

	// Test Redis connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	if err := redisClient.Ping(ctx).Err(); err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}
	fmt.Println("Connected to Redis successfully")
	
	// Initialize common dependencies
	uuidGen := uuid.New()
	clockSvc := clock.New()
	
	// Initialize repositories
	fmt.Println("Initializing repositories...")
	gameRepo, err := game.NewRedis(&game.Config{
		RedisClient: redisClient,
	})
	if err != nil {
		log.Fatalf("Failed to create game repository: %v", err)
	}
	
	playerRepo, err := player.NewRedis(&player.Config{
		RedisClient: redisClient,
	})
	if err != nil {
		log.Fatalf("Failed to create player repository: %v", err)
	}
	
	drinkLedgerRepo, err := drink_ledger.NewRedis(&drink_ledger.Config{
		RedisClient: redisClient,
	})
	if err != nil {
		log.Fatalf("Failed to create drink ledger repository: %v", err)
	}
	
	// Initialize dice roller
	diceRoller := dice.New(&dice.Config{})
	
	// Get game configuration from environment
	maxPlayers := getEnvAsInt("MAX_PLAYERS", 10)
	diceSides := getEnvAsInt("DICE_SIDES", 6)
	criticalHitValue := getEnvAsInt("CRITICAL_HIT_VALUE", 6)
	criticalFailValue := getEnvAsInt("CRITICAL_FAIL_VALUE", 1)
	
	// Initialize game service
	fmt.Println("Initializing game service...")
	gameSvc, err := gameService.New(&gameService.Config{
		GameRepo:       gameRepo,
		PlayerRepo:     playerRepo,
		DrinkLedgerRepo: drinkLedgerRepo,
		DiceRoller:     diceRoller,
		UUIDGenerator:  uuidGen,
		Clock:          clockSvc,
		MaxPlayers:     maxPlayers,
		DiceSides:      diceSides,
		CriticalHitValue: criticalHitValue,
		CriticalFailValue: criticalFailValue,
	})
	if err != nil {
		log.Fatalf("Failed to create game service: %v", err)
	}
	
	// Initialize messaging service
	fmt.Println("Initializing messaging service...")
	msgSvc, err := messagingService.NewService(&messagingService.ServiceConfig{
		// We'll add repository configuration here later when we implement message storage
	})
	if err != nil {
		log.Fatalf("Failed to create messaging service: %v", err)
	}
	
	// Initialize Discord bot
	fmt.Println("Initializing Discord bot...")
	bot, err := discord.New(&discord.Config{
		Token:         discordToken,
		ApplicationID: applicationID,
		GuildID:       guildID,
		GameService:   gameSvc,
		MessagingService: msgSvc,
	})
	if err != nil {
		log.Fatalf("Failed to create Discord bot: %v", err)
	}
	
	// Start the bot
	fmt.Println("Starting Discord bot...")
	if err := bot.Start(); err != nil {
		log.Fatalf("Failed to start Discord bot: %v", err)
	}
	
	// Keep the bot running until interrupted
	fmt.Println("Bot is now running. Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc
	
	// Cleanup before exit
	fmt.Println("Shutting down...")
	
	// Stop the Discord bot
	if err := bot.Stop(); err != nil {
		log.Printf("Error stopping bot: %v", err)
	}
	
	// Close Redis connection
	if err := redisClient.Close(); err != nil {
		log.Printf("Error closing Redis connection: %v", err)
	}
	
	fmt.Println("Shutdown complete. Goodbye!")
}

// getEnv gets an environment variable or returns a default value
func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

// getEnvAsInt gets an environment variable as an integer or returns a default value
func getEnvAsInt(key string, defaultValue int) int {
	valueStr := getEnv(key, "")
	if valueStr == "" {
		return defaultValue
	}
	
	value, err := strconv.Atoi(valueStr)
	if err != nil {
		log.Printf("Warning: Could not parse %s as integer, using default: %d", key, defaultValue)
		return defaultValue
	}
	
	return value
}
