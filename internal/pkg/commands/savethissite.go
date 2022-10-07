package commands

import (
	"dalian-bot/internal/pkg/clients"
	"dalian-bot/internal/pkg/data"
	"dalian-bot/internal/pkg/discord"
	"fmt"
	"github.com/bwmarrin/discordgo"
	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"golang.org/x/net/context"
	"net/url"
	"regexp"
	"strings"
	"time"
)

// websiteRegex: from https://mathiasbynens.be/demo/url-regex, @diegoperini
var websiteRegex = "^(?:https?://)?(?:[^/.\\s]+\\.)*[^/.\\s]+/?$"
var stageOvertime = 30

type SaveThisSiteCommand struct {
	//basic function
	Command
	//manual way of trigger
	PlainCommand
	//multiple args required for manual when necessary
	ArgCommand
	//flag support for manual trigger
	FlagCommand
	//implicit way to trigger
	RegexTextCommand
	//stepped support for implicit trigger
	BotCallingCommand
	//Map containing active implicit collecting process
	ActiveSitetageMap activeSitestageMap
}

func (cm *SaveThisSiteCommand) MatchMessage(m *discordgo.MessageCreate) (bool, bool) {
	//manual
	if matchStatus, _ := cm.MatchText(m.Content); matchStatus {
		return true, true
	}
	//stage progress
	if isCallingBot, _ := cm.IsCallingBot(m.Content); isCallingBot {
		//a stage present, check if it's a stage info
		if _, ok := cm.ActiveSitetageMap[newStageKeyFromMs(*m.Message)]; ok {
			return true, true
		}

	}
	//implicit
	if _, err := url.ParseRequestURI(m.Content); err == nil {
		//go through active stages to make sure no other in process
		if _, ok := cm.ActiveSitetageMap[newStageKeyFromMs(*m.Message)]; ok {
			discord.ChannelMessageSend(m.ChannelID, "Found an active stage, please finish that one first.")
			return false, true
		}
		return true, true
	}
	return false, true
}

type combinedKey string

type activeSitestageMap map[combinedKey]*saveSiteStage

func (m activeSitestageMap) insertStage(ms *discordgo.MessageCreate, cm *SaveThisSiteCommand) error {
	key := newStageKeyFromMs(*ms.Message)
	if stage, ok := m[key]; ok {
		return fmt.Errorf("found an active ask session at stage %d", stage.StageNow)
	}

	stage := newSitestage(ms, cm)
	stage.ProcessMsgChan = make(chan *discordgo.Message, 1)
	stage.URL = ms.Content
	m[key] = &stage
	go stage.process()
	return nil
}

func (m activeSitestageMap) disposeStage(key combinedKey) error {
	if v, ok := m[key]; !ok {
		return fmt.Errorf("disposing non-exist sitestage w/ id: %s", key)
	} else {
		close(v.ProcessMsgChan) // this should immediately trigger dispose
	}

	delete(m, key)
	return nil
}

func newStageKeyFromRaw(channelID, userID string) combinedKey {
	return combinedKey(fmt.Sprintf("%s-%s", channelID, userID))
}

func newStageKeyFromMs(ms discordgo.Message) combinedKey {
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
	URL            string
	ProcessMsgChan chan *discordgo.Message
	MainCommand    *SaveThisSiteCommand
}

