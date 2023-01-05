package plugins

import (
	"dalian-bot/internal/pkg/core"
	"dalian-bot/internal/pkg/services/data"
	"dalian-bot/internal/pkg/services/discord"
	"errors"
	"fmt"
	"github.com/bwmarrin/discordgo"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"golang.org/x/net/context"
	"net/url"
	"strings"
	"time"
)

type ArchivePlugin struct {
	core.Plugin
	DiscordService *discord.Service
	DataService    *data.Service
	discord.SlashCommand
	discord.IDisrocdHelper
	core.ArgParseUtil
	core.StageUtil
}

func (p *ArchivePlugin) handleSaveSite(i *discordgo.Interaction, optionsMap map[string]*discordgo.ApplicationCommandInteractionDataOption) error {
	// must have a valid url
	if _, err := url.ParseRequestURI(optionsMap["url"].StringValue()); err != nil {
		p.DiscordService.InteractionRespond(i, "You must provide a *valid* url!")
		return nil
	}
	aPo := archivePO{
		GuildID:   i.GuildID,
		ChannelID: i.ChannelID,
		UserID:    i.Member.User.ID,
	}
	// set site
	aPo.Site = optionsMap["url"].StringValue()
	// set tags
	if tagsOption, ok := optionsMap["tags"]; ok {
		ephemeralTags := p.SeparateArgs(tagsOption.StringValue(), p.DiscordService.DiscordAccountConfig.Separator)
		aPo.Tags = ephemeralTags
	}
	if tagsOption, ok := optionsMap["note"]; ok {
		aPo.Note = tagsOption.StringValue()
	}
	// set time
	aPo.setTime(true)
	result := p.insertOneArchivePo(aPo)
	if result.Err() != nil {
		core.Logger.Warnf("Error inserting archive document: %v", result.Err())
		p.DiscordService.InteractionRespond(i, "Internal error inserting! Please contact admin for help.")
		return result.Err()
	}
	p.DiscordService.InteractionRespondEmbed(i, &discordgo.MessageEmbed{
		Title:       "Site saved",
		Description: "The following site has been saved",
		Timestamp:   time.Now().Format(time.RFC3339),
		Color:       discord.EmbedColorNormal,
		Fields: []*discordgo.MessageEmbedField{{
			Name:   "Temp Title", // todo: site title through snapshot or other ways
			Value:  aPo.essentialInfoForEmbed(),
			Inline: false,
		}},
	}, nil)
	return nil
}

func (p *ArchivePlugin) DoNamedInteraction(_ *core.Bot, i *discordgo.InteractionCreate) (err error) {
	if match, name := p.DefaultMatchCommand(i); match {
		switch name {
		case "archive":
			cmdOption := i.ApplicationCommandData().Options[0]
			switch cmdOption.Name {
			case "site":
				cmdOption := cmdOption.Options[0]
				switch cmdOption.Name {
				case "save":
					optionsMap := p.ParseOptionsMap(cmdOption.Options)
					return p.handleSaveSite(i.Interaction, optionsMap)
				}
			}
		}
	}
	return nil
}

func (p *ArchivePlugin) Init(reg *core.ServiceRegistry) error {
	// services
	//discordService is a MUST have. return error if not found.
	if err := reg.FetchService(&p.DiscordService); err != nil {
		return err
	}
	// DataService is also a MUST have. return error if not found.
	if err := reg.FetchService(&p.DataService); err != nil {
		return err
	}
	// core plugin type
	p.Plugin = core.Plugin{
		Name:                 "archive",
		AcceptedTriggerTypes: []core.TriggerType{core.TriggerTypeDiscord},
	}
	// utils
	p.ArgParseUtil = core.ArgParseUtil{}
	p.StageUtil = core.NewStageUtil()

	// discord
	p.SlashCommand = discord.SlashCommand{AppCommandsMap: map[string]*discordgo.ApplicationCommand{}}
	p.AppCommandsMap.RegisterCommand(&discordgo.ApplicationCommand{
		Name:        "archive",
		Description: "archive certain things",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Name:        "site",
				Description: "site-saving commands",
				Type:        discordgo.ApplicationCommandOptionSubCommandGroup,
				Options: []*discordgo.ApplicationCommandOption{
					{
						Name:        "save",
						Type:        discordgo.ApplicationCommandOptionSubCommand,
						Description: "Save the given site.",
						Options: []*discordgo.ApplicationCommandOption{
							{
								Type:        discordgo.ApplicationCommandOptionString,
								Name:        "url",
								Description: "The valid Url to be stored into database.",
								Required:    true,
							},
							{
								Type: discordgo.ApplicationCommandOptionString,
								Name: "tags",
								//late init, replace %s with separator
								Description: fmt.Sprintf("Add tags for this site, separated by default separator."+
									" Current separator:[%s]", p.DiscordService.DiscordAccountConfig.Separator),
								Required: false,
							},
							{
								Type: discordgo.ApplicationCommandOptionString,
								Name: "note",
								//same as above
								Description: fmt.Sprintf("Add note for this site."+
									" Current separator:[%s]", p.DiscordService.DiscordAccountConfig.Separator),
								Required: false,
							},
							{
								Type:        discordgo.ApplicationCommandOptionBoolean,
								Name:        "cache",
								Description: "Cache the given site",
								Required:    false,
							},
						},
					}, {
						Name:        "list",
						Type:        discordgo.ApplicationCommandOptionSubCommand,
						Description: "List all sites",
					},
				},
			},
		},
	})

	// discord helps
	formattedHelpSiteSet := `*archive site save*: /archive site save
Save the given website to dalian database. You will have the option to save a snapshot of it.`
	formattedHelpSiteList := `*archive site list*: /archive site list
List all sites archived by dalian. You can filter with tags.`
	p.IDisrocdHelper = discord.GenerateHelper(discord.HelperConfig{
		PluginHelp: "Archive online resources.",
		CommandHelps: []discord.CommandHelp{
			{
				Name:          "archive site save",
				FormattedHelp: formattedHelpSiteSet,
			},
			{
				Name:          "archive site list",
				FormattedHelp: formattedHelpSiteList,
			},
		},
	})
	return p.DiscordService.RegisterSlashCommand(p)
}

