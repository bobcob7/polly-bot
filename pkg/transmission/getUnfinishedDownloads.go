package transmission

import (
	"fmt"
	"strings"

	"github.com/bobcob7/polly/pkg/discord"
	"github.com/bwmarrin/discordgo"
	"go.uber.org/zap"
)

type UnfinishedDownloads struct {
	*Transmission
}

func (p *UnfinishedDownloads) Name() string {
	return "get-unfinished-downloads"
}

func (p *UnfinishedDownloads) Command() *discordgo.ApplicationCommand {
	return &discordgo.ApplicationCommand{
		Name:        p.Name(),
		Description: "Get a list of unfinished downloads",
	}
}

func (p *UnfinishedDownloads) Handle(ctx discord.Context) {
	torrents, err := p.getDownloadingTorrents(ctx)
	if err != nil {
		ctx.InteractionRespond(ctx.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Title:   "Error",
				Content: "Internal server error. Go bother Eric or something.",
			},
		})
		return
	}
	contentList := make([]string, 0, len(torrents))
	for _, torrent := range torrents {
		contentList = append(contentList, fmt.Sprintf("%s: %f", torrent.Name, torrent.PercentDone))
	}
	if err := ctx.InteractionRespond(ctx.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Title:   "Unfinished downloads",
			Content: strings.Join(contentList, "\n"),
		},
	}); err != nil {
		zap.L().Error("Failed to respond to interaction", zap.Error(err))
	}
}
