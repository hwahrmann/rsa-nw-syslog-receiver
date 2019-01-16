//: ----------------------------------------------------------------------------
//: Copyright (C) 2019 Helmut Wahrmann.
//:
//: file:    sysloghandler.go
//: details: Syslog Receiver and Syslog Sender Loop
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
	"errors"
	"net"
	"regexp"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/google/logger"
	"github.com/joncrlsn/dque"
	"gopkg.in/mcuadros/go-syslog.v2"
	"gopkg.in/mcuadros/go-syslog.v2/format"
)

// SyslogHandler Represents a SyslogHandler
type SyslogHandler struct {
	listenPort     int
	listenProtocol string
	logdecoder     string
	workers        int
	stats          SyslogStats
	pool           chan chan struct{}
}

// SyslogStats represents syslogreceiver stats
type SyslogStats struct {
	QueueCount int
	Events     uint64
	Workers    int
}

var (
	log *logger.Logger

	syslogMsgCH = make(chan format.LogParts)
	stopSender  = make(chan struct{})

	server   syslog.Server
	patterns []*regexp.Regexp

	queue *dque.DQue
)

const (
	queueName = "syslogreceiver"

	queueDir  = "/tmp"
	queueSize = 100
)

// Message is what we'll be storing in the queue.
type Message struct {
	Host string
	Msg  string
}

// MessageBuilder creates a new Message and returns a pointer to it.
// This is used when we load a segment of the queue from disk.
func MessageBuilder() interface{} {
	return &Message{}
}

// NewSyslogHandler constructs a SyslogHandler
func NewSyslogHandler() *SyslogHandler {
	log = opts.Logger

	return &SyslogHandler{
		listenPort:     opts.ListenPort,
		listenProtocol: opts.Protocol,
		logdecoder:     opts.LogDecoder,
		workers:        opts.Workers,
		pool:           make(chan chan struct{}, maxWorkers),
	}
}

func (h *SyslogHandler) status() *SyslogStats {
	return &SyslogStats{
		QueueCount: queue.Size(),
		Events:     atomic.LoadUint64(&h.stats.Events),
		Workers:    h.workers,
	}
}

func (h *SyslogHandler) run() error {

	var (
		err error
		p   *regexp.Regexp
	)

	// Compile the Regex Pattern
	for _, search := range opts.Search {
		p, err = regexp.Compile(search.Regex)
		if err == nil {
			patterns = append(patterns, p)
		}
	}

	// Create the Queue to store the messages
	queue, err = dque.NewOrOpen(queueName, queueDir, queueSize, MessageBuilder)
	if err != nil {
		log.Fatal("Error creating new dque ", err)
	}
	log.Infof("Queue Size: %d", queue.Size())

	// Start the Receiver Workers
	for i := 0; i < h.workers; i++ {
		go func() {
			wQuit := make(chan struct{})
			h.pool <- wQuit
			h.syslogWorker(wQuit)
		}()
	}

	// Start the Sender
	go syslogSender(queue)

	// Setup the Syslog Server
	channel := make(syslog.LogPartsChannel)
	handler := syslog.NewChannelHandler(channel)

	addr := "0.0.0.0:" + strconv.Itoa(h.listenPort)
	server := syslog.NewServer()
	server.SetFormat(syslog.RFC3164)
	server.SetHandler(handler)
	if h.listenProtocol == "udp" {
		server.ListenUDP(addr)
	} else {
		server.ListenTCP(addr)
	}

	err = server.Boot()
	if err != nil {
		log.Errorf("Error starting Syslog Server: %s", err)
		return errors.New("Error starting Syslog Server")
	}

	// Start receiver thread
	go func(channel syslog.LogPartsChannel) {
		for logParts := range channel {
			syslogMsgCH <- logParts
		}
	}(channel)

	log.Infof("Syslog Receiver is running (listening on [::]:%d/%s workers#: %d)", h.listenPort, h.listenProtocol, h.workers)

	server.Wait()

	return nil
}