func (p *ArchivePlugin) Trigger(trigger core.Trigger) {

	if !p.AcceptTrigger(trigger.Type) {
		return
	}
	discordEvent := discord.UnboxEvent(trigger)
	switch discordEvent.EventType {
	// only accepting interactionCreate for discord trigers
	case discord.EventTypeInteractionCreate:
		switch discordEvent.InteractionCreate.Type {
		case discordgo.InteractionApplicationCommand:
			// slash command
			p.DoNamedInteraction(trigger.Bot, discordEvent.InteractionCreate)
		default:
			// todo: accept components
			return
		}
	default:
		// does not handle messageCreate or anything like that.
		return
	}
}

type archivePO struct {
	//Display Info
	BsonID primitive.ObjectID `bson:"_id,omitempty"`
	Site   string             `bson:"site"`
	Tags   []string           `bson:"tags"`
	Note   string             `bson:"note"`
	//Retrieved Info
	Title       string `bson:"title"`
	SnapshotURL string `bson:"snapshot_url,omitempty"`
	//Credential Info
	GuildID   string `bson:"guild_id"`
	ChannelID string `bson:"channel_id"`
	UserID    string `bson:"user_id"`
	//Auditing info
	CreatedTime      time.Time `bson:"created_time"`
	LastModifiedTime time.Time `bson:"last_modified_time"`
}

func (ap *archivePO) ToMessageEmbedField(displayID int) *discordgo.MessageEmbedField {
	return &discordgo.MessageEmbedField{
		Name:   fmt.Sprintf("%d. Temporary Title", displayID),
		Value:  ap.essentialInfoForEmbed(),
		Inline: false,
	}
}

func (ap *archivePO) setTime(isCreate bool) {
	currentTime := time.Now()
	if isCreate {
		ap.CreatedTime = currentTime
	}
	ap.LastModifiedTime = currentTime
}

func (ap *archivePO) essentialInfo() string {
	var tags, note string
	if len(ap.Tags) == 0 {
		tags = "*None*"
	} else {
		tags = "[" + strings.Join(ap.Tags, ",") + "]"
	}

	if ap.Note == "" {
		note = "*None*"
	} else {
		note = ap.Note
	}
	essentialInfo := "> Site: %s\r" +
		"> Tags: %s\r" +
		"> Note: %s"
	return fmt.Sprintf(essentialInfo, ap.Site, tags, note)
}

func (ap *archivePO) essentialInfoForEmbed() string {
	var tags, note, optSnapshot string
	if len(ap.Tags) == 0 {
		tags = "*None*"
	} else {
		tags = "[" + strings.Join(ap.Tags, ",") + "]"
	}

	if ap.Note == "" {
		note = "*None*"
	} else {
		note = ap.Note
	}
	if ap.SnapshotURL != "" {
		optSnapshot = fmt.Sprintf("\r[snapshot](%s)", ap.SnapshotURL)
	}
	essentialInfo := "%s\r" +
		"Tags: %s\r" +
		"Note: %s" +
		"%s"
	return fmt.Sprintf(essentialInfo, ap.Site, tags, note, optSnapshot)
}

func (p *ArchivePlugin) getCollection() *mongo.Collection {
	return p.DataService.GetCollection("site_collection")
}

func (p *ArchivePlugin) insertOneArchivePo(po archivePO) data.Result {
	return p.DataService.InsertOne(po, p.getCollection(), context.Background())
}

func (p *ArchivePlugin) findArchivePo(query any) ([]*archivePO, error) {
	var results []*archivePO
	err := p.DataService.Find(results, p.getCollection(), context.Background(), query)
	return results, err
}

func NewArchivePlugin(reg *core.ServiceRegistry) core.INewPlugin {
	var archivePlugin ArchivePlugin
	if err := (&archivePlugin).Init(reg); err != nil && errors.As(err, &core.ErrServiceFetchUnknownService) {
		core.Logger.Panicf("Archive plugin MUST have all required service(s) injected!")
		panic("Archive plugin initialization failed.")
	}
	return &archivePlugin
}
