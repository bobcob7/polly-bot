package commands

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/bobcob7/polly-bot/pkg/discord"
	"github.com/bwmarrin/discordgo"
	"go.uber.org/zap"
)

type Job struct {
	ttl       time.Time
	channelID string
}

type Echo struct {
	lock sync.Mutex
	jobs map[string]Job
}

func (p *Echo) Run(ctx context.Context, s *discordgo.Session) error {
	p.jobs = make(map[string]Job)
	ticker := time.NewTicker(time.Second)
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			p.lock.Lock()
			for id, job := range p.jobs {
				if job.ttl.Before(time.Now()) {
					zap.L().Debug("echo finished", zap.String("id", id))
					delete(p.jobs, id)
					_, err := s.ChannelMessageSend(job.channelID, fmt.Sprintf("Echo %s", id))
					if err != nil {
						return fmt.Errorf("failed to send channel response: %w", err)
					}
				}
			}
			p.lock.Unlock()
		}
	}
}

func (p *Echo) Name() string {
	return "echo"
}

func (p *Echo) Command() *discordgo.ApplicationCommand {
	return &discordgo.ApplicationCommand{
		Name:        p.Name(),
		Description: "Echo returns immediately, then makes an announcement after a specified delay",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "delay",
				Description: "Delay to wait until the announcement is made",
				Required:    true,
			},
		},
	}
}

func (p *Echo) Handle(ctx discord.Context) error {
	input := ctx.Interaction.ApplicationCommandData().Options[0].StringValue()
	delay, err := time.ParseDuration(input)
	if err != nil {
		return fmt.Errorf("failed to parse input duration %q: %w", input, err)
	}
	//nolint: gosec
	id := rand.Int()
	ttl := time.Now().Add(delay)
	p.lock.Lock()
	p.jobs[fmt.Sprintf("%d", id)] = Job{
		ttl:       ttl,
		channelID: ctx.Interaction.ChannelID,
	}
	p.lock.Unlock()
	err = ctx.Session.InteractionRespond(ctx.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Title:   "...",
			Content: fmt.Sprintf("...%d...", id),
		},
	})
	if err != nil {
		return failedResponseInteractionError{err}
	}
	return nil
}
