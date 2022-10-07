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

func (cm *PingCommand) MatchMessage(message *discordgo.MessageCreate) (bool, bool) {
	matchStatus, _ := cm.MatchText(message.Content)
	return matchStatus, true
}

func (cm *PingCommand) New() {
	cm.Name = "ping"
	cm.Identifiers = []string{"ping"}
}

func (cm *PingCommand) DoMessage(m *discordgo.MessageCreate) error {
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
