//: ----------------------------------------------------------------------------
//: Copyright (C) 2019 Helmut Wahrmann.
//:
//: file:    handler.go
//: details: Syslog Handler
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

//Handler receive every syslog entry at Handle method
type Handler interface {
	Handle(LogParts, int64, error)
}

//LogPartsChannel is a map of the result of parsing the log message
type LogPartsChannel chan LogParts

//The ChannelHandler will send all the syslog entries into the given channel
type ChannelHandler struct {
	channel LogPartsChannel
}

//NewChannelHandler returns a new ChannelHandler
func NewChannelHandler(channel LogPartsChannel) *ChannelHandler {
	handler := new(ChannelHandler)
	handler.SetChannel(channel)

	return handler
}

//SetChannel sets channel to be used
func (h *ChannelHandler) SetChannel(channel LogPartsChannel) {
	h.channel = channel
}

//Handle is the Syslog entry receiver
func (h *ChannelHandler) Handle(logParts LogParts, messageLength int64, err error) {
	h.channel <- logParts
}
