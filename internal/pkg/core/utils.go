package core

import (
	"fmt"
	"github.com/kballard/go-shellquote"
	"regexp"
	"strings"
	"sync"
	"time"
)

// StartWithMatchUtil Text command triggers when a specific text is detected, the most common type of command
// start with global identifier, have one or more arguments
type StartWithMatchUtil struct {
	Identifiers []string
}

// MatchText Embedded match method for StartWithMatchUtil
func (cm *StartWithMatchUtil) MatchText(content string, config MessengerConfig) (bool, string) {
	for _, v := range cm.Identifiers {
		//must be a perfect match before the first space
		if strings.TrimSpace(strings.Split(content, " ")[0]) == config.Prefix+v {
			return true, v
		}
	}
	return false, ""
}

// RegexMatchUtil Complicated TextCommand, using Regex to match
// Grants possibility of NOT using identifiers and perform advanced macthing actions.
// embeds one or multiple regex expression(s) for matching purposes.
type RegexMatchUtil struct {
	RegexExpressions []*regexp.Regexp
}

// RegMatchMessage Embedded match method for RegexMatchUtil
func (cm *RegexMatchUtil) RegMatchMessage(content string) (bool, regexp.Regexp) {
	for _, reg := range cm.RegexExpressions {
		if reg.MatchString(content) {
			return true, *reg
		}
	}
	return false, regexp.Regexp{}
}

// ArgParseUtil Text commands with multiple arguments
// embeds a default splitting method for multiple args
type ArgParseUtil struct {
}

// SeparateArgs separate a long string into different args
// when no extra args provided, the string shoud len(1)
func (cm *ArgParseUtil) SeparateArgs(content, separator string) []string {
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
	return args
}

// CommandFlag basic structure for handling command flags
type CommandFlag struct {
	Name             string   // Flag name
	FlagPrefix       []string // Flag prefix(s)
	AcceptsExtraArg  bool     // Acceptance of extra arg
	MultipleExtraArg bool     // Acceptance of multiple extra arg
	MEGroup          []string // Mutually exclusive group
}

// FlagParseUtil Text commands enabling linux flag-like inputs
type FlagParseUtil struct {
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
func (cm *FlagParseUtil) ParseFlags(content string) (FlagArgstatMaps, error) {
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
func (cm *FlagParseUtil) ValidateFlagMap(flagMaps FlagArgstatMaps) (FlagArgstatMaps, error) {
	tempMEMap := make(map[string]CommandFlag)
	validatedArgStatMaps := make(map[string][]string)
	for priKey, priExtra := range flagMaps {
		//first check if the flag exist
		if entry, ok := cm.AvailableFlagMap[priKey]; !ok {
			return nil, fmt.Errorf("unknown flag:[%s]", priKey)
		} else {
			//checking extra arg status
			if !entry.AcceptsExtraArg && len(priExtra) > 0 {
				return nil, fmt.Errorf("flag [%s] does NOT allow ANY extra argument", entry.Name)
			}
			//checking number of extra arg allowed
			//i
			if !entry.MultipleExtraArg && len(priExtra) > 1 {
				return nil, fmt.Errorf("flag [%s] allow exactly ONE extra argument", entry.Name)
			}
			//checking ME status
			for _, v := range entry.MEGroup {
				//CommandFlag of the same ME group must NOT present in the temporary validation map.
				if occupiedFlag, ok := tempMEMap[v]; ok {
					return nil, fmt.Errorf("flag [%s] is mutually exclusive w/ flag [%s]||ME Group Lock [%s]", entry.Name, occupiedFlag.Name, v)
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
					return nil, fmt.Errorf("flag [%s] does NOT allow ANY extra argument", entry.Name)
				}
				validatedArgStatMaps[entry.Name] = tempExtraArr
			}
		}

	}
	// All examination passed!
	return validatedArgStatMaps, nil
}

// RegisterCommandFlag register an valid flag for the flag command.
func (cm *FlagParseUtil) RegisterCommandFlag(theFlag CommandFlag) error {
	for _, v := range theFlag.FlagPrefix {
		cm.AvailableFlagMap[v] = &theFlag
	}
	return nil
}

// InitAvailableFlagMap default method for initalizing available flag map.
func (cm *FlagParseUtil) InitAvailableFlagMap() {
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

type StageMap map[CombinedKey]IStageNew

type StageUtil struct {
	activeStageMap sync.Map
}

func (su *StageUtil) GetStage(key CombinedKey) (stage IStageNew, found bool) {
	v, ok := su.activeStageMap.Load(key)
	return v.(IStageNew), ok
}

func (su *StageUtil) StoretStage(key CombinedKey, stage IStageNew) {
	su.activeStageMap.Store(key, stage)
}

func (su *StageUtil) DeleteStage(key CombinedKey) {
	su.activeStageMap.Delete(key)
}

func (su *StageUtil) IterThroughStage(f func(key, value any) (stopIter bool)) {
	su.activeStageMap.Range(f)
}

func NewStageUtil() StageUtil {
	return StageUtil{}
}

type IStageNew interface {
	Process(t Trigger)
}
type PagerAction int

const (
	PagerPrevPage PagerAction = iota
	PagerNextPage
)

type CombinedKey string

func CombinedKeyFromRaw(args ...string) CombinedKey {
	tempKey := strings.Join(args, "-")
	return CombinedKey(tempKey)
}

type ICacheCommand interface {
	RefreshAllCache() error
	RefreshCache(cacheIdentifier string) error
}

type CacheUtil struct {
	cacheRefreshMap map[string]func() error
}

func (cm *CacheUtil) RefreshAllCache() error {
	for id, functionIn := range cm.cacheRefreshMap {
		if err := functionIn(); err != nil {
			return fmt.Errorf("cache refreshment failed for %s", id)
		}
	}
	return nil
}

func (cm *CacheUtil) RefreshCache(cacheIdentifier string) error {
	if functionIn, exist := cm.cacheRefreshMap[cacheIdentifier]; !exist {
		return fmt.Errorf("no cache named %s", cacheIdentifier)
	} else {
		return functionIn()
	}
}
