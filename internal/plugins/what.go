package plugins

import (
	"dalian-bot/internal/core"
	"dalian-bot/internal/services/discord"
	"errors"
	"fmt"
	"github.com/bwmarrin/discordgo"
	"regexp"
)

// WhatPlugin A for-fun function that will repeat the last message.
// Discord: Can be triggered by `what`
type WhatPlugin struct {
	core.Plugin
	DiscordService *discord.Service
	core.RegexMatchUtil
}

func (p *WhatPlugin) DoMessage(_ *core.Bot, m *discordgo.MessageCreate) (err error) {
	matchStatus, _ := p.RegMatchMessage(m.Content)
	//doing `what`.
	if matchStatus {
		step := 2
		for {
			msgs, err := p.DiscordService.Session.ChannelMessages(m.ChannelID, step, m.ID, "", "")
			if err != nil {
				core.Logger.Warn("Error getting %d message prior to || %s", step, m.Content)
				return err
			}
			msg, foundNonBot := discord.FindFirstNonBotMsg(msgs)
			if foundNonBot {
				p.DiscordService.Session.ChannelMessageSend(msg.ChannelID, fmt.Sprintf("**%s**", msg.Content))
				return nil
			}
			step *= 2
		}
	}
	return nil
}

func (p *WhatPlugin) Init(reg *core.ServiceRegistry) error {
	if err := reg.FetchService(&p.DiscordService); err != nil {
		return err
	}

	p.AcceptedTriggerTypes = []core.TriggerType{discord.TriggerTypeDiscord}
	p.Name = "what"
	p.RegexExpressions = []*regexp.Regexp{regexp.MustCompile("^what$")}
	return nil
}

func (p *WhatPlugin) Trigger(trigger core.Trigger) {
	if !p.AcceptTrigger(trigger.Type) {
		return
	}
	// only accept discord so far, so not using switch.
	// example of accepting other triggers can be found at:
	discordEvent := discord.UnboxEvent(trigger)
	switch discordEvent.EventType {
	case discord.EventTypeMessageCreate:
		if p.DiscordService.IsGuildMessageFromBotOrSelf(discordEvent.MessageCreate.Message) {
			return
		}
		p.DoMessage(trigger.Bot, discordEvent.MessageCreate)
	default:
		//not handling any other type of discordEvent.
		return
	}
}

func NewWhatPlugin(reg *core.ServiceRegistry) core.IPlugin {
	var what WhatPlugin
	if err := (&what).Init(reg); err != nil && errors.As(err, &core.ErrServiceFetchUnknownService) {
		core.Logger.Panicf("What plugin MUST have all required service(s) injected!")
	}
	return &what
}
