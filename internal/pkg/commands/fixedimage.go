package commands

import (
	"dalian-bot/internal/pkg/clients"
	"dalian-bot/internal/pkg/discord"
	"github.com/bwmarrin/discordgo"
	"os"
)

type FixedImageCommand struct {
	Command
	PlainCommand
	ArgCommand
	imageMap map[string]string
}

func (cm *FixedImageCommand) MatchMessage(message *discordgo.MessageCreate) (bool, bool) {
	matchStatus, _ := cm.MatchText(message.Content)
	return matchStatus, true
}

func (cm *FixedImageCommand) New() {
	cm.Name = "fixed-image"
	cm.Identifiers = []string{"fixed-image", "fi"}
	cm.imageMap = make(map[string]string)
	cm.imageMap["hikari-mourn"] = "static/hikari-mourn.gif"
	cm.imageMap["tairitsu-dragon"] = "static/tairitsu-dragon.png"
}

func (cm *FixedImageCommand) DoMessage(m *discordgo.MessageCreate) error {
	args, argCount := cm.SeparateArgs(m.Message.Content, Separator)
	if argCount <= 1 {
		clients.DgSession.ChannelMessageSend(m.ChannelID, "not enough arguments!")
		return nil
	}
	if v, ok := cm.imageMap[args[1]]; !ok {
		clients.DgSession.ChannelMessageSend(m.ChannelID, "unknown emote argument!")
		return nil
	} else {
		if f, err := os.Open(v); err == nil {
			defer f.Close()
			err := discord.ChannelFileSend(m.ChannelID, f.Name(), f)
			return err
		}
	}
	//shouldn't be reached
	return nil
}

func init() {
	var fic FixedImageCommand
	fic.New()
	RegisterCommand(&fic)
}
