package plugins

import (
	"context"
	core2 "dalian-bot/internal/core"
	data2 "dalian-bot/internal/services/data"
	ddtv2 "dalian-bot/internal/services/ddtv"
	discord2 "dalian-bot/internal/services/discord"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/bwmarrin/discordgo"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type DDTVPlugin struct {
	core2.Plugin
	DiscordService *discord2.Service
	DataService    *data2.Service
	discord2.SlashCommand
	discord2.IDisrocdHelper
}

func (p *DDTVPlugin) DoNamedInteraction(_ *core2.Bot, i *discordgo.InteractionCreate) (e error) {
	if isMatched, cmdName := p.DefaultMatchCommand(i); !isMatched {
		fmt.Printf("nothing matched: %v", i)
		return nil
	} else {
		switch cmdName {
		case "ddtv":
			cmdOptions := i.ApplicationCommandData().Options
			switch cmdOptions[0].Name {
			case "webhook-channel":
				cmdOptions := cmdOptions[0].Options
				switch cmdOptions[0].Name {
				case "set":
					updateResult, err := p.upsertOneWebhookNotifyChannel(ddtvNotifyPo{
						AdminDiscordUserID: i.Interaction.Member.User.ID,
						GuildID:            i.Interaction.GuildID,
						NotifyChannelID:    i.Interaction.ChannelID,
					})
					if err != nil {
						core2.Logger.Warnf("Error inserting webhook channel record: %v", err)
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
						core2.Logger.Warnf("Error deleting webhook channel record: %v", err)
						return err
					}
					if deleteResult.DeletedCount > 0 {
						p.DiscordService.InteractionRespond(i.Interaction, "webhook channel removed!")
					} else {
						p.DiscordService.InteractionRespond(i.Interaction, "not a webhook channel yet!")
					}
				}
			}
		}
	}

	return nil
}

func (p *DDTVPlugin) Init(reg *core2.ServiceRegistry) error {
	// DiscordService is a MUST have. return error if not found.
	if err := reg.FetchService(&p.DiscordService); err != nil {
		return err
	}
	// DataService is also a MUST have. return error if not found.
	if err := reg.FetchService(&p.DataService); err != nil {
		return err
	}
	// ddtvService is not used to perform actions actively in the plugin, so not imported.

	p.AcceptedTriggerTypes = []core2.TriggerType{core2.TriggerTypeDiscord, core2.TriggerTypeDDTV}
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
		},
	})

	formattedHelpSetNotifChannel := `*ddtv webhook-channel set*: /ddtv webhook-channel set
Set current channel as a DDTV webhook notification channel.
Dalian will send a message for every incoming DDTV webhook received in this channel`
	formattedHelpRemoveNotifChannel := `ddtv webhook-channel remove*: /ddtv webhook-channel remove
Remove current channel from DDTV webhook notification channel list.
Dalian will no longer send a message for every incoming DDTV webhook received in this channel`

	p.IDisrocdHelper = discord2.GenerateHelper(discord2.HelperConfig{
		PluginHelp: "Helper support for Dalian.",
		CommandHelps: []discord2.CommandHelp{
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

func (p *DDTVPlugin) Trigger(trigger core2.Trigger) {
	if !p.AcceptTrigger(trigger.Type) {
		return
	}
	switch trigger.Type {
	case core2.TriggerTypeDiscord:
		// do NOT accept  messageCreate event
		// do something...
		dcEvent := discord2.UnboxEvent(trigger)
		switch dcEvent.EventType {
		// only accepting interactionCreate for discord trigers
		case discord2.EventTypeInteractionCreate:
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
	case core2.TriggerTypeDDTV:
		// do ddtv webhook thing
		webhook := ddtv2.UnboxEvent(trigger).WebHook
		// if hooktype is channel, restrict non-record channel message to online.
		// todo: add a config option to control this behavior
		if webhook.UserInfo.UID != 0 && !webhook.RoomInfo.IsAutoRec && webhook.Type != ddtv2.HookStartLive {
			return
		}
		p.notifyDDTVWebhookToChannels(webhook)
	default:
		core2.Logger.Warnf(core2.LogPromptUnknownTrigger, trigger.Type)
	}
}

func (p *DDTVPlugin) notifyDDTVWebhookToChannels(webhook ddtv2.WebHook) {
	channelIDs, err := p.findDDTVWebhookNotifyChannels()
	if err != nil {
		core2.Logger.Warnf("Retrieve webhook channels failed!: %v", err)
		return
	}
	for _, channelID := range channelIDs {
		_, err := p.DiscordService.ChannelMessageSendEmbed(channelID, webhook.DigestEmbed())
		if err != nil {
			core2.Logger.Warnf("Embed sent failed: %v", err)
			b, _ := json.Marshal(webhook)
			p.DiscordService.ChannelMessageSendCodeBlock(channelID, string(b))
			return
		}
	}

}

type ddtvNotifyPo struct {
	BsonID             primitive.ObjectID `bson:"_id,omitempty"`
	AdminDiscordUserID string             `bson:"admin_dc_user_id"`
	GuildID            string             `bson:"guild_id"`
	NotifyChannelID    string             `bson:"notify_channel_id"`
}

func (p *DDTVPlugin) getCollection() *mongo.Collection {
	return p.DataService.GetCollection("ddtv_notify_channels")
}

func (p *DDTVPlugin) upsertOneWebhookNotifyChannel(po ddtvNotifyPo) (*mongo.UpdateResult, error) {
	rawResult := p.DataService.UpdateOne(bson.D{{"$set", data2.ToBsonDocForce(po)}}, p.getCollection(), context.Background(), bson.M{"notify_channel_id": po.NotifyChannelID}, options.Update().SetUpsert(true))
	return rawResult.UpdateResult(), rawResult.Err()
}

func (p *DDTVPlugin) deleteOneWebhookNotifyChannel(channelID string) (*mongo.DeleteResult, error) {
	rawResult := p.DataService.DeleteOne(p.getCollection(), context.Background(), bson.M{"notify_channel_id": channelID})
	return rawResult.DeleteResult(), rawResult.Err()
}

func (p *DDTVPlugin) findDDTVWebhookNotifyChannels() (channels []string, er error) {
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

func NewDDTVPlugin(reg *core2.ServiceRegistry) core2.IPlugin {
	var ddtvPlugin DDTVPlugin
	if err := (&ddtvPlugin).Init(reg); err != nil && errors.As(err, &core2.ErrServiceFetchUnknownService) {
		core2.Logger.Panicf("DDTV plugin MUST have all required service(s) injected!")
		panic("DDTV plugin initialization failed.")
	}
	return &ddtvPlugin
}
