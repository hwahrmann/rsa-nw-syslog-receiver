package main

import (
	"errors"
	"net"
	"regexp"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/logger"
	"gopkg.in/mcuadros/go-syslog.v2"
)

// SyslogHandler Represents a SyslogHandler
type SyslogHandler struct {
	listenPort     int
	listenProtocol string
	logdecoder     string
	workers        int
	pool           chan chan struct{}
}

var (
	log *logger.Logger

	syslogMsgCH = make(chan string, 2000)

	// fluentd payload
	syslogBuffer = &sync.Pool{
		New: func() interface{} {
			return make([]byte, 15000)
		},
	}

	server       syslog.Server
	messageCount uint64
	pattern      *regexp.Regexp
)

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

func (h *SyslogHandler) run() error {
	pattern, _ = regexp.Compile(opts.RegexPattern)

	// Start the Workers
	for i := 0; i < h.workers; i++ {
		go func() {
			wQuit := make(chan struct{})
			h.pool <- wQuit
			h.syslogWorker(wQuit)
		}()
	}

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

	err := server.Boot()
	if err != nil {
		log.Errorf("Error starting Syslog Server: %s", err)
		return errors.New("Error starting Syslog Server")
	}

	go func(channel syslog.LogPartsChannel) {
		for logParts := range channel {
			syslogMsgCH <- logParts["content"].(string)
		}
	}(channel)

	log.Infof("Syslog Receiver is running (listening on [::]:%s/%d workers#: %d)", h.listenProtocol, h.listenPort, h.workers)

	server.Wait()

	return nil
}

func (h *SyslogHandler) shutdown() {
	log.Infof("workers served %d", messageCount)
	log.Info("stopping syslog server service gracefully ...")
	for i := 0; i < h.workers; i++ {
		wQuit := h.pool
		close(wQuit)
	}
	server.Kill()
	log.Info("syslogreceiver has been shutdown")
	close(syslogMsgCH)
}

func (h *SyslogHandler) syslogWorker(wQuit chan struct{}) {
	var (
		msg string
		ok  bool
	)

	//Setup network connection
	host := h.logdecoder + ":514"
	var conn net.Conn
	var err error
	if opts.LogDecoderProtocol == "udp" {
		conn, err = net.Dial("udp", host)
		if err != nil {
			log.Errorf("worker could not connect to log decoder: %s\n", err)
			return
		}
	} else {
		conn, err = net.Dial("tcp", host)
		if err != nil {
			log.Errorf("worker could not connect to log decoder: %s\n", err)
			return
		}
	}
	defer conn.Close()
	log.Infof("worker opened connection to %s/%s\n", opts.LogDecoderProtocol, host)

LOOP:
	for {

		select {
		case <-wQuit:
			break LOOP
		case msg, ok = <-syslogMsgCH:
			if !ok {
				break LOOP
			}
		}

		// extract sender and original message
		matches := pattern.FindAllSubmatch([]byte(msg), -1)
		orighost := string(matches[0][1])
		origmsg := string(matches[0][2])

		msg = "[][][" + orighost + "][" + strconv.FormatInt(time.Now().Unix(), 10) + "][]" + origmsg
		atomic.AddUint64(&messageCount, 1)
		if opts.LogDecoderProtocol == "tcp" {
			msg = msg + "\n"
		}
		_, err := conn.Write([]byte(msg))
		if err != nil {
			log.Errorf("worker could not write to log decoder: %s\n", err)
		}
	}
}
