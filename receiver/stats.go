//: ----------------------------------------------------------------------------
//: Copyright (C) 2019 Helmut Wahrmann.
//:
//: file:    stats.go
//: details: Status Hanlder. Provides a basic REST interface
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
	"encoding/json"
	"net"
	"net/http"
	"strconv"
	"time"
)

var startTime = time.Now().Format(time.RFC3339)

// Start the REST HTTP Server
func statsHTTPServer(sysloghandler *SyslogHandler) {
	if !opts.StatsEnabled {
		return
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/stats", StatsHandler(sysloghandler))
	mux.HandleFunc("/stats/events", StatsHandlerEvents(sysloghandler))
	mux.HandleFunc("/stats/queue", StatsHandlerQueue(sysloghandler))

	addr := net.JoinHostPort("0.0.0.0", strconv.Itoa(opts.StatsHTTPPort))

	opts.Logger.Info("starting stats web server ...")
	err := http.ListenAndServe(addr, mux)
	if err != nil {
		opts.Logger.Fatal(err)
	}
}

// StatsHandler returns the stats as part of the REST call
func StatsHandler(h *SyslogHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var data = &struct {
			StartTime   string
			SyslogStats *SyslogStats
		}{
			startTime,
			h.status(),
		}

		j, err := json.Marshal(data)
		if err != nil {
			opts.Logger.Info(err)
		}

		if _, err = w.Write(j); err != nil {
			opts.Logger.Info(err)
		}
	}
}

// StatsHandlerEvents returns the number of Events as part of the REST call
func StatsHandlerEvents(h *SyslogHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var stats = h.status()
		var data = strconv.FormatUint(stats.Events, 10)

		if _, err := w.Write([]byte(data)); err != nil {
			opts.Logger.Info(err)
		}
	}
}

// StatsHandlerQueue returns the number of events in the queue as part of the REST call
func StatsHandlerQueue(h *SyslogHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var stats = h.status()
		var data = strconv.Itoa(stats.QueueCount)

		if _, err := w.Write([]byte(data)); err != nil {
			opts.Logger.Info(err)
		}
	}
}
