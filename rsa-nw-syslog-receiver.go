package main

import (
	"os"
	"os/signal"
	"sync"
	"syscall"
)

var (
	opts *Options
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

	opts.Logger.Info("Stopping Syslog Receiver")

	syslogHandler.shutdown()
}
