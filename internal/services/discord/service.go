package discord

import (
	core2 "dalian-bot/internal/core"
	"fmt"
	"github.com/bwmarrin/discordgo"
	"io"
	"log"
	"reflect"
	"sync"
)

// ChannelMessageSend A wrapper of discordgo ChannelMessageSend function.
func (s *Service) ChannelMessageSend(channelID, content string) (*discordgo.Message, error) {
	return s.Session.ChannelMessageSend(channelID, content)
}

// ChannelMessageSendEmbed A wrapper of discordgo ChannelMessageSendEmbed function.
func (s *Service) ChannelMessageSendEmbed(channelID string, embed *discordgo.MessageEmbed) (*discordgo.Message, error) {
	return s.Session.ChannelMessageSendEmbed(channelID, embed)
}

// ChannelMessageReportError Report the error as a plain message to given gild channel.
func (s *Service) ChannelMessageReportError(channelID string, error error) (*discordgo.Message, error) {
	return s.ChannelMessageSend(channelID, error.Error())
}

// InteractionRespondComplex Basic wrapper for discordgo.InteractionRespond.
func (s *Service) InteractionRespondComplex(i *discordgo.Interaction, resp *discordgo.InteractionResponse) error {
	return s.Session.InteractionRespond(i, resp)
}

// InteractionRespondEmbed Shortcut method for fast reply including a MessageEmbed.
func (s *Service) InteractionRespondEmbed(i *discordgo.Interaction, embed *discordgo.MessageEmbed, components []discordgo.MessageComponent) error {
	return s.InteractionRespondComplex(i, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds:     []*discordgo.MessageEmbed{embed},
			Components: components,
		},
	})
}

// InteractionRespond Shortcut method for a simple message reply.
func (s *Service) InteractionRespond(i *discordgo.Interaction, content string) error {
	return s.InteractionRespondComplex(i, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: content,
		},
	})
}

func (s *Service) InteractionResponse(i *discordgo.Interaction) (*discordgo.Message, error) {
	return s.Session.InteractionResponse(i)
}

func (s *Service) InteractionResponseEdit(i *discordgo.Interaction, newresp *discordgo.WebhookEdit) error {
	return s.InteractionRespondComplex(i, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Content:    *newresp.Content,
			Components: *newresp.Components,
			Embeds:     *newresp.Embeds,
		},
	})
}

func (s *Service) InteractionRespondEditFromMessage(i *discordgo.Interaction, msg *discordgo.Message) error {
	tempWebhookEdit := &discordgo.WebhookEdit{
		Content:    &msg.Content,
		Components: &msg.Components,
		Embeds:     &msg.Embeds,
		//files is not compatible with message. use a new message for edited file.
		//Files:         nil,
		//the same goes AllowedMentions
		//AllowedMentions: nil ,
	}
	return s.InteractionResponseEdit(i, tempWebhookEdit)
}

// ChannelFileSend send a file to given guild channel.
// channelID the id of a channel
// name the display filename to be sent to discord
// r the io reader containing a valid file struct
func (s *Service) ChannelFileSend(channelID, name string, r io.Reader) error {
	if _, err := s.Session.ChannelFileSend(channelID, name, r); err != nil {
		log.Println("Error sending discord message: ", err)
		return err
	}
	return nil
}

type appCommands []*discordgo.ApplicationCommand

func (cmds appCommands) delete(cmd *discordgo.ApplicationCommand) appCommands {
	j := 0
	for _, val := range cmds {
		if val.Name != cmd.Name {
			cmds[j] = val
		}
	}
	return cmds[:j]
}

type Service struct {
	ServiceConfig
	core2.TriggerableEmbedUtil
	Session              *discordgo.Session
	registeredCommands   appCommands
	DiscordAccountConfig core2.MessengerConfig
}

func (s *Service) Init(reg *core2.ServiceRegistry) error {
	reg.RegisterService(s)
	return nil
}

