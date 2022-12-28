package discord

import (
	"dalian-bot/internal/pkg/core"
	"errors"
	"fmt"
	"github.com/bwmarrin/discordgo"
	"sort"
	"strings"
	"time"
)

func GenerateHelper(config HelperConfig) Helper {
	helpMap := make(map[string]string)
	for _, v := range config.CommandHelps {
		helpMap[v.Name] = v.FormattedHelp
	}
	return Helper{
		pluginHelpText: config.PluginHelp,
		commandsHelp:   helpMap,
	}
}

type Helper struct {
	pluginHelpText string
	commandsHelp   map[string]string
}

func (h Helper) DiscordPluginHelp(pluginName string) string {
	var commandsName []string
	for k := range h.commandsHelp {
		commandsName = append(commandsName, k)
	}
	sort.Strings(commandsName)
	return fmt.Sprintf("Commands provided by *%s* plugin: %v", pluginName, commandsName)
}

func (h Helper) DiscordCommandHelp(text string) string {
	if help, ok := h.commandsHelp[text]; ok {
		return fmt.Sprintf("**%s**\r%s", text, help)
	} else {
		return ""
	}
}

type HelperConfig struct {
	PluginHelp   string
	CommandHelps []CommandHelp
}

type CommandHelp struct {
	Name          string
	FormattedHelp string
}

type IDisrocdHelper interface {
	DiscordPluginHelp(pluginName string) string
	DiscordCommandHelp(text string) string
}

type ITextCommand interface {
	// Deprecated: use MatchMessageNew
	MatchMessage(m *discordgo.MessageCreate) (isMatched bool, isTerminated bool)
	// DoMessage All command wlll do something
	// a anything you may need to execute. It is your OWN responsibility to validate before use.
	// err if anything worth *reporting* happened. expected error should not be returned.
	DoMessage(m *discordgo.MessageCreate) (err error)

	DoMessageNew(m *discordgo.MessageCreate)
}

type ITextCommandNew interface {
	DoMessage(b *core.Bot, m *discordgo.MessageCreate) (err error)
}

type AppCommandsMap map[string]*discordgo.ApplicationCommand

func (acm *AppCommandsMap) RegisterCommand(cmd *discordgo.ApplicationCommand) {
	(*acm)[cmd.Name] = cmd
}

type ISlashCommand interface {
	MatchNamedInteraction(i *discordgo.InteractionCreate) (isMatched bool)
	DoNamedInteraction(i *discordgo.InteractionCreate) (err error)
	GetAppCommandsMap() AppCommandsMap
}

type ISlashCommandNew interface {
	DoNamedInteraction(b *core.Bot, i *discordgo.InteractionCreate) (err error)
	GetAppCommandsMap() AppCommandsMap
}

type SlashCommand struct {
	AppCommandsMap AppCommandsMap
}

func (cm *SlashCommand) GetAppCommandsMap() AppCommandsMap {
	return cm.AppCommandsMap
}

func (cm *SlashCommand) DefaultMatchCommand(i *discordgo.InteractionCreate) (bool, string) {
	for _, slashCmd := range cm.AppCommandsMap {
		if i.ApplicationCommandData().Name == slashCmd.Name {
			return true, slashCmd.Name
		}
	}
	return false, ""
}

func (cm *SlashCommand) ParseOptionsMap(options []*discordgo.ApplicationCommandInteractionDataOption) map[string]*discordgo.ApplicationCommandInteractionDataOption {
	optionMap := make(map[string]*discordgo.ApplicationCommandInteractionDataOption)
	for _, opt := range options {
		optionMap[opt.Name] = opt
	}
	return optionMap
}

// IBotCallingCommand TextCommand starts with a @bot
type IBotCallingCommand interface {
	IsCallingBot(content string) bool
}

// BotCallingCommand Functional structure for BotCallingCommand
// embedded a method for identifying @Bot texts
type BotCallingCommand struct {
}

