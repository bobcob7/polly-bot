package discord

import (
	"context"
	"errors"
	"fmt"
	"log"
	"reflect"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/upper/db/v4"
	"go.uber.org/zap"
)

type Config struct {
	Token             string
	GuildID           string `map:"GUILD_ID"`
	PrivateChannelTTL int    `map:"PRIVATE_CHANNEL_TTL"`
}

func (c Config) Valid() (errs []string) {
	if c.Token == "" {
		errs = append(errs, "Discord Token is required")
	}
	if c.PrivateChannelTTL == 0 {
		errs = append(errs, "Private Channel TTL is required")
	}
	return
}

type BaseCommand interface {
	Name() string
	Command() *discordgo.ApplicationCommand
	Handle(ctx Context) error
}

type InitCommand interface {
	BaseCommand
	Run(ctx context.Context, s *discordgo.Session) error
}

type ModalCommand interface {
	BaseCommand
	HasCustomID(id string) bool
	HandleModal(ctx Context, id string) error
}

type registeredCommand struct {
	BaseCommand
	id string
}

var ErrNotFound = errors.New("custom ID not found")

func (b *Bot) registerHandles(commands ...BaseCommand) {
	b.baseHandles = make(map[string]registeredCommand, len(commands))
	for _, rawCommand := range commands {
		base := reflect.ValueOf(rawCommand)
		if !base.IsValid() {
			panic("invalid base handler")
		}
		baseInt := base.Interface()
		if modalCmd, ok := baseInt.(BaseCommand); ok {
			b.baseHandles[modalCmd.Name()] = registeredCommand{
				BaseCommand: modalCmd,
				id:          "",
			}
		}
	}
	b.initHandles = make(map[string]InitCommand, len(commands))
	for _, rawCommand := range commands {
		base := reflect.ValueOf(rawCommand)
		if !base.IsValid() {
			panic("invalid init handler")
		}
		baseInt := base.Interface()
		if modalCmd, ok := baseInt.(InitCommand); ok {
			b.initHandles[modalCmd.Name()] = modalCmd
		}
	}
	b.modalHandles = make(map[string]ModalCommand, len(commands))
	for _, rawCommand := range commands {
		base := reflect.ValueOf(rawCommand)
		if !base.IsValid() {
			panic("invalid modal handler")
		}
		baseInt := base.Interface()
		if modalCmd, ok := baseInt.(ModalCommand); ok {
			b.modalHandles[modalCmd.Name()] = modalCmd
		}
	}
}

type Bot struct {
	config           Config
	privateMessenger PrivateMessenger
	guildRoleMap     map[string][]string
	baseHandles      map[string]registeredCommand
	initHandles      map[string]InitCommand
	modalHandles     map[string]ModalCommand
	onStartHooks     map[string]Starter
}

func New(config Config, sess db.Session, cmds ...BaseCommand) *Bot {
	b := &Bot{
		config: config,
		privateMessenger: PrivateMessenger{
			privateChannelTTL: time.Duration(config.PrivateChannelTTL) * time.Second,
			sess:              sess,
		},
		guildRoleMap: map[string][]string{},
		onStartHooks: make(map[string]Starter),
	}
	b.registerHandles(cmds...)
	return b
}

type Starter interface {
	OnStart(ctx Context, s *discordgo.Session) error
}

