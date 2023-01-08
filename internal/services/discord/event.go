package discord

import (
	"dalian-bot/internal/core"
	"github.com/bwmarrin/discordgo"
)

type EventType string

const (
	EventTypeMessageCreate     EventType = "message-create"
	EventTypeInteractionCreate EventType = "interaction-create"
)

type Event struct {
	EventType EventType
	//will only use one of them
	MessageCreate     *discordgo.MessageCreate
	InteractionCreate *discordgo.InteractionCreate
}

func UnboxEvent(t core.Trigger) Event {
	var e = t.Event.(Event)
	return e
}
