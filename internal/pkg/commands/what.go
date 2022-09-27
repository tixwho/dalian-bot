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

func (cm *WhatCommand) New() {
	cm.Name = "what"
	cm.RegexExpressions = []*regexp.Regexp{}
	cm.RegexExpressions = append(cm.RegexExpressions, regexp.MustCompile("^what$"))
}

func (cm *WhatCommand) Match(a ...any) bool {
	m, isMsgCreate := a[0].(*discordgo.MessageCreate)
	if !isMsgCreate {
		return false
	}
	matchStatus, _ := cm.RegMatchMessage(m.Content)
	return matchStatus
}

func (cm *WhatCommand) Do(a ...any) error {
	m := a[0].(*discordgo.MessageCreate)
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
