package commands

import (
	"github.com/kballard/go-shellquote"
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

type FlagCommand struct {
	// FlagMaps: flag name : ?args required
	FlagArgstatMaps map[string][]string
}

func (cm *FlagCommand) ParseFlags(content string) error {
	//0. initialize map
	flagMap := make(map[string][]string)
	//1. separate
	temp, err := shellquote.Split(content)
	if err != nil {
		return err
	}
	//if no flags ever present
	if len(temp) == 1 {
		cm.FlagArgstatMaps = flagMap
		return nil
	}
	//skipping first bloc
	for i := 1; i < len(temp); i++ {
		//check every argument with "-" if it has a subsequent arg
		if strings.HasPrefix(temp[i], "-") {
			//boundary
			if i == len(temp)-1 {
				//must be a flag without extra
				tryInsertFlagMap([2]string{temp[i], ""}, flagMap)
			} else {
				//checking existence of extra flag
				if !strings.HasPrefix(temp[i+1], "-") {
					tryInsertFlagMap([2]string{temp[i], temp[i+1]}, flagMap)
					//skip one block to make up for the extra arg
					i++
				} else {
					tryInsertFlagMap([2]string{temp[i], ""}, flagMap)
				}
			}
		}
	}
	cm.FlagArgstatMaps = flagMap
	return nil
}

func tryInsertFlagMap(kvPair [2]string, flagMap map[string][]string) {
	if v, ok := flagMap[kvPair[0]]; ok {
		//only add arguments to flags w/ extra args.
		if kvPair[1] != "" {
			flagMap[kvPair[0]] = append(v, kvPair[1])
		}
	} else {
		//create a new string slice and add first extra argument. can be "" if extra unnecessary.
		flagMap[kvPair[0]] = []string{kvPair[1]}
	}
}
