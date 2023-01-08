package plugins

import (
	core2 "dalian-bot/internal/core"
	data2 "dalian-bot/internal/services/data"
	discord2 "dalian-bot/internal/services/discord"
	"errors"
	"fmt"
	"github.com/bwmarrin/discordgo"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"golang.org/x/net/context"
	"net/url"
	"strings"
	"time"
)

type ArchivePlugin struct {
	core2.Plugin
	DiscordService *discord2.Service
	DataService    *data2.Service
	discord2.SlashCommand
	discord2.IDisrocdHelper
	core2.ArgParseUtil
	core2.StageUtil
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
		core2.Logger.Warnf("Error inserting archive document: %v", result.Err())
		p.DiscordService.InteractionRespond(i, "Internal error inserting! Please contact admin for help.")
		return result.Err()
	}
	// todo: replace it with actual title saving
	aPo.Title = "Temporary Title"
	p.DiscordService.InteractionRespondEmbed(i, &discordgo.MessageEmbed{
		Title:       "Site saved",
		Description: "The following site has been saved",
		Timestamp:   time.Now().Format(time.RFC3339),
		Color:       discord2.EmbedColorNormal,
		Fields: []*discordgo.MessageEmbedField{{
			Name:   "Temp Title", // todo: site title through snapshot or other ways
			Value:  aPo.essentialInfoForEmbed(),
			Inline: false,
		}},
	}, nil)
	return nil
}

func (p *ArchivePlugin) handleListSite(i *discordgo.Interaction, optionsMap map[string]*discordgo.ApplicationCommandInteractionDataOption) error {
	query := bson.M{"user_id": i.Member.User.ID, "guild_id": i.GuildID}
	//if found optional tags, add it to the query
	//set tags
	if tagsOption, ok := optionsMap["tags"]; ok {
		parsedTags := p.SeparateArgs(tagsOption.StringValue(), p.DiscordService.DiscordAccountConfig.Separator)
		query["tags"] = bson.M{"$all": parsedTags}

	}
	archiveListPager := discord2.Pager{
		IPagerLoader: &archivePoPagerLoader{
			query:     query,
			queryFunc: p.findArchivePo,
		},
		PageNow: 1,
		Limit:   7,
		PrevPageButton: discordgo.Button{
			Label:    discord2.EmojiLeftArrow,
			Style:    discordgo.PrimaryButton,
			CustomID: lsButtonIDPrev,
		},
		NextPageButton: discordgo.Button{
			Label:    discord2.EmojiRightArrow,
			Style:    discordgo.PrimaryButton,
			CustomID: lsButtonIDNext,
		},
		EmbedFrame: &discordgo.MessageEmbed{
			Title:     "ls-site result",
			Color:     discord2.EmbedColorNormal,
			Timestamp: time.Now().Format(time.RFC3339),
		},
		Overtime: time.Duration(5) * time.Minute,
	}
	if err := archiveListPager.Setup(i, p.DiscordService); err != nil {
		core2.Logger.Warnf("Error setup pager: %v", err)
		return err
	}
	// all stages are now saved regardless of length, to support relative-id
	//if archiveListPager.PageMax > 1 {
	//	var stage archiveQueryStage
	//	stage.Init(&archiveListPager, p)
	//}
	var stage archiveQueryStage
	stage.Init(&archiveListPager, p)
	return nil
}

