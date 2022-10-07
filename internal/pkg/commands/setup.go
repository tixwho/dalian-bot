package commands

import (
	"dalian-bot/internal/pkg/clients"
	"errors"
	"fmt"
	"github.com/bwmarrin/discordgo"
)

var (
	CommandByName = make(map[string]ICommand)
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

	for _, v := range CommandByName {
		//only test TextCommand for MscCreate events
		if iTextCmd, ok := v.(ITextCommand); ok {
			if isMatched, isTerminated := iTextCmd.MatchMessage(m); isMatched {
				iTextCmd.DoMessage(m)
				//end the chain if terminated
				if isTerminated {
					return
				}
			}
		}
	}
}

func RegisterDiscordHandlers() {
	clients.DgSession.AddHandler(messageCreate)
}

func RegisterCommand(command ICommand) error {
	name := command.GetName()
	if _, e := CommandByName[name]; e {
		return errors.New(fmt.Sprintf("command %s already exist!", name))
	}
	CommandByName[name] = command
	fmt.Printf("Registered command:%s\r\n", command.GetName())
	return nil
}
