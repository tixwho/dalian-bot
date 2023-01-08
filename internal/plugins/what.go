package plugins

import (
	core2 "dalian-bot/internal/core"
	discord2 "dalian-bot/internal/services/discord"
	"errors"
	"fmt"
	"github.com/bwmarrin/discordgo"
	"regexp"
)

type WhatPlugin struct {
	core2.Plugin
	DiscordService *discord2.Service
	core2.RegexMatchUtil
}

func (p *WhatPlugin) DoMessage(_ *core2.Bot, m *discordgo.MessageCreate) (err error) {
	matchStatus, _ := p.RegMatchMessage(m.Content)
	//doing `what`.
	if matchStatus {
		step := 2
		for {
			msgs, err := p.DiscordService.Session.ChannelMessages(m.ChannelID, step, m.ID, "", "")
			if err != nil {
				core2.Logger.Warn("Error getting %d message prior to || %s", step, m.Content)
				return err
			}
			msg, foundNonBot := discord2.FindFirstNonBotMsg(msgs)
			if foundNonBot {
				p.DiscordService.Session.ChannelMessageSend(msg.ChannelID, fmt.Sprintf("**%s**", msg.Content))
				return nil
			}
			step *= 2
		}
	}
	return nil
}

func (p *WhatPlugin) Init(reg *core2.ServiceRegistry) error {
	if err := reg.FetchService(&p.DiscordService); err != nil {
		return err
	}

	p.AcceptedTriggerTypes = []core2.TriggerType{core2.TriggerTypeDiscord}
	p.Name = "what"
	p.RegexExpressions = []*regexp.Regexp{regexp.MustCompile("^what$")}
	return nil
}

func (p *WhatPlugin) Trigger(trigger core2.Trigger) {
	if !p.AcceptTrigger(trigger.Type) {
		return
	}
	// only accept discord so far, so not using switch.
	// example of accepting other triggers can be found at:
	discordEvent := discord2.UnboxEvent(trigger)
	switch discordEvent.EventType {
	case discord2.EventTypeMessageCreate:
		if p.DiscordService.IsGuildMessageFromBotOrSelf(discordEvent.MessageCreate.Message) {
			return
		}
		p.DoMessage(trigger.Bot, discordEvent.MessageCreate)
	default:
		//not handling any other type of discordEvent.
		return
	}
}

func NewWhatPlugin(reg *core2.ServiceRegistry) core2.INewPlugin {
	var what WhatPlugin
	if err := (&what).Init(reg); err != nil && errors.As(err, &core2.ErrServiceFetchUnknownService) {
		core2.Logger.Panicf("What plugin MUST have all required service(s) injected!")
	}
	return &what
}
