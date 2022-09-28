package commands

import (
	"dalian-bot/internal/pkg/clients"
	"errors"
	"fmt"
	"github.com/bwmarrin/discordgo"
	"strings"
	"time"
)

type AskCommand struct {
	Command
	//handle the trigger event
	PlainCommand
	//handling subsequent steps
	BotCallingCommand
	//channel:asking
	ActiveAsks map[string]*AskStage
}

type AskStage struct {
	UserId         string
	ChannelId      string
	AskStage       int
	ProcessChan    chan *discordgo.Message
	MainCommand    *AskCommand
	LastActionTime time.Time
}

func (as *AskStage) new(ms *discordgo.MessageCreate, cm *AskCommand) {
	as.UserId = ms.Author.ID
	as.ChannelId = ms.ChannelID
	as.AskStage = 0
	as.LastActionTime = ms.Timestamp
	as.MainCommand = cm
}

func (as *AskStage) process(command *AskCommand) {
	clients.DgSession.ChannelMessageSend(as.ChannelId, "Ask count started in new goroutine!")
	func() {
		for {
			select {
			case msg, ok := <-as.ProcessChan:
				if !ok {
					//channel closed, a termination sign
					fmt.Println("terminating through closed channel")
					return
				}
				if callingBot, content := as.MainCommand.IsCallingBot(msg.Content); callingBot && content == "next" {
					as.AskStage += 1
					as.LastActionTime = time.Now()
					clients.DgSession.ChannelMessageSend(msg.ChannelID, fmt.Sprintf("processed! stage:%d", as.AskStage))
				}
			case <-time.After(15 * time.Second):
				//overtime termination sign
				fmt.Println("terminating through overtime")
				clients.DgSession.ChannelMessageSend(as.ChannelId, "15 seconds overtime")
				return
			}

		}
	}()
	command.disposeAsk(as.UserId)
}

func (cm *AskCommand) New() {
	cm.Name = "ask"
	cm.Identifiers = []string{"ask"}
	cm.ActiveAsks = make(map[string]*AskStage)
}

func (cm *AskCommand) Match(a ...any) bool {
	m, isMsgCreate := a[0].(*discordgo.MessageCreate)
	if !isMsgCreate {
		return false
	}
	if _, ok := cm.ActiveAsks[m.Author.ID]; ok {
		matchStatus, _ := cm.IsCallingBot(m.Content)
		return matchStatus
	}
	matchStatus, _ := cm.MatchMessage(m.Message.Content)
	return matchStatus
}

func (cm *AskCommand) Do(a ...any) error {
	m := a[0].(*discordgo.MessageCreate)
	if aa, ok := cm.ActiveAsks[m.Author.ID]; !ok {
		cm.insertAsk(m)
		return nil
	} else if strings.HasPrefix(m.Content, Prefix) {
		clients.DgSession.ChannelMessageSend(m.ChannelID, "Detected another command, force abort")
		cm.disposeAsk(m.Author.ID)
	} else if callingBot, _ := cm.IsCallingBot(m.Content); callingBot {
		aa.ProcessChan <- m.Message
	}
	return nil
}

func (cm *AskCommand) insertAsk(ms *discordgo.MessageCreate) error {
	if v, ok := cm.ActiveAsks[ms.Author.ID]; ok {
		return errors.New(fmt.Sprintf("Found an active ask session at stage %d", v.AskStage))

	}
	var as AskStage
	as.new(ms, cm)
	cm.ActiveAsks[ms.Author.ID] = &as
	as.ProcessChan = make(chan *discordgo.Message, 1)
	go as.process(cm)
	fmt.Println("Ask inserted")
	return nil
}

func (cm *AskCommand) disposeAsk(userID string) error {
	if v, ok := cm.ActiveAsks[userID]; !ok {
		return errors.New("disposing a non-exist AskStage")
	} else {
		close(v.ProcessChan) // this should immediately trigger dispose
	}

	delete(cm.ActiveAsks, userID)
	fmt.Println("Ask disposed")
	return nil
}

func init() {
	var ask AskCommand
	ask.New()
	RegisterCommand(&ask)
}
