package instances

import (
	"dalian-bot/internal/pkg/clients"
	"dalian-bot/internal/pkg/commands"
	"dalian-bot/internal/pkg/services/data"
	discord2 "dalian-bot/internal/pkg/services/discord"
	"fmt"
	"github.com/bwmarrin/discordgo"
	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"golang.org/x/net/context"
	"net/url"
	"regexp"
	"strings"
	"time"
)

// websiteRegex: from https://mathiasbynens.be/demo/url-regex, @diegoperini
var websiteRegex = "^(?:https?://)?(?:[^/.\\s]+\\.)*[^/.\\s]+/?$"
var stageOvertime = 30

type SaveThisSiteCommand struct {
	//basic function
	commands.Command
	//manual way of trigger
	commands.PlainCommand
	//multiple args required for manual when necessary
	commands.ArgCommand
	//flag support for manual trigger
	commands.FlagCommand
	//implicit way to trigger
	commands.RegexTextCommand
	//stepped support for implicit trigger
	commands.BotCallingCommand
	//Map containing active implicit collecting process
	ActiveSaveSitetageMap saveSitestageMap
	//Map containing active ls-site process
	ActiveListSiteStageMap listSitestageMap
	//Slash Command support
	commands.SlashCommand
	//Component support for page rendering //todo: fill the action
	commands.ComponentCommand
}

func (cm *SaveThisSiteCommand) DoComponent(i *discordgo.InteractionCreate) error {
	//find the customID, do it.
	compFunc := cm.CompActionMap[i.MessageComponentData().CustomID]
	compFunc(i)
	return nil
}

const (
	lsButtonIDPrev = "ls-list_site-prev"
	lsButtonIDNext = "ls-list_site-next"
)

func (cm *SaveThisSiteCommand) MatchNamedInteraction(i *discordgo.InteractionCreate) (isMatched bool) {
	status, _ := cm.DefaultMatchCommand(i)
	return status
}

