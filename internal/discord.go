package internal

import (
	"context"
	"database/sql"
	"math/rand"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/mattn/go-shellwords"
	"go.uber.org/zap"
)

type command struct {
	channelID string
	cmd       string
	args      []string
}

type DiscordController struct {
	dg         *discordgo.Session
	commands   chan command
	controller commander
}

func (d DiscordController) Close() {
	d.dg.Close()
}

func NewDiscordController(token string) (*DiscordController, error) {
	// Create a new Discord session using the provided bot token.
	dg, err := discordgo.New("Bot " + token)
	if err != nil {
		return nil, err
	}
	defer dg.Close()

	d := DiscordController{
		dg:       dg,
		commands: make(chan command, 10),
	}
	dg.AddHandler(d.onMessage)
	// In this example, we only care about receiving message events.
	dg.Identify.Intents = discordgo.IntentsGuildMessages

	// Open a websocket connection to Discord and begin listening.
	err = dg.Open()
	if err != nil {
		return nil, err
	}
	return &d, nil
}

func (d *DiscordController) onMessage(s *discordgo.Session, m *discordgo.MessageCreate) {
	username := m.Message.Author.Username
	// TODO: Check perms

	parts, err := shellwords.Parse(m.Message.Content)
	if err != nil {
		logger.Error("failed to parse shell",
			zap.Error(err))
		return
	}
	if parts[0][0] == '/' {
		newCmd := command{
			channelID: m.ChannelID,
			cmd:       parts[0][1:],
			args:      parts[1:],
		}
		logger.Info("Message received",
			zap.String("username", username),
			zap.String("cmd", newCmd.cmd),
			zap.Strings("args", newCmd.args),
		)
		d.commands <- newCmd
	}
}

var greetings = []string{
	"Fuck off!",
	"What are you still doing here",
	"Can't you see that no body loves you?!",
	"Go away",
	"Stahpit!!!",
	"Why can't you just leave me alone?",
	"Why are you still here?",
	"Don't let the door hit you on the ass",
	"Is $20 enough to get rid of you for an hour?",
}

func randomGreeting() string {
	i := rand.Int() % len(greetings)
	return greetings[i]
}

func (d *DiscordController) processCommand(cmd command) error {
	logger.Info("Processing command")
	keyword := strings.ToLower(cmd.cmd)
	var response string
	switch keyword {
	case "help":
		if len(cmd.args) > 0 {
			switch cmd.args[0] {
			case "agent":
				fallthrough
			case "agents":
				response = d.controller.AgentHelp()
			case "subject":
				fallthrough
			case "subjects":
				response = d.controller.SubjectHelp()
			default:
				response = d.controller.Help()
			}
		} else {
			response = d.controller.Help()
		}
	case "add-agent":
		// Usage: /add-agent [name] [base-url]
		if len(cmd.args) != 2 || (len(cmd.args) > 0 && strings.ToLower(cmd.args[0]) == "help") {
			response = d.controller.CreateAgentHelp()
		} else {
			response = d.controller.CreateAgent(cmd.args[0], cmd.args[1])
		}
	case "delete-agent":
		// Usage: /add-agent [name] [base-url]
		if len(cmd.args) != 1 || (len(cmd.args) > 0 && strings.ToLower(cmd.args[0]) == "help") {
			response = d.controller.DeleteAgentHelp()
		} else {
			response = d.controller.DeleteAgent(cmd.args[0])
		}
	case "list-agents":
		if len(cmd.args) != 0 || (len(cmd.args) > 0 && strings.ToLower(cmd.args[0]) == "help") {
			response = d.controller.ListAgentHelp()
		} else {
			response = d.controller.ListAgent()
		}
	case "add-subject":
		// Usage: /add-subject [name] [agent-name] [search text] (regex)
		if len(cmd.args) != 3 || len(cmd.args) != 4 || (len(cmd.args) > 0 && strings.ToLower(cmd.args[0]) == "help") {
			response = d.controller.CreateSubjectHelp()
		} else {
			response = d.controller.CreateSubject(cmd.args[0], cmd.args[1], cmd.args[2], cmd.args[3])
		}
	case "delete-subject":
		// Usage: /delete-agent [name] [base-url]
		if len(cmd.args) != 1 || (len(cmd.args) > 0 && strings.ToLower(cmd.args[0]) == "help") {
			response = d.controller.DeleteSubjectHelp()
		} else {
			response = d.controller.DeleteSubject(cmd.args[0])
		}
	case "list-subjects":
		if len(cmd.args) != 0 || (len(cmd.args) > 0 && strings.ToLower(cmd.args[0]) == "help") {
			response = d.controller.ListSubjectHelp()
		} else {
			response = d.controller.ListSubject()
		}
	default:
		d.dg.ChannelMessageSend(cmd.channelID, randomGreeting())
	}
	_, err := d.dg.ChannelMessageSend(cmd.channelID, response)
	return err
}

func (d *DiscordController) Run(ctx context.Context, db *sql.DB) {
	d.controller = commander{db}
	for {
		select {
		case <-ctx.Done():
			return
		case cmd := <-d.commands:
			if err := d.processCommand(cmd); err != nil {
				logger.Error("failed to process command",
					zap.Error(err),
				)
			}
		}
	}
}
