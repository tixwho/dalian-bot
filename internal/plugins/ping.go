package plugins

import (
	core2 "dalian-bot/internal/core"
	discord2 "dalian-bot/internal/services/discord"
	"errors"
	"fmt"
	"github.com/bwmarrin/discordgo"
)

// PingPlugin Basic ping support.
// Discord: can be trigggered by $ping, /ping
type PingPlugin struct {
	core2.Plugin
	DiscordService *discord2.Service
	core2.StartWithMatchUtil
	discord2.SlashCommand
	discord2.IDisrocdHelper
}

func (p *PingPlugin) DoNamedInteraction(_ *core2.Bot, i *discordgo.InteractionCreate) (err error) {
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

func (p *PingPlugin) DoMessage(_ *core2.Bot, m *discordgo.MessageCreate) error {
	if matched, _ := p.StartWithMatchUtil.MatchText(m.Content, p.DiscordService.DiscordAccountConfig); matched {
		p.DiscordService.ChannelMessageSend(m.ChannelID, "Pong!")
	}
	return nil
}

func (p *PingPlugin) Init(reg *core2.ServiceRegistry) error {
	//discordService is a MUST have. return error if not found.
	if err := reg.FetchService(&p.DiscordService); err != nil {
		return err
	}

	p.AcceptedTriggerTypes = []core2.TriggerType{core2.TriggerTypeDiscord}
	p.Name = "ping"
	p.Identifiers = []string{"ping"}
	p.AppCommandsMap = make(map[string]*discordgo.ApplicationCommand)
	p.AppCommandsMap.RegisterCommand(&discordgo.ApplicationCommand{
		Name:        "ping",
		Description: "Ping command for Dalian",
	})

	formattedPingHelp := fmt.Sprintf("*Call*: /ping,%sping\rrespond a \"pong\"", p.DiscordService.DiscordAccountConfig.Prefix)
	p.IDisrocdHelper = discord2.GenerateHelper(discord2.HelperConfig{
		PluginHelp: "Basic ping command for dalian over Discord.",
		CommandHelps: []discord2.CommandHelp{
			{
				Name:          "ping",
				FormattedHelp: formattedPingHelp,
			},
		},
	})

	return p.DiscordService.RegisterSlashCommand(p)
}

func (p *PingPlugin) Trigger(trigger core2.Trigger) {
	if !p.AcceptTrigger(trigger.Type) {
		return
	}
	discordEvent := discord2.UnboxEvent(trigger) // not checking because only accept discord.
	switch discordEvent.EventType {
	case discord2.EventTypeMessageCreate:
		if p.DiscordService.IsGuildMessageFromBotOrSelf(discordEvent.MessageCreate.Message) {
			return
		}
		p.DoMessage(trigger.Bot, discordEvent.MessageCreate)
	case discord2.EventTypeInteractionCreate:
		p.DoNamedInteraction(trigger.Bot, discordEvent.InteractionCreate)
	default:
		core2.Logger.Warnf("This should NOT reach!")
	}
}

func NewPingPlugin(reg *core2.ServiceRegistry) core2.INewPlugin {
	var ping PingPlugin
	if err := (&ping).Init(reg); err != nil && errors.As(err, &core2.ErrServiceFetchUnknownService) {
		core2.Logger.Panicf("Ping plugin MUST have all required service(s) injected!")
		panic("Ping plugin initialization failed.")
	}
	return &ping
}