func (s *saveSiteStage) process() {
	//prep
	questionPrompt := "Detected the following url:\r" +
		" %s\r" +
		"DoMessage you wish to add it to SiteCollection? (y/yes/n/no)\r" +
		"all answers should start with **@%s**, expires in %d seconds"
	promptEmbed := &discordgo.MessageEmbed{
		Description: fmt.Sprintf(questionPrompt, s.URL, clients.DgSession.State.User.Username, stageOvertime),
		Color:       discord.EmbedColorQuestion,
	}

	//start the prompt
	discord.ChannelMessageSendEmbed(s.ChannelID, promptEmbed)
	var sitePo SitePO
	func() {
		for {
			select {
			case msg, ok := <-s.ProcessMsgChan:
				if !ok {
					//channel closed, a termination sign
					//debug
					fmt.Println("terminating through closed channel")
					return
				}
				//message in the channel must be calling but.
				_, content := s.MainCommand.IsCallingBot(msg.Content)
				switch s.StageNow {
				case 0:
					if content == "y" || content == "yes" {
						sitePo = newRawSitePO(msg)
						sitePo.Site = s.URL
						prompt := "[1/2] Add tags for this site, separated by default separator, type \"-\" to leave it blank.\r" +
							"Current separator:[%s]"
						discord.ChannelMessageSend(s.ChannelID, fmt.Sprintf(prompt, Separator))
						s.StageNow += 1
					}
					if content == "n" || content == "no" {
						discord.ChannelMessageSend(s.ChannelID, "Site saving cancelled.")
						return
					}
				case 1:
					tags, count := s.MainCommand.SeparateArgs(content, Separator)
					prompt := "[2/2] Add note for this site,type \"-\" to leave it blank."
					if count == 0 {
						discord.ChannelMessageSend(s.ChannelID, "Add at least one tag, or use \"-\" to leave the field blank.")
					} else if count == 1 && tags[0] == "-" {
						//no tags
						discord.ChannelMessageSend(s.ChannelID, prompt)
						s.StageNow += 1
					} else {
						sitePo.Tags = tags
						discord.ChannelMessageSend(s.ChannelID, prompt)
						s.StageNow += 1
					}
				case 2:
					if content != "-" {
						sitePo.Note = content
					}
					//TODO snapshot things
					persistSitePo(sitePo, true)
					discord.ChannelMessageSendEmbed(msg.ChannelID, &discordgo.MessageEmbed{
						Title:       "Site saved",
						Description: "The following site has been saved",
						Timestamp:   time.Now().Format(time.RFC3339),
						Color:       discord.EmbedColorNormal,
						Fields: []*discordgo.MessageEmbedField{{
							Name:   "Temp Title",
							Value:  sitePo.essentialInfoForEmbed(),
							Inline: false,
						}},
					})
					return
				}
			case <-time.After(time.Duration(stageOvertime) * time.Second):
				//overtime termination sign
				fmt.Println("terminating through overtime")
				clients.DgSession.ChannelMessageSend(s.ChannelID, "Time's up.")
				return
			}
		}
	}()
	s.MainCommand.ActiveSitetageMap.disposeStage(newStageKeyFromRaw(s.ChannelID, s.UserID))
}

func (cm *SaveThisSiteCommand) New() {
	cm.Name = "save-this-site"
	cm.Identifiers = []string{"save-site", "list-site"}
	cm.ActiveSitetageMap = make(map[combinedKey]*saveSiteStage)
	cm.RegexExpressions = []*regexp.Regexp{}
	cm.RegexExpressions = append(cm.RegexExpressions, regexp.MustCompile(websiteRegex))
	cm.InitAvailableFlagMap()
	//the flag for taggiong sites
	cm.RegisterCommandFlag(CommandFlag{
		Name:             "tag",
		FlagPrefix:       []string{"tag", "t"},
		AcceptsExtraArg:  true,
		MultipleExtraArg: true,
		MEGroup:          nil,
	})
	//the flag for debugging flag inputs
	cm.RegisterCommandFlag(CommandFlag{
		Name:             "debug",
		FlagPrefix:       []string{"debug"},
		AcceptsExtraArg:  false,
		MultipleExtraArg: false,
		MEGroup:          nil,
	})
	//the flag for adding notes to sites
	cm.RegisterCommandFlag(CommandFlag{
		Name:             "note",
		FlagPrefix:       []string{"note", "n"},
		AcceptsExtraArg:  true,
		MultipleExtraArg: false,
		MEGroup:          nil,
	})
	//the flag for using next-generation interactions.
	cm.RegisterCommandFlag(CommandFlag{
		Name:             "neo",
		FlagPrefix:       []string{"neo"},
		AcceptsExtraArg:  false,
		MultipleExtraArg: false,
		MEGroup:          nil,
	})

}

