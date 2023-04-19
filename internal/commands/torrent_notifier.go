package commands

import (
	"fmt"

	"github.com/bobcob7/polly-bot/internal/models"
	"github.com/bobcob7/polly-bot/pkg/discord"
	"github.com/bwmarrin/discordgo"
	"github.com/upper/db/v4"
	"go.uber.org/zap"
)

type TorrentNotifier struct {
	CompletedTorrents chan *models.Torrent
	dbSession         db.Session
}

func NewTorrentNotifier(dbSession db.Session) *TorrentNotifier {
	return &TorrentNotifier{
		CompletedTorrents: make(chan *models.Torrent),
		dbSession:         dbSession,
	}
}

func (t *TorrentNotifier) OnStart(ctx discord.Context, s *discordgo.Session) error {
	go func() {
		logger := ctx.Logger()
		for {
			select {
			case <-ctx.Done():
				return
			case torrent := <-t.CompletedTorrents:
				logger.Info("completed torrent", zap.String("name", torrent.NameString()))
				notifications, err := models.GetTorrentNotifications(ctx, t.dbSession, torrent.ID)
				if err != nil {
					logger.Error("failed to get notifications", zap.Error(err))
				}
				for _, notification := range notifications {
					content := fmt.Sprintf("Completed download: %s", torrent.NameString())
					if notification.RecipientID != "" {
						if err := ctx.PrivateMessenger.SendMessage(ctx, notification.RecipientID, content); err != nil {
							logger.Error("failed to send notification", zap.Error(err), zap.String("recipientID", notification.RecipientID))
						}
					}
					if notification.ChannelID != "" {
						if _, err := ctx.Session.ChannelMessageSend(notification.ChannelID, content); err != nil {
							logger.Error("failed to send notification", zap.Error(err), zap.String("channelID", notification.ChannelID))
						}
					}
				}
			}
		}
	}()
	return nil
}