func (p *ArchivePlugin) handleModifySite(i *discordgo.Interaction, optionsMap map[string]*discordgo.ApplicationCommandInteractionDataOption) error {
	tempID, ok := optionsMap["relative-id"]
	if !ok {
		p.DiscordService.InteractionRespond(i, "no relative-ID provided!")
		return nil
	}
	id := int(tempID.IntValue())
	////old logic
	//modifyingPo, err := retrieveSitePoByNumericalID(id.IntValue())
	//new logic
	key := p.findActiveRelativeID(i)
	if key == "" {
		p.DiscordService.InteractionRespond(i, "No active query for you! Run a new query first?")
		return nil
	}
	rawStage, _ := p.StageUtil.GetStage(key)
	aqs := rawStage.(*archiveQueryStage)
	if id <= 0 || id > len(aqs.Pager.CompleteItemSlice) {
		p.DiscordService.InteractionRespond(i, "Malformed relative-ID. Check your last query?")
		return nil
	}
	modifyingPo := (*aqs.Pager.CompleteItemSlice[id-1]).(*archivePO)
	tempTags, ok := optionsMap["tags"]
	if ok {
		tagsStr := tempTags.StringValue()
		if tagsStr == "-" {
			//clean up
			modifyingPo.Tags = []string{}
		} else {
			ephemeralTags := p.SeparateArgs(tagsStr, p.DiscordService.DiscordAccountConfig.Separator)
			modifyingPo.Tags = ephemeralTags
		}
	}
	tempNote, ok := optionsMap["note"]
	if ok {
		noteStr := tempNote.StringValue()
		if noteStr == "-" {
			//clean up
			modifyingPo.Note = ""
		} else {
			modifyingPo.Note = noteStr
		}
	}
	res := p.updateArchivePoWithID(*modifyingPo)
	if res.Err() != nil {
		p.DiscordService.InteractionRespond(i, res.Err().Error())
		return nil
	}
	return p.DiscordService.InteractionRespondEmbed(i, &discordgo.MessageEmbed{
		Title:       "Site record updated",
		Description: "The following site has been updated",
		Timestamp:   time.Now().Format(time.RFC3339),
		Color:       discord2.EmbedColorNormal,
		Fields: []*discordgo.MessageEmbedField{{
			Name:   modifyingPo.Title,
			Value:  modifyingPo.essentialInfoForEmbed(),
			Inline: false,
		}},
	}, nil)
}

func (p *ArchivePlugin) handleRemoveSite(i *discordgo.Interaction, optionsMap map[string]*discordgo.ApplicationCommandInteractionDataOption) error {
	tempID, ok := optionsMap["relative-id"]
	if !ok {
		p.DiscordService.InteractionRespond(i, "no relative-ID provided!")
		return nil
	}
	id := int(tempID.IntValue())

	//new logic
	key := p.findActiveRelativeID(i)
	if key == "" {
		p.DiscordService.InteractionRespond(i, "No active query for you! Run a new query first?")
		return nil
	}
	rawStage, _ := p.StageUtil.GetStage(key)
	aqs := rawStage.(*archiveQueryStage)
	if id <= 0 || id > len(aqs.Pager.CompleteItemSlice) {
		p.DiscordService.InteractionRespond(i, "Malformed relative-ID. Check your last query?")
		return nil
	}
	deletingPo := (*aqs.Pager.CompleteItemSlice[id-1]).(*archivePO)
	delResult := p.deleteArchivePoWithID(*deletingPo)
	if delResult.Err() != nil {
		p.DiscordService.InteractionRespond(i, delResult.Err().Error())
		return nil
	}
	p.DiscordService.InteractionRespondEmbed(i, &discordgo.MessageEmbed{
		Title:       "Site record deleted",
		Description: "The following site has been deleted",
		Timestamp:   time.Now().Format(time.RFC3339),
		Color:       discord2.EmbedColorNormal,
		Fields: []*discordgo.MessageEmbedField{{
			Name:   deletingPo.Title,
			Value:  deletingPo.essentialInfoForEmbed(),
			Inline: false,
		}},
	}, nil)
	return nil
}

func (p *ArchivePlugin) findActiveRelativeID(i *discordgo.Interaction) core2.CombinedKey {
	var key core2.CombinedKey
	var latestTime time.Time
	p.StageUtil.IterThroughStage(func(k core2.CombinedKey, v core2.IStage) bool {

		if aqs, ok := (v).(*archiveQueryStage); ok {
			if aqs.OwnerUserID == i.Member.User.ID && aqs.ChannelID == i.ChannelID {
				if aqs.CreatedTime.After(latestTime) {
					latestTime = aqs.CreatedTime
					key = k
				}
			}
		}
		return false
	})
	return key
}

