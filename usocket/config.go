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
	"net"
	"os"
	"time"
)

// Config holds UDS listener options.
type Config struct {
	Perm          os.FileMode          // e.g. 0o660
	OwnerUID      int                  // -1 = no chown
	GroupGID      int                  // -1 = no chown
	RemoveStale   bool                 // remove stale socket file before bind (only if it's a socket)
	AcceptBackoff time.Duration        // backoff for temporary Accept errors (default 100ms)
	MaxConns      int                  // 0 = unlimited concurrent handlers
	ReadTimeout   time.Duration        // per-conn read deadline; 0 = none
	WriteTimeout  time.Duration        // per-conn write deadline; 0 = none
	AllowUIDs     []uint32             // if set, require SO_PEERCRED/XUCRED uid in this list
	AllowGIDs     []uint32             // if set, require SO_PEERCRED/XUCRED gid in this list
	Handler       func(net.Conn) error // REQUIRED: per-conn handler
}

// NewConfig returns a Config with sane defaults.
// - 0660 perms, no chown
// - remove stale socket file safely
// - 100ms accept backoff
// - no read/write deadlines (0)
// - no peer-cred restrictions by default
// Provide a non-nil handler; SetConfig will error if Handler is nil.
func NewConfig(handler func(net.Conn) error) Config {
	return Config{
		Perm:          0o660,
		OwnerUID:      -1,
		GroupGID:      -1,
		RemoveStale:   true,
		AcceptBackoff: 100 * time.Millisecond,
		MaxConns:      0,
		ReadTimeout:   0,
		WriteTimeout:  0,
		AllowUIDs:     nil,
		AllowGIDs:     nil,
		Handler:       handler,
	}
}
