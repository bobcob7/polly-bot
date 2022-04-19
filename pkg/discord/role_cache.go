package discord

import (
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
)

type roleCache struct {
	sync.Mutex
	session   *discordgo.Session
	guildID   string
	ttl       time.Duration
	updatedAt time.Time
	roles     map[string]string
}

// Get will check the cache for the value, if the cache is "too old", then it will be automatically refreshed.
// If the value is not found, then an os.ErrNotExist will be returned.
func (r *roleCache) Get(id string) (string, error) {
	r.Lock()
	defer r.Unlock()
	// Check if cache is too old
	if r.updatedAt.Add(r.ttl).Before(time.Now()) {
		// Refresh cache
		roles, err := r.session.GuildRoles(r.guildID)
		if err != nil {
			return "", fmt.Errorf("failed to get roles: %w", err)
		}
		r.updatedAt = time.Now()
		r.roles = make(map[string]string, len(roles))
		for _, role := range roles {
			r.roles[role.ID] = role.Name
		}
	}
	if v, ok := r.roles[id]; ok {
		return v, nil
	}
	return "", os.ErrNotExist
}
