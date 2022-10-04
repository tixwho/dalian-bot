package commands

import (
	"dalian-bot/internal/pkg/clients"
	"fmt"
	"github.com/bwmarrin/discordgo"
)

type PingCommand struct {
	Command
	PlainCommand
}

func (cm *PingCommand) MatchMessage(message *discordgo.Message) bool {
	matchStatus, _ := cm.MatchText(message.Content)
	return matchStatus
}

func (cm *PingCommand) New() {
	cm.Name = "ping"
	cm.Identifiers = []string{"ping"}
}

func (cm *PingCommand) Match(a ...any) bool {
	m, isMsgCreate := a[0].(*discordgo.MessageCreate)
	if !isMsgCreate {
		return false
	}
	return cm.MatchMessage(m.Message)
}

func (cm *PingCommand) Do(a ...any) error {
	//safely assume that it's a message create event
	m := a[0].(*discordgo.MessageCreate)
	_, err := clients.DgSession.ChannelMessageSend(m.ChannelID, "Pong!")
	if err != nil {
		fmt.Println("error found:", err)
		return err
	}
	return nil
}

func init() {
	var pc PingCommand
	pc.New()
	RegisterCommand(&pc)
}
