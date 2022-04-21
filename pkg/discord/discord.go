package discord

import (
	"context"
	"fmt"
	"log"
	"reflect"
	"time"

	"github.com/bwmarrin/discordgo"
	"go.uber.org/zap"
)

type Context struct {
	context.Context
	*discordgo.Session
	*discordgo.InteractionCreate
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

type SecureCommand interface {
	Command
	ACLs() []string
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
	rootUserID   string
	handles      map[string]registeredCommand
	roleResolver interface{ Get(string) (string, error) }
}

func New(rootUserID string, cmds ...Command) *Bot {
	return &Bot{
		rootUserID: rootUserID,
		handles:    registerHandlers(cmds...),
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

func (b *Bot) Run(ctx context.Context, token, guildID string) error {
	if token == "" {
		return fmt.Errorf("missing token")
	}
	// Create Discord session
	session, err := discordgo.New("Bot " + token)
	if err != nil {
		return fmt.Errorf("failed to create discord session %w", err)
	}
	// Setup role resolver
	b.roleResolver = &roleCache{
		session: session,
		guildID: guildID,
		ttl:     time.Minute,
	}
	// Add handler callbacks
	session.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		if h, ok := b.handles[i.ApplicationCommandData().Name]; ok {
			zap.L().Debug("Handling command", zap.String("name", i.ApplicationCommandData().Name))
			ctx := Context{
				Session:           s,
				InteractionCreate: i,
			}
			if i.Member == nil {
				// Message member doesn't exist
				errorResponse(s, i.Interaction, fmt.Errorf("missing message member"))
				return
			}
			if i.Member.User.ID != b.rootUserID {
				// Check that ACLs match the user's roles
				roleNames, err := b.roleNames(s, i.Member.Roles)
				if err != nil {
					errorResponse(s, i.Interaction, err)
					return
				}
				if !aclMatch(h.acls, roleNames) {
					// ACLs don't match, abort
					errorResponse(s, i.Interaction, fmt.Errorf("permission denied"))
					return
				}
				zap.L().Info("Authorized user executing command", zap.String("name", i.ApplicationCommandData().Name), zap.String("userID", i.Member.User.ID))
			} else {
				zap.L().Info("Root user executing command", zap.String("name", i.ApplicationCommandData().Name))
			}
			h.Handle(ctx)
		} else {
			log.Println("Failed to find command", i.ApplicationCommandData().Name)
		}
	})
	// Add ready callback
	session.AddHandler(func(s *discordgo.Session, r *discordgo.Ready) {
		log.Printf("Logged in as: %v#%v", s.State.User.Username, s.State.User.Discriminator)
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
		if cmd, err := session.ApplicationCommandCreate(session.State.User.ID, guildID, command); err != nil {
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
			if secured, ok := baseInt.(SecureCommand); ok {
				v.acls = secured.ACLs()
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
		err := session.ApplicationCommandDelete(session.State.User.ID, guildID, v.id)
		if err != nil {
			return fmt.Errorf("failed to delete %s command %w", v.Name(), err)
		}
	}
	return nil
}
