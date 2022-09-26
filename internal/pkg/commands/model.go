package commands

import (
	"fmt"
	"github.com/kballard/go-shellquote"
	"regexp"
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

type IImplicitTextCommand interface {
	ICommand
	RegMatchMessage(content string) (bool, regexp.Regexp)
}

type ImplicitCommand struct {
	RegexExpressions []*regexp.Regexp
}

func (cm *ImplicitCommand) RegMatchMessage(content string) (bool, regexp.Regexp) {
	for _, reg := range cm.RegexExpressions {
		if reg.MatchString(content) {
			return true, *reg
		}
	}
	return false, regexp.Regexp{}
}

type IArgCommand interface {
}

type ArgCommand struct {
}

func (cm *ArgCommand) SeparateArgs(content, separator string) ([]string, int) {
	args := strings.Split(content, separator)
	j := 0
	for _, v := range args {
		//delete the element if the string is empty after trim
		if vTrim := strings.TrimSpace(v); vTrim != "" {
			args[j] = vTrim
			j++
		}
	}
	args = args[:j]
	return args, len(args)
}

type IFlagCommand interface {
}

type CommandFlag struct {
	Name             string   // Flag name
	FlagPrefix       []string // Flag prefix(s)
	AcceptsExtraArg  bool     // Acceptance of extra arg
	MultipleExtraArg bool     // Acceptance of multiple extra arg
	MEGroup          []string // Mutually exclusive group
}

type FlagCommand struct {
	// FlagArgstatMaps: flag name : ?args required
	AvailableFlagMap map[string]*CommandFlag
}

type FlagArgstatMaps map[string][]string

func (cm *FlagCommand) ParseFlags(content string) (FlagArgstatMaps, error) {
	//0. initialize map
	flagMap := make(map[string][]string)
	//1. separate
	// todo: make flags compatible with commands with multiple arguments.
	temp, err := shellquote.Split(content)
	if err != nil {
		return nil, err
	}
	//if no flags ever present
	if len(temp) == 1 {
		return flagMap, nil
	}
	//skipping first bloc
	for i := 1; i < len(temp); i++ {
		//check every argument with "-" if it has a subsequent arg
		if strings.HasPrefix(temp[i], "-") {
			//boundary
			if i == len(temp)-1 {
				//must be a flag without extra
				tryInsertFlagMap([2]string{temp[i][1:], ""}, flagMap)
			} else {
				//checking existence of extra flag
				if !strings.HasPrefix(temp[i+1], "-") {
					tryInsertFlagMap([2]string{temp[i][1:], temp[i+1]}, flagMap)
					//skip one block to make up for the extra arg
					i++
				} else {
					tryInsertFlagMap([2]string{temp[i][1:], ""}, flagMap)
				}
			}
		}
	}
	return flagMap, nil
}

func (cm *FlagCommand) ValidateFlagMap(flagMaps FlagArgstatMaps) (FlagArgstatMaps, error) {
	tempMEMap := make(map[string]CommandFlag)
	validatedArgStatMaps := make(map[string][]string)
	for priKey, priExtra := range flagMaps {
		//first check if the flag exist
		if entry, ok := cm.AvailableFlagMap[priKey]; !ok {
			return nil, fmt.Errorf("Unknown flag:[%s]", priKey)
		} else {
			//checking extra arg status
			if !entry.AcceptsExtraArg && len(priExtra) > 0 {
				return nil, fmt.Errorf("Flag [%s] does NOT allow ANY extra argument", entry.Name)
			}
			//checking number of extra arg allowed
			//i
			if !entry.MultipleExtraArg && len(priExtra) > 1 {
				return nil, fmt.Errorf("Flag [%s] allow exactly ONE extra argument", entry.Name)
			}
			//checking ME status
			for _, v := range entry.MEGroup {
				//CommandFlag of the same ME group must NOT present in the temporary validation map.
				if occupiedFlag, ok := tempMEMap[v]; ok {
					return nil, fmt.Errorf("Flag [%s] is mutually exclusive w/ flag [%s]||ME Group Lock [%s]", entry.Name, occupiedFlag.Name, v)
				}
				//validation passed. adding it to temporary ME map for future validation
				tempMEMap[v] = *entry
			}
			// passed the validation, adding to cleaned flag and validate again in case alias used.
			currentFlagExtraArg, ok := validatedArgStatMaps[entry.Name]
			if !ok {
				//first time using this flag. should've passed all examinations.
				validatedArgStatMaps[entry.Name] = priExtra
			} else {
				//alias used, need to examine number of extra argument
				tempExtraArr := append(currentFlagExtraArg, priExtra...)
				if !entry.MultipleExtraArg && len(tempExtraArr) > 1 {
					return nil, fmt.Errorf("Flag [%s] does NOT allow ANY extra argument", entry.Name)
				}
				validatedArgStatMaps[entry.Name] = tempExtraArr
			}
		}

	}
	// All examination passed!
	return validatedArgStatMaps, nil
}

func (cm *FlagCommand) RegisterCommandFlag(theFlag CommandFlag) error {
	for _, v := range theFlag.FlagPrefix {
		cm.AvailableFlagMap[v] = &theFlag
	}
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
		if kvPair[1] != "" {
			flagMap[kvPair[0]] = []string{kvPair[1]}
		} else {
			flagMap[kvPair[0]] = []string{}
		}
	}
}

type IStagedListener interface {
	nextStage()
}
