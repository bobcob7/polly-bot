package models

import (
	"context"
	"fmt"
	"time"

	"github.com/upper/db/v4"
)

const privateChannelsTableName = "private_channels"

type PrivateChannel struct {
	ID            string     `db:"id"`
	RecipientID   string     `db:"recipient_id"`
	CreatedAt     time.Time  `db:"created_at"`
	LastMessageAt *time.Time `db:"last_message_at"`
}

func (t *PrivateChannel) Create(ctx context.Context, sess db.Session) error {
	now := time.Now().UTC()
	t.CreatedAt = now
	t.LastMessageAt = &now
	if err := sess.Collection(privateChannelsTableName).InsertReturning(t); err != nil {
		return fmt.Errorf("failed creating private channel: %w", err)
	}
	return nil
}

func (t *PrivateChannel) Delete(ctx context.Context, sess db.Session) error {
	if err := sess.Collection(privateChannelsTableName).Find("id", t.ID).Delete(); err != nil {
		return fmt.Errorf("failed deleting private channel: %w", err)
	}
	return nil
}

func (t *PrivateChannel) Bump(ctx context.Context, sess db.Session) error {
	now := time.Now().UTC()
	t.LastMessageAt = &now
	if err := sess.Collection(privateChannelsTableName).Find("id", t.ID).Update(t); err != nil {
		return fmt.Errorf("failed deleting private channel: %w", err)
	}
	return nil
}

func GetPrivateChannels(ctx context.Context, sess db.Session) ([]*PrivateChannel, error) {
	output := make([]*PrivateChannel, 0)
	if err := sess.Collection(privateChannelsTableName).Find().Limit(100).All(&output); err != nil {
		return nil, fmt.Errorf("failed getting private channels: %w", err)
	}
	return output, nil
}

func GetPrivateChannel(ctx context.Context, sess db.Session, recipientID string) (*PrivateChannel, error) {
	output := &PrivateChannel{}
	if err := sess.Collection(privateChannelsTableName).Find("recipient_id", recipientID).One(output); err != nil {
		return nil, fmt.Errorf("failed getting private channel: %w", err)
	}
	return output, nil
}
