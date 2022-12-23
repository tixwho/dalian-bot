package ddtv

import (
	"dalian-bot/internal/pkg/core"
)

type EventType string

const (
	EventTypeWebhook EventType = "webhook"
)

type Event struct {
	EventType EventType
	//will only use one of them
	WebHook WebHook
}

func UnboxEvent(t core.Trigger) Event {
	var e = t.Event.(Event)
	return e
}
