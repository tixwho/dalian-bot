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
	c.AvailableFlagMap = make(map[string]*CommandFlag)
	c.RegisterCommandFlag(CommandFlag{
		Name:             "create",
		FlagPrefix:       []string{"c", "create"},
		RequiresExtraArg: false,
		MultipleExtraArg: false,
		MEGroup:          []string{"o"},
	})
	c.RegisterCommandFlag(CommandFlag{
		Name:             "delete",
		FlagPrefix:       []string{"d", "delete"},
		RequiresExtraArg: false,
		MultipleExtraArg: false,
		MEGroup:          []string{"o"},
	})
	c.RegisterCommandFlag(CommandFlag{
		Name:             "free",
		FlagPrefix:       []string{"f", "free"},
		RequiresExtraArg: true,
		MultipleExtraArg: false,
		MEGroup:          []string{},
	})
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
	if err := c.ValidateFlagMap(); err != nil {
		clients.DgSession.ChannelMessageSend(m.ChannelID, err.Error())
		return err
	}
	clients.DgSession.ChannelMessageSend(m.ChannelID, fmt.Sprintf("Successfully read arguments w/ flag! \r %v", c.FlagArgstatMaps))
	return nil
}

func init() {
	var crud CrudCommand
	crud.New()
	RegisterCommand(&crud)
}
