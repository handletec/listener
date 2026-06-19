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
	"log/slog"
	"sync"

	"github.com/coder/websocket"
)

type hubMsg struct {
	typ  websocket.MessageType
	data []byte
}

type WSHub struct {
	log       *slog.Logger
	cfg       *WSConfig
	mu        sync.RWMutex
	cl        map[*wsClient]struct{}
	inbox     chan hubMsg
	closed    chan struct{}
	closeOnce sync.Once
}

type wsClient struct {
	conn      *websocket.Conn
	out       chan hubMsg // bounded queue per client
	cfg       *WSConfig
	log       *slog.Logger
	closed    chan struct{}
	closeOnce sync.Once
}

func NewWSHub(log *slog.Logger, cfg *WSConfig) *WSHub {
	return &WSHub{
		log:    log,
		cfg:    cfg,
		cl:     make(map[*wsClient]struct{}),
		inbox:  make(chan hubMsg, 1024), // drop if hub is overloaded
		closed: make(chan struct{}),
	}
}

func (h *WSHub) Run() {
	for {
		select {
		case m, ok := <-h.inbox:
			if !ok {
				return
			}
			h.mu.RLock()
			for c := range h.cl {
				select {
				case c.out <- m:
				default:
					// backpressure: drop to slow client (avoid blocking hub)
					// h.log.Warn("ws: drop to slow client")
				}
			}
			h.mu.RUnlock()
		case <-h.closed:
			return
		}
	}
}

func (h *WSHub) BroadcastText(b []byte) {
	cp := append([]byte(nil), b...)
	select {
	case h.inbox <- hubMsg{typ: websocket.MessageText, data: cp}:
	default:
		// drop if hub is overloaded
	}
}

func (h *WSHub) BroadcastBinary(b []byte) {
	cp := append([]byte(nil), b...)
	select {
	case h.inbox <- hubMsg{typ: websocket.MessageBinary, data: cp}:
	default:
	}
}

func (h *WSHub) Add(c *websocket.Conn) *wsClient {
	cl := &wsClient{
		conn:   c,
		out:    make(chan hubMsg, max(1, h.cfg.Backlog)),
		cfg:    h.cfg,
		log:    h.log,
		closed: make(chan struct{}),
	}
	h.mu.Lock()
	h.cl[cl] = struct{}{}
	h.mu.Unlock()
	go cl.writer()
	return cl
}

func (h *WSHub) Remove(cl *wsClient) {
	h.mu.Lock()
	delete(h.cl, cl) // unnecessary guard removed — delete is safe on missing keys
	h.mu.Unlock()
	cl.close()
}

func (h *WSHub) Close() {
	h.closeOnce.Do(func() {
		close(h.closed)
		h.mu.Lock()
		for cl := range h.cl {
			cl.close()
		}
		h.cl = map[*wsClient]struct{}{}
		h.mu.Unlock()
		close(h.inbox)
	})
}

func (c *wsClient) writer() {
	defer func() {
		_ = c.conn.Close(websocket.StatusNormalClosure, "bye")
		close(c.closed)
	}()
	// Each write uses a fresh context so that an individual write timeout does
	// not cancel the entire connection. The connection lifetime is managed by
	// the hub: wsHub.Remove closes c.out (via wsClient.close), which terminates
	// this range loop and triggers the deferred conn.Close above.
	//
	// Design note: writes are not tied to a per-connection or server shutdown
	// context. On Stop(), wsHub.Close() calls cl.close() for every registered
	// client, which closes c.out and stops this goroutine. In-flight writes
	// may complete before the loop exits, but they are bounded by WriteTimeout.
	// This is an intentional trade-off: avoiding a shared context prevents a
	// single slow write from cancelling pending messages for other clients.
	for m := range c.out {
		var (
			ctx    context.Context = context.Background()
			cancel context.CancelFunc
		)
		if c.cfg.WriteTimeout > 0 {
			ctx, cancel = context.WithTimeout(ctx, c.cfg.WriteTimeout)
		}
		err := c.conn.Write(ctx, m.typ, m.data)
		if cancel != nil {
			cancel()
		}
		if err != nil {
			return
		}
	}
}

func (c *wsClient) TrySend(t websocket.MessageType, b []byte) bool {
	cp := append([]byte(nil), b...)
	select {
	case c.out <- hubMsg{typ: t, data: cp}:
		return true
	default:
		return false
	}
}

func (c *wsClient) close() {
	c.closeOnce.Do(func() { close(c.out) })
}

