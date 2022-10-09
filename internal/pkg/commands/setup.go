package commands

import (
	"dalian-bot/internal/pkg/clients"
	"errors"
	"fmt"
	"github.com/bwmarrin/discordgo"
	"log"
)

var (
	CommandByName        = make(map[string]*ICommand)
	registeredCommands   []*discordgo.ApplicationCommand
	CommandByComponentID = make(map[string]*IComponentCommand)
)

func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	//debugging
	fmt.Printf("%s:%s \r\n", m.Author.Username, m.Content)

	// Ignore all messages created by the bot itself
	// This isn't required in this specific example, but it's a good practice.
	if m.Author.ID == s.State.User.ID {
		return
	}
	// Ignore chain requests from other bots
	if m.Author.Bot {
		return
	}

	for _, v := range CommandByName {
		//only test TextCommand for MscCreate events
		if iTextCmd, ok := (*v).(ITextCommand); ok {
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

func interactionCreate(s *discordgo.Session, i *discordgo.InteractionCreate) {
	//debugging
	fmt.Printf("Int: %s:%s:%v \r\n", i.Member.User.Username, i.Data, i.Message)
	switch i.Type {
	case discordgo.InteractionApplicationCommand:
		for _, v := range CommandByName {
			//only test TextCommand for MscCreate events
			if iSlashCmd, ok := (*v).(ISlashCommand); ok {
				if iSlashCmd.MatchInteraction(i) {
					iSlashCmd.DoInteraction(i)
					return
				}
			}
		}
	case discordgo.InteractionMessageComponent:
		if compCmd, ok := CommandByComponentID[i.MessageComponentData().CustomID]; ok {
			(*compCmd).(IComponentCommand).DoComponent(i)
		}
	}

}

func RegisterDiscordHandlers() {
	clients.DgSession.AddHandler(messageCreate)
	clients.DgSession.AddHandler(interactionCreate)
}

func RegisterSlashCommands() {
	for _, v := range CommandByName {
		if ISlashCmd, ok := (*v).(ISlashCommand); ok {
			cmd, err := clients.DgSession.ApplicationCommandCreate(clients.DgSession.State.User.ID, "", ISlashCmd.GetAppCommand())
			if err != nil {
				log.Panicf("Cannot create '%v' command: %v", ISlashCmd.GetAppCommand(), err)
			}
			registeredCommands = append(registeredCommands, cmd)
		}
	}
}

func DisposeSlashCommands() {
	for _, v := range registeredCommands {
		err := clients.DgSession.ApplicationCommandDelete(clients.DgSession.State.User.ID, "", v.ID)
		if err != nil {
			log.Panicf("Cannot delete '%v' command: %v", v.Name, err)
		}
	}
}

func RegisterCommand(command ICommand) error {
	name := command.GetName()
	if _, e := CommandByName[name]; e {
		return errors.New(fmt.Sprintf("command %s already exist!", name))
	}
	CommandByName[name] = &command
	if ICompCmd, ok := command.(IComponentCommand); ok {
		for actionName := range ICompCmd.GetCompActionMap() {
			if _, e := CommandByComponentID[actionName]; e {
				return errors.New(fmt.Sprintf("component %s already exist!", actionName))
			}
			CommandByComponentID[actionName] = &ICompCmd
		}
	}
	fmt.Printf("Registered command:%s\r\n", command.GetName())
	return nil
}
