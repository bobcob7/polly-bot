package reader

import (
	"errors"
	"fmt"
	"strings"

	"github.com/bobcob7/polly-bot/internal/models"
	"github.com/bobcob7/polly-bot/internal/torrent"
	"github.com/bobcob7/polly-bot/pkg/discord"
	"github.com/bobcob7/transmission-rpc"
	"github.com/bwmarrin/discordgo"
	"github.com/upper/db/v4"
	"go.uber.org/zap"
)

type AddCommand struct {
	sess      db.Session
	tx        *transmission.Client
	customIDs map[string]struct{}
}

func NewAddCommand(sess db.Session, tx *transmission.Client) *AddCommand {
	return &AddCommand{
		sess:      sess,
		tx:        tx,
		customIDs: make(map[string]struct{}),
	}
}

func (p *AddCommand) Name() string {
	return "add-torrent"
}

func (p *AddCommand) Command() *discordgo.ApplicationCommand {
	return &discordgo.ApplicationCommand{
		Name:        p.Name(),
		Description: "Add a new torrent",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "magnet",
				Description: "Magnet link to add",
				Required:    true,
			},
		},
	}
}

func (p *AddCommand) HasCustomID(customID string) bool {
	_, ok := p.customIDs[customID]
	return ok
}

func selectOptions(options []string) []discordgo.SelectMenuOption {
	output := make([]discordgo.SelectMenuOption, 0, len(options))
	for _, category := range options {
		output = append(output,
			discordgo.SelectMenuOption{
				Label:       category,
				Value:       category,
				Description: category,
				// Emoji: discordgo.ComponentEmoji{
				// 	Name: "🟨",
				// },
			},
		)
	}
	return output
}

var validCategories = []string{
	"MOVIE",
	"TV SHOW",
	"MUSIC",
	"AUDIOBOOK",
	"SOFTWARE",
}

func (p *AddCommand) Handle(ctx discord.Context) error {
	// Get finished input
	magnetURI := ctx.Interaction.ApplicationCommandData().Options[0].StringValue()
	// Get display name from URI
	displayName, err := torrent.MagnetURIDisplayName(magnetURI)
	if err != nil {
		return errors.New("Failed parsing magnet link")
	}
	logger := ctx.Logger().With(zap.String("displayName", displayName))

	customID := ctx.Interaction.ID
	p.customIDs[customID] = struct{}{}
	logger.Debug("sending interaction")

	if len(displayName) > 100 {
		displayName = displayName[:99]
	}

	return ctx.Session.InteractionRespond(ctx.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseModal,
		Data: &discordgo.InteractionResponseData{
			Title:    "Add torrent dialog",
			CustomID: customID,
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.TextInput{
							CustomID:  "name",
							Label:     "Name",
							Style:     discordgo.TextInputShort,
							Value:     displayName,
							Required:  true,
							MaxLength: 100,
							MinLength: 5,
						},
					},
				},
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.TextInput{
							CustomID:    "category",
							Label:       "Category",
							Placeholder: `"Movie", "TV Show", "Music", "Audiobook", "Book", "Software"`,
							Style:       discordgo.TextInputShort,
							Required:    false,
						},
					},
				},
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.TextInput{
							CustomID:  "link",
							Label:     "Link",
							Style:     discordgo.TextInputShort,
							Value:     magnetURI,
							Required:  true,
							MinLength: 5,
						},
					},
				},
			},
		},
	})
}

func (p *AddCommand) HandleModal(ctx discord.Context, id string) error {
	if _, ok := p.customIDs[id]; !ok {
		return discord.NotFoundError
	}
	defer delete(p.customIDs, id)
	// Add torent with link and friendly name
	data := ctx.Interaction.ModalSubmitData()
	logger := ctx.Logger()
	name := data.Components[0].(*discordgo.ActionsRow).Components[0].(*discordgo.TextInput).Value
	rawCategory := data.Components[1].(*discordgo.ActionsRow).Components[0].(*discordgo.TextInput).Value
	link := data.Components[2].(*discordgo.ActionsRow).Components[0].(*discordgo.TextInput).Value

	meta := &models.TorrentMetadata{
		FriendlyName: name,
	}
	// Validate that categories are correct
	var opts transmission.AddMagnetLinkOption
	if rawCategory != "" {
		var found bool
		category := strings.ToUpper(strings.TrimSpace(rawCategory))
		for _, knownCategory := range validCategories {
			if category == knownCategory {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("Unknown category: %q", rawCategory)
		}
		opts = transmission.DownloadSubDirOption(strings.ToLower(category))
		meta.Categories = []string{category}
	}
	// Adding magnet link
	torrentID, err := p.tx.AddMagnetLink(ctx, link, opts)
	if err != nil {
		return err
	}
	// Scrape new torrent
	torrents, err := p.tx.GetTorrents(ctx, torrentID)
	if err != nil {
		return err
	}
	logger.Debug("scraped torrent from transmission", zap.Int("id", torrentID))
	if len(torrents) != 1 {
		return fmt.Errorf("Scraped %d torrents intead of 1", len(torrents))
	}
	newTorrent := models.FromTransmission(torrents[0])
	newTorrent.TorrentMetadata = meta
	if err := newTorrent.Set(ctx, p.sess); err != nil {
		return err
	}
	return ctx.InteractionRespond(ctx.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "Thank you sharing",
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
}
