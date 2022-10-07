package commands

import (
	"dalian-bot/internal/pkg/clients"
	"fmt"
	"github.com/bwmarrin/discordgo"
	"regexp"
)

type WhatCommand struct {
	Command
	RegexTextCommand
}

func (cm *WhatCommand) MatchMessage(message *discordgo.MessageCreate) (bool, bool) {
	matchStatus, _ := cm.RegMatchMessage(message.Content)
	return matchStatus, true
}

func (cm *WhatCommand) New() {
	cm.Name = "what"
	cm.RegexExpressions = []*regexp.Regexp{}
	cm.RegexExpressions = append(cm.RegexExpressions, regexp.MustCompile("^what$"))
}

func (cm *WhatCommand) DoMessage(m *discordgo.MessageCreate) error {
	step := 2
	for {
		msgs, err := clients.DgSession.ChannelMessages(m.ChannelID, step, m.ID, "", "")
		if err != nil {
			fmt.Printf("Error getting %d message prior to || %s", step, m.Content)
			return err
		}
		msg, foundNonBot := findFirstNonBotMsg(msgs)
		if foundNonBot {
			clients.DgSession.ChannelMessageSend(msg.ChannelID, fmt.Sprintf("**%s**", msg.Content))
			break
		}
		step *= 2
	}

	return nil
}

func findFirstNonBotMsg(messages []*discordgo.Message) (*discordgo.Message, bool) {
	for _, v := range messages {
		if !v.Author.Bot {
			return v, true
		}
	}
	return nil, false
}

func init() {
	var whatCommand WhatCommand
	whatCommand.New()
	RegisterCommand(&whatCommand)
}