func (cm *SaveThisSiteCommand) DoNamedInteraction(i *discordgo.InteractionCreate) (err error) {
	switch _, interactionName := cm.DefaultMatchCommand(i); interactionName {
	/* Save-site */
	case "save-site":
		optionsMap := cm.ParseOptionsMap(i.ApplicationCommandData().Options)
		if _, err := url.ParseRequestURI(optionsMap["url"].StringValue()); err != nil {
			discord2.InteractionRespond(i.Interaction, "You must provide a *valid* url!")
			return nil
		}
		//validation passed, start the logic
		sitePO := newRawSitePoFromInteraction(i.Interaction)
		//set site
		sitePO.Site = optionsMap["url"].StringValue()
		//set tags
		if tagsOption, ok := optionsMap["tags"]; ok {
			ephemeralTags, _ := cm.SeparateArgs(tagsOption.StringValue(), commands.Separator)
			sitePO.Tags = ephemeralTags
		}
		if tagsOption, ok := optionsMap["note"]; ok {
			sitePO.Note = tagsOption.StringValue()
		}
		//save it to the database
		go insertSitePo(sitePO, true)
		// discord.ChannelMessageSend(m.ChannelID, fmt.Sprintf("Site saved:%s", sitePO.essentialInfo()))
		//TODO: snapshot things.
		discord2.InteractionRespondEmbed(i.Interaction, &discordgo.MessageEmbed{
			Title:       "Site saved",
			Description: "The following site has been saved",
			Timestamp:   time.Now().Format(time.RFC3339),
			Color:       discord2.EmbedColorNormal,
			Fields: []*discordgo.MessageEmbedField{{
				Name:   "Temp Title",
				Value:  sitePO.essentialInfoForEmbed(),
				Inline: false,
			}},
		}, nil)
		return nil
	case "list-site":
		optionsMap := cm.ParseOptionsMap(i.ApplicationCommandData().Options)
		query := bson.M{"user_id": i.Member.User.ID, "guild_id": i.GuildID}
		//if found optional tags, add it to the query
		//set tags
		if tagsOption, ok := optionsMap["tags"]; ok {
			parsedTags, _ := cm.SeparateArgs(tagsOption.StringValue(), commands.Separator)
			query["tags"] = bson.M{"$all": parsedTags}

		}
		findOpts := options.Find().SetSort(bson.D{{"id", 1}})
		siteCollectionPager := commands.Pager{
			IPagerLoader: &sitePoPagerLoader{
				context:        context.Background(),
				query:          query,
				queryOptions:   []*options.FindOptions{findOpts},
				resultsStorage: []*SitePO{},
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
		siteCollectionPager.Setup(i.Interaction)
		if siteCollectionPager.PageMax > 1 {
			cm.ActiveListSiteStageMap.insertListSiteStage(i.Member.User.ID, siteCollectionPager, cm)
		}
	case "modify-site":
		//todo: finish this
		optionsMap := cm.ParseOptionsMap(i.ApplicationCommandData().Options)
		id, ok := optionsMap["id"]
		if !ok {
			discord2.InteractionRespond(i.Interaction, "no ID provided!")
			return nil
		}
		modifyingPo, err := retrieveSitePoByNumericalID(id.IntValue())
		if err != nil {
			if err == mongo.ErrNoDocuments {
				discord2.InteractionRespond(i.Interaction, "ID doesn't exist!")
				return nil
			}
			discord2.InteractionRespond(i.Interaction, err.Error())
			return nil
		}
		if modifyingPo.UserID != i.Member.User.ID {
			discord2.InteractionRespond(i.Interaction, "Sorry, this document belongs to another user.")
			return nil
		}
		tags, ok := optionsMap["tags"]
		if ok {
			tagsStr := tags.StringValue()
			if tagsStr == "-" {
				//clean up
				modifyingPo.Tags = []string{}
			} else {
				ephemeralTags, _ := cm.SeparateArgs(tagsStr, commands.Separator)
				modifyingPo.Tags = ephemeralTags
			}
		}
		note, ok := optionsMap["note"]
		if ok {
			noteStr := note.StringValue()
			if noteStr == "-" {
				//clean up
				modifyingPo.Note = ""
			} else {
				modifyingPo.Note = noteStr
			}
		}
		_, err = updateSitePo(*modifyingPo)
		if err != nil {
			discord2.InteractionRespond(i.Interaction, err.Error())
			return nil
		}
		discord2.InteractionRespondEmbed(i.Interaction, &discordgo.MessageEmbed{
			Title:       "Site record updated",
			Description: "The following site has been updated",
			Timestamp:   time.Now().Format(time.RFC3339),
			Color:       discord2.EmbedColorNormal,
			Fields: []*discordgo.MessageEmbedField{{
				Name:   "Temp title",
				Value:  modifyingPo.essentialInfoForEmbed(),
				Inline: false,
			}},
		}, nil)

	case "update-site-snapshot":
		opts := i.ApplicationCommandData().Options
		switch opts[0].Name {
		case "refresh-snapshot":
			//optionsMap := cm.ParseOptionsMap(opts[0].Options)
			//id, ok := optionsMap["id"]
			//if !ok {
			//	discord.InteractionRespond(i.Interaction, "no ID provided!")
			//	return nil
			//}
		case "update-snapshot-url":
			optionsMap := cm.ParseOptionsMap(opts[0].Options)
			id, ok := optionsMap["id"]
			if !ok {
				discord2.InteractionRespond(i.Interaction, "no ID provided!")
				return nil
			}
			snapshotUrl, ok := optionsMap["snapshot-url"]
			if !ok {
				discord2.InteractionRespond(i.Interaction, "no snapshot url provided!")
				return nil
			}
			if _, err := url.ParseRequestURI(snapshotUrl.StringValue()); err != nil {
				discord2.InteractionRespond(i.Interaction, "invalid snapshot url!")
				return nil
			}
			retrievedSitePo, err := retrieveSitePoByNumericalID(id.IntValue())
			if err != nil {
				discord2.InteractionRespond(i.Interaction, err.Error())
				return nil
			}
			if retrievedSitePo.UserID != i.Member.User.ID {
				discord2.InteractionRespond(i.Interaction, fmt.Sprintf("Sorry, this document belongs to another user."))
				return nil
			}
			retrievedSitePo.SnapshotURL = snapshotUrl.StringValue()
			_, err = updateSitePo(*retrievedSitePo)
			if err != nil {
				discord2.InteractionRespond(i.Interaction, err.Error())
				return nil
			}
			discord2.InteractionRespondEmbed(i.Interaction, &discordgo.MessageEmbed{
				Title:       "Site url updated",
				Description: "The following site has been updated",
				Timestamp:   time.Now().Format(time.RFC3339),
				Color:       discord2.EmbedColorNormal,
				Fields: []*discordgo.MessageEmbedField{{
					Name:   "Temp title",
					Value:  retrievedSitePo.essentialInfoForEmbed(),
					Inline: false,
				}},
			}, nil)
		case "upload-snapshot-attachment":
		}
	}

	return nil
}

func (cm *SaveThisSiteCommand) MatchMessage(m *discordgo.MessageCreate) (bool, bool) {
	//manual
	if matchStatus, _ := cm.MatchText(m.Content); matchStatus {
		return true, true
	}
	//stage progress
	if isCallingBot, _ := cm.IsCallingBot(m.Content); isCallingBot {
		//a stage present, check if it's a stage info
		if _, ok := cm.ActiveSaveSitetageMap[newSaveStageKeyFromMs(*m.Message)]; ok {
			return true, true
		}

	}
	//implicit
	if _, err := url.ParseRequestURI(m.Content); err == nil {
		//go through active stages to make sure no other in process
		if _, ok := cm.ActiveSaveSitetageMap[newSaveStageKeyFromMs(*m.Message)]; ok {
			discord2.ChannelMessageSend(m.ChannelID, "Found an active stage, please finish that one first.")
			return false, true
		}
		return true, true
	}
	return false, true
}

type saveSitestageMap map[commands.CombinedKey]*saveSiteStage

func (m saveSitestageMap) insertSaveSiteStage(ms *discordgo.MessageCreate, cm *SaveThisSiteCommand) error {
	key := newSaveStageKeyFromMs(*ms.Message)
	if stage, ok := m[key]; ok {
		return fmt.Errorf("found an active ask session at stage %d", stage.StageNow)
	}

	stage := newSaveSitestage(ms, cm)
	stage.ProcessMsgChan = make(chan *discordgo.Message, 1)
	stage.URL = ms.Content
	m[key] = &stage
	go stage.process()
	return nil
}

func (m saveSitestageMap) disposeSaveSiteStage(key commands.CombinedKey) error {
	if v, ok := m[key]; !ok {
		return fmt.Errorf("disposing non-exist sitestage w/ id: %s", key)
	} else {
		close(v.ProcessMsgChan) // this should immediately trigger dispose
	}

	delete(m, key)
	return nil
}

func newSaveStageKeyFromRaw(channelID, userID string) commands.CombinedKey {
	return commands.CombinedKeyFromRaw(channelID, userID)
}

func newSaveStageKeyFromMs(ms discordgo.Message) commands.CombinedKey {
	return newSaveStageKeyFromRaw(ms.ChannelID, ms.Author.ID)
}

func newSaveSitestage(ms *discordgo.MessageCreate, cm *SaveThisSiteCommand) saveSiteStage {
	stage := saveSiteStage{
		BasicStageInfo: commands.BasicStageInfo{
			ChannelID:      ms.ChannelID,
			UserID:         ms.Author.ID,
			StageNow:       0,
			LastActionTime: time.Now(),
		},
		ProcessMsgChan: nil,
		MainCommand:    cm,
	}
	return stage
}

type saveSiteStage struct {
	commands.BasicStageInfo
	URL            string
	ProcessMsgChan chan *discordgo.Message
	MainCommand    *SaveThisSiteCommand
}

func (s *saveSiteStage) process() {
	//prep
	questionPrompt := "Detected the following url:\r" +
		" %s\r" +
		"DoMessage you wish to add it to SiteCollection? (y/yes/n/no)\r" +
		"all answers should start with **@%s**, expires in %d seconds"
	promptEmbed := &discordgo.MessageEmbed{
		Description: fmt.Sprintf(questionPrompt, s.URL, clients.DgSession.State.User.Username, stageOvertime),
		Color:       discord2.EmbedColorQuestion,
	}

	//start the prompt
	discord2.ChannelMessageSendEmbed(s.ChannelID, promptEmbed)
	var sitePo SitePO
	func() {
		for {
			select {
			case msg, ok := <-s.ProcessMsgChan:
				if !ok {
					//channel closed, a termination sign
					//debug
					fmt.Println("terminating through closed channel")
					return
				}
				//message in the channel must be calling but.
				_, content := s.MainCommand.IsCallingBot(msg.Content)
				switch s.StageNow {
				case 0:
					if content == "y" || content == "yes" {
						sitePo = newRawSitePOFromMessage(msg)
						sitePo.Site = s.URL
						prompt := "[1/2] Add tags for this site, separated by default separator, type \"-\" to leave it blank.\r" +
							"Current separator:[%s]"
						discord2.ChannelMessageSend(s.ChannelID, fmt.Sprintf(prompt, commands.Separator))
						s.StageNow += 1
					}
					if content == "n" || content == "no" {
						discord2.ChannelMessageSend(s.ChannelID, "Site saving cancelled.")
						return
					}
				case 1:
					tags, count := s.MainCommand.SeparateArgs(content, commands.Separator)
					prompt := "[2/2] Add note for this site,type \"-\" to leave it blank."
					if count == 0 {
						discord2.ChannelMessageSend(s.ChannelID, "Add at least one tag, or use \"-\" to leave the field blank.")
					} else if count == 1 && tags[0] == "-" {
						//no tags
						discord2.ChannelMessageSend(s.ChannelID, prompt)
						s.StageNow += 1
					} else {
						sitePo.Tags = tags
						discord2.ChannelMessageSend(s.ChannelID, prompt)
						s.StageNow += 1
					}
				case 2:
					if content != "-" {
						sitePo.Note = content
					}
					//TODO snapshot things
					insertSitePo(sitePo, true)
					discord2.ChannelMessageSendEmbed(msg.ChannelID, &discordgo.MessageEmbed{
						Title:       "Site saved",
						Description: "The following site has been saved",
						Timestamp:   time.Now().Format(time.RFC3339),
						Color:       discord2.EmbedColorNormal,
						Fields: []*discordgo.MessageEmbedField{{
							Name:   "Temp Title",
							Value:  sitePo.essentialInfoForEmbed(),
							Inline: false,
						}},
					})
					return
				}
			case <-time.After(time.Duration(stageOvertime) * time.Second):
				//overtime termination sign
				fmt.Println("terminating through overtime")
				clients.DgSession.ChannelMessageSend(s.ChannelID, "Time's up.")
				return
			}
		}
	}()
	s.MainCommand.ActiveSaveSitetageMap.disposeSaveSiteStage(newSaveStageKeyFromRaw(s.ChannelID, s.UserID))
}

func newListSiteStage(ms *discordgo.Message, userID string, cm *SaveThisSiteCommand) listSiteStage {
	stage := listSiteStage{
		BasicStageInfo: commands.BasicStageInfo{
			ChannelID:      ms.ChannelID,
			UserID:         userID,
			StageNow:       0,
			LastActionTime: time.Now(),
		},
		LsPager:                nil,
		ProcessInteractionChan: nil,
		MainCommand:            cm,
	}
	return stage
}

type listSiteStage struct {
	commands.BasicStageInfo
	LsPager                *commands.Pager
	ProcessInteractionChan chan *discordgo.Interaction
	MainCommand            *SaveThisSiteCommand
}

func (l *listSiteStage) process() {
	func() {
		for {
			select {
			case interaction, ok := <-l.ProcessInteractionChan:
				if !ok {
					fmt.Println("Aborted")
					return
				}
				//this should never be called
				fmt.Println(interaction.Data)
				return
			case <-time.After(l.LsPager.Overtime):
				//overtime termination sign
				fmt.Println("terminating through overtime")
				return
			}
		}
	}()
	//lock buttons. requires additional resources
	l.LsPager.LockPagerButtons()
	l.MainCommand.ActiveListSiteStageMap.disposeListSiteStage(newListStageKeyFromRaw(l.LsPager.AttachedMessage.ID, l.UserID))
}

func newListStageKeyFromRaw(messageID, userID string) commands.CombinedKey {
	return commands.CombinedKeyFromRaw(messageID, userID)
}

type listSitestageMap map[commands.CombinedKey]*listSiteStage

func (m listSitestageMap) insertListSiteStage(userID string, pager commands.Pager, cm *SaveThisSiteCommand) error {
	key := newListStageKeyFromRaw(pager.AttachedMessage.ID, userID)
	//can have multiple active stage
	if stage, ok := m[key]; ok {
		return fmt.Errorf("found an identical session at stage %d", stage.StageNow)
	}
	//todo function of new ListSiteStage
	stage := newListSiteStage(pager.AttachedMessage, userID, cm)
	stage.ProcessInteractionChan = make(chan *discordgo.Interaction, 1)
	stage.LsPager = &pager
	m[key] = &stage
	go stage.process()
	return nil
}

func (m listSitestageMap) disposeListSiteStage(key commands.CombinedKey) error {
	if v, ok := m[key]; !ok {
		return fmt.Errorf("disposing non-exist ls-site-stage w/ id: %s", key)
	} else {
		close(v.ProcessInteractionChan) // this should immediately trigger dispose
	}
	delete(m, key)
	return nil
}

func (cm *SaveThisSiteCommand) New() {
	cm.Name = "save-this-site"
	cm.Identifiers = []string{"save-site", "list-site"}
	cm.ActiveSaveSitetageMap = make(saveSitestageMap)
	cm.ActiveListSiteStageMap = make(listSitestageMap)
	cm.RegexExpressions = []*regexp.Regexp{}
	cm.RegexExpressions = append(cm.RegexExpressions, regexp.MustCompile(websiteRegex))
	cm.InitAvailableFlagMap()
	//the flag for taggiong sites
	cm.RegisterCommandFlag(commands.CommandFlag{
		Name:             "tag",
		FlagPrefix:       []string{"tag", "t"},
		AcceptsExtraArg:  true,
		MultipleExtraArg: true,
		MEGroup:          nil,
	})
	//the flag for debugging flag inputs
	cm.RegisterCommandFlag(commands.CommandFlag{
		Name:             "debug",
		FlagPrefix:       []string{"debug"},
		AcceptsExtraArg:  false,
		MultipleExtraArg: false,
		MEGroup:          nil,
	})
	//the flag for adding notes to sites
	cm.RegisterCommandFlag(commands.CommandFlag{
		Name:             "note",
		FlagPrefix:       []string{"note", "n"},
		AcceptsExtraArg:  true,
		MultipleExtraArg: false,
		MEGroup:          nil,
	})
	//the flag for using next-generation interactions.
	cm.RegisterCommandFlag(commands.CommandFlag{
		Name:             "neo",
		FlagPrefix:       []string{"neo"},
		AcceptsExtraArg:  false,
		MultipleExtraArg: false,
		MEGroup:          nil,
	})

	//Slash Commands
	cm.AppCommandsMap = make(map[string]*discordgo.ApplicationCommand)
	cm.AppCommandsMap["save-site"] = &discordgo.ApplicationCommand{
		Name:        "save-site",
		Description: "Request Dalian to save the site.",
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
				Description: "Add tags for this site, separated by default separator." +
					" Current separator:[%s]",
				Required: false,
			},
			{
				Type: discordgo.ApplicationCommandOptionString,
				Name: "note",
				//same as above
				Description: "Add note for this site." +
					" Current separator:[%s]",
				Required: false,
			},
			{
				Type:        discordgo.ApplicationCommandOptionBoolean,
				Name:        "cache",
				Description: "Cache the given site",
				Required:    false,
			},
		},
	}
	cm.AppCommandsMap["list-site"] = &discordgo.ApplicationCommand{
		Name:        "list-site",
		Description: "Retrieve sites saved by Dalian",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type: discordgo.ApplicationCommandOptionString,
				Name: "tags",
				//late init, replace %s with separator
				Description: "Add tags for this site, separated by default separator." +
					" Current separator:[%s]",
				Required: false,
			},
		},
	}

	cm.CompActionMap = make(commands.ComponentActionMap)
	cm.CompActionMap[lsButtonIDPrev] = func(i *discordgo.InteractionCreate) {
		matchKey := newListStageKeyFromRaw(i.Message.ID, i.Member.User.ID)
		if listStage, ok := cm.ActiveListSiteStageMap[matchKey]; ok {
			listStage.LsPager.SwitchPage(commands.PagerPrevPage, i.Interaction)
		}
	}
	cm.CompActionMap[lsButtonIDNext] = func(i *discordgo.InteractionCreate) {
		matchKey := newListStageKeyFromRaw(i.Message.ID, i.Member.User.ID)
		if listStage, ok := cm.ActiveListSiteStageMap[matchKey]; ok {
			listStage.LsPager.SwitchPage(commands.PagerNextPage, i.Interaction)
		}
	}
	//todo: complete the modification
	cm.AppCommandsMap["modify-site"] = &discordgo.ApplicationCommand{
		Name:        "modify-site",
		Description: "Modify site logs saved by dalian",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type: discordgo.ApplicationCommandOptionInteger,
				Name: "id",
				//late init, replace %s with separator
				Description: "ID of the modifying site log.",
				Required:    true,
			},
			{
				Type: discordgo.ApplicationCommandOptionString,
				Name: "tags",
				//late init, replace %s with separator
				Description: "Modify tags for this site, separated by default separator." +
					" If you want to clear it up, type [-].",
				Required: false,
			},
			{
				Type: discordgo.ApplicationCommandOptionString,
				Name: "note",
				//late init, replace %s with separator
				Description: "Modify note for this site." +
					" If you want to clear it up, type [-].",
				Required: false,
			},
		},
	}
	cm.AppCommandsMap["update-site-snapshot"] = &discordgo.ApplicationCommand{
		Name:        "update-site-snapshot",
		Description: "Update the snapshot for a given site record.",

		Options: []*discordgo.ApplicationCommandOption{
			{
				Name:        "refresh-snapshot",
				Description: "Force refresh the snapshot. Old snapshot (if present) will be discarded.",
				Type:        discordgo.ApplicationCommandOptionSubCommand,
				Options: []*discordgo.ApplicationCommandOption{
					{
						Type: discordgo.ApplicationCommandOptionInteger,
						Name: "id",
						//late init, replace %s with separator
						Description: "ID of the modifying site log.",
						Required:    true,
					},
				},
			},
			{
				Name:        "update-snapshot-url",
				Description: "Manually update the url for snapshot. Old snapshot (if present) will be discarded.",
				Type:        discordgo.ApplicationCommandOptionSubCommand,
				Options: []*discordgo.ApplicationCommandOption{
					{
						Type: discordgo.ApplicationCommandOptionInteger,
						Name: "id",
						//late init, replace %s with separator
						Description: "ID of the modifying site log.",
						Required:    true,
					},
					{
						Type: discordgo.ApplicationCommandOptionString,
						Name: "snapshot-url",
						Description: "Modify the valid share URL for the site snapshot." +
							" If you want to clear it up, type [-].",
						Required: true,
					},
				},
			},
			{
				Name:        "upload-snapshot-attachment",
				Description: "Manually update the snapshot attachment to cloud. Old snapshot (if present) will be discarded.",
				Type:        discordgo.ApplicationCommandOptionSubCommand,
				Options: []*discordgo.ApplicationCommandOption{
					{
						Type: discordgo.ApplicationCommandOptionInteger,
						Name: "id",
						//late init, replace %s with separator
						Description: "ID of the modifying site log.",
						Required:    true,
					},
					{
						Type: discordgo.ApplicationCommandOptionString,
						Name: "snapshot-attachment",
						Description: "Modify the valid share URL for the site snapshot." +
							" If you want to clear it up, type [-].",
						Required: true,
					},
				},
			},
		},
	}

}

