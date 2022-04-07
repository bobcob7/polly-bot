package transmission

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
	"go.uber.org/zap"
	"golang.org/x/exp/maps"
)

type SubscribeDownloads struct {
	*Transmission
	sync.RWMutex
	subscribedChannels map[string]struct{}
}

func (p *SubscribeDownloads) Name() string {
	return "subscribe-downloads"
}

func (p *SubscribeDownloads) Command() *discordgo.ApplicationCommand {
	return &discordgo.ApplicationCommand{
		Name:        p.Name(),
		Description: "Subscribe to download notifications",
	}
}

func (p *SubscribeDownloads) Handle(s *discordgo.Session, i *discordgo.InteractionCreate) {
	var deleted bool
	p.Lock()
	if _, deleted = p.subscribedChannels[i.ChannelID]; deleted {
		delete(p.subscribedChannels, i.ChannelID)
	} else {
		p.subscribedChannels[i.ChannelID] = struct{}{}
	}
	p.Unlock()
	var err error
	if deleted {
		err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Title:   "Successfully unsubscribed channel",
				Content: "Unsubscribed channel from download notifications",
			},
		})
	} else {
		err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Title:   "Successfully subscribed channel",
				Content: "Subscribed channel to download notifications",
			},
		})
	}
	if err != nil {
		zap.L().Error("Failed to respond to interaction", zap.Error(err))
	}
}

func (p *SubscribeDownloads) Run(ctx context.Context, s *discordgo.Session) error {
	p.subscribedChannels = make(map[string]struct{})
	ticker := time.NewTicker(time.Second)
	finishedDownloads := make(chan string, 10)
	defer close(finishedDownloads)
	go p.sendNotifications(s, finishedDownloads)
	// Get list of all torrents that are in progress
	downloadingTorrents, err := p.getDownloadingTorrents(ctx)
	if err != nil {
		return fmt.Errorf("failed to get initial downloading torrents: %w", err)
	}
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			// Get a list of torrents that have been completed
			if completedDownloads, err := p.getCompletedTorrents(ctx, maps.Keys(downloadingTorrents)); err != nil {
				return fmt.Errorf("failed to get completed torrents: %w", err)
			} else {
				// Send notifications for completed torrents
				for _, torrent := range completedDownloads {
					finishedDownloads <- fmt.Sprintf("%s finished downloading", torrent.Name)
				}
			}
			// Update list of all torrents that are in progress
			if downloadingTorrents, err = p.getDownloadingTorrents(ctx); err != nil {
				return fmt.Errorf("failed to get downloading torrents: %w", err)
			}
		}
	}
}

func (p *SubscribeDownloads) sendNotifications(s *discordgo.Session, notifications <-chan string) {
	for notification := range notifications {
		p.sendNotification(s, notification)
	}
}

func (p *SubscribeDownloads) sendNotification(s *discordgo.Session, notification string) {
	p.RLock()
	defer p.RUnlock()
	for channelID := range p.subscribedChannels {
		_, err := s.ChannelMessageSend(channelID, notification)
		if err != nil {
			zap.L().Error("failed to send notification", zap.Error(err))
		}
	}
}
