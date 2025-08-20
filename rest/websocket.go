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
package rest

import (
	"context"
	"time"

	"github.com/coder/websocket"
)

// WSHandler - app-level handler per accepted connection.
// Your application sets this; it runs per-connection after a successful upgrade.
type WSHandler func(ctx context.Context, c *websocket.Conn)

// WSConfig - WebSocket settings
type WSConfig struct {
	Enabled        bool
	Path           string        // e.g. "/ws"
	AllowedOrigins []string      // exact origins; "*" = allow all (not recommended)
	BearerSecret   string        // optional: if set, require Authorization: Bearer <secret>
	IdleTimeout    time.Duration // optional: idle limit for a conn; 0 = disabled
	MaxReadBytes   int64         // per-message read limit; 0 = library default
	WriteTimeout   time.Duration // per write; 0 = no extra timeout
	KeepAlive      time.Duration // ping interval; 0 = disabled
	Backlog        int           // outbound queue per client; drop if full
}

func (c *Config) EnableWebSockets(path string, allowedOrigins []string) {
	if c.WS == nil {
		c.WS = &WSConfig{
			Path:           "/ws",
			AllowedOrigins: nil,
			IdleTimeout:    60 * time.Second,
			MaxReadBytes:   1 << 20, // 1 MiB
			WriteTimeout:   10 * time.Second,
			KeepAlive:      30 * time.Second,
			Backlog:        64,
		}
	}
	c.WS.Enabled = true
	if path != "" {
		c.WS.Path = path
	}

	if allowedOrigins == nil {
		allowedOrigins = []string{"https://*", "http://*"} // if no origins are given, we default to these safe options
	}
	c.WS.AllowedOrigins = allowedOrigins

}
