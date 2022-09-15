package discord

import (
	"dalian-bot/internal/pkg/clients"
	"io"
	"log"
)

func SendFile(channel, name string, r io.Reader) error {
	if _, err := clients.DgSession.ChannelFileSend(channel, name, r); err != nil {
		log.Println("Error sending discord message: ", err)
		return err
	}
	return nil
}
