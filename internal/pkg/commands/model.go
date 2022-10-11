/*
Package commands includes all interfaces, strucs and implementations of various discord commands.
*/
package commands

import (
	"fmt"
	"github.com/bwmarrin/discordgo"
	"github.com/kballard/go-shellquote"
	"regexp"
	"strings"
	"time"
)

// Prefix Designated prefix for regular text command
var Prefix string

// Separator Designated separator for separating arguments
var Separator string

// BotID Current id for the bot user, for command identification purpose.
var BotID string

// SetPrefix Set the global prefix for regular text command.
func SetPrefix(prefix string) {
	Prefix = prefix
}

// SetSeparator Set the global separator for regular text command.
func SetSeparator(separator string) {
	Separator = separator
}

// SetBotID Set the BotID of the current bot user.
func SetBotID(botID string) {
	BotID = botID
}

// ICommand The highest level interface for all commands
type ICommand interface {
	// New All command must have a valid pointer initialization method
	New()

	// GetName All command must have a name (unique identifier)
	GetName() string
}

// Command Basic command struct with no function
type Command struct {
	Name string
}

// GetName Return the name (unique identifier) of the command.
func (cm *Command) GetName() string {
	return cm.Name
}

type ITextCommand interface {
	MatchMessage(m *discordgo.MessageCreate) (isMatched bool, isTerminated bool)
	// DoMessage All command wlll do something
	// a anything you may need to execute. It is your OWN responsibility to validate before use.
	// err if anything worth *reporting* happened. expected error should not be returned.
	DoMessage(m *discordgo.MessageCreate) (err error)
}

type AppCommandsMap map[string]*discordgo.ApplicationCommand

type ISlashCommand interface {
	MatchInteraction(i *discordgo.InteractionCreate) (isMatched bool)
	DoInteraction(i *discordgo.InteractionCreate) (err error)
	GetAppCommandsMap() AppCommandsMap
}

type SlashCommand struct {
	AppCommandsMap AppCommandsMap
}

func (cm *SlashCommand) GetAppCommandsMap() AppCommandsMap {
	return cm.AppCommandsMap
}

func (cm *SlashCommand) DefaultMatchCommand(i *discordgo.InteractionCreate) (bool, string) {
	for _, slashCmd := range cm.AppCommandsMap {
		if i.ApplicationCommandData().Name == slashCmd.Name {
			return true, slashCmd.Name
		}
	}
	return false, ""
}

func (cm *SlashCommand) ParseOptionsMap(options []*discordgo.ApplicationCommandInteractionDataOption) map[string]*discordgo.ApplicationCommandInteractionDataOption {
	optionMap := make(map[string]*discordgo.ApplicationCommandInteractionDataOption)
	for _, opt := range options {
		optionMap[opt.Name] = opt
	}
	return optionMap
}

// PlainCommand Text command triggers when a specific text is detected, the most common type of command
// start with global identifier, have one or more arguments
type PlainCommand struct {
	Identifiers []string
}

// MatchText Embedded match method for PlainCommand
func (cm *PlainCommand) MatchText(content string) (bool, string) {
	for _, v := range cm.Identifiers {
		//must be a perfect match before the first space
		if strings.TrimSpace(strings.Split(content, " ")[0]) == Prefix+v {
			return true, v
		}
	}
	return false, ""
}

// RegexTextCommand Complicated TextCommand, using Regex to match
// Grants possibility of NOT using identifiers and perform advanced macthing actions.
// embeds one or multiple regex expression(s) for matching purposes.
type RegexTextCommand struct {
	RegexExpressions []*regexp.Regexp
}

// RegMatchMessage Embedded match method for RegexTextCommand
func (cm *RegexTextCommand) RegMatchMessage(content string) (bool, regexp.Regexp) {
	for _, reg := range cm.RegexExpressions {
		if reg.MatchString(content) {
			return true, *reg
		}
	}
	return false, regexp.Regexp{}
}

// ArgCommand Text commands with multiple arguments
// embeds a default splitting method for multiple args
type ArgCommand struct {
}

// SeparateArgs separate a long string into different args
// when no extra args provided, the string shoud len(1)
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

// CommandFlag basic structure for handling command flags
type CommandFlag struct {
	Name             string   // Flag name
	FlagPrefix       []string // Flag prefix(s)
	AcceptsExtraArg  bool     // Acceptance of extra arg
	MultipleExtraArg bool     // Acceptance of multiple extra arg
	MEGroup          []string // Mutually exclusive group
}

// FlagCommand Text commands enabling linux flag-like inputs
type FlagCommand struct {
	// FlagArgstatMaps: flag name : ?args required
	AvailableFlagMap map[string]*CommandFlag
}

// FlagArgstatMaps Defined structure for storing flag info for a given trigger
type FlagArgstatMaps map[string][]string

// HasFlag A helper function for checking simple existence of a flag.
// often equivalent to len(flagMap[flagName])>0
func (flagMap FlagArgstatMaps) HasFlag(flagName string) bool {
	_, exist := flagMap[flagName]
	return exist
}

// ParseFlags read the input flag from given text message.
// Does NOT handle the validation part,only return err if the input is invalid structuralwise
// Will produce unexpected result if using with multiple args command, sanitize before calling.
func (cm *FlagCommand) ParseFlags(content string) (FlagArgstatMaps, error) {
	//0. initialize map
	flagMap := make(map[string][]string)
	//1. separate
	temp, err := shellquote.Split(content)
	if err != nil {
		return nil, err
	}
	//if no flags ever presentI
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

// ValidateFlagMap handle the validation of flags for a given flag command.
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

// RegisterCommandFlag register an valid flag for the flag command.
func (cm *FlagCommand) RegisterCommandFlag(theFlag CommandFlag) error {
	for _, v := range theFlag.FlagPrefix {
		cm.AvailableFlagMap[v] = &theFlag
	}
	return nil
}

// InitAvailableFlagMap default method for initalizing available flag map.
func (cm *FlagCommand) InitAvailableFlagMap() {
	cm.AvailableFlagMap = make(map[string]*CommandFlag)
}

// tryInsertFlagMap Supportive function for parsing flags from text.
func tryInsertFlagMap(kvPair [2]string, flagMap FlagArgstatMaps) {
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

// IBotCallingCommand TextCommand starts with a @bot
type IBotCallingCommand interface {
	IsCallingBot(content string) bool
}

// BotCallingCommand Functional structure for BotCallingCommand
// embedded a method for identifying @Bot texts
type BotCallingCommand struct {
}

// IsCallingBot Return true if the text starts with @{BotID}
func (b BotCallingCommand) IsCallingBot(content string) (isCalling bool, sanitizedContent string) {
	callingStr := fmt.Sprintf("<@%s>", BotID)
	if strings.HasPrefix(content, callingStr) {
		return true, strings.TrimSpace(strings.Replace(content, callingStr, "", 1))
	}
	return false, ""
}

// BasicStageInfo Include shared information for staged actions
type BasicStageInfo struct {
	ChannelID      string
	UserID         string
	StageNow       int
	LastActionTime time.Time
}

type IStage interface {
	process()
}

type ComponentActionMap map[string]func(i *discordgo.InteractionCreate)

type IComponentCommand interface {
	GetCompActionMap() ComponentActionMap
	DoComponent(i *discordgo.InteractionCreate) error
}

type ComponentCommand struct {
	CompActionMap ComponentActionMap
}

func (cm *ComponentCommand) GetCompActionMap() ComponentActionMap {
	return cm.CompActionMap
}