// IsCallingBot Return true if the text starts with @{BotID}
func (b BotCallingCommand) IsCallingBot(content string) (isCalling bool, sanitizedContent string) {
	callingStr := fmt.Sprintf("<@%s>", core.BotID)
	if strings.HasPrefix(content, callingStr) {
		return true, strings.TrimSpace(strings.Replace(content, callingStr, "", 1))
	}
	return false, ""
}

type ComponentActionMap map[string]func(i *discordgo.InteractionCreate)

type IComponentCommand interface {
	GetCompActionMap() ComponentActionMap
	IsComponentInteraction(i *discordgo.InteractionCreate) bool
	DoComponent(i *discordgo.InteractionCreate) error
}

type ComponentCommand struct {
	CompActionMap ComponentActionMap
}

func (cm *ComponentCommand) GetCompActionMap() ComponentActionMap {
	return cm.CompActionMap
}

func (cm *ComponentCommand) IsComponentInteraction(i *discordgo.InteractionCreate) bool {
	if i.Type == discordgo.InteractionMessageComponent {
		return true
	}
	return false
}

type IPagerLoader interface {
	//LoadPager initialize the pager
	LoadPager(pager *Pager) error
	//RenderPage render the given page
	//index order of the first item
	//limit max number of items in a page
	//embedFrame given embed frame of render
	//renderedEmbed *discordgo.MessageEmbed rendered, unsent.
	RenderPage(pager *Pager, toPage, limit int, embedFrame discordgo.MessageEmbed) (renderedEmbed *discordgo.MessageEmbed, err error)
}

type Pager struct {
	//core page loading functions, to be implemented
	IPagerLoader
	discordService *Service
	//autofilled later
	AttachedMessage *discordgo.Message
	//pagination cache. need fo fill Limit
	PageNow, PageMax, Limit int
	//embed rendering skeleton
	EmbedFrame *discordgo.MessageEmbed
	//Customized pagination button
	PrevPageButton, NextPageButton discordgo.Button
	//Overtime time to expire the pager
	Overtime time.Duration
	//only used when not lazy loading
	CompleteItemSlice []*IPagerPart
	displayItemSlice  []*IPagerPart
	//calculated actionsRow
	actionsRow discordgo.ActionsRow
}

// Setup initialize a pager AND send an initial message with interaction components
func (bp *Pager) Setup(trigger any, service *Service) error {

	bp.discordService = service
	//initialize pager
	if err := bp.IPagerLoader.LoadPager(bp); err != nil {
		return err
	}

	//initialize first page
	filledFrame, err := bp.IPagerLoader.RenderPage(bp, bp.PageNow, bp.Limit, *bp.EmbedFrame)
	if err != nil {
		return err
	}

	//initialize buttons
	var components []discordgo.MessageComponent
	if bp.PageMax <= 1 {
		//no buttons rendered for only one page
		components = nil
	} else {
		//setup pagination buttons otherwise
		bp.actionsRow = discordgo.ActionsRow{Components: []discordgo.MessageComponent{bp.PrevPageButton, bp.NextPageButton}}
		components = append(components, bp.actionsRow)
	}

	//work for both Interaction(Slash commands) and raw trigger
	if i, ok := trigger.(*discordgo.Interaction); ok {
		//Interaction (Slash)
		if err := bp.discordService.InteractionRespondEmbed(i, filledFrame, components); err != nil {
			return err
		}
		if attachedMsg, err := bp.discordService.InteractionResponse(i); err != nil {
			return fmt.Errorf("failed loading attached message from interaction%w", err)
		} else {
			bp.AttachedMessage = attachedMsg
		}
	} else if m, ok := trigger.(*discordgo.Message); ok {
		//Raw command (Message)
		if attachedMessage, err := bp.discordService.ChannelMessageSendEmbed(m.ChannelID, filledFrame); err != nil {
			return fmt.Errorf("failed loading attached message from message%w", err)
		} else {
			bp.AttachedMessage = attachedMessage
		}
	} else {
		return errors.New("unknown trigger type, pager initialization failed")
	}

	return nil
}