func (p *ArchivePlugin) DoNamedInteraction(_ *core2.Bot, i *discordgo.InteractionCreate) (err error) {
	if match, name := p.DefaultMatchCommand(i); match {
		switch name {
		case "archive":
			cmdOption := i.ApplicationCommandData().Options[0]
			switch cmdOption.Name {
			case "site":
				cmdOption := cmdOption.Options[0]
				optionsMap := p.ParseOptionsMap(cmdOption.Options)
				switch cmdOption.Name {
				case "save":
					return p.handleSaveSite(i.Interaction, optionsMap)
				case "list":
					return p.handleListSite(i.Interaction, optionsMap)
				case "modify":
					return p.handleModifySite(i.Interaction, optionsMap)
				case "remove":
					return p.handleRemoveSite(i.Interaction, optionsMap)
				}
			}
		}
	}
	return nil
}

func (p *ArchivePlugin) Init(reg *core2.ServiceRegistry) error {
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
	p.Plugin = core2.Plugin{
		Name:                 "archive",
		AcceptedTriggerTypes: []core2.TriggerType{core2.TriggerTypeDiscord},
	}
	// utils
	p.ArgParseUtil = core2.ArgParseUtil{}
	p.StageUtil = core2.NewStageUtil()

	// discord
	p.SlashCommand = discord2.SlashCommand{AppCommandsMap: map[string]*discordgo.ApplicationCommand{}}
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
						Options: []*discordgo.ApplicationCommandOption{
							{
								Type: discordgo.ApplicationCommandOptionString,
								Name: "tags",
								//late init, replace %s with separator
								Description: fmt.Sprintf("Search tags for this site, separated by default separator."+
									" Current separator:[%s]", p.DiscordService.DiscordAccountConfig.Separator),
								Required: false,
							},
						},
					},
					{
						Name:        "modify",
						Type:        discordgo.ApplicationCommandOptionSubCommand,
						Description: "Modify the given site.",
						Options: []*discordgo.ApplicationCommandOption{
							{
								Type:        discordgo.ApplicationCommandOptionInteger,
								Name:        "relative-id",
								Description: "The relative id of item in the last query",
								Required:    true,
							},
							{
								Type:        discordgo.ApplicationCommandOptionString,
								Name:        "url",
								Description: "The valid Url to be stored into database.",
								Required:    false,
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
								Description: "Re-cache the given site",
								Required:    false,
							},
						},
					}, {
						Name:        "remove",
						Type:        discordgo.ApplicationCommandOptionSubCommand,
						Description: "Remove the given site.",
						Options: []*discordgo.ApplicationCommandOption{
							{
								Type:        discordgo.ApplicationCommandOptionInteger,
								Name:        "relative-id",
								Description: "The relative id of item in the last query",
								Required:    true,
							},
							{
								Type:        discordgo.ApplicationCommandOptionBoolean,
								Name:        "cache",
								Description: "Should cache be deleted, if present",
								Required:    false,
							},
						},
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
	formattedHelpSiteModify := `*archive site modify*: /archive site modify
Modify a site archived by dalian.
You MUST first run a query with *archive site list* to get an active relative-ID for the site`
	formattedHelpSiteDelete := `*archive site delete*: /archive site delete
Delete a site archived by dalian.
You MUST first run a query with *archive site list* to get an active relative-ID for the site`

	p.IDisrocdHelper = discord2.GenerateHelper(discord2.HelperConfig{
		PluginHelp: "Archive online resources.",
		CommandHelps: []discord2.CommandHelp{
			{
				Name:          "archive site save",
				FormattedHelp: formattedHelpSiteSet,
			},
			{
				Name:          "archive site list",
				FormattedHelp: formattedHelpSiteList,
			},
			{
				Name:          "archive site modify",
				FormattedHelp: formattedHelpSiteModify,
			},
			{
				Name:          "archive site delete",
				FormattedHelp: formattedHelpSiteDelete,
			},
		},
	})
	return p.DiscordService.RegisterSlashCommand(p)
}

