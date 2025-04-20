package discord

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sort"
	"strings"
	"time"

	"github.com/KirkDiggler/ronnied/internal/models"
	"github.com/KirkDiggler/ronnied/internal/services/game"
	"github.com/bwmarrin/discordgo"
)

// RonniedCommand handles the /ronnied command
type RonniedCommand struct {
	BaseCommand
	gameService game.Service
}

// NewRonniedCommand creates a new ronnied command handler
func NewRonniedCommand(gameService game.Service) *RonniedCommand {
	return &RonniedCommand{
		BaseCommand: BaseCommand{
			Name:        "ronnied",
			Description: "Dice rolling drinking game commands",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "start",
					Description: "Create a new game",
				},
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "leaderboard",
					Description: "Show the current session leaderboard",
				},
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "newsession",
					Description: "Start a new drinking session",
				},
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "abandon",
					Description: "Abandon the current game",
				},
			},
		},
		gameService: gameService,
	}
}

// Handle processes a Discord interaction for the ronnied command
func (c *RonniedCommand) Handle(s *discordgo.Session, i *discordgo.InteractionCreate) error {
	if i.Type != discordgo.InteractionApplicationCommand {
		return nil
	}

	data := i.ApplicationCommandData()
	if data.Name != c.Name {
		return nil
	}

	// Get the channel ID and user information
	channelID := i.ChannelID
	userID := i.Member.User.ID
	username := i.Member.User.Username
	if i.Member.Nick != "" {
		username = i.Member.Nick
	}

	// Handle the appropriate subcommand
	var err error
	switch data.Options[0].Name {
	case "start":
		err = c.handleStart(s, i, channelID, userID, username)
	case "leaderboard":
		err = c.handleSessionboard(s, i, channelID)
	case "newsession":
		err = c.handleNewSession(s, i, channelID)
	case "abandon":
		err = c.handleAbandon(s, i, channelID, userID)
	default:
		err = errors.New("unknown subcommand")
	}

	return err
}

// handleStart handles the start subcommand
func (c *RonniedCommand) handleStart(s *discordgo.Session, i *discordgo.InteractionCreate, channelID, userID, username string) error {
	ctx := context.Background()

	// Check if there's already a game in this channel
	existingGame, err := c.gameService.GetGameByChannel(ctx, &game.GetGameByChannelInput{
		ChannelID: channelID,
	})

	// Handle errors from the service
	if err != nil {
		// If it's a "game not found" error, we can proceed to create a new game
		if errors.Is(err, game.ErrGameNotFound) {
			// Continue with game creation
		} else {
			// It's a real error, log and return
			log.Printf("Error checking for existing game: %v", err)
			return RespondWithError(s, i, fmt.Sprintf("Error checking for existing game: %v", err))
		}
	} else if existingGame != nil && existingGame.Game != nil {
		// There's an existing game, check if it's active or waiting
		if existingGame.Game.Status == models.GameStatusActive || existingGame.Game.Status == models.GameStatusWaiting || existingGame.Game.Status == models.GameStatusRollOff {
			return RespondWithError(s, i, "There's already a game in progress in this channel. Use `/ronnied abandon` to clear it if needed.")
		}
		// If the game exists but is completed, we can proceed to create a new game
	}

	// Create a new game
	createOutput, err := c.gameService.CreateGame(ctx, &game.CreateGameInput{
		ChannelID:   channelID,
		CreatorID:   userID,
		CreatorName: username,
	})
	if err != nil {
		log.Printf("Error creating game: %v", err)
		return RespondWithError(s, i, fmt.Sprintf("Failed to create game: %v", err))
	}

	// Join the creator to the game
	_, err = c.gameService.JoinGame(ctx, &game.JoinGameInput{
		GameID:     createOutput.GameID,
		PlayerID:   userID,
		PlayerName: username,
	})
	if err != nil {
		log.Printf("Error joining game: %v", err)
		return RespondWithError(s, i, fmt.Sprintf("Failed to join game: %v", err))
	}

	// Create buttons for joining and starting the game
	joinButton := discordgo.Button{
		Label:    "Join Game",
		Style:    discordgo.SuccessButton,
		CustomID: ButtonJoinGame,
		Emoji: discordgo.ComponentEmoji{
			Name: "🎲",
		},
	}

	startButton := discordgo.Button{
		Label:    "Begin Game",
		Style:    discordgo.PrimaryButton,
		CustomID: ButtonBeginGame,
		Emoji: discordgo.ComponentEmoji{
			Name: "🎮",
		},
	}

	// Create fields for the message
	fields := []*discordgo.MessageEmbedField{
		{
			Name:   "Created By",
			Value:  username,
			Inline: true,
		},
		{
			Name:   "Status",
			Value:  "Waiting for players",
			Inline: true,
		},
	}

	// Send the response message
	err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{
				{
					Title:       "New Game Started!",
					Description: "Click the Join button to join the game. Once everyone has joined, the creator can click Begin to start the game.",
					Color:       0x00ff00, // Green color
					Fields:      fields,
				},
			},
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{joinButton, startButton},
				},
			},
		},
	})
	if err != nil {
		log.Printf("Error sending response message: %v", err)
		return err
	}

	// Get the message ID from the interaction response
	// We need to wait a moment for Discord to process the interaction response
	time.Sleep(500 * time.Millisecond)

	// Get the channel messages to find our message
	messages, err := s.ChannelMessages(channelID, 5, "", "", "")
	if err != nil {
		log.Printf("Error getting channel messages: %v", err)
		// This is not critical, so we'll continue
	} else {
		// Find our message (should be the most recent one)
		for _, msg := range messages {
			if msg.Author.ID == s.State.User.ID {
				// Update the game with the message ID
				_, err = c.gameService.UpdateGameMessage(ctx, &game.UpdateGameMessageInput{
					GameID:    createOutput.GameID,
					MessageID: msg.ID,
				})
				if err != nil {
					log.Printf("Error updating game message ID: %v", err)
					// Not critical, continue
				}
				break
			}
		}
	}

	return nil
}

