package transmission

import (
	"context"
	"time"

	"github.com/bwmarrin/discordgo"
	"go.uber.org/zap"
)

type AddDownload struct {
	*Transmission
}

func (p *AddDownload) Name() string {
	return "add-downloads"
}

func (p *AddDownload) Command() *discordgo.ApplicationCommand {
	return &discordgo.ApplicationCommand{
		Name:        p.Name(),
		Description: "Add a new download using it's magnet link",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "magnet-link",
				Description: "Torrent magnet link",
				Required:    true,
			},
		},
	}
}

func (p *AddDownload) Handle(s *discordgo.Session, i *discordgo.InteractionCreate) {
	// Get argument
	magnetLink := i.ApplicationCommandData().Options[0].StringValue()
	ctx, done := context.WithTimeout(context.Background(), time.Second*10)
	defer done()
	zap.L().Info("Downloading torrent", zap.String("magnet-link", magnetLink))

	if err := p.AddLink(ctx, magnetLink); err != nil {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Title:   "Error",
				Content: "Failed to add link. " + err.Error(),
			},
		})
		return
	}
	if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Title:   "Successfully added download",
			Content: "Added link",
		},
	}); err != nil {
		zap.L().Error("Failed to respond to interaction", zap.Error(err))
	}
}
