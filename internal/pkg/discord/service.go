package discord

import (
	"dalian-bot/internal/pkg/clients"
	"dalian-bot/internal/pkg/commands"
	"fmt"
	"github.com/bwmarrin/discordgo"
)

func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	//debugging
	fmt.Printf("%s:%s \r\n", m.Author.Username, m.Content)

	// Ignore all messages created by the bot itself
	// This isn't required in this specific example but it's a good practice.
	if m.Author.ID == s.State.User.ID {
		return
	}
	// Ignore chain requests from all other bots
	if m.Author.Bot {
		return
	}

	for _, v := range commands.CommandByName {
		//only test TextCommand for MscCreate events
		if tCommand, ok := v.(commands.ITextCommand); ok {
			if tCommand.Match(m) {
				tCommand.Do(m)
				//stop right there.
				return
			}
		}
		//after that, test ImplicitCommand
		if inCommand, ok := v.(commands.IImplicitTextCommand); ok {
			if inCommand.Match(m) {
				inCommand.Do(m)
				//stop right there
				return
			}
		}
	}
}

func RegisterHandlers() {
	clients.DgSession.AddHandler(messageCreate)
}
