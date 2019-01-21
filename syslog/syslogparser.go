//: ----------------------------------------------------------------------------
//: Copyright (C) 2019 Helmut Wahrmann.
//:
//: file:    syslogparser.go
//: details: Syslog Parser Helper
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
	"fmt"
	"strconv"
	"strings"
	"time"
)

const (
	PRI_PART_START = '<'
	PRI_PART_END   = '>'

	// The special enVision Header
	PRI_PART_ENVISION = '['

	NO_VERSION = -1
)

var (
	ErrEOL     = &ParserError{"End of log line"}
	ErrNoSpace = &ParserError{"No space found"}

	ErrEnvisionFormat = &ParserError{"Envision Format"}

	ErrPriorityNoStart  = &ParserError{"No start char found for priority"}
	ErrPriorityEmpty    = &ParserError{"Priority field empty"}
	ErrPriorityNoEnd    = &ParserError{"No end char found for priority"}
	ErrPriorityTooShort = &ParserError{"Priority field too short"}
	ErrPriorityTooLong  = &ParserError{"Priority field too long"}
	ErrPriorityNonDigit = &ParserError{"Non digit found in priority"}

	ErrVersionNotFound = &ParserError{"Can not find version"}

	ErrTimestampUnknownFormat = &ParserError{"Timestamp format unknown"}
)

type LogParser interface {
	Parse() error
	Dump() LogParts
	Location(*time.Location)
}

type ParserError struct {
	ErrorString string
}

type Priority struct {
	P int
	F Facility
	S Severity
}

type Facility struct {
	Value int
}

type Severity struct {
	Value int
}

// https://tools.ietf.org/html/rfc3164#section-4.1
func ParsePriority(buff []byte, cursor *int, l int) (Priority, error) {
	pri := newPriority(0)

	if l <= 0 {
		return pri, ErrPriorityEmpty
	}

	if buff[*cursor] == PRI_PART_ENVISION {
		return pri, ErrEnvisionFormat
	}

	if buff[*cursor] != PRI_PART_START {
		return pri, ErrPriorityNoStart
	}

	i := 1
	priDigit := 0

	for i < l {
		if i >= 5 {
			return pri, ErrPriorityTooLong
		}

		c := buff[i]

		if c == PRI_PART_END {
			if i == 1 {
				return pri, ErrPriorityTooShort
			}

			*cursor = i + 1
			return newPriority(priDigit), nil
		}

		if IsDigit(c) {
			v, e := strconv.Atoi(string(c))
			if e != nil {
				return pri, e
			}

			priDigit = (priDigit * 10) + v
		} else {
			return pri, ErrPriorityNonDigit
		}

		i++
	}

	return pri, ErrPriorityNoEnd
}

// https://tools.ietf.org/html/rfc5424#section-6.2.2
func ParseVersion(buff []byte, cursor *int, l int) (int, error) {
	if *cursor >= l {
		return NO_VERSION, ErrVersionNotFound
	}

	c := buff[*cursor]
	*cursor++

	// XXX : not a version, not an error though as RFC 3164 does not support it
	if !IsDigit(c) {
		return NO_VERSION, nil
	}

	v, e := strconv.Atoi(string(c))
	if e != nil {
		*cursor--
		return NO_VERSION, e
	}

	return v, nil
}

func IsDigit(c byte) bool {
	return c >= '0' && c <= '9'
}

func newPriority(p int) Priority {
	// The Priority value is calculated by first multiplying the Facility
	// number by 8 and then adding the numerical value of the Severity.

	return Priority{
		P: p,
		F: Facility{Value: p / 8},
		S: Severity{Value: p % 8},
	}
}

func FindNextSpace(buff []byte, from int, l int) (int, error) {
	var to int

	for to = from; to < l; to++ {
		if buff[to] == ' ' {
			to++
			return to, nil
		}
	}

	return 0, ErrNoSpace
}

func Parse2Digits(buff []byte, cursor *int, l int, min int, max int, e error) (int, error) {
	digitLen := 2

	if *cursor+digitLen > l {
		return 0, ErrEOL
	}

	sub := string(buff[*cursor : *cursor+digitLen])

	*cursor += digitLen

	i, err := strconv.Atoi(sub)
	if err != nil {
		return 0, e
	}

	if i >= min && i <= max {
		return i, nil
	}

	return 0, e
}

func ParseHostname(buff []byte, cursor *int, l int) (string, error) {
	from := *cursor
	var to int

	// Handle the envision Header time
	if string(buff[*cursor:*cursor+4]) == "[][]" {
		from = *cursor + 5
		for to = from; to < l; to++ {
			if buff[to] == ']' {
				break
			}
		}
		hostname := buff[from:to]
		// Now search for the last pair of brackets to set the cursor
		ix := bytes.Index(buff[to:], []byte("[]"))
		to += ix + 2
		*cursor = to
		return string(hostname), nil
	}

	// Hostname shall be parsed by regex
	*cursor = 0

	return "", nil
}

func ShowCursorPos(buff []byte, cursor int) {
	fmt.Println(string(buff))
	padding := strings.Repeat("-", cursor)
	fmt.Println(padding + "↑\n")
}

func (err *ParserError) Error() string {
	return err.ErrorString
}
