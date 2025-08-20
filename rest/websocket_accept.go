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
	"net/http"
	"strings"
	"time"

	"github.com/coder/websocket"
)

func (l *Listener) wsAccept(w http.ResponseWriter, r *http.Request) {
	if !wsOriginAllowed(r, l.config.CORS.AllowedOrigins) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	ws := l.config.WS
	if ws == nil || !ws.Enabled {
		http.Error(w, "websocket disabled", http.StatusNotFound)
		return
	}

	// 2) (Optional) Bearer token check BEFORE upgrade
	if ws.BearerSecret != "" && !checkBearer(r.Header.Get("Authorization"), ws.BearerSecret) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	// 3) Upgrade
	c, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		// Subprotocols: []string{"your-proto"}, // if you negotiate any
		//OriginPatterns: []string{"http://*", "https://*"},
	})
	if err != nil {
		l.logger.Error("ws accept failed", "err", err)
		return
	}
	// NOTE: do not defer Close here — writer goroutine does it on exit.

	// 4) Per-connection limits
	if ws.MaxReadBytes > 0 {
		c.SetReadLimit(ws.MaxReadBytes)
	}

	// 5) Connection-scoped context with optional idle timeout
	ctx := r.Context()
	var cancel context.CancelFunc
	if ws.IdleTimeout > 0 {
		ctx, cancel = context.WithTimeout(ctx, ws.IdleTimeout)
	} else {
		ctx, cancel = context.WithCancel(ctx)
	}
	defer cancel()

	// 6) Register with hub to get a bounded outbound queue
	client := l.wsHub.Add(c)
	defer l.wsHub.Remove(client)

	// 7) Keepalive pings
	stopKA := startKeepAlive(ctx, c, ws.KeepAlive)
	defer stopKA()

	// 8) Hand off to app handler (or default passthrough echo)
	if l.wsHandler != nil {
		l.wsHandler(ctx, c)
		return
	}

	// Default handler: echo back raw frames; drop if backpressure
	for {
		typ, data, err := c.Read(ctx)
		if err != nil {
			return
		}
		client.TrySend(typ, data)

	}
}

// wsOriginAllowed mimics go-chi/cors matching rules for WS.
func wsOriginAllowed(r *http.Request, allowed []string) bool {
	origin := r.Header.Get("Origin")
	if origin == "" {
		return false // No Origin header — deny unless you want to allow CLI tools
	}
	if len(allowed) == 0 {
		return false
	}

	origin = strings.ToLower(origin)

	for _, o := range allowed {
		pattern := strings.ToLower(o)

		if pattern == "*" {
			return true
		}

		// scheme://*
		if strings.HasSuffix(pattern, "://*") {
			if strings.HasPrefix(origin, strings.TrimSuffix(pattern, "*")) {
				return true
			}
			continue
		}

		// exact match
		if origin == pattern {
			return true
		}
	}

	return false
}

func checkBearer(authHeader, secret string) bool {
	const pfx = "Bearer "
	if !strings.HasPrefix(authHeader, pfx) {
		return false
	}
	return strings.EqualFold(strings.TrimSpace(authHeader[len(pfx):]), secret)
}

func startKeepAlive(ctx context.Context, c *websocket.Conn, every time.Duration) func() {
	if every <= 0 {
		return func() {}
	}
	t := time.NewTicker(every)
	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			select {
			case <-t.C:
				_ = c.Ping(ctx) // ignore error; next read/write surfaces closure
			case <-ctx.Done():
				return
			}
		}
	}()
	return func() { t.Stop(); <-done }
}
