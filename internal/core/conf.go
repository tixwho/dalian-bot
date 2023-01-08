package core

import (
	"gopkg.in/yaml.v3"
	"os"
)

type Cred struct {
	Version      string `yaml:"version"`
	DiscordCred  `yaml:"discord-cred"`
	MongoCred    `yaml:"mongo-cred"`
	OnedriveCred `yaml:"onedrive-cred"`
}

type DiscordCred struct {
	DiscordToken yaml.Node `yaml:"token"`
}

type MongoCred struct {
	MongoURI yaml.Node `yaml:"uri"`
}

type OnedriveCred struct {
	OnedriveClientID yaml.Node `yaml:"client-id"`
	OnedriveSecret   yaml.Node `yaml:"secret"`
}

var credInternal Cred

func GetCred(fileLocation string) (*Cred, error) {
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
