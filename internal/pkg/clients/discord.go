package clients

import "github.com/bwmarrin/discordgo"

var DgSession *discordgo.Session

func RegisterDiscordClient(session *discordgo.Session) {
	DgSession = session
	SetupIntents()
}

func SetupIntents() {
	DgSession.Identify.Intents = discordgo.IntentGuildMessages
}
