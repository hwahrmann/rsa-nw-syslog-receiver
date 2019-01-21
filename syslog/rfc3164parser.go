//: ----------------------------------------------------------------------------
//: Copyright (C) 2019 Helmut Wahrmann.
//:
//: file:    rfc3164parser.go
//: details: Syslog Parser for RFC 3164 compatible messages
//: author:  Helmut Wahrmann
//: date:    21/01/2019
//:
//: Base on the work of Máximo Cuadros  as documented in
//: gopkg.in/mcuadros/go-syslog.v2
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

package syslog

import (
	"bytes"
	"strconv"
	"time"
)

// Parser returns information about a Parser
type Parser struct {
	buff           []byte
	cursor         int
	l              int
	priority       Priority
	version        int
	header         header
	message        string
	location       *time.Location
	envisionFormat bool
}

type header struct {
	timestamp time.Time
	hostname  string
}

// NewParser returns a new Parser instance
func NewParser(buff []byte) *Parser {
	return &Parser{
		buff:     buff,
		cursor:   0,
		l:        len(buff),
		location: time.UTC,
	}
}

// Location returns a new Tiome location
func (p *Parser) Location(location *time.Location) {
	p.location = location
}

// Parse invokes parsing of the received syslog message
func (p *Parser) Parse() error {
	pri, err := p.parsePriority()
	if err != nil {
		if err == ErrEnvisionFormat {
			p.cursor = 0
			p.envisionFormat = true
		} else {
			return err
		}
	}

	tcursor := p.cursor
	hdr, err := p.parseHeader()
	if err == ErrTimestampUnknownFormat {
		// RFC3164 sec 4.3.2.
		hdr.timestamp = time.Now().Round(time.Second)
		// Reset cursor for content read
		p.cursor = tcursor
	} else if err != nil {
		return err
	} else {
		if !p.envisionFormat {
			p.cursor = tcursor
		}
	}

	msg, err := p.parsemessage()
	if err != ErrEOL {
		return err
	}

	p.priority = pri
	p.version = NO_VERSION
	p.header = hdr
	p.message = msg

	return nil
}

// Dump dumps the parsed message into LogParts struct
func (p *Parser) Dump() LogParts {
	return LogParts{
		"timestamp": p.header.timestamp,
		"hostname":  p.header.hostname,
		"content":   p.message,
		"priority":  p.priority.P,
		"facility":  p.priority.F.Value,
		"severity":  p.priority.S.Value,
	}
}

func (p *Parser) parsePriority() (Priority, error) {
	return ParsePriority(p.buff, &p.cursor, p.l)
}

func (p *Parser) parseHeader() (header, error) {
	hdr := header{}
	var err error

	tcursor := p.cursor
	ts, err := p.parseTimestamp()
	if err != nil {
		// Set current Time
		ts = time.Now()
	}

	if string(p.buff[p.cursor:p.cursor+4]) == "[][]" {
		p.envisionFormat = true
		p.cursor = tcursor
	}
	hostname, err := p.parseHostname()
	if err != nil {
		return hdr, err
	}

	hdr.timestamp = ts
	hdr.hostname = hostname

	return hdr, nil
}

func (p *Parser) parsemessage() (string, error) {
	var err error

	content, err := p.parseContent()
	if err != ErrEOL {
		return "", err
	}
	return content, err
}

// https://tools.ietf.org/html/rfc3164#section-4.1.2
func (p *Parser) parseTimestamp() (time.Time, error) {
	var ts time.Time
	var err error
	var tsFmtLen int
	var sub []byte

	tsFmts := []string{
		"Jan 02 15:04:05",
		"Jan  2 15:04:05",
	}

	tcursor := p.cursor
	found := false
	for _, tsFmt := range tsFmts {
		tsFmtLen = len(tsFmt)

		if p.cursor+tsFmtLen > p.l {
			continue
		}

		sub = p.buff[p.cursor : tsFmtLen+p.cursor]
		ts, err = time.ParseInLocation(tsFmt, string(sub), p.location)
		if err == nil {
			found = true
			break
		}
	}

	if !found {
		// Handle the envision Header time
		p.cursor = tcursor
		if p.envisionFormat || string(p.buff[p.cursor:p.cursor+1]) == "[" {
			tmpBuf := p.buff[bytes.Index(p.buff, []byte("[][]["))+5:]
			openBracket := bytes.Index(tmpBuf, []byte("["))
			tmpBuf = tmpBuf[openBracket:]
			closingBracket := bytes.Index(tmpBuf, []byte("]"))
			unxTime, _ := strconv.ParseInt(string(tmpBuf[1:closingBracket]), 10, 64)
			ts = time.Unix(unxTime/1000, 0)
			return ts, nil
		}

		p.cursor = tsFmtLen
		// XXX : If the timestamp is invalid we try to push the cursor one byte
		// XXX : further, in case it is a space
		if (p.cursor < p.l) && (p.buff[p.cursor] == ' ') {
			p.cursor++
		}

		return ts, ErrTimestampUnknownFormat
	}

	fixTimestampIfNeeded(&ts)

	p.cursor += tsFmtLen

	if (p.cursor < p.l) && (p.buff[p.cursor] == ' ') {
		p.cursor++
	}

	return ts, nil
}

func (p *Parser) parseHostname() (string, error) {
	return ParseHostname(p.buff, &p.cursor, p.l)
}

// http://tools.ietf.org/html/rfc3164#section-4.1.3
func (p *Parser) parseTag() (string, error) {
	var b byte
	var endOfTag bool
	var bracketOpen bool
	var tag []byte
	var err error
	var found bool

	from := p.cursor

	for {
		if p.cursor == p.l {
			// no tag found, reset cursor for content
			p.cursor = from
			return "", nil
		}

		b = p.buff[p.cursor]
		bracketOpen = (b == '[')
		endOfTag = (b == ':' || b == ' ')

		// XXX : parse PID ?
		if bracketOpen {
			tag = p.buff[from:p.cursor]
			found = true
		}

		if endOfTag {
			if !found {
				tag = p.buff[from:p.cursor]
				found = true
			}

			p.cursor++
			break
		}

		p.cursor++
	}

	if (p.cursor < p.l) && (p.buff[p.cursor] == ' ') {
		p.cursor++
	}

	return string(tag), err
}

func (p *Parser) parseContent() (string, error) {
	if p.cursor > p.l {
		return "", ErrEOL
	}

	content := bytes.Trim(p.buff[p.cursor:p.l], " ")
	p.cursor += len(content)

	return string(content), ErrEOL
}

func fixTimestampIfNeeded(ts *time.Time) {
	now := time.Now()
	y := ts.Year()

	if ts.Year() == 0 {
		y = now.Year()
	}

	newTs := time.Date(y, ts.Month(), ts.Day(), ts.Hour(), ts.Minute(),
		ts.Second(), ts.Nanosecond(), ts.Location())

	*ts = newTs
}
