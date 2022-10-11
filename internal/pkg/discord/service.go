package discord

import (
	"dalian-bot/internal/pkg/clients"
	"github.com/bwmarrin/discordgo"
	"io"
	"log"
)

const (
	EmbedColorNormal   = 0x33acff
	EmbedColorQuestion = 0xffcb66
	EmbedColorSuccess  = 0x3df53d
	EmbedColorDanger   = 0xfa3838
)

// ChannelMessageSend A wrapper of discordgo ChannelMessageSend function.
func ChannelMessageSend(channelID, content string) (*discordgo.Message, error) {
	return clients.DgSession.ChannelMessageSend(channelID, content)
}

// ChannelMessageSendEmbed A wrapper of discordgo ChannelMessageSendEmbed function.
func ChannelMessageSendEmbed(channelID string, embed *discordgo.MessageEmbed) (*discordgo.Message, error) {
	return clients.DgSession.ChannelMessageSendEmbed(channelID, embed)
}

// ChannelMessageReportError Report the error as a plain message to given gild channel.
func ChannelMessageReportError(channelID string, error error) (*discordgo.Message, error) {
	return ChannelMessageSend(channelID, error.Error())
}

// InteractionRespondComplex Basic wrapper for discordgo.InteractionRespond.
func InteractionRespondComplex(i *discordgo.Interaction, resp *discordgo.InteractionResponse) error {
	return clients.DgSession.InteractionRespond(i, resp)
}

// InteractionRespondEmbed Shortcut method for fast reply including a MessageEmbed.
func InteractionRespondEmbed(i *discordgo.Interaction, embed *discordgo.MessageEmbed) error {
	return InteractionRespondComplex(i, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{embed},
		},
	})
}

// InteractionRespond Shortcut method for a simple message reply.
func InteractionRespond(i *discordgo.Interaction, content string) error {
	return InteractionRespondComplex(i, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: content,
		},
	})
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
