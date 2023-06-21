package plugins

import (
	"context"
	"dalian-bot/internal/core"
	"dalian-bot/internal/services/data"
	"dalian-bot/internal/services/ddtv"
	"dalian-bot/internal/services/discord"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/bwmarrin/discordgo"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"golang.org/x/exp/slices"
	"sort"
	"strconv"
	"strings"
)

// DDTVPlugin Receives DDTV Webhook and notify in channel
// Discord: related command can be found under command group of `ddtv`
type DDTVPlugin struct {
	core.Plugin
	DiscordService *discord.Service
	DataService    *data.Service
	discord.SlashCommandUtil
	core.ArgParseUtil
	discord.IDiscordHelper
}

func (p *DDTVPlugin) DoNamedInteraction(_ *core.Bot, i *discordgo.InteractionCreate) (e error) {
	if isMatched, cmdName := p.DefaultMatchCommand(i); !isMatched {
		//fmt.Printf("nothing matched: %v", i)
		return nil
	} else {
		switch cmdName {
		case "ddtv":
			cmdOption := i.ApplicationCommandData().Options[0]
			switch cmdOption.Name {
			case "webhook-channel":
				cmdOption := cmdOption.Options[0]
				switch cmdOption.Name {
				case "set":
					updateResult, err := p.upsertOneWebhookNotifyChannel(ddtvNotifyPo{
						AdminDiscordUserID: i.Interaction.Member.User.ID,
						GuildID:            i.Interaction.GuildID,
						NotifyChannelID:    i.Interaction.ChannelID,
					})
					if err != nil {
						core.Logger.Warnf("Error inserting webhook channel record: %v", err)
						return err
					}
					if updateResult.UpsertedCount > 0 {
						p.DiscordService.InteractionRespond(i.Interaction, "webhook channel created!")
					} else {
						p.DiscordService.InteractionRespond(i.Interaction, "already a webhook channel!")
					}
				case "remove":
					deleteResult, err := p.deleteOneWebhookNotifyChannel(i.Interaction.ChannelID)
					if err != nil {
						core.Logger.Warnf("Error deleting webhook channel record: %v", err)
						return err
					}
					if deleteResult.DeletedCount > 0 {
						p.DiscordService.InteractionRespond(i.Interaction, "webhook channel removed!")
					} else {
						p.DiscordService.InteractionRespond(i.Interaction, "not a webhook channel yet!")
					}
				}
			case "streamers":
				cmdOption := cmdOption.Options[0]
				optionsMap := p.ParseOptionsMap(cmdOption.Options)
				switch cmdOption.Name {
				case "addone-by-uid":
					// fetch uid (int64)
					uid := optionsMap["uid"].IntValue()
					// fetch and validate ddtvNotifyPo
					notifyPo, err := p.findOneWebhookNotifyChannelByChannelID(i.Interaction.ChannelID)
					if err != nil {
						if err == mongo.ErrNoDocuments {
							p.DiscordService.InteractionRespond(i.Interaction, "This is not a notification channel yet! Consider making it one by using *ddtv webhook-channel set*?")
							return nil
						}
						core.Logger.Warnf("Error finding webhook channel record: %v", err)
						return err
					}
					// avoid duplicate
					if slices.Contains(notifyPo.FeaturedUIDs, uid) {
						p.DiscordService.InteractionRespond(i.Interaction, "This streamer is already featured.")
						return nil
					}
					// good, add the uid to slices
					notifyPo.FeaturedUIDs = append(notifyPo.FeaturedUIDs, uid)
					// database persistance
					if _, err := p.upsertOneWebhookNotifyChannel(notifyPo); err != nil {
						core.Logger.Warnf("Error updating webhook channel featured list: %v", err)
						return err
					}
					p.DiscordService.InteractionRespond(i.Interaction, fmt.Sprintf("Added the following streamer to featured list: %d", uid))
				case "batch-modify":
					// parse uids (string -> int64)
					var uids []int64
					uidsStr := optionsMap["uids"].StringValue()
					appendFlag := optionsMap["append"].BoolValue()
					if uidsStr == "-" {
						//clean up
						uids = []int64{}
					} else {
						rawUidsStrings := p.SeparateArgs(uidsStr, p.DiscordService.DiscordAccountConfig.Separator)
						// iter through and validate uids
						for _, v := range rawUidsStrings {
							parsedInt64, err := strconv.ParseInt(v, 10, 64)
							if err != nil {
								p.DiscordService.InteractionRespond(i.Interaction, fmt.Sprintf("\"%s\" is not a valid int64!", v))
								return nil
							}
							if !slices.Contains(uids, parsedInt64) {
								uids = append(uids, parsedInt64)
							}
						}
					}
					// find and modify notifyPo when necessary
					notifyPo, err := p.findOneWebhookNotifyChannelByChannelID(i.Interaction.ChannelID)
					if err != nil {
						if err == mongo.ErrNoDocuments {
							p.DiscordService.InteractionRespond(i.Interaction, "This is not a notification channel yet! Consider making it one by using *ddtv webhook-channel set*?")
							return nil
						}
						core.Logger.Warnf("Error finding webhook channel record: %v", err)
						return err
					}
					// replace or append uid list
					if !appendFlag {
						// a complete replace.
						notifyPo.FeaturedUIDs = uids
					} else {
						for _, v := range uids {
							if !slices.Contains(notifyPo.FeaturedUIDs, v) {
								notifyPo.FeaturedUIDs = append(notifyPo.FeaturedUIDs, v)
							}
						}
					}
					// database persistance
					if _, err := p.upsertOneWebhookNotifyChannel(notifyPo); err != nil {
						core.Logger.Warnf("Error updating webhook channel featured list: %v", err)
						return err
					}
					p.DiscordService.InteractionRespond(i.Interaction, fmt.Sprintf("Updated featured list: %v", notifyPo.FeaturedUIDs))

				case "status":
					dumpFlag := false
					if dump, ok := optionsMap["dump"]; ok {
						dumpFlag = dump.BoolValue()
					}
					// fetch and validate ddtvNotifyPo
					notifyPo, err := p.findOneWebhookNotifyChannelByChannelID(i.Interaction.ChannelID)
					if err != nil {
						if err == mongo.ErrNoDocuments {
							p.DiscordService.InteractionRespond(i.Interaction, "This is not a notification channel yet! Consider making it one by using *ddtv webhook-channel set*?")
							return nil
						}
						core.Logger.Warnf("Error finding webhook channel record: %v", err)
						return err
					}
					currentUIDs := notifyPo.FeaturedUIDs
					// if nothing to show
					if len(notifyPo.FeaturedUIDs) == 0 {
						p.DiscordService.InteractionRespond(i.Interaction, "Featured list empty. Push ALL webhook notifications by default.")
						return nil
					}
					sort.Slice(notifyPo.FeaturedUIDs, func(i, j int) bool { return notifyPo.FeaturedUIDs[i] < notifyPo.FeaturedUIDs[j] })
					ansStr := fmt.Sprintf("%d streamers featured: %v", len(currentUIDs), currentUIDs)
					if dumpFlag {
						var strSlice []string
						for _, v := range currentUIDs {
							strSlice = append(strSlice, strconv.FormatInt(v, 10))
						}
						ansStr += fmt.Sprintf("\rHere's the dump for you:\r```%s```", strings.Join(strSlice[:], p.DiscordService.DiscordAccountConfig.Separator))
					}
					p.DiscordService.InteractionRespond(i.Interaction, ansStr)
				}
			case "webhooks":
				cmdOption := cmdOption.Options[0]
				optionsMap := p.ParseOptionsMap(cmdOption.Options)
				switch cmdOption.Name {
				case "addone-by-code":
					// fetch hook code (int)
					hookCode := int(optionsMap["webhook-code"].IntValue())
					// fetch and validate ddtvNotifyPo
					notifyPo, err := p.findOneWebhookNotifyChannelByChannelID(i.Interaction.ChannelID)
					if err != nil {
						if err == mongo.ErrNoDocuments {
							p.DiscordService.InteractionRespond(i.Interaction, "This is not a notification channel yet! Consider making it one by using *ddtv webhook-channel set*?")
							return nil
						}
						core.Logger.Warnf("Error finding webhook channel record: %v", err)
						p.DiscordService.InteractionRespond(i.Interaction, "internal error.")
						return err
					}
					// avoid duplicate
					if slices.Contains(notifyPo.FeaturedHookTypes, hookCode) {
						p.DiscordService.InteractionRespond(i.Interaction, "This streamer is already featured.")
						return nil
					}
					// good, add the hook code to slices
					notifyPo.FeaturedHookTypes = append(notifyPo.FeaturedHookTypes, hookCode)
					// database persistance
					if _, err := p.upsertOneWebhookNotifyChannel(notifyPo); err != nil {
						core.Logger.Warnf("Error updating webhook channel featured list: %v", err)
						return err
					}
					p.DiscordService.InteractionRespond(i.Interaction, fmt.Sprintf("Added the following hooktype code to featured list: %d", hookCode))
				case "batch-modify":
					// parse hook types (string -> int)
					var hookTypes []int
					hookTypesStr := optionsMap["webhook-codes"].StringValue()
					appendFlag := optionsMap["append"].BoolValue()
					if hookTypesStr == "-" {
						//clean up
						hookTypes = []int{}
					} else {
						rawHooksStrings := p.SeparateArgs(hookTypesStr, p.DiscordService.DiscordAccountConfig.Separator)
						// iter through and validate webhook types
						for _, v := range rawHooksStrings {
							parsedInt, err := strconv.Atoi(v)
							if err != nil {
								p.DiscordService.InteractionRespond(i.Interaction, fmt.Sprintf("\"%s\" is not a valid int!", v))
								return nil
							}
							if !slices.Contains(hookTypes, parsedInt) {
								hookTypes = append(hookTypes, parsedInt)
							}
						}
					}
					// find and modify notifyPo when necessary
					notifyPo, err := p.findOneWebhookNotifyChannelByChannelID(i.Interaction.ChannelID)
					if err != nil {
						if err == mongo.ErrNoDocuments {
							p.DiscordService.InteractionRespond(i.Interaction, "This is not a notification channel yet! Consider making it one by using *ddtv webhook-channel set*?")
							return nil
						}
						core.Logger.Warnf("Error finding webhook channel record: %v", err)
						return err
					}
					// replace or append webhook types list
					if !appendFlag {
						// a complete replace.
						notifyPo.FeaturedHookTypes = hookTypes
					} else {
						for _, v := range hookTypes {
							if !slices.Contains(notifyPo.FeaturedHookTypes, v) {
								notifyPo.FeaturedHookTypes = append(notifyPo.FeaturedHookTypes, v)
							}
						}
					}
					// database persistance
					if _, err := p.upsertOneWebhookNotifyChannel(notifyPo); err != nil {
						core.Logger.Warnf("Error updating webhook types featured list: %v", err)
						return err
					}
					p.DiscordService.InteractionRespond(i.Interaction, fmt.Sprintf("Updated featured list: %v", notifyPo.FeaturedHookTypes))

				case "status":
					dumpFlag := false
					if dump, ok := optionsMap["dump"]; ok {
						dumpFlag = dump.BoolValue()
					}
					// fetch and validate ddtvNotifyPo
					notifyPo, err := p.findOneWebhookNotifyChannelByChannelID(i.Interaction.ChannelID)
					if err != nil {
						if err == mongo.ErrNoDocuments {
							p.DiscordService.InteractionRespond(i.Interaction, "This is not a notification channel yet! Consider making it one by using *ddtv webhook-channel set*?")
							return nil
						}
						core.Logger.Warnf("Error finding webhook channel record: %v", err)
						return err
					}
					currentWebhookTypes := notifyPo.FeaturedHookTypes
					// if nothing to show
					if len(currentWebhookTypes) == 0 {
						p.DiscordService.InteractionRespond(i.Interaction, "Featured list empty. Push ALL webhook notifications by default.")
						return nil
					}
					sort.Slice(currentWebhookTypes, func(i, j int) bool { return currentWebhookTypes[i] < currentWebhookTypes[j] })
					ansStr := fmt.Sprintf("%d webhook types featured: %v", len(currentWebhookTypes), currentWebhookTypes)
					if dumpFlag {
						var strSlice []string
						for _, v := range currentWebhookTypes {
							strSlice = append(strSlice, strconv.Itoa(v))
						}
						ansStr += fmt.Sprintf("\rHere's the dump for you:\r```%s```", strings.Join(strSlice[:], p.DiscordService.DiscordAccountConfig.Separator))
					}
					p.DiscordService.InteractionRespond(i.Interaction, ansStr)
				}
			}

		}
	}

	return nil
}

