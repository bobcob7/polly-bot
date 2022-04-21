package discord

import (
	"context"
	"fmt"
	"log"
	"reflect"
	"strings"

	"github.com/bwmarrin/discordgo"
	"go.uber.org/zap"
	"golang.org/x/exp/maps"
)

type Config struct {
	Token      string
	GuildID    string   `map:"GUILD_ID"`
	RootUserID string   `map:"ROOT_USER_ID"`
	RolePrefix string   `map:"ROLE_PREFIX"`
	RoleLevels []string `map:"ROLE_LEVEL"`
}

func (c Config) Valid() (errs []string) {
	if c.Token == "" {
		errs = append(errs, "Discord Token is required")
	}
	return
}

type Context struct {
	context.Context
	*discordgo.Session
	*discordgo.InteractionCreate
	logger    *zap.Logger
	userLevel int
}

func (c *Context) HasLevel(level int) bool {
	return level <= c.userLevel
}

func (c *Context) Logger() *zap.Logger {
	return c.logger
}

func (c *Context) Error(err error) {
	c.logger.Info("Failed with error", zap.Error(err))
	if err := c.Session.InteractionRespond(c.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Title:   "Error",
			Content: err.Error(),
		},
	}); err != nil {
		c.logger.Error("Failed to respond with error", zap.Error(err))
	}
}

type Command interface {
	Name() string
	Command() *discordgo.ApplicationCommand
	Handle(ctx Context)
}

type AdvancedCommand interface {
	Command
	Run(ctx context.Context, s *discordgo.Session) error
}

type registeredCommand struct {
	Command
	id   string
	acls []string
}

func registerHandlers(commands ...Command) map[string]registeredCommand {
	cmds := []Command{}
	cmds = append(cmds, commands...)
	output := make(map[string]registeredCommand, len(cmds))
	for _, cmd := range cmds {
		output[cmd.Name()] = registeredCommand{Command: cmd}
	}
	return output
}

type Bot struct {
	config       Config
	guildRoleMap map[string][]string
	handles      map[string]registeredCommand
	roleResolver interface{ Get(string) (string, error) }
}

func New(config Config, cmds ...Command) *Bot {
	return &Bot{
		config:       config,
		guildRoleMap: map[string][]string{},
		handles:      registerHandlers(cmds...),
	}
}

func GetGuilds(token string) (map[string]string, error) {
	output := make(map[string]string)
	// Create Discord session
	session, err := discordgo.New("Bot " + token)
	if err != nil {
		return output, fmt.Errorf("failed to create discord session %w", err)
	}
	var afterID string
	const pageSize = 2
	for {
		guilds, err := session.UserGuilds(pageSize, "", afterID)
		if err != nil {
			return output, fmt.Errorf("failed to get guilds %w", err)
		}
		for _, guild := range guilds {
			output[guild.ID] = guild.Name
		}
		if len(guilds) < pageSize {
			// Exit if the returned guilds are less than the max
			break
		}
		// Set afterID to the last guild ID
		afterID = guilds[len(guilds)-1].ID
	}
	return output, nil
}

func errorResponse(s *discordgo.Session, i *discordgo.Interaction, err error) {
	zap.L().Error("Failed to get role names", zap.Error(err))
	if err := s.InteractionRespond(i, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Title:   "Internal error?",
			Content: "Internal error",
		},
	}); err != nil {
		zap.L().Error("Failed to respond with error", zap.Error(err))
	}
}

func aclMatch(allowed, available []string) bool {
	for _, v := range available {
		for _, a := range allowed {
			if v == a {
				return true
			}
		}
	}
	return false
}

func (b *Bot) roleNames(s *discordgo.Session, roleIDs []string) ([]string, error) {
	output := []string{}
	for _, id := range roleIDs {
		roleName, err := b.roleResolver.Get(id)
		if err != nil {
			return nil, err
		}
		output = append(output, roleName)
	}
	return output, nil
}

func (b *Bot) updateGuildRoles(s *discordgo.Session, guildID string) {
	logger := zap.L().With(zap.String("guildID", guildID))
	roles, err := s.GuildRoles(guildID)
	if err != nil {
		logger.Error("failed to get roles", zap.Error(err))
		return
	}
	found := make(map[string]struct{}, len(b.config.RoleLevels))
	for _, role := range roles {
		if strings.HasPrefix(role.Name, b.config.RolePrefix) {
			for _, suffix := range b.config.RoleLevels {
				if strings.HasSuffix(role.Name, suffix) {
					found[role.ID] = struct{}{}
				}
			}
		}
	}
	b.guildRoleMap[guildID] = maps.Keys(found)
	if len(b.guildRoleMap[guildID]) != len(b.config.RoleLevels) {
		logger.Warn("Failed to find all roles", zap.Int("gotNum", len(b.guildRoleMap[guildID])), zap.Int("wantNum", len(b.config.RoleLevels)))
	}
}

