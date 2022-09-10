package commands

import (
	"strings"
)

var Prefix string
var Separator string

func SetPrefix(prefix string) {
	Prefix = prefix
}

func SetSeparator(separator string) {
	Separator = separator
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
	Identifiers []string
}

func (cm *PlainCommand) MatchMessage(content string) (bool, string) {
	for _, v := range cm.Identifiers {
		//must be a perfect match before the first space
		if strings.TrimSpace(strings.Split(content, " ")[0]) == Prefix+v {
			return true, v
		}
	}
	return false, ""
}

type IArgCommand interface {
}

type ArgCommand struct {
	Args []string
}

func (cm *ArgCommand) SeparateArgs(content, separator string) int {
	cm.Args = strings.Split(content, separator)
	j := 0
	for _, v := range cm.Args {
		//delete the element if the string is empty after trim
		if vTrim := strings.TrimSpace(v); vTrim != "" {
			cm.Args[j] = vTrim
			j++
		}
	}
	cm.Args = cm.Args[:j]
	return len(cm.Args)
}

type IFlagCommand interface {
}