func (p *DDTVPlugin) Init(reg *core.ServiceRegistry) error {
	// DiscordService is a MUST have. return error if not found.
	if err := reg.FetchService(&p.DiscordService); err != nil {
		return err
	}
	// DataService is also a MUST have. return error if not found.
	if err := reg.FetchService(&p.DataService); err != nil {
		return err
	}
	// ddtvService is not used to perform actions actively in the plugin, so not imported.

	p.AcceptedTriggerTypes = []core.TriggerType{discord.TriggerTypeDiscord, ddtv.TriggerTypeDDTV}
	p.Name = "ddtv"
	p.AppCommandsMap = make(map[string]*discordgo.ApplicationCommand)
	p.AppCommandsMap.RegisterCommand(&discordgo.ApplicationCommand{
		Name:        "ddtv",
		Description: "ddtv commands",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Name:        "webhook-channel",
				Description: "webhook channel commands",
				Type:        discordgo.ApplicationCommandOptionSubCommandGroup,
				Options: []*discordgo.ApplicationCommandOption{
					{
						Name:        "set",
						Type:        discordgo.ApplicationCommandOptionSubCommand,
						Description: "Set current channel as ddtv webhook channel",
					}, {
						Name:        "remove",
						Type:        discordgo.ApplicationCommandOptionSubCommand,
						Description: "Remove current channel as ddtv webhook channel",
					},
				},
			},
			{
				Name:        "streamers",
				Description: "featuring webhook notifications by streamers and/or types",
				Type:        discordgo.ApplicationCommandOptionSubCommandGroup,
				Options: []*discordgo.ApplicationCommandOption{
					{
						//todo: add helper
						Name:        "addone-by-uid",
						Type:        discordgo.ApplicationCommandOptionSubCommand,
						Description: "Add a streamer to current channel's featured list",
						Options: []*discordgo.ApplicationCommandOption{
							{
								Type:        discordgo.ApplicationCommandOptionInteger,
								Name:        "uid",
								Required:    true,
								Description: "The bilibili UID of the streamer (not RoomID!)",
							},
						},
					}, {
						//todo: add helper
						Name:        "batch-modify",
						Type:        discordgo.ApplicationCommandOptionSubCommand,
						Description: "Append or replace the featured list with input",
						Options: []*discordgo.ApplicationCommandOption{
							{
								Type:        discordgo.ApplicationCommandOptionString,
								Name:        "uids",
								Required:    true,
								Description: fmt.Sprintf("The bilibili UID of the streamer, separated by default separator (%s)", p.DiscordService.DiscordAccountConfig.Separator),
							},
							{
								Type:        discordgo.ApplicationCommandOptionBoolean,
								Name:        "append",
								Required:    true,
								Description: "Whether dalian should append or DISCARD existing lists and use new one.",
							},
						},
					}, {
						//todo: add helper
						Name:        "status",
						Type:        discordgo.ApplicationCommandOptionSubCommand,
						Description: "Display the current featured list for this channel",
						Options: []*discordgo.ApplicationCommandOption{
							{
								Type:        discordgo.ApplicationCommandOptionBoolean,
								Name:        "dump",
								Required:    false,
								Description: "Whether dalian should dump all existing featured streamers",
							},
						},
					},
				},
			}, {
				Name:        "webhooks",
				Description: "featuring webhook notifications by webhook types",
				Type:        discordgo.ApplicationCommandOptionSubCommandGroup,
				Options: []*discordgo.ApplicationCommandOption{
					{
						//todo: add helper
						Name:        "addone-by-code",
						Type:        discordgo.ApplicationCommandOptionSubCommand,
						Description: "Add a webbhook type code to featured list",
						Options: []*discordgo.ApplicationCommandOption{
							{
								Type:        discordgo.ApplicationCommandOptionInteger,
								Name:        "webhook-code",
								Required:    true,
								Description: "The webhook type code of the DDTV Webhook.",
							},
						},
					}, {
						//todo: add helper
						Name:        "batch-modify",
						Type:        discordgo.ApplicationCommandOptionSubCommand,
						Description: "Append or replace the featured list with input",
						Options: []*discordgo.ApplicationCommandOption{
							{
								Type:        discordgo.ApplicationCommandOptionString,
								Name:        "webhook-codes",
								Required:    true,
								Description: fmt.Sprintf("TThe webhook type code of the DDTV Webhook, separated by default separator (%s)", p.DiscordService.DiscordAccountConfig.Separator),
							},
							{
								Type:        discordgo.ApplicationCommandOptionBoolean,
								Name:        "append",
								Required:    true,
								Description: "Whether dalian should append or DISCARD existing lists and use new one.",
							},
						},
					}, {
						//todo: add helper
						Name:        "status",
						Type:        discordgo.ApplicationCommandOptionSubCommand,
						Description: "Display the current featured list for this channel",
						Options: []*discordgo.ApplicationCommandOption{
							{
								Type:        discordgo.ApplicationCommandOptionBoolean,
								Name:        "dump",
								Required:    false,
								Description: "Whether dalian should dump all existing featured webhook type codes",
							},
						},
					},
				},
			},
		},
	})

	formattedHelpSetNotifChannel := `*ddtv webhook-channel set*: /ddtv webhook-channel set
Set current channel as a DDTV webhook notification channel.
Dalian will send a message for every incoming DDTV webhook received in this channel`
	formattedHelpRemoveNotifChannel := `ddtv webhook-channel remove*: /ddtv webhook-channel remove
Remove current channel from DDTV webhook notification channel list.
Dalian will no longer send a message for every incoming DDTV webhook received in this channel`

	p.IDiscordHelper = discord.GenerateHelper(discord.HelperConfig{
		PluginHelp: "HelperUtil support for Dalian.",
		CommandHelps: []discord.CommandHelp{
			{
				Name:          "ddtv webhook-channel set",
				FormattedHelp: formattedHelpSetNotifChannel,
			},
			{
				Name:          "ddtv webhook-channel remove",
				FormattedHelp: formattedHelpRemoveNotifChannel,
			},
		},
	})
	return p.DiscordService.RegisterSlashCommand(p)
}

