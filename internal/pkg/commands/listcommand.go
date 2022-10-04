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

func (m *ListCommand) MatchMessage(message *discordgo.Message) bool {
	matchStatus, _ := m.MatchText(message.Content)
	return matchStatus
}

func (m *ListCommand) New() {
	m.Name = "list"
	m.Identifiers = []string{"list", "l"}
}

func (m *ListCommand) Match(a ...any) bool {
	msg, isMsgCreate := a[0].(*discordgo.MessageCreate)
	if !isMsgCreate {
		return false
	}
	return m.MatchMessage(msg.Message)
}

func (m *ListCommand) Do(a ...any) error {
	msg := a[0].(*discordgo.MessageCreate)
	names := make([]string, 0, len(CommandByName))
	for k := range CommandByName {
		names = append(names, k)
	}
	clients.DgSession.ChannelMessageSend(msg.ChannelID, fmt.Sprintf("**Listing Registered Commands**\r\n%v", names))
	return nil
}

func init() {
	var lc ListCommand
	lc.New()
	RegisterCommand(&lc)
}
