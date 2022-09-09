package commands

import (
	"errors"
	"fmt"
)

var (
	CommandByName = make(map[string]ICommand)
)

func RegisterCommand(command ICommand) error {
	name := command.GetName()
	if _, e := CommandByName[name]; e {
		return errors.New(fmt.Sprintf("command %s already exist!", name))
	}
	CommandByName[name] = command
	return nil
}
