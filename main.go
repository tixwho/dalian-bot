package main

import (
	"os"
	"os/signal"
	"syscall"
)

func main() {

	InitDalian()

	//graceful shutdown
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc

	GracefulShutDalian()
}
