package bot

import (
	"context"
	"fmt"

	"github.com/bobcob7/polly/pkg/db"
	"github.com/bobcob7/polly/pkg/discord"
	"github.com/bobcob7/polly/pkg/transmission"
)

type Bot struct {
	TransmissionEndpoint string
	DiscordToken         string
	DiscordGuildID       string
	Database             db.Database
}

func (b *Bot) Start(ctx context.Context) <-chan error {
	errChan := make(chan error, 1)
	// Start transmission interface
	tr, err := transmission.New(ctx, b.TransmissionEndpoint)
	if err != nil {
		errChan <- fmt.Errorf("failed to connect to tranmission RPC: %w", err)
		close(errChan)
		return errChan
	}
	// Start discord interface
	bot := discord.New(
		&transmission.AddDownload{Transmission: tr},
		&transmission.UnfinishedDownloads{Transmission: tr},
		&transmission.SubscribeDownloads{Transmission: tr},
	)
	go func() {
		defer close(errChan)
		errChan <- bot.Run(ctx, b.DiscordToken, b.DiscordGuildID)
	}()
	return nil
}