// handleSessionboard handles the sessionboard subcommand
func (c *RonniedCommand) handleSessionboard(s *discordgo.Session, i *discordgo.InteractionCreate, channelID string) error {
	ctx := context.Background()

	// Get the session leaderboard
	sessionboard, err := c.gameService.GetSessionLeaderboard(ctx, &game.GetSessionLeaderboardInput{
		ChannelID: channelID,
	})
	if err != nil {
		log.Printf("Error getting session leaderboard: %v", err)
		return RespondWithError(s, i, fmt.Sprintf("Failed to get session leaderboard: %v", err))
	}

	// Build the session leaderboard description
	var description strings.Builder
	
	// Session info header
	if sessionboard.Session != nil {
		sessionAge := time.Since(sessionboard.Session.CreatedAt).Round(time.Minute)
		description.WriteString(fmt.Sprintf("🍻 **Session Age:** %s\n\n", sessionAge))
	}
	
	if len(sessionboard.Entries) == 0 {
		description.WriteString("🏜️ **The Sahara is less dry than this session!** No drinks have been assigned yet.")
	} else {
		// Find the player with the most drinks for ranking
		maxDrinks := 0
		for _, entry := range sessionboard.Entries {
			if entry.DrinkCount > maxDrinks {
				maxDrinks = entry.DrinkCount
			}
		}
		
		// Sort entries by drink count (descending)
		sort.Slice(sessionboard.Entries, func(i, j int) bool {
			return sessionboard.Entries[i].DrinkCount > sessionboard.Entries[j].DrinkCount
		})
		
		// Add a header
		description.WriteString("🏆 **DRINK LEADERBOARD** 🏆\n\n")
		
		// Add each player with rank emoji and progress bar
		rankEmojis := []string{"🥇", "🥈", "🥉", "4️⃣", "5️⃣", "6️⃣", "7️⃣", "8️⃣", "9️⃣", "🔟"}
		
		for i, entry := range sessionboard.Entries {
			// Rank emoji
			rankEmoji := "🍺"
			if i < len(rankEmojis) {
				rankEmoji = rankEmojis[i]
			}
			
			// Progress bar (10 segments)
			progressBarLength := 10
			filledSegments := 0
			if maxDrinks > 0 {
				filledSegments = (entry.DrinkCount * progressBarLength) / maxDrinks
				if filledSegments == 0 && entry.DrinkCount > 0 {
					filledSegments = 1 // Show at least one segment if they have any drinks
				}
			}
			
			progressBar := ""
			for j := 0; j < progressBarLength; j++ {
				if j < filledSegments {
					progressBar += "🟥" // Filled segment
				} else {
					progressBar += "⬜" // Empty segment
				}
			}
			
			// Payment status
			paymentStatus := ""
			if entry.PaidCount > 0 {
				paymentRatio := float64(entry.PaidCount) / float64(entry.DrinkCount)
				if paymentRatio >= 1.0 {
					paymentStatus = " ✅ **PAID IN FULL!**"
				} else if paymentRatio >= 0.5 {
					paymentStatus = fmt.Sprintf(" ⏳ (%d/%d paid)", entry.PaidCount, entry.DrinkCount)
				} else {
					paymentStatus = fmt.Sprintf(" 💸 (%d/%d paid)", entry.PaidCount, entry.DrinkCount)
				}
			}
			
			// Add the entry with all components
			description.WriteString(fmt.Sprintf("%s **%s**: %d drinks%s\n%s\n\n", 
				rankEmoji, 
				entry.PlayerName, 
				entry.DrinkCount,
				paymentStatus,
				progressBar))
		}
		
		// Add a fun message at the end based on total drinks
		totalDrinks := 0
		for _, entry := range sessionboard.Entries {
			totalDrinks += entry.DrinkCount
		}
		
		description.WriteString("\n")
		if totalDrinks > 20 {
			description.WriteString("🔥 **LEGENDARY SESSION!** Your livers will be remembered for generations to come!")
		} else if totalDrinks > 10 {
			description.WriteString("🥴 **IMPRESSIVE!** Tomorrow's hangover is going to be epic!")
		} else if totalDrinks > 5 {
			description.WriteString("😎 **GOOD START!** Keep the drinks flowing!")
		} else {
			description.WriteString("🐣 **JUST WARMING UP!** The night is young!")
		}
	}

	// Create fields for additional info
	fields := []*discordgo.MessageEmbedField{
		{
			Name:   "Commands",
			Value:  "`/ronnied newsession` - Start a new session",
			Inline: false,
		},
	}

	// Respond with the session leaderboard
	return RespondWithEmbed(s, i, "🍻 Session Leaderboard 🍻", description.String(), fields)
}

