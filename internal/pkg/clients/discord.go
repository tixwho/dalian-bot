// Package clients the package store lower level clients for different services
// ideally, you will want to avoid directly using clients here without a good reason.
package clients

import (
	"github.com/bwmarrin/discordgo"
)

var DgSession *discordgo.Session

func RegisterDiscordClient(session *discordgo.Session) {
	DgSession = session
	SetupIntents()
}

func SetupIntents() {
	DgSession.Identify.Intents = discordgo.IntentGuildMessages
}
