package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"runtime"

	"gopkg.in/yaml.v2"
)

var (
	version    string
	maxWorkers = runtime.NumCPU() * 1e4
)

// Options represents options
type Options struct {
	// global options
	Verbose            bool   `yaml:"verbose"`
	LogFile            string `yaml:"log-file"`
	PIDFile            string `yaml:"pid-file"`
	Logger             *log.Logger
	version            bool
	LogDecoder         string `yaml:"logdecoder"`
	LogDecoderProtocol string `yaml:"logdecoderprotocol"`
	ListenPort         int    `yaml:"listenport"`
	Protocol           string `yaml:"listenprotocol"`
	Workers            int    `yaml:"workers"`
}

func init() {
	if version == "" {
		version = "1.0"
	}
}

// NewOptions constructs new options
func NewOptions() *Options {
	return &Options{
		Verbose: false,
		version: false,
		PIDFile: "/var/run/rsa-nw-syslog-receiver.pid",
		Logger:  log.New(os.Stderr, "[syslogreceiver] ", log.Ldate|log.Ltime),

		ListenPort:         5514,
		LogDecoder:         "",
		LogDecoderProtocol: "tcp",
		Protocol:           "tcp",
		Workers:            5,
	}
}

// GetOptions gets options through cmd and file
func GetOptions() *Options {
	opts := NewOptions()

	opts.syslogreceiverFlagSet()
	opts.syslogreceiverVersion()

	if opts.LogFile != "" {
		f, err := os.OpenFile(opts.LogFile, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
		if err != nil {
			opts.Logger.Println(err)
		} else {
			opts.Logger.SetOutput(f)
		}
	}

	if ok := opts.receiverIsRunning(); ok {
		opts.Logger.Fatal("The Syslog Receiver is already running!")
	}

	opts.pidWrite()

	opts.Logger.Printf("Welcome to Syslog Receiver v.%s GPL v3", version)
	opts.Logger.Printf("Copyright (C) 2019 Helmut Wahrmann.")

	return opts
}

func (opts Options) pidWrite() {
	f, err := os.OpenFile(opts.PIDFile, os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		opts.Logger.Println(err)
		return
	}

	_, err = fmt.Fprintf(f, "%d", os.Getpid())
	if err != nil {
		opts.Logger.Println(err)
	}
}

func (opts Options) receiverIsRunning() bool {
	b, err := ioutil.ReadFile(opts.PIDFile)
	if err != nil {
		return false
	}

	cmd := exec.Command("kill", "-0", string(b))
	_, err = cmd.Output()
	if err != nil {
		return false
	}

	return true
}

func (opts Options) syslogreceiverVersion() {
	if opts.version {
		fmt.Printf("Syslog Receiver version: %s\n", version)
		os.Exit(0)
	}
}

func (opts *Options) syslogreceiverFlagSet() {

	var config string
	flag.StringVar(&config, "config", "/etc/syslogreceiver/syslogreceiver.conf", "path to config file")

	syslogreceiverLoadCfg(opts)

	// global options
	flag.BoolVar(&opts.Verbose, "verbose", opts.Verbose, "enable/disable verbose logging")
	flag.BoolVar(&opts.version, "version", opts.version, "show version")
	flag.StringVar(&opts.LogFile, "log-file", opts.LogFile, "log file name")
	flag.StringVar(&opts.PIDFile, "pid-file", opts.PIDFile, "pid file name")
	flag.StringVar(&opts.LogDecoder, "logdecoder", opts.LogDecoder, "the address of the log decoder")
	flag.StringVar(&opts.LogDecoderProtocol, "logdecoderprotocol", opts.LogDecoderProtocol, "the protcol to connect to the log decoder")
	flag.IntVar(&opts.ListenPort, "listenport", opts.ListenPort, "syslogreceiver listening port number")
	flag.StringVar(&opts.Protocol, "listenprotocol", opts.Protocol, "the protocol to listen for incoming traffic")
	flag.IntVar(&opts.Workers, "workers", opts.Workers, "the number of workers forwarding messages to the log decoder")

	flag.Usage = func() {
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, `
    Example:
	# listen on default port tcp/5514
	rsa-nw-syslog-receiver -logdecoder 192.168.1.53

	# specify port and protocol
	rsa-nw-syslog-receiver -logdecoder 192.168.1.53 -listenport 6514 -listenprotocol udp
	`)
	}

	flag.Parse()
}

func syslogreceiverLoadCfg(opts *Options) {
	var file = "/etc/syslogreceiver/syslogreceiver.conf"

	for i, flag := range os.Args {
		if flag == "-config" {
			file = os.Args[i+1]
			break
		}
	}

	b, err := ioutil.ReadFile(file)
	if err != nil {
		opts.Logger.Println(err)
		return
	}
	err = yaml.Unmarshal(b, opts)
	if err != nil {
		opts.Logger.Println(err)
	}
}