func (s *Service) Start(wg *sync.WaitGroup) {
	/* Setup DiscordGo Session */
	discordSession, err := discordgo.New("Bot " + s.Token)
	if err != nil {
		core2.Logger.Panicf("error creating Discord session:%v", err)
	}
	discordSession.Identify.Intents = discordgo.IntentGuildMessages
	err = discordSession.Open()
	if err != nil {
		core2.Logger.Panicf("error opening Discord connection:%v", err)
	}
	discordSession.AddHandler(s.messageCreate)
	discordSession.AddHandler(s.interactionCreate)
	s.Session = discordSession
	//Todo: move it to config file
	s.DiscordAccountConfig = core2.MessengerConfig{
		Prefix:    "$",
		Separator: "$",
		BotID:     s.Session.State.User.ID,
	}
	core2.Logger.Debugf("Service [%s] is now online.", reflect.TypeOf(s))
	wg.Done()
}

func (s *Service) Stop(wg *sync.WaitGroup) error {
	s.DisposeAllSlashCommand()
	core2.Logger.Debugf("Service [%s] is successfully closed.", reflect.TypeOf(s))
	wg.Done()
	return nil
}

func (s *Service) Status() error {
	//TODO implement me
	panic("implement me")
}

func (s *Service) Name() string {
	return "discord"
}

func (s *Service) messageCreate(_ *discordgo.Session, m *discordgo.MessageCreate) {
	t := core2.Trigger{
		Type: core2.TriggerTypeDiscord,
		Event: Event{
			EventType:     EventTypeMessageCreate,
			MessageCreate: m,
		},
	}
	s.TriggerChan <- t
}

func (s *Service) interactionCreate(_ *discordgo.Session, i *discordgo.InteractionCreate) {
	//debugging
	fmt.Printf("Int: %s:%s:%v \r\n", i.Member.User.Username, i.Data, i.Message)
	t := core2.Trigger{
		Type: core2.TriggerTypeDiscord,
		Event: Event{
			EventType:         EventTypeInteractionCreate,
			InteractionCreate: i,
		},
	}
	s.TriggerChan <- t
}

func (s *Service) DisposeAllSlashCommand() error {
	for _, v := range s.registeredCommands {
		err := s.Session.ApplicationCommandDelete(s.Session.State.User.ID, "", v.ID)
		if err != nil {
			log.Panicf("Cannot delete '%v' command: %v", v.Name, err)
		} else {
			core2.Logger.Debugf("disposed slash command: %s", v.Name)
		}
	}
	return nil
}

func (s *Service) DisposeSlashCommand(command core2.INewPlugin) error {
	if slash, ok := command.(ISlashCommand); ok {
		for _, cmd := range slash.GetAppCommandsMap() {
			s.registeredCommands = s.registeredCommands.delete(cmd)
			s.Session.ApplicationCommandDelete(s.Session.State.User.ID, "", cmd.ID)
		}
	}
	return nil
}

func (s *Service) RegisterSlashCommand(plugin core2.INewPlugin) error {
	if slash, ok := plugin.(ISlashCommandNew); ok {
		for _, cmd := range slash.GetAppCommandsMap() {
			cmd, err := s.Session.ApplicationCommandCreate(s.Session.State.User.ID, "", cmd)
			if err != nil {
				log.Panicf("Cannot register Command %v", err)
			} else {
				core2.Logger.Debugf("Installed slash command: %s", cmd.Name)
				s.registeredCommands = append(s.registeredCommands, cmd)
			}
		}
	} else {
		core2.Logger.Errorf("NOT A SLASH CMD")
	}
	core2.Logger.Debugf("Registered slash command for plugin:%s", plugin.GetName())
	return nil
}

func (s *Service) IsGuildMessageFromBotOrSelf(m *discordgo.Message) bool {
	// Ignore all messages created by the bot itself
	// This isn't required in this specific example, but it's a good practice.
	if m.Author.ID == s.DiscordAccountConfig.BotID {
		return true
	}
	// Ignore chain requests from other bots
	if m.Author.Bot {
		return true
	}
	return false
}

type ServiceConfig struct {
	Token string
}
