package commands

import (
	"dalian-bot/internal/pkg/clients"
	"fmt"
	"github.com/bwmarrin/discordgo"
)

type ListCommand struct {
	Command
	PlainCommand
}

func (cm *ListCommand) MatchMessage(message *discordgo.MessageCreate) (bool, bool) {
	matchStatus, _ := cm.MatchText(message.Content)
	return matchStatus, true
}

func (cm *ListCommand) New() {
	cm.Name = "list"
	cm.Identifiers = []string{"list", "l"}
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
