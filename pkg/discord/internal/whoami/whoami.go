package whoami

import (
	"fmt"

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

func (p *WhoAmI) Handle(s *discordgo.Session, i *discordgo.InteractionCreate) {
	var user *discordgo.User
	var roles []string
	if i.User != nil {
		user = i.User
	} else if i.Member != nil && i.Member.User != nil {
		user = i.Member.User
		roles = i.Member.Roles
	}
	content := fmt.Sprintf(`ID:		%s
Name:	%s
Email:	%s
Roles:	%v`,
		user.ID,
		user.Username,
		user.Email,
		roles,
	)
	if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Title:   "Who Am I?",
			Content: content,
		},
	}); err != nil {
		zap.L().Error("failed to respond to interaction", zap.Error(err))
	}
}