// handleNewSession handles the newsession subcommand
func (c *RonniedCommand) handleNewSession(s *discordgo.Session, i *discordgo.InteractionCreate, channelID string) error {
	ctx := context.Background()

	// Start a new session
	_, err := c.gameService.StartNewSession(ctx, &game.StartNewSessionInput{
		ChannelID: channelID,
	})
	if err != nil {
		log.Printf("Error starting new session: %v", err)
		return RespondWithError(s, i, fmt.Sprintf("Failed to start new session: %v", err))
	}

	// Respond with success message
	return RespondWithMessage(s, i, "New session started successfully.")
}

// handleAbandon handles the abandon subcommand
func (c *RonniedCommand) handleAbandon(s *discordgo.Session, i *discordgo.InteractionCreate, channelID, userID string) error {
	ctx := context.Background()

	// Get the game in this channel
	existingGame, err := c.gameService.GetGameByChannel(ctx, &game.GetGameByChannelInput{
		ChannelID: channelID,
	})

	// Handle errors or missing game
	if err != nil {
		if err == game.ErrGameNotFound {
			return RespondWithError(s, i, "No game found in this channel to abandon.")
		}
		log.Printf("Error getting game: %v", err)
		return RespondWithError(s, i, fmt.Sprintf("Error getting game: %v", err))
	}

	// Abandon the game
	_, err = c.gameService.AbandonGame(ctx, &game.AbandonGameInput{
		GameID: existingGame.Game.ID,
	})
	if err != nil {
		log.Printf("Error abandoning game: %v", err)
		return RespondWithError(s, i, fmt.Sprintf("Failed to abandon game: %v", err))
	}

	// Respond with success message
	return RespondWithMessage(s, i, "Game abandoned successfully. You can start a new game with `/ronnied start`.")
}

// handlePay handles the pay button interaction
func (c *RonniedCommand) handlePay(s *discordgo.Session, i *discordgo.InteractionCreate, channelID, userID string, count int) error {
	ctx := context.Background()

	// Get the game in this channel
	existingGame, err := c.gameService.GetGameByChannel(ctx, &game.GetGameByChannelInput{
		ChannelID: channelID,
	})

	// Handle errors or missing game
	if err != nil {
		if errors.Is(err, game.ErrGameNotFound) {
			return RespondWithError(s, i, "No game found in this channel. Use `/ronnied start` to create a new game.")
		}
		log.Printf("Error getting game: %v", err)
		return RespondWithError(s, i, fmt.Sprintf("Error getting game: %v", err))
	}

	// Track how many drinks were successfully paid
	paidCount := 0
	
	// Pay one drink at a time
	for j := 0; j < count; j++ {
		_, err := c.gameService.PayDrink(ctx, &game.PayDrinkInput{
			GameID:   existingGame.Game.ID,
			PlayerID: userID,
		})
		
		if err != nil {
			// If we've paid at least one drink, consider it a partial success
			if paidCount > 0 {
				return RespondWithMessage(s, i, fmt.Sprintf("You've paid %d drinks. No more unpaid drinks found!", paidCount))
			}
			
			// Check for specific error about no unpaid drinks
			if strings.Contains(err.Error(), "no unpaid drinks found") {
				return RespondWithMessage(s, i, "You're all caught up! No drinks to pay right now. 🎉")
			}
			
			log.Printf("Error paying drink: %v", err)
			return RespondWithError(s, i, fmt.Sprintf("Failed to pay drinks: %v", err))
		}
		
		paidCount++
	}

	// Respond with success message
	if paidCount == 1 {
		return RespondWithMessage(s, i, "You've paid 1 drink. Cheers! 🍻")
	}
	return RespondWithMessage(s, i, fmt.Sprintf("You've paid %d drinks. Cheers! 🍻", paidCount))
}
