package instances

import (
	"dalian-bot/internal/pkg/clients"
	"dalian-bot/internal/pkg/commands"
	"errors"
	"fmt"
	"github.com/bwmarrin/discordgo"
	"strings"
	"time"
)

type AskCommand struct {
	commands.Command
	//handle the trigger event
	commands.PlainCommand
	//handling subsequent steps
	commands.BotCallingCommand
	//channel:asking
	ActiveAsks map[string]*AskStage
}

func (cm *AskCommand) MatchMessage(message *discordgo.MessageCreate) (bool, bool) {
	if _, ok := cm.ActiveAsks[message.Author.ID]; ok {
		matchStatus, _ := cm.IsCallingBot(message.Content)
		return matchStatus, true
	}
	matchStatus, _ := cm.MatchText(message.Content)
	return matchStatus, true
}

type AskStage struct {
	commands.BasicStageInfo
	ProcessMsgChan chan *discordgo.Message
	MainCommand    *AskCommand
}

func (as *AskStage) new(ms *discordgo.MessageCreate, cm *AskCommand) {
	as.UserID = ms.Author.ID
	as.ChannelID = ms.ChannelID
	as.StageNow = 0
	as.LastActionTime = ms.Timestamp
	as.MainCommand = cm
}

func (as *AskStage) process() {
	clients.DgSession.ChannelMessageSend(as.ChannelID, "Ask count started in new goroutine!")
	func() {
		for {
			select {
			case msg, ok := <-as.ProcessMsgChan:
				if !ok {
					//channel closed, a termination sign
					fmt.Println("terminating through closed channel")
					return
				}
				if callingBot, content := as.MainCommand.IsCallingBot(msg.Content); callingBot && content == "next" {
					as.StageNow += 1
					as.LastActionTime = time.Now()
					clients.DgSession.ChannelMessageSend(msg.ChannelID, fmt.Sprintf("processed! stage:%d", as.StageNow))
				}
			case <-time.After(15 * time.Second):
				//overtime termination sign
				fmt.Println("terminating through overtime")
				clients.DgSession.ChannelMessageSend(as.ChannelID, "15 seconds overtime")
				return
			}
		}
	}()
	as.MainCommand.disposeAsk(as.UserID)
}

func (cm *AskCommand) New() {
	cm.Name = "ask"
	cm.Identifiers = []string{"ask"}
	cm.ActiveAsks = make(map[string]*AskStage)
}

func (cm *AskCommand) DoMessage(m *discordgo.MessageCreate) error {
	if aa, ok := cm.ActiveAsks[m.Author.ID]; !ok {
		cm.insertAsk(m)
		return nil
	} else if strings.HasPrefix(m.Content, commands.Prefix) {
		clients.DgSession.ChannelMessageSend(m.ChannelID, "Detected another command, force abort")
		cm.disposeAsk(m.Author.ID)
	} else if callingBot, _ := cm.IsCallingBot(m.Content); callingBot {
		aa.ProcessMsgChan <- m.Message
	}
	return nil
}

func (cm *AskCommand) insertAsk(ms *discordgo.MessageCreate) error {
	if v, ok := cm.ActiveAsks[ms.Author.ID]; ok {
		return errors.New(fmt.Sprintf("Found an active ask session at stage %d", v.StageNow))

	}
	var as AskStage
	as.new(ms, cm)
	cm.ActiveAsks[ms.Author.ID] = &as
	as.ProcessMsgChan = make(chan *discordgo.Message, 1)
	go as.process()
	fmt.Println("Ask inserted")
	return nil
}

func (cm *AskCommand) disposeAsk(userID string) error {
	if v, ok := cm.ActiveAsks[userID]; !ok {
		return errors.New("disposing a non-exist StageNow")
	} else {
		close(v.ProcessMsgChan) // this should immediately trigger dispose
	}

	delete(cm.ActiveAsks, userID)
	fmt.Println("Ask disposed")
	return nil
}

func init() {
	var ask AskCommand
	ask.New()
	commands.RegisterCommand(&ask)
}