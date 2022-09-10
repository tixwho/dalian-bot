package commands

import "github.com/bwmarrin/discordgo"

type FixedImageCommand struct {
	Command
	PlainCommand
	ArgCommand
}

func (cm *FixedImageCommand) New() {
	cm.Name = "fixed-image"
	cm.Identifiers = []string{"fixed-image", "fi"}
}

func (cm *FixedImageCommand) Match(a ...any) bool {
	m, isMsgCreate := a[0].(*discordgo.MessageCreate)
	if !isMsgCreate {
		return false
	}
	matchStatus, _ := cm.MatchMessage(m.Message.Content)
	return matchStatus
}

func (cm *FixedImageCommand) Do(a ...any) error {
	//TODO implement me
	panic("implement me")
}
