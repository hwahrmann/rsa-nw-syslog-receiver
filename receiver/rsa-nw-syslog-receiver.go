//: ----------------------------------------------------------------------------
//: Copyright (C) 2019 Helmut Wahrmann.
//:
//: file:    rsa-nw-syslog-receiver.go
//: details: Main Program
//: author:  Helmut Wahrmann
//: date:    08/01/2019
//:
//: Licensed under the Apache License, Version 2.0 (the "License");
//: you may not use this file except in compliance with the License.
//: You may obtain a copy of the License at
//:
//:     http://www.apache.org/licenses/LICENSE-2.0
//:
//: Unless required by applicable law or agreed to in writing, software
//: distributed under the License is distributed on an "AS IS" BASIS,
//: WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//: See the License for the specific language governing permissions and
//: limitations under the License.
//: ----------------------------------------------------------------------------

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
