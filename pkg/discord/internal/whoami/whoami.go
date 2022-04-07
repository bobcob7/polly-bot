package whoami

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
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
	content := fmt.Sprintf(`ID:		%s
Name:	%s
Email:	%s`,
		i.User.ID,
		i.User.Username,
		i.User.Email,
	)
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Title:   "Who Am I?",
			Content: content,
		},
	})
}
