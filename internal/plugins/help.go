package plugins

import (
	core2 "dalian-bot/internal/core"
	discord2 "dalian-bot/internal/services/discord"
	"errors"
	"fmt"
	"github.com/bwmarrin/discordgo"
)

// HelpPlugin Plugin for collecting help info of registered commands.
// For Discord: can be triggered by $help or /help
type HelpPlugin struct {
	core2.Plugin                               // basic plugin basetype
	DiscordService           *discord2.Service // currently support discord
	core2.StartWithMatchUtil                   // plain message support
	core2.ArgParseUtil                         // command argument support
	discord2.SlashCommand                      // discord slash command support
	discord2.IDisrocdHelper                    // the plugin itself needs to display help texts.
}

// DoNamedInteraction `/help [command-name]` support
func (p *HelpPlugin) DoNamedInteraction(b *core2.Bot, i *discordgo.InteractionCreate) (err error) {
	switch i.Type {
	case discordgo.InteractionApplicationCommand:
		if match, name := p.DefaultMatchCommand(i); match {
			switch name {
			case "help":
				optionsMap := p.ParseOptionsMap(i.ApplicationCommandData().Options)
				if commandName, ok := optionsMap["command-name"]; ok {
					p.DiscordService.InteractionRespond(i.Interaction, parseHelpText(b, commandName.StringValue()))
				} else {
					p.DiscordService.InteractionRespond(i.Interaction, parseHelpText(b, ""))
				}
			}
		}
	}

	return nil
}

// parseHelpText browse through all plugins registered with bot and match help texts available.
func parseHelpText(b *core2.Bot, commandName string) string {
	helpText := ""
	if commandName == "" {
		helpText += "**Available Commands**"
	}
	for _, plugin := range b.PluginRegistry.GetPlugins() {
		if helpPlugin, ok := plugin.(discord2.IDisrocdHelper); ok {
			if commandName == "" {
				//general help
				helpText += "\r" + helpPlugin.DiscordPluginHelp(plugin.GetName())
			} else {
				//specific help
				if text := helpPlugin.DiscordCommandHelp(commandName); text != "" {
					helpText += text
					break
				}
			}
		}
	}
	//no specific command matchted
	if helpText == "" {
		helpText += fmt.Sprintf("Cant find help of command %s.", commandName)
	}
	return helpText
}

// DoMessage `$help [command-name]` support
func (p *HelpPlugin) DoMessage(b *core2.Bot, m *discordgo.MessageCreate) (err error) {
	if matched, _ := p.StartWithMatchUtil.MatchText(m.Content, p.DiscordService.DiscordAccountConfig); matched {
		args := p.ArgParseUtil.SeparateArgs(m.Content, p.DiscordService.DiscordAccountConfig.Separator)
		if len(args) == 1 {
			p.DiscordService.ChannelMessageSend(m.ChannelID, parseHelpText(b, ""))
		} else {
			p.DiscordService.ChannelMessageSend(m.ChannelID, parseHelpText(b, args[1]))
		}
	}
	return nil
}

func (p *HelpPlugin) Init(reg *core2.ServiceRegistry) error {
	//discordService is a MUST have. return error if not found.
	if err := reg.FetchService(&p.DiscordService); err != nil {
		return err
	}
	p.AcceptedTriggerTypes = []core2.TriggerType{core2.TriggerTypeDiscord}
	p.Name = "help"
	p.Identifiers = []string{"help"}
	p.AppCommandsMap = make(map[string]*discordgo.ApplicationCommand)
	p.AppCommandsMap.RegisterCommand(&discordgo.ApplicationCommand{
		Name:        "help",
		Description: "Display help messages.",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type: discordgo.ApplicationCommandOptionString,
				Name: "command-name",
				//late init, replace %s with separator
				Description: "Name of the command.",
				Required:    false,
			},
		},
	})

	// Help text for HelpPlugin itself.
	formattedHelpHelp := fmt.Sprintf(
		`*Call*: /help,%shelp
*Optional Argument*: [command-name]
Display the help message.
If command-name not provided, list the names of all available commands; Otherwise, provide detailed explaination of the specific command.`,
		p.DiscordService.DiscordAccountConfig.Prefix)
	p.IDisrocdHelper = discord2.GenerateHelper(discord2.HelperConfig{
		PluginHelp: "Helper support for Dalian.",
		CommandHelps: []discord2.CommandHelp{
			{
				Name:          "help",
				FormattedHelp: formattedHelpHelp,
			},
		},
	})
	return p.DiscordService.RegisterSlashCommand(p)
}

func (p *HelpPlugin) Trigger(trigger core2.Trigger) {
	if !p.AcceptTrigger(trigger.Type) {
		return
	}
	discordEvent := discord2.UnboxEvent(trigger)
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

func NewHelpPlugin(reg *core2.ServiceRegistry) core2.INewPlugin {
	var help HelpPlugin
	if err := (&help).Init(reg); err != nil && errors.As(err, &core2.ErrServiceFetchUnknownService) {
		core2.Logger.Panicf("Help plugin MUST have all required service(s) injected!")
		panic("Help plugin initialization failed.")
	}
	return &help
}
