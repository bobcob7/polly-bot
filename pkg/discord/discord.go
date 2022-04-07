package discord

import (
	"context"
	"fmt"
	"log"
	"reflect"

	"github.com/bobcob7/polly/pkg/discord/internal/echo"
	"github.com/bobcob7/polly/pkg/discord/internal/ping"
	"github.com/bwmarrin/discordgo"
	"go.uber.org/zap"
)

type Command interface {
	Name() string
	Command() *discordgo.ApplicationCommand
	Handle(s *discordgo.Session, i *discordgo.InteractionCreate)
}

type AdvancedCommand interface {
	Command
	Run(ctx context.Context, s *discordgo.Session) error
}

type registeredCommand struct {
	Command
	id string
}

func registerHandlers(commands ...Command) map[string]registeredCommand {
	cmds := []Command{
		&ping.Ping{},
		&echo.Echo{},
	}
	cmds = append(cmds, commands...)
	output := make(map[string]registeredCommand, len(cmds))
	for _, cmd := range cmds {
		output[cmd.Name()] = registeredCommand{Command: cmd}
	}
	return output
}

type Bot struct {
	handles map[string]registeredCommand
}

func New(cmds ...Command) *Bot {
	return &Bot{
		handles: registerHandlers(cmds...),
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

func (b *Bot) Run(ctx context.Context, token, guildID string) error {
	if token == "" {
		return fmt.Errorf("missing token")
	}
	// Create Discord session
	session, err := discordgo.New("Bot " + token)
	if err != nil {
		return fmt.Errorf("failed to create discord session %w", err)
	}
	// Add handler callbacks
	session.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		if h, ok := b.handles[i.ApplicationCommandData().Name]; ok {
			zap.L().Debug("Handling command", zap.String("name", i.ApplicationCommandData().Name))
			h.Handle(s, i)
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
