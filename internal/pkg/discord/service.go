package discord

import (
	"dalian-bot/internal/pkg/clients"
	"github.com/bwmarrin/discordgo"
	"io"
	"log"
)

// ChannelMessageSend a wrapper of discordgo ChannelMessageSend function
func ChannelMessageSend(channelID, content string) (*discordgo.Message, error) {
	return clients.DgSession.ChannelMessageSend(channelID, content)
}

// ChannelFileSend send a file to given guild channel.
// channelID the id of a channel
// name the display filename to be sent to discord
// r the io reader containing a valid file struct
func ChannelFileSend(channelID, name string, r io.Reader) error {
	if _, err := clients.DgSession.ChannelFileSend(channelID, name, r); err != nil {
		log.Println("Error sending discord message: ", err)
		return err
	}
	return nil
}