func (b *Bot) OnStartHook(name string, hook Starter) {
	b.onStartHooks[name] = hook
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

var errMissingMessageMember = errors.New("missing message member")
var errMissingToken = errors.New("missing token")

func (b *Bot) Run(ctx context.Context) error {
	if b.config.Token == "" {
		return errMissingToken
	}
	// Create Discord session
	session, err := discordgo.New("Bot " + b.config.Token)
	if err != nil {
		return fmt.Errorf("failed to create discord session %w", err)
	}
	// Add handler callbacks
	session.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		switch i.Type {
		case discordgo.InteractionApplicationCommand:
			logger := zap.L().With(zap.String("guildID", i.GuildID), zap.String("commandName", i.ApplicationCommandData().Name))
			if h, ok := b.baseHandles[i.ApplicationCommandData().Name]; ok {
				logger.Info("Handling command")
				handleContext := Context{
					Session:           s,
					InteractionCreate: i,
					PrivateMessenger:  &b.privateMessenger,
				}
				if i.Member == nil || i.Member.User == nil {
					// Message member doesn't exist
					errorResponse(s, i.Interaction, errMissingMessageMember)
					return
				}
				handleContext.logger = logger.With(zap.String("userID", i.Member.User.ID))
				var done context.CancelFunc
				handleContext.Context, done = context.WithTimeout(ctx, time.Second*10)
				defer done()
				func() {
					defer func() {
						if r := recover(); r != nil {
							logger.Error("Recovering from panic", zap.Error(err))
							_ = handleContext.Session.InteractionRespond(handleContext.Interaction, &discordgo.InteractionResponse{
								Type: discordgo.InteractionResponseChannelMessageWithSource,
								Data: &discordgo.InteractionResponseData{
									Flags:   discordgo.MessageFlagsEphemeral,
									Title:   "Panic",
									Content: fmt.Sprintf("Panic while processing command: %s", r),
								},
							})
						}
					}()
					if err := h.Handle(handleContext); err != nil {
						logger.Info("Handler error", zap.Error(err))
						var msg string
						//nolint: errorlint
						if pubErr, ok := err.(interface{ Public() string }); ok {
							msg = pubErr.Public()
						} else {
							msg = err.Error()
						}
						_ = handleContext.Session.InteractionRespond(handleContext.Interaction, &discordgo.InteractionResponse{
							Type: discordgo.InteractionResponseChannelMessageWithSource,
							Data: &discordgo.InteractionResponseData{
								Flags:   discordgo.MessageFlagsEphemeral,
								Title:   "Error",
								Content: msg,
							},
						})
					}
				}()
			} else {
				logger.Error("failed to find command")
			}

		case discordgo.InteractionModalSubmit:
			customID := i.Interaction.ModalSubmitData().CustomID
			logger := zap.L().With(zap.String("guildID", i.GuildID), zap.String("customID", customID))
			var handle ModalCommand
			for _, h := range b.modalHandles {
				if h.HasCustomID(customID) {
					handle = h
					break
				}
			}
			if handle != nil {
				logger.Info("Handling modal submission")
				handleContext := Context{
					Session:           s,
					InteractionCreate: i,
					PrivateMessenger:  &b.privateMessenger,
				}
				if i.Member == nil || i.Member.User == nil {
					// Message member doesn't exist
					errorResponse(s, i.Interaction, errMissingMessageMember)
					return
				}
				handleContext.logger = logger.With(zap.String("userID", i.Member.User.ID))
				var done context.CancelFunc
				handleContext.Context, done = context.WithTimeout(ctx, time.Second*10)
				defer done()
				func() {
					defer func() {
						if r := recover(); r != nil {
							_ = handleContext.Session.InteractionRespond(handleContext.Interaction, &discordgo.InteractionResponse{
								Type: discordgo.InteractionResponseChannelMessageWithSource,
								Data: &discordgo.InteractionResponseData{
									Flags:   discordgo.MessageFlagsEphemeral,
									Title:   "Panic",
									Content: fmt.Sprintf("Panic while processing modal: %s", r),
								},
							})
						}
					}()
					// Get Modal handle
					if err := handle.HandleModal(handleContext, customID); err != nil {
						var msg string
						//nolint: errorlint
						if pubErr, ok := err.(interface{ Public() string }); ok {
							msg = pubErr.Public()
						} else {
							msg = err.Error()
						}
						_ = handleContext.Session.InteractionRespond(handleContext.Interaction, &discordgo.InteractionResponse{
							Type: discordgo.InteractionResponseChannelMessageWithSource,
							Data: &discordgo.InteractionResponseData{
								Flags:   discordgo.MessageFlagsEphemeral,
								Title:   "Error",
								Content: msg,
							},
						})
					}
				}()
			} else {
				logger.Error("failed to find interaction")
			}
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
	errChan := make(chan error, len(b.baseHandles))
	// Register application commands
	for name, v := range b.baseHandles {
		command := v.Command()
		command.Name = name
		cmd, err := session.ApplicationCommandCreate(session.State.User.ID, b.config.GuildID, command)
		if err != nil {
			return fmt.Errorf("failed to create %s command %w", name, err)
		}
		v.id = cmd.ID
		// If the command is an advanced command, start it
		base := reflect.ValueOf(v.BaseCommand)
		if base.IsValid() {
			baseInt := base.Interface()
			if advanced, ok := baseInt.(InitCommand); ok {
				go func() {
					err := advanced.Run(ctx, session)
					errChan <- err
				}()
			}
		}
	}
	// Cleanup Session
	defer func() {
		for _, v := range b.baseHandles {
			err := session.ApplicationCommandDelete(session.State.User.ID, b.config.GuildID, v.id)
			if err != nil {
				zap.L().Error("failed to delete command", zap.String("name", v.Name()), zap.Error(err))
			}
		}
	}()
	// Run onStart hooks
	for name, hook := range b.onStartHooks {
		hookContext := Context{
			Context:          ctx,
			Session:          session,
			PrivateMessenger: &b.privateMessenger,
			logger:           zap.L().With(zap.String("onStartHook", name)),
		}
		if err := hook.OnStart(hookContext, session); err != nil {
			return fmt.Errorf("failed onStart hook %q: %w", name, err)
		}
	}
	// Wait for context to be cancelled
	ticker := time.NewTicker(time.Second * 10)
	defer ticker.Stop()
botLoop:
	for {
		select {
		case <-ctx.Done():
			break botLoop
		case <-ticker.C:
			if err := b.privateMessenger.garbageCollect(ctx, session); err != nil {
				zap.L().Error("Error collecting private messenger garbage", zap.Error(err))
			}
		case err := <-errChan:
			zap.L().Error("Error running command handler", zap.Error(err))
		}
	}
	return nil
}
