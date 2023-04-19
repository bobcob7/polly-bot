package models

import (
	"context"
	"fmt"

	"github.com/upper/db/v4"
)

const torrentNotificationTableName = "torrent_notifications"

type TorrentNotification struct {
	ID          string `db:"id"`
	TorrentID   string `db:"torrent_id"`
	RecipientID string `db:"recipient_id"`
	ChannelID   string `db:"channel_id"`
}

func (t *TorrentNotification) Create(ctx context.Context, sess db.Session) error {
	if err := sess.Collection(torrentNotificationTableName).InsertReturning(t); err != nil {
		return fmt.Errorf("failed deleting torrent notification: %w", err)
	}
	return nil
}

func (t *TorrentNotification) Delete(ctx context.Context, sess db.Session) error {
	if err := sess.Collection(torrentNotificationTableName).Find("id", t.ID).Delete(); err != nil {
		return fmt.Errorf("failed deleting torrent notification: %w", err)
	}
	return nil
}

func (t Torrent) DeleteNotification(ctx context.Context, sess db.Session) error {
	if err := sess.Collection(torrentNotificationTableName).Find("torrent_id", t.ID).Delete(); err != nil {
		return fmt.Errorf("failed deleting torrent notifications: %w", err)
	}
	return nil
}

func GetTorrentNotifications(ctx context.Context, sess db.Session, torrentID string) ([]*TorrentNotification, error) {
	output := make([]*TorrentNotification, 0)
	if err := sess.Collection(torrentNotificationTableName).Find("torrent_id", torrentID).Limit(100).All(&output); err != nil {
		return nil, fmt.Errorf("failed getting torrent notifications: %w", err)
	}
	return output, nil
}

func (t *Torrent) AddPrivateNotification(ctx context.Context, sess db.Session, recipientID string) error {
	notification := TorrentNotification{
		TorrentID:   t.ID,
		RecipientID: recipientID,
	}
	if _, err := sess.Collection(torrentNotificationTableName).Insert(&notification); err != nil {
		return fmt.Errorf("failed creating new torrent notification: %w", err)
	}
	return nil
}

func (t *Torrent) AddPublicNotification(ctx context.Context, sess db.Session, channelID string) error {
	notification := TorrentNotification{
		TorrentID: t.ID,
		ChannelID: channelID,
	}
	if _, err := sess.Collection(torrentNotificationTableName).Insert(&notification); err != nil {
		return fmt.Errorf("failed creating new torrent notification: %w", err)
	}
	return nil
}