// Shutdown the Syslog Receiver
func (h *SyslogHandler) shutdown() {
	log.Infof("Workers received %d messages", &h.stats.Events)
	log.Info("Stopping syslog server service gracefully ...")
	for i := 0; i < h.workers; i++ {
		wQuit := h.pool
		close(wQuit)
	}
	server.Kill()
	log.Info("Syslogreceiver has been shutdown")
	close(syslogMsgCH)
	close(stopSender)
}

// Worker, which receives Syslog Events and Queues the message
func (h *SyslogHandler) syslogWorker(wQuit chan struct{}) {
	var (
		syslogmsg format.LogParts
		msg       string
		ok        bool
		orighost  string
		origmsg   string
	)

LOOP:
	for {

		select {
		case <-wQuit:
			break LOOP
		case syslogmsg, ok = <-syslogMsgCH:
			if !ok {
				break LOOP
			}
		}

		// As a fallback the message and host as received by the relay is stored
		msg = syslogmsg["content"].(string)
		orighost = syslogmsg["hostname"].(string)
		origmsg = msg
		// extract sender and original message
		var (
			m       map[string]string
			matches [][]string
		)

		for _, pattern := range patterns {
			matches = pattern.FindAllStringSubmatch(msg, -1)
			if matches != nil {
				m = findNamedMatches(pattern, matches)
				orighost = m["host"]
				origmsg = m["message"]
				break
			}
		}
		atomic.AddUint64(&h.stats.Events, 1)

		// Add an item to the queue
		if err := queue.Enqueue(&Message{orighost, origmsg}); err != nil {
			log.Fatal("Error enqueueing item ", err)
		}
	}
}

// Map all the Submatches
func findNamedMatches(regex *regexp.Regexp, matches [][]string) map[string]string {
	results := map[string]string{}
	for i, name := range matches[0] {
		results[regex.SubexpNames()[i]] = name
	}
	return results
}

// This Worker extracts messages from the queue and sends them to RSA Netwitness
func syslogSender(queue *dque.DQue) {
	var (
		conn  net.Conn
		err   error
		iface interface{}
	)

	log.Infof("Starting Syslog Sender with a Queue Size of %d", queue.Size())
	//Setup network connection
	host := opts.LogDecoder + ":514"
	if opts.LogDecoderProtocol == "udp" {
		conn, err = net.Dial("udp", host)
		if err != nil {
			log.Errorf("Worker could not connect to log decoder: %s\n", err)
			return
		}
	} else {
		conn, err = net.Dial("tcp", host)
		if err != nil {
			log.Errorf("Worker could not connect to log decoder: %s\n", err)
			log.Info("Leaving Sylog Sender")
			go checkConnection()
			return
		}
	}
	defer conn.Close()
	log.Infof("Worker opened connection to %s/%s\n", opts.LogDecoderProtocol, host)

LOOP:
	for {
		select {
		case <-stopSender:
			log.Info("Stopping Syslog Sender")
			break LOOP
		default:
			// Dequeue the next message in the queue
			if iface, err = queue.Dequeue(); err != nil && err != dque.ErrEmpty {
				log.Fatal("Error dequeuing item:", err)
			}

			// On an empty queue sleeLogpartsp 1 second before rerying
			if err == dque.ErrEmpty {
				time.Sleep(1000 * time.Millisecond)
				continue
			}

			message := iface.(*Message)
			msg := "[][][" + message.Host + "][" + strconv.FormatInt(time.Now().Unix(), 10) + "][]" + message.Msg
			if opts.LogDecoderProtocol == "tcp" {
				msg = msg + "\n"
			}
			_, err = conn.Write([]byte(msg))
			if err != nil {
				log.Errorf("worker could not write to log decoder: %s\n", err)
				// Check for decoder coming up again and leave worker
				go checkConnection()
				break LOOP
			}
		}
	}
}

// Check for Log Decoder capturing again
func checkConnection() {
	log.Info("Starting connection check for Log Decoder")
	host := opts.LogDecoder + ":514"
	for {
		tcpAddr, _ := net.ResolveTCPAddr("tcp", host)
		conn, err := net.DialTCP("tcp", nil, tcpAddr)
		if err != nil {
			time.Sleep(5000 * time.Millisecond)
			continue
		}
		conn.Close()
		// Start Sender Worker
		go syslogSender(queue)
		log.Info("Log Decoder capture interface up")
		break
	}
}
