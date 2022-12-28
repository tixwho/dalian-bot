package core

import (
	"fmt"
	"gopkg.in/yaml.v3"
	"os"
)

type Cred struct {
	Version     string `yaml:"version"`
	DiscordCred `yaml:"discord-cred"`
	MongoCred   `yaml:"mongo-cred"`
}

type DiscordCred struct {
	DiscordToken yaml.Node `yaml:"token"`
}

type MongoCred struct {
	MongoURI yaml.Node `yaml:"uri"`
}

var credInternal Cred

func GetCred(cred *Cred, fileLocation string) error {
	yamlFile, err := os.ReadFile(fileLocation)
	if err != nil {
		fmt.Println(err.Error())
	}
	err = yaml.Unmarshal(yamlFile, cred)
	if err != nil {
		return err
	}
	return nil
}

func GetCredNew(fileLocation string) (*Cred, error) {
	if credInternal.Version == "" {
		yamlFile, err := os.ReadFile(fileLocation)
		if err != nil {
			Logger.Panicf("Error reading cred file from [%s]: %v", fileLocation, err)
			return nil, err
		}
		err = yaml.Unmarshal(yamlFile, &credInternal)
		if err != nil {
			Logger.Panicf("Error unmarshalling cred file: %v", err)
			return nil, err
		}
	}

	return &credInternal, nil
}
