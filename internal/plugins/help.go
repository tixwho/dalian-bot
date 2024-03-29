package plugins

import (
	"dalian-bot/internal/core"
	"dalian-bot/internal/services/discord"
	"errors"
	"fmt"
	"github.com/bwmarrin/discordgo"
)

// HelpPlugin Plugin for collecting help info of registered commands.
// Discord: can be triggered by `$help` or `/help`
type HelpPlugin struct {
	core.Plugin                               // basic plugin basetype
	DiscordService           *discord.Service // currently support discord
	core.StartWithMatchUtil                   // plain message support
	core.ArgParseUtil                         // command argument support
	discord.SlashCommandUtil                  // discord slash command support
	discord.IDiscordHelper                    // the plugin itself needs to display help texts.
}

// DoNamedInteraction `/help [command-name]` support
func (p *HelpPlugin) DoNamedInteraction(b *core.Bot, i *discordgo.InteractionCreate) (err error) {
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
func parseHelpText(b *core.Bot, commandName string) string {
	helpText := ""
	if commandName == "" {
		helpText += "**Available Commands**"
	}
	for _, plugin := range b.PluginRegistry.GetPlugins() {
		if helpPlugin, ok := plugin.(discord.IDiscordHelper); ok {
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

// DoPlainMessage `$help [command-name]` support
func (p *HelpPlugin) DoPlainMessage(b *core.Bot, m *discordgo.MessageCreate) (err error) {
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

func (p *HelpPlugin) Init(reg *core.ServiceRegistry) error {
	//discordService is a MUST have. return error if not found.
	if err := reg.FetchService(&p.DiscordService); err != nil {
		return err
	}
	p.AcceptedTriggerTypes = []core.TriggerType{discord.TriggerTypeDiscord}
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
	p.IDiscordHelper = discord.GenerateHelper(discord.HelperConfig{
		PluginHelp: "HelperUtil support for Dalian.",
		CommandHelps: []discord.CommandHelp{
			{
				Name:          "help",
				FormattedHelp: formattedHelpHelp,
			},
		},
	})
	return p.DiscordService.RegisterSlashCommand(p)
}

func (p *HelpPlugin) Trigger(trigger core.Trigger) {
	if !p.AcceptTrigger(trigger.Type) {
		return
	}
	discordEvent := discord.UnboxEvent(trigger)
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

func NewHelpPlugin(reg *core.ServiceRegistry) core.IPlugin {
	var help HelpPlugin
	if err := (&help).Init(reg); err != nil && errors.As(err, &core.ErrServiceFetchUnknownService) {
		core.Logger.Panicf("Help plugin MUST have all required service(s) injected!")
		panic("Help plugin initialization failed.")
	}
	return &help
}
