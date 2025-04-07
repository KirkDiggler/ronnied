package discord

import (
	"github.com/KirkDiggler/ronnied/internal/services/game"
	"github.com/bwmarrin/discordgo"
	"fmt"
	"github.com/KirkDiggler/ronnied/internal/models"
)

// renderRollDiceResponse renders the response for a roll dice action
func renderRollDiceResponse(s *discordgo.Session, i *discordgo.InteractionCreate, output *game.RollDiceOutput) error {
	var components []discordgo.MessageComponent

	// Build components based on the roll result
	if output.IsCriticalHit {
		// Create player selection dropdown for critical hits
		if len(output.EligiblePlayers) > 0 {
			var playerOptions []discordgo.SelectMenuOption
			
			for _, player := range output.EligiblePlayers {
				playerOptions = append(playerOptions, discordgo.SelectMenuOption{
					Label:       player.PlayerName,
					Value:       player.PlayerID,
					Description: "Assign a drink to this player",
					Emoji: &discordgo.ComponentEmoji{
						Name: "🍺",
					},
				})
			}
			
			playerSelect := discordgo.SelectMenu{
				CustomID:    SelectAssignDrink,
				Placeholder: "Select a player to drink",
				Options:     playerOptions,
			}
			
			components = append(components, discordgo.SelectMenu(playerSelect))
		}
	} else {
		// Create roll again button for non-critical hits
		rollButton := discordgo.Button{
			Label:    "Roll Again",
			Style:    discordgo.PrimaryButton,
			CustomID: ButtonRollDice,
			Emoji: &discordgo.ComponentEmoji{
				Name: "🎲",
			},
		}
		
		components = append(components, rollButton)
	}

	// Create action row for components if we have any
	var messageComponents []discordgo.MessageComponent
	if len(components) > 0 {
		messageComponents = append(messageComponents, discordgo.ActionsRow{
			Components: components,
		})
	}

	// Create embeds
	embeds := []*discordgo.MessageEmbed{
		{
			Title:       output.Result,
			Description: output.Details,
			Color:       0x00ff00, // Green color
		},
	}

	// Check if this is a component interaction (button click)
	if i.Type == discordgo.InteractionMessageComponent {
		// Update the existing message instead of sending a new one
		return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseUpdateMessage,
			Data: &discordgo.InteractionResponseData{
				Content:    output.Result,
				Embeds:     embeds,
				Components: messageComponents,
			},
		})
	} else {
		// For the initial interaction, create a new ephemeral message
		return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content:    output.Result,
				Embeds:     embeds,
				Components: messageComponents,
				Flags:      discordgo.MessageFlagsEphemeral,
			},
		})
	}
}

// renderGameMessage creates the game message with appropriate components based on game status
func renderGameMessage(game *models.Game) (*discordgo.MessageSend, error) {
	// Create the base message
	message := &discordgo.MessageSend{
		Content: fmt.Sprintf("Game ID: %s", game.ID),
	}

	// Create the game info embed
	gameEmbed := &discordgo.MessageEmbed{
		Title: "Ronnied Dice Game",
		Color: 0x00ff00, // Green color
	}

	// Set description based on game status
	switch game.Status {
	case models.GameStatusWaiting:
		gameEmbed.Description = "Waiting for players to join. Click the Join button to join the game!"
	case models.GameStatusActive:
		gameEmbed.Description = "Game in progress. Click the Roll button to roll the dice!"
	case models.GameStatusRollOff:
		gameEmbed.Description = "Roll-off in progress! Players in the roll-off need to roll again."
	case models.GameStatusCompleted:
		gameEmbed.Description = "Game completed. Check the leaderboard to see who owes drinks!"
	default:
		gameEmbed.Description = "Unknown game status."
	}

	// Add fields for game info
	gameEmbed.Fields = append(gameEmbed.Fields, &discordgo.MessageEmbedField{
		Name:   "Status",
		Value:  string(game.Status),
		Inline: true,
	})

	// Add participant list
	var participantList string
	if len(game.Participants) == 0 {
		participantList = "No players yet"
	} else {
		for _, p := range game.Participants {
			participantList += fmt.Sprintf("• %s\n", p.PlayerName)
		}
	}

	gameEmbed.Fields = append(gameEmbed.Fields, &discordgo.MessageEmbedField{
		Name:   "Players",
		Value:  participantList,
		Inline: true,
	})

	// Add the embed to the message
	message.Embeds = append(message.Embeds, gameEmbed)

	// Add components based on game status
	var components []discordgo.MessageComponent

	// Only show Join and Begin buttons when the game is in waiting status
	if game.Status == models.GameStatusWaiting {
		// Join button
		joinButton := discordgo.Button{
			Label:    "Join Game",
			Style:    discordgo.SuccessButton,
			CustomID: ButtonJoinGame,
			Emoji: &discordgo.ComponentEmoji{
				Name: "👋",
			},
		}
		components = append(components, joinButton)

		// Begin button
		beginButton := discordgo.Button{
			Label:    "Begin Game",
			Style:    discordgo.PrimaryButton,
			CustomID: ButtonBeginGame,
			Emoji: &discordgo.ComponentEmoji{
				Name: "🎮",
			},
		}
		components = append(components, beginButton)
	}

	// Show Roll button when the game is active or in roll-off
	if game.Status == models.GameStatusActive || game.Status == models.GameStatusRollOff {
		// Roll button
		rollButton := discordgo.Button{
			Label:    "Roll Dice",
			Style:    discordgo.PrimaryButton,
			CustomID: ButtonRollDice,
			Emoji: &discordgo.ComponentEmoji{
				Name: "🎲",
			},
		}
		components = append(components, rollButton)
	}

	// Show Leaderboard button for all game states
	leaderboardButton := discordgo.Button{
		Label:    "Leaderboard",
		Style:    discordgo.SecondaryButton,
		CustomID: ButtonLeaderboard,
		Emoji: &discordgo.ComponentEmoji{
			Name: "📊",
		},
	}
	components = append(components, leaderboardButton)

	// Add components to the message if we have any
	if len(components) > 0 {
		message.Components = append(message.Components, discordgo.ActionsRow{
			Components: components,
		})
	}

	return message, nil
}
