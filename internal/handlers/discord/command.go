package discord

import (
	"github.com/bwmarrin/discordgo"
)

// CommandHandler defines the interface for Discord command handlers
type CommandHandler interface {
	// GetName returns the command name
	GetName() string
	
	// GetCommand returns the application command definition
	GetCommand() *discordgo.ApplicationCommand
	
	// Handle processes a Discord interaction
	Handle(s *discordgo.Session, i *discordgo.InteractionCreate) error
}

// BaseCommand provides common functionality for all commands
type BaseCommand struct {
	Name        string
	Description string
	Options     []*discordgo.ApplicationCommandOption
}

// GetName returns the command name
func (c *BaseCommand) GetName() string {
	return c.Name
}

// GetCommand returns the application command definition
func (c *BaseCommand) GetCommand() *discordgo.ApplicationCommand {
	return &discordgo.ApplicationCommand{
		Name:        c.Name,
		Description: c.Description,
		Options:     c.Options,
	}
}

// RespondWithMessage sends a simple text message response to an interaction
func RespondWithMessage(s *discordgo.Session, i *discordgo.InteractionCreate, message string) error {
	return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: message,
		},
	})
}

// RespondWithEmbed sends an embed response to an interaction
func RespondWithEmbed(s *discordgo.Session, i *discordgo.InteractionCreate, title, description string, fields []*discordgo.MessageEmbedField) error {
	embed := &discordgo.MessageEmbed{
		Title:       title,
		Description: description,
		Color:       0x00ff00, // Green color
		Fields:      fields,
	}
	
	return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{embed},
		},
	})
}

// RespondWithError sends an error response to an interaction
func RespondWithError(s *discordgo.Session, i *discordgo.InteractionCreate, errorMessage string) error {
	embed := &discordgo.MessageEmbed{
		Title:       "Error",
		Description: errorMessage,
		Color:       0xff0000, // Red color
	}
	
	return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{embed},
		},
	})
}

// RespondWithEmbedAndButtons sends an embed response with buttons to an interaction
func RespondWithEmbedAndButtons(s *discordgo.Session, i *discordgo.InteractionCreate, title, description string, fields []*discordgo.MessageEmbedField, buttons []discordgo.MessageComponent) error {
	embed := &discordgo.MessageEmbed{
		Title:       title,
		Description: description,
		Color:       0x00ff00, // Green color
		Fields:      fields,
	}
	
	// Create action row for buttons
	actionRow := discordgo.ActionsRow{
		Components: buttons,
	}
	
	return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds:     []*discordgo.MessageEmbed{embed},
			Components: []discordgo.MessageComponent{actionRow},
		},
	})
}

// RespondWithEphemeralEmbedAndButtons sends an ephemeral embed response with buttons to an interaction
func RespondWithEphemeralEmbedAndButtons(s *discordgo.Session, i *discordgo.InteractionCreate, title, description string, fields []*discordgo.MessageEmbedField, buttons []discordgo.MessageComponent) error {
	embed := &discordgo.MessageEmbed{
		Title:       title,
		Description: description,
		Color:       0x00ff00, // Green color
		Fields:      fields,
	}
	
	// Create action row for buttons
	actionRow := discordgo.ActionsRow{
		Components: buttons,
	}
	
	return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds:     []*discordgo.MessageEmbed{embed},
			Components: []discordgo.MessageComponent{actionRow},
			Flags:      discordgo.MessageFlagsEphemeral, // Make the message ephemeral (only visible to the user)
		},
	})
}

// RespondWithEphemeralMessage sends an ephemeral message response to an interaction
func RespondWithEphemeralMessage(s *discordgo.Session, i *discordgo.InteractionCreate, message string) error {
	return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: message,
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
}
