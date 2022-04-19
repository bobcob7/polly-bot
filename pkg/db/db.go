package db

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type AuthLevel string

const (
	AuthLevelNone      AuthLevel = "none"
	AuthLevelUser      AuthLevel = "user"
	AuthLevelModerator AuthLevel = "moderator"
	AuthLevelAdmin     AuthLevel = "admin"
	AuthLevelGod       AuthLevel = "god"
)

type User struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	AuthLevel AuthLevel `json:"auth_level"`
	Created   time.Time
	Updated   *time.Time
}

type DatabaseUser interface {
	GetUsers(ctx context.Context) ([]*User, error)
	CreateUser(ctx context.Context, name string) (*User, error)
	UpdateUser(ctx context.Context, user *User) error
	DeleteUser(ctx context.Context, id uuid.UUID) error
}

type Database interface {
	Close() error
	DatabaseUser
}
