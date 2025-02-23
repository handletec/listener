/*
Copyright © 2024 Vicknesh Suppramaniam <vicknesh@handletec.my>

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package listener

import (
	"fmt"
	"strings"

	"github.com/handletec/listener/rest"
)

// Protocol - custom protocol definitions
type Protocol uint8

const (
	ProtoNone Protocol = iota
	ProtoREST
	ProtoMQTT
)

// Listener - returns listener implementation for this protocol
func (proto Protocol) Listener() (l Listener, err error) {

	if !proto.IsValid() {
		return nil, fmt.Errorf("proto listener: unknown protocol given, cannot create listener")
	}

	switch proto {
	case ProtoREST:
		l = new(rest.Listener)
	case ProtoMQTT:
		//l = new(mqtt.Listener)
	}

	return
}

// IsValid - determines if a valid protocol is given
func (proto Protocol) IsValid() (valid bool) {
	if proto.String() != "NONE" {
		valid = true
	}

	return valid
}

func (proto Protocol) String() (str string) {

	protoName := []string{"NONE", "REST", "MQTT"}
	protoInt := int(proto)

	if protoInt < 0 || protoInt >= len(protoName) {
		protoInt = 0
	}

	return protoName[protoInt]
}

// ParseProto - returns protocol type from given string
func ParseProto(protoStr string) (proto Protocol) {

	switch strings.ToUpper(protoStr) {
	case "REST":
		proto = ProtoREST
	case "MQTT":
		proto = ProtoMQTT
	default:
		// if something unrecognized is given, set it to none
		proto = ProtoNone
	}

	return
}
