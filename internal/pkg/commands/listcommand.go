package commands

import (
	"dalian-bot/internal/pkg/clients"
	"dalian-bot/internal/pkg/discord"
	"fmt"
	"github.com/bwmarrin/discordgo"
	"strings"
	"time"
)

type ListCommand struct {
	Command
	PlainCommand
	SlashCommand
	ComponentCommand
}

func (cm *ListCommand) DoComponent(i *discordgo.InteractionCreate) error {
	if componentDo, ok := cm.CompActionMap[i.MessageComponentData().CustomID]; ok {
		componentDo(i)
	}
	return nil
}

func (cm *ListCommand) MatchInteraction(i *discordgo.InteractionCreate) (isMatched bool) {
	if i.ApplicationCommandData().Name == cm.AppCommand.Name {
		return true
	}
	return false
}

func (cm *ListCommand) DoInteraction(i *discordgo.InteractionCreate) (err error) {

	optionsMap := cm.ParseOptionsMap(i.ApplicationCommandData().Options)
	names := make([]string, 0, len(CommandByName))
	if option, ok := optionsMap["qualifier"]; ok {
		for k := range CommandByName {
			if strings.Contains(k, option.StringValue()) {
				names = append(names, k)
			}
		}
	} else {
		for k := range CommandByName {
			names = append(names, k)
		}
	}
	clients.DgSession.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("Commands: %v", names),
		},
	})
	return nil
}

func (cm *ListCommand) MatchMessage(message *discordgo.MessageCreate) (bool, bool) {
	matchStatus, _ := cm.MatchText(message.Content)
	return matchStatus, true
}

func (cm *ListCommand) New() {
	cm.Name = "list-command"
	cm.Identifiers = []string{"list", "l"}
	cm.AppCommand = &discordgo.ApplicationCommand{
		Name:        "list-command",
		Description: "List the name of all available commands.",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "qualifier",
				Description: "Online commands include the string will be shown",
				Required:    false,
			},
		},
	}
	cm.CompActionMap = make(ComponentActionMap)
	cm.CompActionMap["list-command-good"] = func(i *discordgo.InteractionCreate) {
		if i.Message != nil {
			i.Message.Embeds[0].Color = discord.EmbedColorSuccess
			clients.DgSession.ChannelMessageEditEmbeds(i.Message.ChannelID, i.Message.ID, i.Message.Embeds)
		}
		clients.DgSession.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "Thank you for your approve!",
			},
		})
	}
	cm.CompActionMap["list-command-bad"] = func(i *discordgo.InteractionCreate) {
		if i.Message != nil {
			i.Message.Embeds[0].Color = discord.EmbedColorDanger
			clients.DgSession.ChannelMessageEditEmbeds(i.Message.ChannelID, i.Message.ID, i.Message.Embeds)
		}
		clients.DgSession.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseUpdateMessage,
			Data: &discordgo.InteractionResponseData{
				Content: "Gah.",
			},
		})
		time.AfterFunc(5*time.Second, func() {
			msg, err := clients.DgSession.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{Content: "fuckyou."})
			if err != nil {
				fmt.Println(err)
			} else {
				time.AfterFunc(2*time.Second, func() {
					clients.DgSession.FollowupMessageDelete(i.Interaction, msg.ID)
				})
			}
		})

	}
}

func (cm *ListCommand) DoMessage(m *discordgo.MessageCreate) error {
	names := make([]string, 0, len(CommandByName))
	for k := range CommandByName {
		names = append(names, k)
	}
	var fields []*discordgo.MessageEmbedField
	for _, v := range names {
		fields = append(fields, &discordgo.MessageEmbedField{
			Name:   v,
			Value:  "a command.",
			Inline: false,
		})
	}
	_, err := clients.DgSession.ChannelMessageSendComplex(m.ChannelID, &discordgo.MessageSend{
		Content: "Listing registered commands",
		Components: []discordgo.MessageComponent{
			discordgo.ActionsRow{Components: []discordgo.MessageComponent{
				discordgo.Button{
					Label:    "Good",
					Style:    discordgo.SuccessButton,
					CustomID: "list-command-good",
				},
				discordgo.Button{
					Label:    "Bad",
					Style:    discordgo.DangerButton,
					CustomID: "list-command-bad",
				},
			}},
		},
		Embeds: []*discordgo.MessageEmbed{
			{
				Type:        discordgo.EmbedTypeRich,
				Title:       "Embed",
				Description: "Desc",
				Fields:      fields,
			},
		},
	})
	if err != nil {
		fmt.Println(err)
	}
	return nil
}

func init() {
	var lc ListCommand
	lc.New()
	RegisterCommand(&lc)
}
