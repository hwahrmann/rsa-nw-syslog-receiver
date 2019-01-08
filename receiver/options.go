//: ----------------------------------------------------------------------------
//: Copyright (C) 2019 Helmut Wahrmann.
//:
//: file:    options.go
//: details: Handle the Configuration
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
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"regexp"
	"runtime"

	"github.com/google/logger"
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
	PIDFile            string `yaml:"pid-file"`
	Logger             *logger.Logger
	version            bool
	StatsEnabled       bool
	StatsHTTPPort      int
	LogDecoder         string `yaml:"logdecoder"`
	LogDecoderProtocol string `yaml:"logdecoderprotocol"`
	ListenPort         int    `yaml:"listenport"`
	Protocol           string `yaml:"listenprotocol"`
	Workers            int    `yaml:"workers"`
	RegexRFC3164       string `yaml:"rfc3164"`
	RegexRFC5424       string `yaml:"rfc5424"`
}

func init() {
	if version == "" {
		version = "1.0"
	}
}

// NewOptions constructs new options
func NewOptions() *Options {
	options := Options{}
	options.Verbose = false
	options.version = false
	options.PIDFile = "/var/run/rsa-nw-syslog-receiver.pid"
	options.ListenPort = 5514
	options.LogDecoder = "127.0.0.1"
	options.LogDecoderProtocol = "tcp"
	options.Protocol = "tcp"
	options.Workers = 5
	options.Logger = logger.Init("", options.Verbose, true, ioutil.Discard)
	options.RegexRFC3164 = "(?P<time>[A-Z][a-z][a-z]\\s{1,2}\\d{1,2}\\s\\d{2}[:]\\d{2}[:]\\d{2})\\s(?P<host>[\\w][\\w\\d\\.@-]*)\\s(?P<message>.*)$"
	options.RegexRFC5424 = "^[1-9]\\d{0,2} (?P<time>(\\d{4}[-]\\d{2}[-]\\d{2}[T]\\d{2}[:]\\d{2}[:]\\d{2}(?:\\.\\d{1,6})?(?:[+-]\\d{2}[:]\\d{2}|Z)?)|-)\\s(?P<host>([\\w][\\w\\d\\.@-]*)|-)\\s(?P<ident>[^ ]+)\\s(?P<pid>[-0-9]+)\\s(?P<msgid>[^ ]+)\\s?(?P<extradata>(\\[(.*)\\]|[^ ]))?\\s(?P<message>.*)$"
	options.StatsEnabled = true
	options.StatsHTTPPort = 8081
	logger.SetFlags(0)
	return &options
}

// GetOptions gets options through cmd and file
func GetOptions() *Options {
	opts := NewOptions()

	opts.syslogreceiverFlagSet()
	opts.syslogreceiverVersion()

	if opts.RegexRFC3164 == "" {
		opts.Logger.Fatal("Missing Regex Pattern")
	}

	_, err := regexp.Compile(opts.RegexRFC3164)
	if err != nil {
		opts.Logger.Fatalf("Error in Regex Pattern: %s", err)
	}
	_, err = regexp.Compile(opts.RegexRFC5424)
	if err != nil {
		opts.Logger.Fatalf("Error in Regex Pattern: %s", err)
	}

	if ok := opts.receiverIsRunning(); ok {
		opts.Logger.Fatal("The Syslog Receiver is already running!")
	}

	opts.pidWrite()

	opts.Logger.Infof("Welcome to Syslog Receiver v.%s GPL v3", version)
	opts.Logger.Info("Copyright (C) 2019 Helmut Wahrmann.")

	return opts
}

func (opts Options) pidWrite() {
	f, err := os.OpenFile(opts.PIDFile, os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		opts.Logger.Info(err)
		return
	}

	_, err = fmt.Fprintf(f, "%d", os.Getpid())
	if err != nil {
		opts.Logger.Info(err)
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
	flag.StringVar(&opts.PIDFile, "pid-file", opts.PIDFile, "pid file name")
	flag.StringVar(&opts.LogDecoder, "logdecoder", opts.LogDecoder, "the address of the log decoder")
	flag.StringVar(&opts.LogDecoderProtocol, "logdecoderprotocol", opts.LogDecoderProtocol, "the protcol to connect to the log decoder")
	flag.IntVar(&opts.ListenPort, "listenport", opts.ListenPort, "syslogreceiver listening port number")
	flag.StringVar(&opts.Protocol, "listenprotocol", opts.Protocol, "the protocol to listen for incoming traffic")
	flag.IntVar(&opts.Workers, "workers", opts.Workers, "the number of workers forwarding messages to the log decoder")
	flag.StringVar(&opts.RegexRFC3164, "rfc3164", opts.RegexRFC3164, "The Regex Pattern to parse RFC3164 for sending host and message")
	flag.StringVar(&opts.RegexRFC5424, "rfc5424", opts.RegexRFC5424, "The Regex Pattern to parse RFC5424 for sending host and message")
	flag.BoolVar(&opts.StatsEnabled, "stats-enabled", opts.StatsEnabled, "enable REST interface for status query")
	flag.IntVar(&opts.StatsHTTPPort, "stats-http-port", opts.StatsHTTPPort, "The Listen Port for the REST Interface")

	flag.Usage = func() {
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, `
    Example:
	# listen on default port tcp/5514
	rsa-nw-syslog-receiver -logdecoder 192.168.1.53 -regexpattern "^\\d\\s.*?\\s(.*?)\\s.*?\\[.*\\]\\s(.*)"

	# specify port and protocol
	rsa-nw-syslog-receiver -logdecoder 192.168.1.53 -listenport 6514 -listenprotocol udp -regexpattern "^\\d\\s.*?\\s(.*?)\\s.*?\\[.*\\]\\s(.*)"
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
		opts.Logger.Info(err)
		return
	}
	err = yaml.Unmarshal(b, opts)
	if err != nil {
		opts.Logger.Info(err)
	}
}