func (cm *SaveThisSiteCommand) DoMessage(m *discordgo.MessageCreate) error {
	//first check manual
	//one-step insertion, or list / other operations
	if matchStatus, matchedCommand := cm.MatchText(m.Content); matchStatus {
		args, length := cm.SeparateArgs(m.Content, Separator)
		//read the flags
		flagMap, err := cm.ParseFlags(args[0])
		if err != nil {
			discord.ChannelReportError(m.ChannelID, err)
			return nil
		}
		//validate the flags
		flagMap, err = cm.ValidateFlagMap(flagMap)
		if err != nil {
			discord.ChannelReportError(m.ChannelID, err)
			return nil
		}
		if flagMap.HasFlag("debug") {
			discord.ChannelMessageSend(m.ChannelID, fmt.Sprintf("flagMap:%v", flagMap))
		}
		//execute command body
		switch matchedCommand {
		case "save-site":
			//make sure arg[1] has a valid url
			if length <= 1 {
				discord.ChannelMessageSend(m.ChannelID, "You need an url as the second argument!")
				return nil
			}
			if _, err := url.ParseRequestURI(args[1]); err != nil {
				discord.ChannelMessageSend(m.ChannelID, "The second argument must be a VALID url!")
				return nil
			}
			//validation passed, start the logic
			sitePO := newRawSitePO(m.Message)
			//set site
			sitePO.Site = args[1]
			//set tags
			if len(flagMap["tag"]) > 0 {
				sitePO.Tags = flagMap["tag"]
			}
			//set note
			if len(flagMap["note"]) > 0 {
				sitePO.Note = flagMap["note"][0]
			}
			//save it to the database
			go persistSitePo(sitePO, true)
			// discord.ChannelMessageSend(m.ChannelID, fmt.Sprintf("Site saved:%s", sitePO.essentialInfo()))
			//TODO: snapshot things.
			discord.ChannelMessageSendEmbed(m.ChannelID, &discordgo.MessageEmbed{
				Title:       "Site saved",
				Description: "The following site has been saved",
				Timestamp:   time.Now().Format(time.RFC3339),
				Color:       discord.EmbedColorNormal,
				Fields: []*discordgo.MessageEmbedField{{
					Name:   "Temp Title",
					Value:  sitePO.essentialInfoForEmbed(),
					Inline: false,
				}},
			})
			//no subsequent check
			return nil
		case "list-site":
			query := bson.M{"user_id": m.Author.ID, "guild_id": m.GuildID}
			//optional url
			if len(flagMap["tag"]) > 0 {
				if flagMap["tag"][0] == "-" {
					query["tags"] = nil
				} else {
					query["tags"] = bson.M{"$all": flagMap["tag"]}
				}
			}
			findOpts := options.Find().SetSort(bson.D{{"id", -1}})
			findCursor, err := data.GetCollection("site_collection").Find(context.TODO(), query, findOpts)
			var results sitePoArr
			if err = findCursor.All(context.TODO(), &results); err != nil {
				fmt.Println("ERROR:" + err.Error())
				return err
			}
			// clients.DgSession.ChannelMessageSend(m.ChannelID, fmt.Sprintf("Found Doc(s):%s", results.digestInfo()))
			_, err = discord.ChannelMessageSendEmbed(m.ChannelID, newListSiteEmbed(results))
			if err != nil {
				fmt.Println(err)
			}
		}

	}

	//then check implicit
	if _, err1 := url.ParseRequestURI(m.Content); err1 == nil {
		//calling insertStage to start a goroutine for stepped Q&A
		//the stage will auto dispose.
		if err := cm.ActiveSitetageMap.insertStage(m, cm); err != nil {
			discord.ChannelReportError(m.ChannelID, err)
		}
		//no subsequent check
		return nil
	}

	//finally check stage progress
	if activeStage, ok := cm.ActiveSitetageMap[newStageKeyFromMs(*m.Message)]; ok {
		//a stage present, check if it's a stage info
		if isCallingBot, _ := cm.IsCallingBot(m.Content); isCallingBot {
			//@Bot content, pass to process for further check
			activeStage.ProcessMsgChan <- m.Message
		}

	}

	return nil
}

