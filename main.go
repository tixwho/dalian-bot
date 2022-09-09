package main

import (
	"dalian-bot/internal/pkg"
	"os"
	"os/signal"
	"syscall"
)

func main() {

	pkg.InitDalian()

	//graceful shutdown
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc

	pkg.GracefulShutDalian()
}
