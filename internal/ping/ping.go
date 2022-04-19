package ping

import (
	"fmt"
	"sync"

	"github.com/bobcob7/polly/pkg/discord"
	"github.com/bwmarrin/discordgo"
)

type Ping struct {
	sync.Mutex
	count int
}

func (p *Ping) Name() string {
	return "ping"
}

func (p *Ping) Command() *discordgo.ApplicationCommand {
	return &discordgo.ApplicationCommand{
		Name:        p.Name(),
		Description: "Echo back pings with pongs along with a count",
	}
}

func (p *Ping) Handle(ctx discord.Context) {
	p.Lock()
	p.count++
	currentCount := p.count
	p.Unlock()
	ctx.Session.InteractionRespond(ctx.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Title:   fmt.Sprintf("Ping#%d", currentCount),
			Content: fmt.Sprintf("Pong - %d", currentCount),
		},
	})
}
