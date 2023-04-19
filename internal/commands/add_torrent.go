package commands

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

var validCategories = []string{
	"MOVIE",
	"TV SHOW",
	"MUSIC",
	"AUDIOBOOK",
	"SOFTWARE",
}

var errInvalidMagnetLink = errors.New("invalid magnet link")

func (p *AddCommand) Handle(ctx discord.Context) error {
	// Get finished input
	magnetURI := ctx.Interaction.ApplicationCommandData().Options[0].StringValue()
	// Get display name from URI
	displayName, err := torrent.MagnetURIDisplayName(magnetURI)
	if err != nil {
		return errInvalidMagnetLink
	}
	logger := ctx.Logger().With(zap.String("displayName", displayName))

	customID := ctx.Interaction.ID
	p.customIDs[customID] = struct{}{}
	logger.Debug("sending interaction")

	if len(displayName) > 100 {
		displayName = displayName[:99]
	}

	if err := ctx.Session.InteractionRespond(ctx.Interaction, &discordgo.InteractionResponse{
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
	}); err != nil {
		return failedResponseInteractionError{err}
	}
	return nil
}

func (p *AddCommand) HandleModal(ctx discord.Context, id string) error {
	if _, ok := p.customIDs[id]; !ok {
		return discord.ErrNotFound
	}
	defer delete(p.customIDs, id)
	// Add torent with link and friendly name
	data := ctx.Interaction.ModalSubmitData()
	logger := ctx.Logger()
	nameInput, ok := data.Components[0].(*discordgo.ActionsRow).Components[0].(*discordgo.TextInput)
	if !ok {
		return errFailedTypeAssertion
	}
	name := nameInput.Value
	rawCategoryInput, ok := data.Components[1].(*discordgo.ActionsRow).Components[0].(*discordgo.TextInput)
	if !ok {
		return errFailedTypeAssertion
	}
	rawCategory := rawCategoryInput.Value
	linkInput, ok := data.Components[2].(*discordgo.ActionsRow).Components[0].(*discordgo.TextInput)
	if !ok {
		return errFailedTypeAssertion
	}
	link := linkInput.Value

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
			return unexpectedCategoryError{rawCategory}
		}
		opts = transmission.DownloadSubDirOption(strings.ToLower(category))
		meta.Categories = []string{category}
	}
	// Adding magnet link
	torrentID, err := p.tx.AddMagnetLink(ctx, link, opts)
	if err != nil {
		return fmt.Errorf("failed to add magnet link: %w", err)
	}
	// Scrape new torrent
	torrents, err := p.tx.GetTorrents(ctx, torrentID)
	if err != nil {
		return fmt.Errorf("failed to get torrents from db: %w", err)
	}
	logger.Debug("scraped torrent from transmission", zap.Int("id", torrentID))
	if len(torrents) != 1 {
		return unexpectedNumberOfTorrentsError{
			want: 1,
			got:  len(torrents),
		}
	}
	newTorrent := models.FromTransmission(torrents[0])
	newTorrent.TorrentMetadata = meta
	if err := newTorrent.Set(ctx, p.sess); err != nil {
		return fmt.Errorf("failed to set in db: %w", err)
	}
	if err := ctx.InteractionRespond(ctx.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "Thank you sharing",
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	}); err != nil {
		return failedResponseInteractionError{err}
	}
	return nil
}
