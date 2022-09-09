package commands

import (
	"strings"
)

var Prefix string

func SetPrefix(prefix string) {
	Prefix = prefix
}

type ICommand interface {
	New()
	Match(a ...any) bool
	Do(a ...any) error
	GetName() string
}

type Command struct {
	Name string
}

func (cm *Command) GetName() string {
	return cm.Name
}

type ITextCommand interface {
	ICommand
	MatchMessage(content string) (bool, string)
}
type PlainCommand struct {
	Command
	Identifiers []string
}

func (cm *PlainCommand) MatchMessage(content string) (bool, string) {
	for _, v := range cm.Identifiers {
		if strings.HasPrefix(content, Prefix+v) {
			return true, v
		}
	}
	return false, ""
}