func (b *Bot) Run(ctx context.Context) error {
	if b.config.Token == "" {
		return fmt.Errorf("missing token")
	}
	// Create Discord session
	session, err := discordgo.New("Bot " + b.config.Token)
	if err != nil {
		return fmt.Errorf("failed to create discord session %w", err)
	}
	// Add handler callbacks
	session.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		if h, ok := b.handles[i.ApplicationCommandData().Name]; ok {
			logger := zap.L().With(zap.String("guildID", i.GuildID), zap.String("commandName", i.ApplicationCommandData().Name))
			logger.Debug("Handling command")
			ctx := Context{
				Session:           s,
				InteractionCreate: i,
				userLevel:         9999,
			}
			if i.Member == nil || i.Member.User == nil {
				// Message member doesn't exist
				errorResponse(s, i.Interaction, fmt.Errorf("missing message member"))
				return
			}
			ctx.logger = logger.With(zap.String("userID", i.Member.User.ID))
			if i.Member.User.ID == b.config.RootUserID {
				ctx.logger = ctx.logger.With(zap.Int("userLevel", 0))
				ctx.userLevel = 0
			} else {
				roles, ok := b.guildRoleMap[i.GuildID]
				if !ok {
					// Guild isn't setup correctly
					errorResponse(s, i.Interaction, fmt.Errorf("guild roles not setup"))
					return
				}
				// Get min user level
				for level, roleID := range roles {
					if contains(roleID, i.Member.Roles) {
						ctx.userLevel = level + 1
						break
					}
				}
			}
			h.Handle(ctx)
		} else {
			log.Println("Failed to find command", i.ApplicationCommandData().Name)
		}
	})
	session.AddHandler(func(s *discordgo.Session, g *discordgo.GuildRoleCreate) {
		b.updateGuildRoles(s, g.GuildID)
	})
	session.AddHandler(func(s *discordgo.Session, g *discordgo.GuildRoleDelete) {
		b.updateGuildRoles(s, g.GuildID)
	})
	session.AddHandler(func(s *discordgo.Session, g *discordgo.GuildRoleUpdate) {
		b.updateGuildRoles(s, g.GuildID)
	})
	// Add ready callback
	session.AddHandler(func(s *discordgo.Session, r *discordgo.Ready) {
		log.Printf("Logged in as: %v#%v", s.State.User.Username, s.State.User.Discriminator)
		for _, guild := range r.Guilds {
			b.updateGuildRoles(s, guild.ID)
		}
	})
	// Open session
	if err := session.Open(); err != nil {
		log.Fatalf("Cannot open the session: %v", err)
	}
	defer session.Close()
	errChan := make(chan error, len(b.handles))
	// Register application commands
	for name, v := range b.handles {
		command := v.Command.Command()
		command.Name = name
		if cmd, err := session.ApplicationCommandCreate(session.State.User.ID, b.config.GuildID, command); err != nil {
			return fmt.Errorf("failed to create %s command %w", name, err)
		} else {
			v.id = cmd.ID
		}
		// If the command is an advanced command, start it
		base := reflect.ValueOf(v.Command)
		if base.IsValid() {
			baseInt := base.Interface()
			if advanced, ok := baseInt.(AdvancedCommand); ok {
				go func() {
					err := advanced.Run(ctx, session)
					errChan <- err
				}()
			}
		}
	}
	// Wait for context to be cancelled
	select {
	case <-ctx.Done():
	case err := <-errChan:
		zap.L().Error("Error running command handler", zap.Error(err))
	}
	// Cleanup Session
	for _, v := range b.handles {
		err := session.ApplicationCommandDelete(session.State.User.ID, b.config.GuildID, v.id)
		if err != nil {
			return fmt.Errorf("failed to delete %s command %w", v.Name(), err)
		}
	}
	return nil
}

func contains[T comparable](x T, xs []T) bool {
	for _, y := range xs {
		if x == y {
			return true
		}
	}
	return false
}