func (p *DDTVPlugin) Trigger(trigger core.Trigger) {
	if !p.AcceptTrigger(trigger.Type) {
		return
	}
	switch trigger.Type {
	case discord.TriggerTypeDiscord:
		// do NOT accept  messageCreate event
		// do something...
		dcEvent := discord.UnboxEvent(trigger)
		switch dcEvent.EventType {
		// only accepting interactionCreate for discord trigers
		case discord.EventTypeInteractionCreate:
			switch dcEvent.InteractionCreate.Type {
			case discordgo.InteractionApplicationCommand:
				// slash command
				p.DoNamedInteraction(trigger.Bot, dcEvent.InteractionCreate)
			default:
				// not accepting other type of interaction for this plugin
				return
			}
		default:
			// does not handle messageCreate or anything like that.
			return
		}
	case ddtv.TriggerTypeDDTV:
		// do ddtv webhook thing
		webhook := ddtv.UnboxEvent(trigger).WebHook
		// if hooktype is channel, restrict non-record channel message to online.
		// todo: add a config option to control this behavior
		if webhook.UserInfo.UID != 0 && !webhook.RoomInfo.IsAutoRec && webhook.Type != ddtv.HookStartLive {
			return
		}
		p.notifyDDTVWebhookToChannels(webhook)
	default:
		core.Logger.Warnf(core.LogPromptUnknownTrigger, trigger.Type)
	}
}

