package discord

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/bobcob7/polly-bot/internal/models"
	"github.com/bwmarrin/discordgo"
	"github.com/upper/db/v4"
	"go.uber.org/zap"
)

type PrivateMessenger struct {
	privateChannelTTL time.Duration
	sess              db.Session
}

func (p *PrivateMessenger) garbageCollect(ctx context.Context, dis *discordgo.Session) error {
	channels, err := models.GetPrivateChannels(ctx, p.sess)
	if err != nil {
		return fmt.Errorf("failed getting private channels from db: %w", err)
	}
	now := time.Now().UTC()
	for _, channel := range channels {
		if channel.LastMessageAt.Add(p.privateChannelTTL).Before(now) {
			zap.L().Info("Deleting private channel", zap.String("channelID", channel.ID))
			if _, err := dis.ChannelDelete(channel.ID); err != nil {
				return fmt.Errorf("failed delete private channel: %w", err)
			}

			if err := channel.Delete(ctx, p.sess); err != nil {
				return fmt.Errorf("failed deleting private channel from db: %w", err)
			}
		}
	}
	return nil
}

func (p *PrivateMessenger) SendMessage(ctx Context, recipientID, content string) error {
	var channel *models.PrivateChannel
	if err := p.sess.Tx(func(sess db.Session) error {
		var err error
		channel, err = models.GetPrivateChannel(ctx, sess, recipientID)
		if err != nil {
			if !errors.Is(err, db.ErrNoMoreRows) {
				return fmt.Errorf("unknown error getting private channels from db: %w", err)
			}
			// Need to make a new channel
			userChannel, err := ctx.Session.UserChannelCreate(recipientID)
			if err != nil {
				return fmt.Errorf("failed creating new private channel: %w", err)
			}
			channel = &models.PrivateChannel{
				ID:          userChannel.ID,
				RecipientID: recipientID,
			}
			if err := channel.Create(ctx, sess); err != nil {
				return fmt.Errorf("failed creating new private channel in db: %w", err)
			}
		}
		return nil
	}); err != nil {
		return fmt.Errorf("failed db transaction: %w", err)
	}
	if _, err := ctx.Session.ChannelMessageSend(channel.ID, content); err != nil {
		return fmt.Errorf("failed sending private message: %w", err)
	}
	if err := channel.Bump(ctx, p.sess); err != nil {
		return fmt.Errorf("failed bumping private channel: %w", err)
	}
	return nil
}
