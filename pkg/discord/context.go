package discord

import (
	"context"

	"github.com/bwmarrin/discordgo"
	"go.uber.org/zap"
)

type Context struct {
	//nolint: containedctx
	context.Context
	*discordgo.Session
	*discordgo.InteractionCreate
	*PrivateMessenger
	logger *zap.Logger
}

func (c *Context) Logger() *zap.Logger {
	return c.logger
}

func (c *Context) Error(err error) {
	c.logger.Info("Failed with error", zap.Error(err))
	if err := c.Session.InteractionRespond(c.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Title:   "Error",
			Content: err.Error(),
		},
	}); err != nil {
		c.logger.Error("Failed to respond with error", zap.Error(err))
	}
}

func (c *Context) UserID() string {
	if c.Interaction == nil {
		return ""
	}
	var user *discordgo.User
	if c.Interaction.User != nil {
		user = c.Interaction.User
	} else if c.Interaction.Member != nil && c.Interaction.Member.User != nil {
		user = c.Interaction.Member.User
	}
	if user != nil {
		return user.ID
	}
	return ""
}

func (c *Context) ChannelID() string {
	if c.Interaction == nil {
		return ""
	}
	return c.Interaction.ChannelID
}
