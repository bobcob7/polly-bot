package ping

import (
	"fmt"
	"sync"

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

func (p *Ping) Handle(s *discordgo.Session, i *discordgo.InteractionCreate) {
	p.Lock()
	p.count++
	currentCount := p.count
	p.Unlock()
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Title:   fmt.Sprintf("Ping#%d", currentCount),
			Content: fmt.Sprintf("Pong - %d", currentCount),
		},
	})
}
