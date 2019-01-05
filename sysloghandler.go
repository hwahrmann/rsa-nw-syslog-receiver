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
	"github.com/joncrlsn/dque"
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
	stopSender  = make(chan struct{})

	// fluentd payload
	syslogBuffer = &sync.Pool{
		New: func() interface{} {
			return make([]byte, 15000)
		},
	}

	server       syslog.Server
	messageCount uint64
	pattern3164  *regexp.Regexp
	pattern5424  *regexp.Regexp
	queue        *dque.DQue
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

func (h *SyslogHandler) run() error {

	// Compile the Regex Pattern
	pattern3164, _ = regexp.Compile(opts.RegexRFC3164)
	pattern5424, _ = regexp.Compile(opts.RegexRFC5424)

	// Create the Queue to store the messages
	var err error
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
	log.Infof("Workers received %d messages", messageCount)
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

func (h *SyslogHandler) syslogWorker(wQuit chan struct{}) {
	var (
		msg      string
		ok       bool
		orighost string
		origmsg  string
	)

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
		var m map[string]string
		matches := pattern3164.FindAllStringSubmatch(msg, -1)
		if matches != nil {
			m = findNamedMatches(pattern3164, matches)
			orighost = m["host"]
			origmsg = msg[2:]
		} else {
			matches = pattern5424.FindAllStringSubmatch(msg, -1)
			if matches != nil {
				m = findNamedMatches(pattern5424, matches)
				orighost = m["host"]
				origmsg = m["message"]
			}
		}
		atomic.AddUint64(&messageCount, 1)

		// Add an item to the queue
		if err := queue.Enqueue(&Message{orighost, origmsg}); err != nil {
			log.Fatal("Error enqueueing item ", err)
		}
	}
}

func findNamedMatches(regex *regexp.Regexp, matches [][]string) map[string]string {
	results := map[string]string{}
	for i, name := range matches[0] {
		results[regex.SubexpNames()[i]] = name
	}
	return results
}

func syslogSender(queue *dque.DQue) {
	var (
		conn  net.Conn
		err   error
		iface interface{}
	)

	log.Infof("Starting Syslog Sender qith Queue Size of %d", queue.Size())
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

			// On an empty queue sleep 1 second before rerying
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
				go checkConnection()
				break LOOP
			}
		}
	}
}

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
		go syslogSender(queue)
		log.Info("Log Decoder capture interface up")
		break
	}
}
