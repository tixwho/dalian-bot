package clients

import (
	"github.com/bwmarrin/discordgo"
	"io"
	"log"
)

var DgSession *discordgo.Session

func RegisterDiscordClient(session *discordgo.Session) {
	DgSession = session
	SetupIntents()
}

func SetupIntents() {
	DgSession.Identify.Intents = discordgo.IntentGuildMessages
}

func SendFile(channel, name string, r io.Reader) error {
	if _, err := DgSession.ChannelFileSend(channel, name, r); err != nil {
		log.Println("Error sending discord message: ", err)
		return err
	}
	return nil
}