func (p *DDTVPlugin) notifyDDTVWebhookToChannels(webhook ddtv.WebHook) {
	channels, err := p.fetchDDTVWebhookNotifyChannels()
	if err != nil {
		core.Logger.Warnf("Retrieve webhook channels failed!: %v", err)
		return
	}
	for _, channel := range channels {
		// check feature group only when it's not empty
		if len(channel.FeaturedUIDs) != 0 {
			// do NOT display if the user is not in the list
			if !slices.Contains(channel.FeaturedUIDs, webhook.Uid) {
				//SKIP this channel
				continue
			}
		}
		// check feature group only when it's not empty
		if len(channel.FeaturedHookTypes) != 0 {
			// do NOT display if the webhook type is incorrect
			if !slices.Contains(channel.FeaturedHookTypes, webhook.Type.Value()) {
				//SKIP this webhook type
				continue
			}
		}
		_, err := p.DiscordService.ChannelMessageSendEmbed(channel.NotifyChannelID, webhook.DigestEmbed())
		if err != nil {
			core.Logger.Warnf("Embed sent failed: %v", err)
			b, _ := json.Marshal(webhook)
			p.DiscordService.ChannelMessageSendCodeBlock(channel.NotifyChannelID, string(b))
			return
		}
	}

}

type ddtvNotifyPo struct {
	BsonID             primitive.ObjectID `bson:"_id,omitempty"`
	AdminDiscordUserID string             `bson:"admin_dc_user_id"`
	GuildID            string             `bson:"guild_id"`
	NotifyChannelID    string             `bson:"notify_channel_id"`
	FeaturedUIDs       []int64            `bson:"featured_uid_list"`
	FeaturedHookTypes  []int              `bson:"featured_hook_types"`
}