func (cm *SaveThisSiteCommand) DoMessage(m *discordgo.MessageCreate) error {
	//first check manual
	//one-step insertion, or list / other operations
	if matchStatus, matchedCommand := cm.MatchText(m.Content); matchStatus {
		args, length := cm.SeparateArgs(m.Content, commands.Separator)
		//read the flags
		flagMap, err := cm.ParseFlags(args[0])
		if err != nil {
			discord2.ChannelMessageReportError(m.ChannelID, err)
			return nil
		}
		//validate the flags
		flagMap, err = cm.ValidateFlagMap(flagMap)
		if err != nil {
			discord2.ChannelMessageReportError(m.ChannelID, err)
			return nil
		}
		if flagMap.HasFlag("debug") {
			discord2.ChannelMessageSend(m.ChannelID, fmt.Sprintf("flagMap:%v", flagMap))
		}
		//execute command body
		switch matchedCommand {
		case "save-site":
			//make sure arg[1] has a valid url
			if length <= 1 {
				discord2.ChannelMessageSend(m.ChannelID, "You need an url as the second argument!")
				return nil
			}
			if _, err := url.ParseRequestURI(args[1]); err != nil {
				discord2.ChannelMessageSend(m.ChannelID, "The second argument must be a VALID url!")
				return nil
			}
			//validation passed, start the logic
			sitePO := newRawSitePOFromMessage(m.Message)
			//set site
			sitePO.Site = args[1]
			//set tags
			if len(flagMap["tag"]) > 0 {
				sitePO.Tags = flagMap["tag"]
			}
			//set note
			if len(flagMap["note"]) > 0 {
				sitePO.Note = flagMap["note"][0]
			}
			//save it to the database
			go insertSitePo(sitePO, true)
			// discord.ChannelMessageSend(m.ChannelID, fmt.Sprintf("Site saved:%s", sitePO.essentialInfo()))
			//TODO: snapshot things.
			discord2.ChannelMessageSendEmbed(m.ChannelID, &discordgo.MessageEmbed{
				Title:       "Site saved",
				Description: "The following site has been saved",
				Timestamp:   time.Now().Format(time.RFC3339),
				Color:       discord2.EmbedColorNormal,
				Fields: []*discordgo.MessageEmbedField{{
					Name:   "Temp Title",
					Value:  sitePO.essentialInfoForEmbed(),
					Inline: false,
				}},
			})
			//no subsequent check
			return nil
		case "list-site":
			query := bson.M{"user_id": m.Author.ID, "guild_id": m.GuildID}
			//optional url
			if len(flagMap["tag"]) > 0 {
				if flagMap["tag"][0] == "-" {
					query["tags"] = nil
				} else {
					query["tags"] = bson.M{"$all": flagMap["tag"]}
				}
			}
			findOpts := options.Find().SetSort(bson.D{{"id", 1}})
			findCursor, err := data.GetCollection("site_collection").Find(context.TODO(), query, findOpts)
			var results sitePoArr
			if err = findCursor.All(context.TODO(), &results); err != nil {
				fmt.Println("ERROR:" + err.Error())
				return err
			}
			// clients.DgSession.ChannelMessageSend(m.ChannelID, fmt.Sprintf("Found Doc(s):%s", results.digestInfo()))
			_, err = discord2.ChannelMessageSendEmbed(m.ChannelID, newListSiteEmbed(results))
			if err != nil {
				fmt.Println(err)
			}
		}

	}

	//then check implicit
	if _, err1 := url.ParseRequestURI(m.Content); err1 == nil {
		//calling insertSaveSiteStage to start a goroutine for stepped Q&A
		//the stage will auto dispose.
		if err := cm.ActiveSaveSitetageMap.insertSaveSiteStage(m, cm); err != nil {
			discord2.ChannelMessageReportError(m.ChannelID, err)
		}
		//no subsequent check
		return nil
	}

	//finally check stage progress
	if activeStage, ok := cm.ActiveSaveSitetageMap[newSaveStageKeyFromMs(*m.Message)]; ok {
		//a stage present, check if it's a stage info
		if isCallingBot, _ := cm.IsCallingBot(m.Content); isCallingBot {
			//@Bot content, pass to process for further check
			activeStage.ProcessMsgChan <- m.Message
		}

	}

	return nil
}

