package commands

import (
	"dalian-bot/internal/pkg/clients"
	"fmt"
	"github.com/bwmarrin/discordgo"
	"strings"
)

type ListCommand struct {
	Command
	PlainCommand
	SlashCommand
}

func (cm *ListCommand) MatchInteraction(i *discordgo.InteractionCreate) (isMatched bool) {
	if i.ApplicationCommandData().Name == cm.AppCommand.Name {
		return true
	}
	return false
}

func (cm *ListCommand) DoInteraction(i *discordgo.InteractionCreate) (err error) {

	optionsMap := cm.ParseOptionsMap(i.ApplicationCommandData().Options)
	names := make([]string, 0, len(CommandByName))
	if option, ok := optionsMap["qualifier"]; ok {
		for k := range CommandByName {
			if strings.Contains(k, option.StringValue()) {
				names = append(names, k)
			}
		}
	} else {
		for k := range CommandByName {
			names = append(names, k)
		}
	}
	clients.DgSession.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("Commands: %v", names),
		},
	})
	return nil
}

func (cm *ListCommand) MatchMessage(message *discordgo.MessageCreate) (bool, bool) {
	matchStatus, _ := cm.MatchText(message.Content)
	return matchStatus, true
}

func (cm *ListCommand) New() {
	cm.Name = "list-command"
	cm.Identifiers = []string{"list", "l"}
	cm.AppCommand = &discordgo.ApplicationCommand{
		Name:        "list-command",
		Description: "List the name of all available commands.",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "qualifier",
				Description: "Online commands include the string will be shown",
				Required:    false,
			},
		},
	}
}

func (cm *ListCommand) DoMessage(m *discordgo.MessageCreate) error {
	names := make([]string, 0, len(CommandByName))
	for k := range CommandByName {
		names = append(names, k)
	}
	clients.DgSession.ChannelMessageSend(m.ChannelID, fmt.Sprintf("**Listing Registered Commands**\r\n%v", names))
	return nil
}

func init() {
	var lc ListCommand
	lc.New()
	RegisterCommand(&lc)
}
