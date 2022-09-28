package commands

import (
	"fmt"
	"github.com/bwmarrin/discordgo"
	"time"
)

type SaveThisSiteCommand struct {
	//basic function
	Command
	//manual way of trigger
	PlainCommand
	//flag support for manual trigger
	FlagCommand
	//implicit way to trigger
	RegexTextCommand
	//stepped support for implicit trigger
	BotCallingCommand
	//Map containing active implicit collecting process
	ActiveSitetageMap activeSitestageMap
}

type combinedKey string

type activeSitestageMap map[combinedKey]*saveSiteStage

func (m activeSitestageMap) insertStage(ms *discordgo.MessageCreate, cm *SaveThisSiteCommand) error {
	key := newStageKeyFromMsCreate(*ms)
	if stage, ok := m[key]; ok {
		return fmt.Errorf("found an active ask session at stage %d", stage.StageNow)
	}

	stage := newSitestage(ms, cm)
	stage.ProcessMsgChan = make(chan *discordgo.Message, 1)
	go stage.process()
	//debug
	fmt.Println("Sitestage inserted")
	return nil
}

func (m activeSitestageMap) disposeStage(key combinedKey) error {
	if v, ok := m[key]; !ok {
		return fmt.Errorf("disposing non-exist sitestage w/ id: %s", key)
	} else {
		close(v.ProcessMsgChan) // this should immediately trigger dispose
	}

	delete(m, key)
	//debug
	fmt.Printf("sitestage[%s] disposed.", key)
	return nil
}

func newStageKeyFromRaw(channelID, userID string) combinedKey {
	return combinedKey(fmt.Sprintf("%s-%s", channelID, userID))
}

func newStageKeyFromMsCreate(ms discordgo.MessageCreate) combinedKey {
	return newStageKeyFromRaw(ms.ChannelID, ms.Author.ID)
}

func newSitestage(ms *discordgo.MessageCreate, cm *SaveThisSiteCommand) saveSiteStage {
	stage := saveSiteStage{
		BasicStageInfo: BasicStageInfo{
			ChannelID:      ms.ChannelID,
			UserID:         ms.Author.ID,
			StageNow:       0,
			LastActionTime: time.Now(),
		},
		ProcessMsgChan: nil,
		MainCommand:    cm,
	}
	return stage
}

type saveSiteStage struct {
	BasicStageInfo
	ProcessMsgChan chan *discordgo.Message
	MainCommand    *SaveThisSiteCommand
}

func (s *saveSiteStage) process() {
	//TODO implement me
	panic("implement me")
}

func (s *SaveThisSiteCommand) New() {
	s.Name = "save-this-site"
	s.Identifiers = []string{"save-site"}
	s.ActiveSitetageMap = make(map[combinedKey]*saveSiteStage)
	//TODO implement me
	panic("implement me")
}

func (s *SaveThisSiteCommand) Match(a ...any) bool {
	//TODO implement me
	panic("implement me")
}

func (s *SaveThisSiteCommand) Do(a ...any) error {
	//TODO implement me
	panic("implement me")
}