type SitePO struct {
	//Display Info
	ID   int64    `bson:"id"`
	Site string   `bson:"site"`
	Tags []string `bson:"tags"`
	Note string   `bson:"note"`
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

func (sp *SitePO) ToMessageEmbedField() *discordgo.MessageEmbedField {
	return &discordgo.MessageEmbedField{
		Name:   fmt.Sprintf("%d. Temporary Title", sp.ID),
		Value:  sp.essentialInfoForEmbed(),
		Inline: false,
	}
}

func (sp *SitePO) setTime(isCreate bool) {
	currentTime := time.Now()
	if isCreate {
		sp.CreatedTime = currentTime
	}
	sp.LastModifiedTime = currentTime
}

func (sp *SitePO) essentialInfo() string {
	var tags, note string
	if len(sp.Tags) == 0 {
		tags = "*None*"
	} else {
		tags = "[" + strings.Join(sp.Tags, ",") + "]"
	}

	if sp.Note == "" {
		note = "*None*"
	} else {
		note = sp.Note
	}
	essentialInfo := "> Site: %s\r" +
		"> Tags: %s\r" +
		"> Note: %s"
	return fmt.Sprintf(essentialInfo, sp.Site, tags, note)
}

func (sp *SitePO) essentialInfoForEmbed() string {
	var tags, note, optSnapshot string
	if len(sp.Tags) == 0 {
		tags = "*None*"
	} else {
		tags = "[" + strings.Join(sp.Tags, ",") + "]"
	}

	if sp.Note == "" {
		note = "*None*"
	} else {
		note = sp.Note
	}
	if sp.SnapshotURL != "" {
		optSnapshot = fmt.Sprintf("\r[snapshot](%s)", sp.SnapshotURL)
	}
	essentialInfo := "%s\r" +
		"Tags: %s\r" +
		"Note: %s" +
		"%s"
	return fmt.Sprintf(essentialInfo, sp.Site, tags, note, optSnapshot)
}

func newRawSitePOFromMessage(message *discordgo.Message) SitePO {
	sitepo := SitePO{
		Site:      "",
		Tags:      []string{},
		Note:      "",
		GuildID:   message.GuildID,
		ChannelID: message.ChannelID,
		UserID:    message.Author.ID,
	}
	return sitepo
}

func newRawSitePoFromInteraction(i *discordgo.Interaction) SitePO {
	sitepo := SitePO{
		Site:      "",
		Tags:      []string{},
		Note:      "",
		GuildID:   i.GuildID,
		ChannelID: i.ChannelID,
		UserID:    i.Member.User.ID,
	}
	return sitepo
}

func getLastNumericalID() int64 {

	singleResult := data.GetCollection("site_collection").FindOne(context.TODO(), bson.M{}, options.FindOne().SetSort(bson.M{"id": -1}))
	if singleResult.Err() == mongo.ErrNoDocuments {
		return 0
	}
	var doc SitePO
	if err := singleResult.Decode(&doc); err != nil {
		fmt.Println("Unable to Decode doc into SitePo", err)
		return -1
	}
	return doc.ID
}

func insertSitePo(po SitePO, isCreate bool) error {
	idNow := getLastNumericalID() + 1
	if idNow == 0 {
		return errors.New("Something wrong retrieving last id")
	}
	po.ID = idNow
	po.setTime(isCreate)
	_, err := data.GetCollection("site_collection").InsertOne(context.TODO(), po)
	if err != nil {
		return err
	}
	//TODO need to get a numerical id first
	return nil
}

func retrieveSitePoByNumericalID(id int64) (*SitePO, error) {
	query := bson.M{"id": id}
	res := data.GetCollection("site_collection").FindOne(context.Background(), query)
	if res.Err() != nil {
		return nil, res.Err()
	}
	var foundPo SitePO
	if err := res.Decode(&foundPo); err != nil {
		return nil, errors.Wrap(err, "unknown error")
	}
	return &foundPo, nil
}

func updateSitePo(po SitePO) (*SitePO, error) {
	query := bson.M{"id": po.ID}
	po.setTime(false)
	//FindOneAndUpdate return the modified document before the update, not the document AFTER the update
	res := data.GetCollection("site_collection").FindOneAndUpdate(context.Background(), query, bson.M{"$set": po})
	if res.Err() != nil {
		return nil, res.Err()
	}
	var updatedPo SitePO
	if err := res.Decode(&updatedPo); err != nil {
		return nil, errors.Wrap(err, "unknown error")
	}
	return &updatedPo, nil
}

type sitePoArr []*SitePO

func (arr sitePoArr) digestInfo() string {
	info := ""
	for _, v := range arr {
		info += "\r" + v.essentialInfo() + "\r"
	}
	info = fmt.Sprintf("{%s}", info)
	return info
}

func newListSiteEmbed(arr sitePoArr) *discordgo.MessageEmbed {
	embed := &discordgo.MessageEmbed{
		Title:       "ls result",
		Description: fmt.Sprintf("Your query rendered %d results.", len(arr)),
		Timestamp:   time.Now().Format(time.RFC3339),
		Color:       discord2.EmbedColorNormal,
		Fields:      nil,
	}
	var fields []*discordgo.MessageEmbedField
	for _, sitePo := range arr {
		fields = append(fields, sitePo.ToMessageEmbedField())
	}
	embed.Fields = fields
	return embed
}

type sitePoPagerLoader struct {
	context        context.Context
	query          any
	queryOptions   []*options.FindOptions
	resultsStorage []*SitePO
	commands.DefaultPageRenderer
}

func (s *sitePoPagerLoader) LoadPager(pager *commands.Pager) error {
	findCursor, err := data.GetCollection("site_collection").Find(s.context, s.query, s.queryOptions...)
	if err != nil {
		return errors.Wrap(err, "error querying site_collections")
	}
	if err = findCursor.All(context.TODO(), &s.resultsStorage); err != nil {
		return errors.Wrap(err, "internal Error: wrong cursor type")
	}
	for _, v := range s.resultsStorage {
		var tempVar commands.IPagerPart
		tempVar = v
		pager.CompleteItemSlice = append(pager.CompleteItemSlice, &tempVar)
	}
	return nil
}

func init() {
	var saveThisCommand SaveThisSiteCommand
	saveThisCommand.New()
	commands.RegisterCommand(&saveThisCommand)
}

func (cm *SaveThisSiteCommand) LateInit() {
	//late-init separator in prompt
	for _, cmdOption := range cm.AppCommandsMap["save-site"].Options {
		if cmdOption.Name == "tags" || cmdOption.Name == "note" {
			cmdOption.Description = fmt.Sprintf(cmdOption.Description, commands.Separator)
		}
	}
}
