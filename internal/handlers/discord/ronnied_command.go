package discord

import (
	"context"
	"errors"
	"fmt"
	"log"
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
			Description: "Start a new dice rolling drinking game",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "start",
					Description: "Create a new game",
				},
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "join",
					Description: "Join the current game",
				},
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "begin",
					Description: "Begin the game after players have joined",
				},
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "roll",
					Description: "Roll the dice",
				},
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "leaderboard",
					Description: "Show the current game leaderboard",
				},
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "sessionboard",
					Description: "Show the session leaderboard (drinks across all games)",
				},
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "newsession",
					Description: "Start a new drinking session (resets the session leaderboard)",
				},
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "abandon",
					Description: "Abandon the current game (for debugging)",
				},
			},
		},
		gameService: gameService,
	}
}

// Handle processes a Discord interaction for the ronnied command
func (c *RonniedCommand) Handle(s *discordgo.Session, i *discordgo.InteractionCreate) error {
	// Get the subcommand
	options := i.ApplicationCommandData().Options
	if len(options) == 0 {
		return RespondWithError(s, i, "Please specify a subcommand")
	}

	subcommand := options[0].Name

	// Get the channel ID
	channelID := i.ChannelID

	// Get the user ID
	userID := i.Member.User.ID
	username := i.Member.User.Username
	if i.Member.Nick != "" {
		username = i.Member.Nick
	}

	// Handle the subcommand
	switch subcommand {
	case "start":
		return c.handleStart(s, i, channelID, userID, username)
	case "join":
		return c.handleJoin(s, i, channelID, userID, username)
	case "begin":
		return c.handleBegin(s, i, channelID, userID)
	case "roll":
		return c.handleRoll(s, i, channelID, userID)
	case "leaderboard":
		return c.handleLeaderboard(s, i, channelID)
	case "sessionboard":
		return c.handleSessionboard(s, i, channelID)
	case "newsession":
		return c.handleNewSession(s, i, channelID)
	case "abandon":
		return c.handleAbandon(s, i, channelID, userID)
	default:
		return RespondWithError(s, i, fmt.Sprintf("Unknown subcommand: %s", subcommand))
	}
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
		Emoji: &discordgo.ComponentEmoji{
			Name: "🎲",
		},
	}

	startButton := discordgo.Button{
		Label:    "Begin Game",
		Style:    discordgo.PrimaryButton,
		CustomID: ButtonBeginGame,
		Emoji: &discordgo.ComponentEmoji{
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

// handleJoin handles the join subcommand
func (c *RonniedCommand) handleJoin(s *discordgo.Session, i *discordgo.InteractionCreate, channelID, userID, username string) error {
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

	// Check if the game is in a state where players can join
	if existingGame.Game.Status != models.GameStatusWaiting {
		return RespondWithError(s, i, "Cannot join game. Game is not in waiting state.")
	}

	// Join the game
	_, err = c.gameService.JoinGame(ctx, &game.JoinGameInput{
		GameID:     existingGame.Game.ID,
		PlayerID:   userID,
		PlayerName: username,
	})
	if err != nil {
		log.Printf("Error joining game: %v", err)
		return RespondWithError(s, i, fmt.Sprintf("Failed to join game: %v", err))
	}

	// Respond with success message
	return RespondWithMessage(s, i, fmt.Sprintf("You've joined the game! Wait for the creator to start the game."))
}

// handleBegin handles the begin subcommand
func (c *RonniedCommand) handleBegin(s *discordgo.Session, i *discordgo.InteractionCreate, channelID, userID string) error {
	ctx := context.Background()

	// Check if there's a game in this channel
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

	// Check if the game is in a state where it can be started
	if existingGame.Game.Status != models.GameStatusWaiting {
		return RespondWithError(s, i, "Cannot start game. Game is not in waiting state.")
	}

	// Start the game
	startOutput, err := c.gameService.StartGame(ctx, &game.StartGameInput{
		GameID:   existingGame.Game.ID,
		PlayerID: userID,
	})
	if err != nil {
		log.Printf("Error starting game: %v", err)
		return RespondWithError(s, i, fmt.Sprintf("Failed to start game: %v", err))
	}

	if !startOutput.Success {
		return RespondWithError(s, i, "Failed to start the game. Make sure you are the creator of the game.")
	}

	// Create roll button
	rollButton := discordgo.Button{
		Label:    "Roll Dice",
		Style:    discordgo.PrimaryButton,
		CustomID: ButtonRollDice,
		Emoji: &discordgo.ComponentEmoji{
			Name: "🎲",
		},
	}

	// Send an ephemeral message to the user who started the game
	err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "Game Started! Click the button below to roll your dice.",
			Flags:   discordgo.MessageFlagsEphemeral,
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{rollButton},
				},
			},
		},
	})
	if err != nil {
		log.Printf("Error sending ephemeral message: %v", err)
		return err
	}

	// Update the main message in the channel if it exists
	if existingGame.Game.MessageID != "" {
		// Create fields for the updated message
		fields := []*discordgo.MessageEmbedField{
			{
				Name:   "Status",
				Value:  "Game in progress",
				Inline: true,
			},
			{
				Name:   "Players",
				Value:  fmt.Sprintf("%d", len(existingGame.Game.Participants)),
				Inline: true,
			},
		}

		// Update the message to remove buttons and show game status
		embeds := []*discordgo.MessageEmbed{
			{
				Title:       "Game Started!",
				Description: "The game has begun! Each player has received a private message with a button to roll their dice.",
				Color:       0x00ff00, // Green color
				Fields:      fields,
			},
		}

		components := []discordgo.MessageComponent{}

		_, err = s.ChannelMessageEditComplex(&discordgo.MessageEdit{
			Channel:    channelID,
			ID:         existingGame.Game.MessageID,
			Embeds:     &embeds,
			Components: &components,
		})
		if err != nil {
			// Just log the error and continue, this isn't critical
			log.Printf("Error updating game message: %v", err)
		}
	}

	// Send direct messages to other players with roll buttons
	for _, participant := range existingGame.Game.Participants {
		// Skip the creator as they already got the ephemeral message
		if participant.PlayerID == userID {
			continue
		}

		// Create a DM channel with the player
		dmChannel, err := s.UserChannelCreate(participant.PlayerID)
		if err != nil {
			log.Printf("Error creating DM channel with player %s: %v", participant.PlayerID, err)
			continue
		}

		// Send a message with the roll button
		_, err = s.ChannelMessageSendComplex(dmChannel.ID, &discordgo.MessageSend{
			Embeds: []*discordgo.MessageEmbed{
				{
					Title:       "Game Started!",
					Description: fmt.Sprintf("A game has started in the channel. Click the button below to roll your dice."),
					Color:       0x00ff00, // Green color
				},
			},
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{rollButton},
				},
			},
		})
		if err != nil {
			log.Printf("Error sending DM to player %s: %v", participant.PlayerID, err)
		}
	}

	return nil
}

