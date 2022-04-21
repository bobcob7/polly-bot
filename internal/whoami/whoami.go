package whoami

import (
	"fmt"

	"github.com/bobcob7/polly/pkg/discord"
	"github.com/bwmarrin/discordgo"
	"go.uber.org/zap"
)

type WhoAmI struct{}

func (p *WhoAmI) Name() string {
	return "whoami"
}

func (p *WhoAmI) Command() *discordgo.ApplicationCommand {
	return &discordgo.ApplicationCommand{
		Name:        p.Name(),
		Description: "Returns back information about the caller",
	}
}

func (p *WhoAmI) Handle(ctx discord.Context) {
	var user *discordgo.User
	var roles []string
	if ctx.Interaction.User != nil {
		user = ctx.Interaction.User
	} else if ctx.Interaction.Member != nil && ctx.Interaction.Member.User != nil {
		user = ctx.Interaction.Member.User
		roles = ctx.Interaction.Member.Roles
	}
	content := fmt.Sprintf(`ID:		%s
Name:	%s
Email:	%s
Level:  %d
Roles:	%v`,
		user.ID,
		user.Username,
		user.Email,
		ctx.MinLevel(),
		roles,
	)
	if err := ctx.Session.InteractionRespond(ctx.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Title:   "Who Am I?",
			Content: content,
		},
	}); err != nil {
		zap.L().Error("failed to respond to interaction", zap.Error(err))
	}
}
