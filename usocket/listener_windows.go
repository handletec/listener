//go:build windows

/*
Copyright © 2025 Vicknesh Suppramaniam <vicknesh@handletec.my>

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

package usocket

import (
	"crypto/tls"
	"errors"
	"log/slog"
)

// Windows stub so the module builds on Windows. UDS features aren’t supported here.
type Listener struct{}

func New() *Listener                                                { return &Listener{} }
func (*Listener) Name() string                                      { return "unix" }
func (*Listener) Init(*slog.Logger, string, int, *tls.Config) error { return nil }
func (*Listener) SetConfig(any) error                               { return nil }
func (*Listener) Start() error {
	return errors.New("usocket: UNIX sockets unsupported on Windows build")
}
func (*Listener) Close() error { return nil }
