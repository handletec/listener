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
	"crypto/tls"
	"fmt"
	"log/slog"
)

// Listener - generic interface that specific interfaces must implement
type Listener interface {
	Name() string
	Init(logger *slog.Logger, address string, port int, tlsConfig *tls.Config) error
	SetConfig(config any) error
	Start() error
}

// Listeners - slice of listeners for specific protocols
type Listeners []Listener

// Add - add a protocol to the listeners type
func (ls *Listeners) Add(l Listener) (err error) {
	*ls = append(*ls, l)
	return
}

// StartAll - start all configured listeners
func (ls Listeners) StartAll() (err error) {

	for _, l := range ls {
		err = l.Start()
		if nil != err {
			return fmt.Errorf("listeners startall: %w", err)
		}
	}

	return
}

func (ls Listeners) String() (str string) {

	if len(ls) == 0 {
		return "no listeners configured"
	}

	str = "configured listeners - "

	for _, l := range ls {
		str += l.Name() + ", "
	}

	return str[:len(str)-2]
}

/*
// Instance - initialize the listeners with the specific protocols
func Instance(proto ...Protocol) (ls Listeners, err error) {

	ls = make(Listeners, 0)

	var l Listener
	for _, p := range proto {
		l, err = p.Listener()
		if nil != err {
			return nil, fmt.Errorf("init: %w", err)
		}

		ls = append(ls, l)
	}

	return
}
*/