func (p *ArchivePlugin) Trigger(trigger core2.Trigger) {

	if !p.AcceptTrigger(trigger.Type) {
		return
	}
	discordEvent := discord2.UnboxEvent(trigger)
	switch discordEvent.EventType {
	// only accepting interactionCreate for discord trigers
	case discord2.EventTypeInteractionCreate:
		switch discordEvent.InteractionCreate.Type {
		case discordgo.InteractionApplicationCommand:
			// slash command
			if err := p.DoNamedInteraction(trigger.Bot, discordEvent.InteractionCreate); err != nil {
				core2.Logger.Warnf("Error executing slash command: %v", err)
			}
		case discordgo.InteractionMessageComponent:
			// message component (pager)
			if stage, ok := p.StageUtil.GetStage(core2.CombinedKeyFromRaw(discordEvent.InteractionCreate.Message.ID)); ok {
				stage.Process(discordEvent.InteractionCreate.Interaction)
			}
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

func (p *ArchivePlugin) insertOneArchivePo(po archivePO) data2.Result {
	return p.DataService.InsertOne(po, p.getCollection(), context.Background())
}

func (p *ArchivePlugin) findArchivePo(query any) ([]*archivePO, error) {
	var results []*archivePO
	err := p.DataService.Find(&results, p.getCollection(), context.Background(), query)
	return results, err
}

func (p *ArchivePlugin) updateArchivePoWithID(po archivePO) data2.Result {
	return p.DataService.UpdateByID(bson.D{{"$set", data2.ToBsonDocForce(po)}}, po.BsonID, p.getCollection(), context.Background())
}

func (p *ArchivePlugin) deleteArchivePoWithID(po archivePO) data2.Result {
	return p.DataService.DeleteOne(p.getCollection(), context.Background(), bson.M{"_id": po.BsonID})
}

type archiveQueryStage struct {
	*discord2.Pager
	UserID      string
	ChannelID   string
	GuildID     string
	CreatedTime time.Time
	triggerChan chan *discordgo.Interaction
	plugin      *ArchivePlugin
}

func (a *archiveQueryStage) Process(t any) {
	a.triggerChan <- t.(*discordgo.Interaction)
}

func (a *archiveQueryStage) Init(pager *discord2.Pager, plugin *ArchivePlugin) {
	a.Pager = pager
	a.UserID = pager.OwnerUserID
	a.ChannelID = pager.AttachedMessage.ChannelID
	a.GuildID = pager.AttachedMessage.GuildID
	a.CreatedTime = time.Now()
	a.triggerChan = make(chan *discordgo.Interaction, 1)
	a.plugin = plugin
	key := core2.CombinedKeyFromRaw(pager.AttachedMessage.ID)
	go func() {
		plugin.StageUtil.StoreStage(key, a)
		func() {
			for {
				select {
				case interaction, ok := <-a.triggerChan:
					if !ok {
						fmt.Println("Aborted")
						return
					}
					switch interaction.MessageComponentData().CustomID {
					case lsButtonIDPrev:
						a.Pager.SwitchPage(core2.PagerPrevPage, interaction)
					case lsButtonIDNext:
						a.Pager.SwitchPage(core2.PagerNextPage, interaction)
					default:
						core2.Logger.Warnf("Unknown customID: %n" + interaction.MessageComponentData().CustomID)
						return
					}
				case <-time.After(a.Pager.Overtime):
					//overtime termination sign
					fmt.Println("terminating through overtime")
					return
				}
			}
		}()
		a.Pager.LockPagerButtons()
		plugin.StageUtil.DeleteStage(key)
	}()
}

const (
	lsButtonIDPrev = "ls-archive-prev"
	lsButtonIDNext = "ls-archive-next"
)

type archivePoPagerLoader struct {
	query          any
	queryFunc      func(query any) ([]*archivePO, error)
	resultsStorage []*archivePO
	discord2.DefaultPageRenderer
}

func (s *archivePoPagerLoader) LoadPager(pager *discord2.Pager) error {
	var err error
	s.resultsStorage, err = s.queryFunc(s.query)
	if err != nil {
		return err
	}
	for _, v := range s.resultsStorage {
		var tempVar discord2.IPagerPart
		tempVar = v
		pager.CompleteItemSlice = append(pager.CompleteItemSlice, &tempVar)
	}
	return nil
}

func NewArchivePlugin(reg *core2.ServiceRegistry) core2.IPlugin {
	var archivePlugin ArchivePlugin
	if err := (&archivePlugin).Init(reg); err != nil && errors.As(err, &core2.ErrServiceFetchUnknownService) {
		core2.Logger.Panicf("Archive plugin MUST have all required service(s) injected!")
		panic("Archive plugin initialization failed.")
	}
	return &archivePlugin
}
