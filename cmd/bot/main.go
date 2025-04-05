package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/KirkDiggler/ronnied/internal/dice"
	"github.com/KirkDiggler/ronnied/internal/handlers/discord"
	"github.com/KirkDiggler/ronnied/internal/repositories/drink_ledger"
	"github.com/KirkDiggler/ronnied/internal/repositories/game"
	"github.com/KirkDiggler/ronnied/internal/repositories/player"
	gameService "github.com/KirkDiggler/ronnied/internal/services/game"
	"github.com/redis/go-redis/v9"
)

func main() {
	// Initialize Redis client
	redisClient := redis.NewClient(&redis.Options{
		Addr:     getEnv("REDIS_ADDR", "localhost:6379"),
		Password: getEnv("REDIS_PASSWORD", ""),
		DB:       0,
	})

	// Test Redis connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	if err := redisClient.Ping(ctx).Err(); err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}
	
	// Initialize repositories
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
	
	// Initialize game service
	gameSvc, err := gameService.New(&gameService.Config{
		GameRepo:       gameRepo,
		PlayerRepo:     playerRepo,
		DrinkLedgerRepo: drinkLedgerRepo,
		DiceRoller:     diceRoller,
		MaxPlayers:     10,
	})
	if err != nil {
		log.Fatalf("Failed to create game service: %v", err)
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
	
	// Initialize Discord bot
	bot, err := discord.New(&discord.Config{
		Token:         discordToken,
		ApplicationID: applicationID,
		GuildID:       guildID,
		GameService:   gameSvc,
	})
	if err != nil {
		log.Fatalf("Failed to create Discord bot: %v", err)
	}
	
	// Start the bot
	if err := bot.Start(); err != nil {
		log.Fatalf("Failed to start Discord bot: %v", err)
	}
	
	// Wait for interrupt signal to gracefully shutdown
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc
	
	// Shutdown the bot
	if err := bot.Stop(); err != nil {
		log.Printf("Error stopping bot: %v", err)
	}
	
	log.Println("Bot has been shut down")
}

// getEnv gets an environment variable or returns a default value
func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}