func (p *DDTVPlugin) getCollection() *mongo.Collection {
	return p.DataService.GetCollection("ddtv_notify_channels")
}

func (p *DDTVPlugin) findOneWebhookNotifyChannelByChannelID(channelID string) (ddtvNotifyPo, error) {
	var result ddtvNotifyPo
	rawResult := p.DataService.FindOne(&result, p.getCollection(), context.Background(), bson.M{"notify_channel_id": channelID})
	return result, rawResult.Err()
}

func (p *DDTVPlugin) upsertOneWebhookNotifyChannel(po ddtvNotifyPo) (*mongo.UpdateResult, error) {
	rawResult := p.DataService.UpdateOne(bson.D{{"$set", data.ToBsonDocForce(po)}}, p.getCollection(), context.Background(), bson.M{"notify_channel_id": po.NotifyChannelID}, options.Update().SetUpsert(true))
	return rawResult.UpdateResult(), rawResult.Err()
}

func (p *DDTVPlugin) deleteOneWebhookNotifyChannel(channelID string) (*mongo.DeleteResult, error) {
	rawResult := p.DataService.DeleteOne(p.getCollection(), context.Background(), bson.M{"notify_channel_id": channelID})
	return rawResult.DeleteResult(), rawResult.Err()
}

func (p *DDTVPlugin) findDDTVWebhookNotifyChannelIDs() (channels []string, er error) {
	var results []ddtvNotifyPo
	if err := p.DataService.Find(&results, p.getCollection(), context.Background(), bson.M{}); err != nil {
		return nil, err
	}
	var channelIDs []string
	for _, v := range results {
		channelIDs = append(channelIDs, v.NotifyChannelID)
	}
	return channelIDs, nil

}

func (p *DDTVPlugin) fetchDDTVWebhookNotifyChannels() (channels []ddtvNotifyPo, er error) {
	if err := p.DataService.Find(&channels, p.getCollection(), context.Background(), bson.M{}); err != nil {
		return nil, err
	}
	return channels, nil
}

func NewDDTVPlugin(reg *core.ServiceRegistry) core.IPlugin {
	var ddtvPlugin DDTVPlugin
	if err := (&ddtvPlugin).Init(reg); err != nil && errors.As(err, &core.ErrServiceFetchUnknownService) {
		core.Logger.Panicf("DDTV plugin MUST have all required service(s) injected!")
		panic("DDTV plugin initialization failed.")
	}
	return &ddtvPlugin
}
