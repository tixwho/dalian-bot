package plugins

import (
	"dalian-bot/internal/pkg/core"
	"dalian-bot/internal/pkg/services/data"
	"dalian-bot/internal/pkg/services/ddtv"
	"dalian-bot/internal/pkg/services/discord"
	"errors"
	"github.com/bwmarrin/discordgo"
)

type DDTVPlugin struct {
	core.Plugin
	DiscordService *discord.Service
	DataService    *data.Service
	discord.SlashCommand
	discord.IDisrocdHelper
}

const (
	NotifChannelID = "920708761600028713" // todo: use database instead
)

func (p *DDTVPlugin) DoNamedInteraction(b *core.Bot, i *discordgo.InteractionCreate) (err error) {
	//TODO implement me
	panic("implement me")
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

	p.AcceptedTriggerTypes = []core.TriggerType{core.TriggerTypeDiscord, core.TriggerTypeDDTV}
	p.Name = "ddtv"
	p.AppCommandsMap = make(map[string]*discordgo.ApplicationCommand)
	p.AppCommandsMap.RegisterCommand(&discordgo.ApplicationCommand{
		Name:        "ddtv-set-notification-channel",
		Description: "Set this channel as a DDTV webhook notification channel.",
		/* No access-token validation for now
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type: discordgo.ApplicationCommandOptionString,
				Name: "command-name",
				//late init, replace %s with separator
				Description: "Name of the command.",
				Required:    false,
			},
		},
		*/
	})
	p.AppCommandsMap.RegisterCommand(&discordgo.ApplicationCommand{
		Name:        "ddtv-remove-notification-channel",
		Description: "Remove this channel from DDTV webhook notification channel list.",
		/* No access-token validation for now
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type: discordgo.ApplicationCommandOptionString,
				Name: "command-name",
				//late init, replace %s with separator
				Description: "Name of the command.",
				Required:    false,
			},
		},
		*/
	})

	formattedHelpSetNotifChannel := `*DDTV set-notification-channel*: /DDTV set-notification-channel
Set current channel as a DDTV webhook notification channel.
Dalian will send a message for every incoming DDTV webhook received in this channel`
	formattedHelpRemoveNotifChannel := `*DDTV remove-notification-channel*: /DDTV set-notification-channel
Remove current channel from DDTV webhook notification channel list.
Dalian will no longer send a message for every incoming DDTV webhook received in this channel`

	p.IDisrocdHelper = discord.GenerateHelper(discord.HelperConfig{
		PluginHelp: "Helper support for Dalian.",
		CommandHelps: []discord.CommandHelp{
			{
				Name:          "ddtv-set-notification-channel",
				FormattedHelp: formattedHelpSetNotifChannel,
			},
			{
				Name:          "ddtv-remove-notification-channel",
				FormattedHelp: formattedHelpRemoveNotifChannel,
			},
		},
	})
	return p.DiscordService.RegisterSlashCommand(p)
}

func (p *DDTVPlugin) Trigger(trigger core.Trigger) {
	//TODO implement me
	if !p.AcceptTrigger(trigger.Type) {
		return
	}
	switch trigger.Type {
	case core.TriggerTypeDiscord:
		// do NOT accept  messageCreate event
		// do something...
		dcEvent := discord.UnboxEvent(trigger)
		switch dcEvent.EventType {
		// only accepting interactionCreate for discord trigers
		case discord.EventTypeInteractionCreate:
			switch dcEvent.InteractionCreate.Type {
			case discordgo.InteractionApplicationCommand:
				// slash command
			default:
				// not accepting other type of interaction for this plugin
				return
			}
		default:
			// does not handle messageCreate or anything like that.
			return
		}
	case core.TriggerTypeDDTV:
		// do ddtv webhook thing
		ddtvEvent := ddtv.UnboxEvent(trigger)
		p.notifyDDTVWebhookToChannel(NotifChannelID, ddtvEvent.WebHook) // todo: use database instead
	default:
		core.Logger.Warnf(core.LogPromptUnknownTrigger, trigger.Type)
	}
}

func (p *DDTVPlugin) notifyDDTVWebhookToChannel(channelID string, webhook ddtv.WebHook) {
	_, err := p.DiscordService.ChannelMessageSendEmbed(channelID, webhook.DigestEmbed())
	if err != nil {
		core.Logger.Warnf("Embed sent failed: %v", err)
		return
	}
	core.Logger.Debugf("Webhook Info Sent to Channel [%s]", channelID) // todo: remove.
}

func NewDDTVPlugin(reg *core.ServiceRegistry) core.INewPlugin {
	var ddtvPlugin DDTVPlugin
	if err := (&ddtvPlugin).Init(reg); err != nil && errors.As(err, &core.ErrServiceFetchUnknownService) {
		core.Logger.Panicf("DDTV plugin MUST have all required service(s) injected!")
		panic("DDTV plugin initialization failed.")
	}
	return &ddtvPlugin
}
