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
	"net/http"
	"time"
)

// Config - listener specific configuration
type Config struct {
	CORS          *CORS
	MaxConcurrent int
	Timeout       time.Duration
	compress      bool  // compress response to requester
	maxBody       int64 // maximum body size to be received
	router        *Router
	WS            *WSConfig
}

// NewConfig - creates new instance of config
func NewConfig() (cfg *Config) {
	cfg = new(Config)

	// set default configuration
	cfg.MaxConcurrent = 4096 // maximum concurrent requests per second
	cfg.Timeout = time.Duration(15 * time.Second)
	cfg.CORS = NewCORS()
	cfg.router = nil      // default create a nil instance of handler for error checking
	cfg.maxBody = 1 << 20 // default 1 MiB hard cap

	return
}

// SetCORS - sets CORS information
func (cfg *Config) SetCORS(c *CORS) {
	cfg.CORS = c
}

// SetRouter - sets the required router with handlers for HTTP requests
func (cfg *Config) SetRouter(router *Router) (err error) {
	cfg.router = router
	return
}

// EnableCompress - enable or disable gzip compression
func (cfg *Config) EnableCompress(compress bool) {
	cfg.compress = compress
}

// MaxBody - sets maximum body size in MiB to prevent resource drainage
func (cfg *Config) MaxBody(maxBody int64) {
	cfg.maxBody = maxBody << 20
}

// limitRequestBody enforces a hard max on the request body size.
// If the limit is exceeded (with or without Content-Length), the server
// responds 413 and stops reading further.
func limitRequestBody(max int64) func(http.Handler) http.Handler {
	if max <= 0 {
		return func(next http.Handler) http.Handler { return next }
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r.Body = http.MaxBytesReader(w, r.Body, max)
			// If Content-Length is known and already too big, fail fast.
			if r.ContentLength != -1 && r.ContentLength > max {
				http.Error(w, http.StatusText(http.StatusRequestEntityTooLarge), http.StatusRequestEntityTooLarge)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
