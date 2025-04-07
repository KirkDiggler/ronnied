package discord

import (
	"github.com/KirkDiggler/ronnied/internal/services/game"
	"github.com/bwmarrin/discordgo"
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
			Title:       output.Title,
			Description: output.Description,
			Color:       0x00ff00, // Green color
		},
	}

	// Check if this is a component interaction (button click)
	if i.Type == discordgo.InteractionMessageComponent {
		// Update the existing message instead of sending a new one
		return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseUpdateMessage,
			Data: &discordgo.InteractionResponseData{
				Content:    output.Title,
				Embeds:     embeds,
				Components: messageComponents,
			},
		})
	} else {
		// For the initial interaction, create a new ephemeral message
		return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content:    output.Title,
				Embeds:     embeds,
				Components: messageComponents,
				Flags:      discordgo.MessageFlagsEphemeral,
			},
		})
	}
}
