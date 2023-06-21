package plugins

import (
	"dalian-bot/internal/core"
	"dalian-bot/internal/services/discord"
	"errors"
	"fmt"
	"github.com/bwmarrin/discordgo"
)

// PingPlugin Basic ping support.
// Discord: can be trigggered by `$ping`, `/ping`
type PingPlugin struct {
	core.Plugin
	DiscordService *discord.Service
	core.StartWithMatchUtil
	discord.SlashCommandUtil
	discord.IDiscordHelper
}

func (p *PingPlugin) DoNamedInteraction(_ *core.Bot, i *discordgo.InteractionCreate) (err error) {
	switch i.Type {
	case discordgo.InteractionApplicationCommand:
		if match, name := p.DefaultMatchCommand(i); match {
			switch name {
			case "ping":
				p.DiscordService.ChannelMessageSend(i.ChannelID, "pong response not using interaction!")
				p.DiscordService.InteractionRespondComplex(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: "pong response with discord interaction!!",
					},
				})
			}
		}
	}
	return nil
}

func (p *PingPlugin) DoPlainMessage(_ *core.Bot, m *discordgo.MessageCreate) error {
	if matched, _ := p.StartWithMatchUtil.MatchText(m.Content, p.DiscordService.DiscordAccountConfig); matched {
		p.DiscordService.ChannelMessageSend(m.ChannelID, "Pong!")
	}
	return nil
}

func (p *PingPlugin) Init(reg *core.ServiceRegistry) error {
	//discordService is a MUST have. return error if not found.
	if err := reg.FetchService(&p.DiscordService); err != nil {
		return err
	}

	p.AcceptedTriggerTypes = []core.TriggerType{discord.TriggerTypeDiscord}
	p.Name = "ping"
	p.Identifiers = []string{"ping"}
	p.AppCommandsMap = make(map[string]*discordgo.ApplicationCommand)
	p.AppCommandsMap.RegisterCommand(&discordgo.ApplicationCommand{
		Name:        "ping",
		Description: "Ping command for Dalian",
	})

	formattedPingHelp := fmt.Sprintf("*Call*: /ping,%sping\rrespond a \"pong\"", p.DiscordService.DiscordAccountConfig.Prefix)
	p.IDiscordHelper = discord.GenerateHelper(discord.HelperConfig{
		PluginHelp: "Basic ping command for dalian over Discord.",
		CommandHelps: []discord.CommandHelp{
			{
				Name:          "ping",
				FormattedHelp: formattedPingHelp,
			},
		},
	})

	return p.DiscordService.RegisterSlashCommand(p)
}

func (p *PingPlugin) Trigger(trigger core.Trigger) {
	if !p.AcceptTrigger(trigger.Type) {
		return
	}
	discordEvent := discord.UnboxEvent(trigger) // not checking because only accept discord.
	switch discordEvent.EventType {
	case discord.EventTypeMessageCreate:
		if p.DiscordService.IsGuildMessageFromBotOrSelf(discordEvent.MessageCreate.Message) {
			return
		}
		p.DoPlainMessage(trigger.Bot, discordEvent.MessageCreate)
	case discord.EventTypeInteractionCreate:
		p.DoNamedInteraction(trigger.Bot, discordEvent.InteractionCreate)
	default:
		core.Logger.Warnf("This should NOT reach!")
	}
}

func NewPingPlugin(reg *core.ServiceRegistry) core.IPlugin {
	var ping PingPlugin
	if err := (&ping).Init(reg); err != nil && errors.As(err, &core.ErrServiceFetchUnknownService) {
		core.Logger.Panicf("Ping plugin MUST have all required service(s) injected!")
		panic("Ping plugin initialization failed.")
	}
	return &ping
}
