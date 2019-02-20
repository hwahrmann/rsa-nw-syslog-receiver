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
	version      = "1.0.1"
	maxWorkers   = runtime.NumCPU() * 1e4
	regexRFC3164 = "^(?P<message>(?P<time>[A-Z][a-z][a-z]\\s{1,2}\\d{1,2}\\s\\d{2}[:]\\d{2}[:]\\d{2})\\s(?P<host>[\\w][\\w\\d\\.@-]*)\\s.*)$"
	regexRFC5424 = "^[1-9]\\d{0,2} (?P<time>(\\d{4}[-]\\d{2}[-]\\d{2}[T]\\d{2}[:]\\d{2}[:]\\d{2}(?:\\.\\d{1,6})?(?:[+-]\\d{2}[:]\\d{2}|Z)?)|-)\\s(?P<host>([\\w][\\w\\d\\.@-]*)|-)\\s(?P<ident>[^ ]+)\\s(?P<pid>[-0-9]+)\\s(?P<msgid>[^ ]+)\\s?(?P<extradata>(\\[(.*)\\]|[^ ]))?\\s(?P<message>.*)$"
)

// Options represents options
type Options struct {
	// global options
	Verbose            bool   `yaml:"verbose"`
	PIDFile            string `yaml:"pid-file"`
	Logger             *logger.Logger
	version            bool
	StatsEnabled       bool     `yaml:"statsenabled"`
	StatsHTTPPort      int      `yaml:"statsport"`
	LogDecoder         string   `yaml:"logdecoder"`
	LogDecoderProtocol string   `yaml:"logdecoderprotocol"`
	ListenPort         int      `yaml:"listenport"`
	Protocol           string   `yaml:"listenprotocol"`
	Workers            int      `yaml:"workers"`
	Search             []Search `yaml:"search"`
}

// Search represents a Search structure
type Search struct {
	Regex   string
	Type    string
	Mapping []string
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
	options.PIDFile = "/var/run/rsa-nw-syslog-receiver.pid"
	options.ListenPort = 5514
	options.LogDecoder = "127.0.0.1"
	options.LogDecoderProtocol = "tcp"
	options.Protocol = "tcp"
	options.Workers = 5
	options.Logger = logger.Init("", options.Verbose, true, ioutil.Discard)
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

	var err error
	for _, search := range opts.Search {
		_, err = regexp.Compile(search.Regex)
		if err != nil {
			opts.Logger.Fatalf("Error in Regex Pattern: %s", err)
		}
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

	flag.Usage = func() {
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, `
    Example:
	rsa-nw-syslog-receiver -config /etc/syslogreceiver/syslogreceiver.conf"
	`)
	}

	flag.Parse()
}

// Load the configuration from the config file
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

	// Check, if we have only the default searches
	if len(opts.Search) == 0 {
		opts.Logger.Info("No Search strings found. Using default Syslog Regex")
	}

	// Adding the default regexes to the end
	s := Search{Regex: regexRFC3164, Type: "syslog"}
	opts.Search = append(opts.Search, s)
	s = Search{Regex: regexRFC5424, Type: "syslog"}
	opts.Search = append(opts.Search, s)
}