type SitePO struct {
	//Display Info
	ID   int      `bson:"id"`
	Site string   `bson:"site"`
	Tags []string `bson:"tags"`
	Note string   `bson:"note"`
	//Retrieved Info
	Title       string `bson:"title"`
	SnapshotURL string `bson:"snapshot_url,omitempty"`
	//Credential Info
	GuildID   string `bson:"guild_id"`
	ChannelID string `bson:"channel_id"`
	UserID    string `bson:"user_id"`
	//Auditing info
	CreatedTime      time.Time `bson:"created_time"`
	LastModifiedTime time.Time `bson:"last_modified_time"`
}

func (sp *SitePO) setTime(isCreate bool) {
	currentTime := time.Now()
	if isCreate {
		sp.CreatedTime = currentTime
	}
	sp.LastModifiedTime = currentTime
}

func (sp *SitePO) essentialInfo() string {
	var tags, note string
	if len(sp.Tags) == 0 {
		tags = "*None*"
	} else {
		tags = "[" + strings.Join(sp.Tags, ",") + "]"
	}

	if sp.Note == "" {
		note = "*None*"
	} else {
		note = sp.Note
	}
	essentialInfo := "> Site: %s\r" +
		"> Tags: %s\r" +
		"> Note: %s"
	return fmt.Sprintf(essentialInfo, sp.Site, tags, note)
}

func (sp *SitePO) essentialInfoForEmbed() string {
	var tags, note string
	if len(sp.Tags) == 0 {
		tags = "*None*"
	} else {
		tags = "[" + strings.Join(sp.Tags, ",") + "]"
	}

	if sp.Note == "" {
		note = "*None*"
	} else {
		note = sp.Note
	}
	essentialInfo := "%s\r" +
		"Tags: %s\r" +
		" Note: %s"
	return fmt.Sprintf(essentialInfo, sp.Site, tags, note)
}

func newRawSitePO(message *discordgo.Message) SitePO {
	sitepo := SitePO{
		Site:      "",
		Tags:      []string{},
		Note:      "",
		GuildID:   message.GuildID,
		ChannelID: message.ChannelID,
		UserID:    message.Author.ID,
	}
	return sitepo
}

func getLastNumericalID() int {

	singleResult := data.GetCollection("site_collection").FindOne(context.TODO(), bson.M{}, options.FindOne().SetSort(bson.M{"id": -1}))
	if singleResult.Err() == mongo.ErrNoDocuments {
		return 0
	}
	var doc SitePO
	if err := singleResult.Decode(&doc); err != nil {
		fmt.Println("Unable to Decode doc into SitePo", err)
		return -1
	}
	return doc.ID
}

func persistSitePo(po SitePO, isCreate bool) error {
	idNow := getLastNumericalID() + 1
	if idNow == 0 {
		return errors.New("Something wrong retrieving last id")
	}
	po.ID = idNow
	po.setTime(isCreate)
	_, err := data.GetCollection("site_collection").InsertOne(context.TODO(), po)
	if err != nil {
		return err
	}
	//TODO need to get a numerical id first
	return nil
}

type sitePoArr []*SitePO

func (arr sitePoArr) digestInfo() string {
	info := ""
	for _, v := range arr {
		info += "\r" + v.essentialInfo() + "\r"
	}
	info = fmt.Sprintf("{%s}", info)
	return info
}

func newListSiteEmbed(arr sitePoArr) *discordgo.MessageEmbed {
	embed := &discordgo.MessageEmbed{
		Title:       "ls result",
		Description: fmt.Sprintf("Your query rendered %d results.", len(arr)),
		Timestamp:   time.Now().Format(time.RFC3339),
		Color:       discord.EmbedColorNormal,
		Fields:      nil,
	}
	var fields []*discordgo.MessageEmbedField
	for _, sitePo := range arr {
		fields = append(fields, &discordgo.MessageEmbedField{
			Name:   fmt.Sprintf("Temporary Title"),
			Value:  sitePo.essentialInfoForEmbed(),
			Inline: false,
		})
	}
	embed.Fields = fields
	return embed
}

func init() {
	var saveThisCommand SaveThisSiteCommand
	saveThisCommand.New()
	RegisterCommand(&saveThisCommand)
}
