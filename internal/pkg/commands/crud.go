package commands

import (
	"dalian-bot/internal/pkg/clients"
	"fmt"
	"github.com/bwmarrin/discordgo"
)

type CrudCommand struct {
	Command
	PlainCommand
	ArgCommand
	FlagCommand
}

func (c *CrudCommand) New() {
	c.Name = "crud"
	c.Identifiers = []string{"crud", "crud-second"}
}

func (c *CrudCommand) Match(a ...any) bool {
	m, isMsgCreate := a[0].(*discordgo.MessageCreate)
	if !isMsgCreate {
		return false
	}
	matchStatus, _ := c.MatchMessage(m.Message.Content)
	return matchStatus
}

func (c *CrudCommand) Do(a ...any) error {
	m := a[0].(*discordgo.MessageCreate)
	if err := c.ParseFlags(m.Message.Content); err != nil {
		fmt.Println(err)
		return err
	}
	clients.DgSession.ChannelMessageSend(m.ChannelID, fmt.Sprintf("Successfully read arguments w/ flag! \r\n %v", c.FlagArgstatMaps))
	return nil
}

func init() {
	var crud CrudCommand
	crud.New()
	RegisterCommand(&crud)
}
