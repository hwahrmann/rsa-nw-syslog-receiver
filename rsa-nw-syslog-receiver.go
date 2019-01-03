package main

import (
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

var (
	opts   *Options
	logger *log.Logger
)

type handler interface {
	run() error
	shutdown()
}

func main() {
	var (
		wg       sync.WaitGroup
		signalCh = make(chan os.Signal, 1)
	)

	opts = GetOptions()

	signal.Notify(signalCh, syscall.SIGINT, syscall.SIGTERM)

	syslogHandler := NewSyslogHandler()
	wg.Add(1)

	go func(syslogHandler handler) {
		defer wg.Done()
		err := syslogHandler.run()
		if err != nil {
			close(signalCh)
		}
	}(syslogHandler)

	<-signalCh

	logger.Printf("Stopping Syslog Receiver")

	syslogHandler.shutdown()
}
