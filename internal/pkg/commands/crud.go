package commands

import (
	"dalian-bot/internal/pkg/clients"
	"dalian-bot/internal/pkg/data"
	"fmt"
	"github.com/bwmarrin/discordgo"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
	"golang.org/x/net/context"
	"time"
)

type CrudCommand struct {
	Command
	PlainCommand
	ArgCommand
	FlagCommand
}

const (
	CRUDOperation = "o" // CRUDOperation: unique operation flag
)

func (c *CrudCommand) New() {
	c.Name = "crud"
	c.Identifiers = []string{"crud", "crud-second"}
	c.AvailableFlagMap = make(map[string]*CommandFlag)
	c.RegisterCommandFlag(CommandFlag{
		Name:             "create",
		FlagPrefix:       []string{"c", "create"},
		AcceptsExtraArg:  false,
		MultipleExtraArg: false,
		MEGroup:          []string{CRUDOperation},
	})
	c.RegisterCommandFlag(CommandFlag{
		Name:             "read",
		FlagPrefix:       []string{"r", "read"},
		AcceptsExtraArg:  false,
		MultipleExtraArg: false,
		MEGroup:          []string{CRUDOperation},
	})
	c.RegisterCommandFlag(CommandFlag{
		Name:             "delete",
		FlagPrefix:       []string{"d", "delete"},
		AcceptsExtraArg:  false,
		MultipleExtraArg: false,
		MEGroup:          []string{CRUDOperation},
	})
	c.RegisterCommandFlag(CommandFlag{
		Name:             "free",
		FlagPrefix:       []string{"f", "free"},
		AcceptsExtraArg:  true,
		MultipleExtraArg: false,
		MEGroup:          []string{},
	})
	c.RegisterCommandFlag(CommandFlag{
		Name:             "one_argument",
		FlagPrefix:       []string{"one", "one_argument"},
		AcceptsExtraArg:  true,
		MultipleExtraArg: false,
		MEGroup:          []string{},
	})
}

type CrudTestStruct struct {
	MsgID        string
	ChannelID    string
	MsgInfo      string
	AuthorID     string
	MsgTimestamp time.Time
}

func newTestStruct(createEvent *discordgo.MessageCreate) CrudTestStruct {
	return CrudTestStruct{
		MsgID:        createEvent.ID,
		ChannelID:    createEvent.ChannelID,
		MsgInfo:      createEvent.Content,
		AuthorID:     createEvent.Author.ID,
		MsgTimestamp: createEvent.Timestamp,
	}
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
	/* Handle Flags */
	if err := c.ParseFlags(m.Message.Content); err != nil {
		fmt.Println(err)
		return err
	}
	if err := c.ValidateFlagMap(); err != nil {
		clients.DgSession.ChannelMessageSend(m.ChannelID, err.Error())
		return err
	}
	/* Flag parsed without error. Now follows various actions. */
	if _, ok := c.FlagArgstatMaps["free"]; ok {
		clients.DgSession.ChannelMessageSend(m.ChannelID, fmt.Sprintf("Successfully read arguments w/ flag! \r %v", c.FlagArgstatMaps))
	}
	if _, ok := c.FlagArgstatMaps["create"]; ok {
		data.GetCollection("test_crud").InsertOne(context.TODO(), newTestStruct(m))
		//debug
		fmt.Println("Inserted message: " + m.ID)
	} else if _, ok := c.FlagArgstatMaps["read"]; ok {
		//single struct
		/*
			var singleResult *mongo.SingleResult
			if singleResult = data.GetCollection("test_crud").FindOne(context.TODO(), bson.M{"authorid": m.Author.ID}); singleResult.Err() != nil {
				if singleResult.Err() == mongo.ErrNoDocuments {
					clients.DgSession.ChannelMessageSend(m.ChannelID, "Unable to find any testStruct for the given userID!")
				}
				fmt.Println("Unable to find any doc for authorid:" + m.Author.ID)
				return errors.Wrap(singleResult.Err(), "Doc query failed")
			}
			var doc CrudTestStruct
			if err := singleResult.Decode(&doc); err != nil {
				fmt.Println("Unable to Decode doc into TestCrudStruct")
				return errors.Wrap(err, "Decode doc failed")
			}
		*/
		//multiple Struct
		findOpts := options.Find().SetSort(bson.D{{"msgtimestamp", 1}})
		findCursor, err := data.GetCollection("test_crud").Find(context.TODO(), bson.M{"authorid": m.Author.ID}, findOpts)
		var results []*CrudTestStruct
		if err = findCursor.All(context.TODO(), &results); err != nil {
			fmt.Println("ERROR:" + err.Error())
			return err
		}
		for _, result := range results {
			fmt.Printf("Result: %v\r\n", result)
		}
		clients.DgSession.ChannelMessageSend(m.ChannelID, fmt.Sprintf("Found Doc(s):%v", results))
	}
	if err := clients.DgSession.MessageReactionAdd(m.ChannelID, m.ID, "\u2705"); err != nil {
		fmt.Println("Error reacting: " + err.Error())
	}
	return nil
}

func init() {
	var crud CrudCommand
	crud.New()
	RegisterCommand(&crud)
}
