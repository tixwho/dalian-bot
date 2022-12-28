package plugins

import (
	"dalian-bot/internal/pkg/core"
	"dalian-bot/internal/pkg/services/discord"
	"errors"
	"fmt"
	"github.com/bwmarrin/discordgo"
)

type PingPlugin struct {
	core.Plugin
	DiscordService *discord.Service
	core.StartWithMatchUtil
	discord.SlashCommand
	discord.IDisrocdHelper
}

func (p *PingPlugin) DoNamedInteraction(_ *core.Bot, i *discordgo.InteractionCreate) (err error) {
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
	return nil
}

func (p *PingPlugin) DoMessage(_ *core.Bot, m *discordgo.MessageCreate) error {
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

	p.AcceptedTriggerTypes = []core.TriggerType{core.TriggerTypeDiscord}
	p.Name = "ping"
	p.Identifiers = []string{"ping"}
	p.AppCommandsMap = make(map[string]*discordgo.ApplicationCommand)
	p.AppCommandsMap.RegisterCommand(&discordgo.ApplicationCommand{
		Name:        "ping",
		Description: "Ping command for Dalian",
	})

	formattedPingHelp := fmt.Sprintf("*Call*: /ping,%sping\rrespond a \"pong\"", p.DiscordService.DiscordAccountConfig.Prefix)
	p.IDisrocdHelper = discord.GenerateHelper(discord.HelperConfig{
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
		p.DoMessage(trigger.Bot, discordEvent.MessageCreate)
	case discord.EventTypeInteractionCreate:
		p.DoNamedInteraction(trigger.Bot, discordEvent.InteractionCreate)
	default:
		core.Logger.Warnf("This should NOT reach!")
	}
}

func NewPingPlugin(reg *core.ServiceRegistry) core.INewPlugin {
	var ping PingPlugin
	if err := (&ping).Init(reg); err != nil && errors.As(err, &core.ErrServiceFetchUnknownService) {
		core.Logger.Panicf("Ping plugin MUST have all required service(s) injected!")
		panic("Ping plugin initialization failed.")
	}
	return &ping
}