// handleRoll handles the roll subcommand
func (c *RonniedCommand) handleRoll(s *discordgo.Session, i *discordgo.InteractionCreate, channelID, userID string) error {
	ctx := context.Background()

	// Get the game in this channel
	existingGame, err := c.gameService.GetGameByChannel(ctx, &game.GetGameByChannelInput{
		ChannelID: channelID,
	})

	// Handle errors or missing game
	if err != nil {
		if err == game.ErrGameNotFound {
			return RespondWithError(s, i, "No active game found in this channel. Use `/ronnied start` to create a new game.")
		}
		log.Printf("Error getting game: %v", err)
		return RespondWithError(s, i, fmt.Sprintf("Error getting game: %v", err))
	}

	// Check if the game is in a state where players can roll
	if existingGame.Game.Status != models.GameStatusActive {
		return RespondWithError(s, i, "Cannot roll dice. Game is not active.")
	}

	// Roll the dice
	rollOutput, err := c.gameService.RollDice(ctx, &game.RollDiceInput{
		GameID:   existingGame.Game.ID,
		PlayerID: userID,
	})
	if err != nil {
		log.Printf("Error rolling dice: %v", err)
		return RespondWithError(s, i, fmt.Sprintf("Failed to roll dice: %v", err))
	}

	// Build the response message
	var description string
	if rollOutput.IsCriticalHit {
		description = "Critical hit! You can assign a drink to another player."
	} else if rollOutput.IsCriticalFail {
		description = "Critical fail! Take a drink."
	} else {
		description = "You rolled the dice."
	}

	// Respond with the roll result
	return RespondWithMessage(s, i, fmt.Sprintf("You rolled a %d. %s", rollOutput.Value, description))
}

// handleLeaderboard handles the leaderboard subcommand
func (c *RonniedCommand) handleLeaderboard(s *discordgo.Session, i *discordgo.InteractionCreate, channelID string) error {
	ctx := context.Background()

	// Get the game in this channel
	existingGame, err := c.gameService.GetGameByChannel(ctx, &game.GetGameByChannelInput{
		ChannelID: channelID,
	})

	// Handle errors or missing game
	if err != nil {
		if err == game.ErrGameNotFound {
			return RespondWithError(s, i, "No game found in this channel. Use `/ronnied start` to create a new game.")
		}
		log.Printf("Error getting game: %v", err)
		return RespondWithError(s, i, fmt.Sprintf("Error getting game: %v", err))
	}

	// Get the leaderboard
	leaderboard, err := c.gameService.GetLeaderboard(ctx, &game.GetLeaderboardInput{
		GameID: existingGame.Game.ID,
	})
	if err != nil {
		log.Printf("Error getting leaderboard: %v", err)
		return RespondWithError(s, i, fmt.Sprintf("Failed to get leaderboard: %v", err))
	}

	// Build the leaderboard description
	var description strings.Builder
	if len(leaderboard.Entries) == 0 {
		description.WriteString("No drinks have been assigned yet.")
	} else {
		for _, entry := range leaderboard.Entries {
			description.WriteString(fmt.Sprintf("**%s**: %d drinks\n", entry.PlayerName, entry.DrinkCount))
		}
	}

	// Respond with the leaderboard
	return RespondWithEmbed(s, i, "Leaderboard", description.String(), nil)
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
	if len(sessionboard.Entries) == 0 {
		description.WriteString("No drinks have been assigned yet.")
	} else {
		for _, entry := range sessionboard.Entries {
			description.WriteString(fmt.Sprintf("**%s**: %d drinks\n", entry.PlayerName, entry.DrinkCount))
		}
	}

	// Respond with the session leaderboard
	return RespondWithEmbed(s, i, "Session Leaderboard", description.String(), nil)
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
