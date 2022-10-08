package commands

import (
	"dalian-bot/internal/pkg/clients"
	"dalian-bot/internal/pkg/discord"
	"fmt"
	"github.com/bwmarrin/discordgo"
)

type PingCommand struct {
	Command
	PlainCommand
	SlashCommand
}

func (cm *PingCommand) MatchInteraction(i *discordgo.InteractionCreate) (isMatched bool) {
	if i.ApplicationCommandData().Name == cm.AppCommand.Name {
		return true
	}
	return false
}

func (cm *PingCommand) DoInteraction(i *discordgo.InteractionCreate) (err error) {
	discord.ChannelMessageSend(i.ChannelID, "pong response not using interaction!")
	clients.DgSession.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "pong response with discord interaction!!",
		},
	})
	return nil
}

func (cm *PingCommand) MatchMessage(message *discordgo.MessageCreate) (bool, bool) {
	matchStatus, _ := cm.MatchText(message.Content)
	return matchStatus, true
}

func (cm *PingCommand) New() {
	cm.Name = "ping"
	cm.Identifiers = []string{"ping"}
	cm.AppCommand = &discordgo.ApplicationCommand{
		Name:        "ping",
		Description: "Ping command for Dalian",
	}
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
