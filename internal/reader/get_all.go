package reader

import (
	"fmt"
	"strings"

	"github.com/bobcob7/polly-bot/internal/models"
	"github.com/bobcob7/polly-bot/pkg/discord"
	"github.com/bwmarrin/discordgo"
	"github.com/upper/db/v4"
)

type GetAllCommand struct {
	sess db.Session
}

func NewGetAllCommand(sess db.Session) *GetAllCommand {
	return &GetAllCommand{
		sess: sess,
	}
}

func (p *GetAllCommand) Name() string {
	return "get-all"
}

func (p *GetAllCommand) Command() *discordgo.ApplicationCommand {
	return &discordgo.ApplicationCommand{
		Name:        p.Name(),
		Description: "Gets all torrents",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionBoolean,
				Name:        "finished",
				Description: "Only get finished torrents",
				Required:    false,
			},
		},
	}
}

func (p *GetAllCommand) Handle(ctx discord.Context) error {
	// Get finished input
	args := make([]interface{}, 0)
	if len(ctx.Interaction.ApplicationCommandData().Options) != 0 {
		finished := ctx.Interaction.ApplicationCommandData().Options[0].BoolValue()
		if finished {
			args = append(args, "completed_at IS NOT NULL")
		} else {
			args = append(args, "completed_at IS NULL")
		}
	}

	torrents, err := models.GetTorrents(ctx, p.sess, args...)
	if err != nil {
		return err
	}
	title := fmt.Sprintf("Found %d torrents", len(torrents))
	content := make([]string, 0, len(torrents))

	for _, torrent := range torrents {
		content = append(content, torrent.String())
	}
	if len(content) == 0 {
		content = append(content, "No torrents found")
	}

	return ctx.Session.InteractionRespond(ctx.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Title:   title,
			Flags:   discordgo.MessageFlagsEphemeral,
			Content: strings.Join(content, "\n"),
		},
	})
}
