package pkg

import (
	"dalian-bot/internal/pkg/clients"
	"dalian-bot/internal/pkg/commands"
	"fmt"
	"github.com/bwmarrin/discordgo"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"golang.org/x/net/context"
	"gopkg.in/yaml.v2"
	"io/ioutil"
)

func InitDalian() error {

	/* Read Config files */
	var cred = new(Cred)
	if err := GetCred(cred, "config/credentials.yaml"); err != nil {
		fmt.Println("failed opening credentials file!")
		panic(err)
	}

	/* Setup Mongo Clients & DB */
	mongoClient, err := mongo.Connect(context.TODO(), options.Client().ApplyURI(cred.Uri))
	if err != nil {
		fmt.Println("failed opening Mongo connection.")
		panic(err)
	}
	clients.RegisterMongoClient(mongoClient)
	clients.ConnectToDB("dalian")

	/* Setup DiscordGo Session */
	discordSession, err := discordgo.New("Bot " + cred.Token)
	if err != nil {
		fmt.Println("error creating Discord session")
		panic(err)
	}

	clients.RegisterDiscordClient(discordSession)

	err = discordSession.Open()
	if err != nil {
		fmt.Println("error opening Discord connection")
		panic(err)
	}

	/* Setup Cron from database --reserved-- */

	/* Dalian specific setups */
	commands.SetPrefix("$")
	commands.SetSeparator("$")
	commands.SetBotID(discordSession.State.User.ID)

	commands.RegisterDiscordHandlers()

	fmt.Println("Bot is now running. Press Ctrl+C to exit.")
	return nil
}

func GracefulShutDalian() error {
	//cleanly close down the Discord session
	defer func() {
		if err := clients.MongoClient.Disconnect(context.TODO()); err != nil {
			fmt.Println("error closing Mongo connection!")
		}
		fmt.Println("Mongo connection closed.")
	}()
	clients.DgSession.Close()
	fmt.Println("Connection closed!")
	return nil
}

type Cred struct {
	Discord
	Mongo
}

type Discord struct {
	Token string `yaml:"token"`
}

type Mongo struct {
	Uri string `yaml:"uri"`
}

func GetCred(cred *Cred, fileLocation string) error {
	yamlFile, err := ioutil.ReadFile(fileLocation)
	if err != nil {
		fmt.Println(err.Error())
	}
	err = yaml.Unmarshal(yamlFile, cred)
	if err != nil {
		return err
	}
	return nil
}
