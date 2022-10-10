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

// ChannelMessageSend a wrapper of discordgo ChannelMessageSend function
func ChannelMessageSend(channelID, content string) (*discordgo.Message, error) {
	return clients.DgSession.ChannelMessageSend(channelID, content)
}

func ChannelMessageSendEmbed(channelID string, embed *discordgo.MessageEmbed) (*discordgo.Message, error) {
	return clients.DgSession.ChannelMessageSendEmbed(channelID, embed)
}

func ChannelReportError(channelID string, error error) (*discordgo.Message, error) {
	return ChannelMessageSend(channelID, error.Error())
}

func InteractionRespondComplex(i *discordgo.Interaction, resp *discordgo.InteractionResponse) error {
	return clients.DgSession.InteractionRespond(i, resp)
}

func InteractionRespondEmbed(i *discordgo.Interaction, embed *discordgo.MessageEmbed) error {
	return InteractionRespondComplex(i, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{embed},
		},
	})
}

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