// SwitchPage switch the page for a given pager.
// no verification process involved
func (bp *Pager) SwitchPage(a core.PagerAction, i *discordgo.Interaction) error {
	//render page
	switch a {
	case core.PagerPrevPage:
		newEmbed, err := bp.RenderPage(bp, bp.PageNow-1, bp.Limit, *bp.EmbedFrame)
		if err != nil {
			return err
		}
		bp.AttachedMessage.Embeds[0] = newEmbed
	case core.PagerNextPage:
		newEmbed, err := bp.RenderPage(bp, bp.PageNow+1, bp.Limit, *bp.EmbedFrame)
		if err != nil {
			return err
		}
		bp.AttachedMessage.Embeds[0] = newEmbed
	}
	//edit response
	err := bp.discordService.InteractionRespondEditFromMessage(i, bp.AttachedMessage)
	if err != nil {
		return err
	}
	return nil
}

// LockPagerButtons disable buttons of the pager
func (bp *Pager) LockPagerButtons() error {
	//new array with disabled buttons
	var components []discordgo.MessageComponent
	bp.PrevPageButton.Disabled = true
	bp.NextPageButton.Disabled = true
	bp.actionsRow = discordgo.ActionsRow{Components: []discordgo.MessageComponent{bp.PrevPageButton, bp.NextPageButton}}
	components = append(components, bp.actionsRow)

	//raw edit.
	editedMsg, err := bp.discordService.Session.ChannelMessageEditComplex(&discordgo.MessageEdit{
		Content:    &bp.AttachedMessage.Content,
		Components: components,
		Embeds:     bp.AttachedMessage.Embeds,
		ID:         bp.AttachedMessage.ID,
		Channel:    bp.AttachedMessage.ChannelID,
	})
	if err != nil {
		return err
	}
	bp.AttachedMessage = editedMsg
	return nil
}

type IPagerPart interface {
	ToMessageEmbedField() *discordgo.MessageEmbedField
}

type DefaultPageRenderer struct{}

func (DefaultPageRenderer) RenderPage(pager *Pager, toPage, limit int, embedFrame discordgo.MessageEmbed) (renderedEmbed *discordgo.MessageEmbed, err error) {
	//prepare
	totalSize := len(pager.CompleteItemSlice)
	maxPage := totalSize / limit
	//page logic
	if totalSize%limit != 0 {
		maxPage += 1
	}
	pager.PageMax = maxPage
	//boundary limit
	if toPage > maxPage {
		toPage = 1
	} else if toPage < 1 {
		toPage = maxPage
	}
	//boundary limit 2: nothing to show
	if totalSize == 0 {
		embedFrame.Description = "Your query rendered 0 result. Nothing to show."
		return &embedFrame, nil
	}
	//split slice
	lowerLimit := (toPage - 1) * limit
	upperLimit := toPage * limit
	if toPage == maxPage {
		upperLimit = len(pager.CompleteItemSlice)
	}
	pager.displayItemSlice = pager.CompleteItemSlice[lowerLimit:upperLimit]
	//rendering
	var alterFields []*discordgo.MessageEmbedField
	for _, pagerPart := range pager.displayItemSlice {
		var part = *pagerPart
		alterFields = append(alterFields, part.ToMessageEmbedField())
	}
	embedFrame.Fields = alterFields
	embedFrame.Footer = &discordgo.MessageEmbedFooter{Text: fmt.Sprintf("page: %d/%d", toPage, maxPage)}
	//setup pageNow
	pager.PageNow = toPage
	return &embedFrame, nil
}

func FindFirstNonBotMsg(messages []*discordgo.Message) (*discordgo.Message, bool) {
	for _, v := range messages {
		if !v.Author.Bot {
			return v, true
		}
	}
	return nil, false
}
