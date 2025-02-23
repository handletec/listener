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
	"time"
)

// Config - listener specific configuration
type Config struct {
	CORS     *CORS
	RPS      int
	Timeout  time.Duration
	compress bool // compress response to requester
	//handlers http.Handler
	router *Router
}

// NewConfig - creates new instance of config
func NewConfig() (cfg *Config) {
	cfg = new(Config)

	// set default configuration
	cfg.RPS = 4096 // default request per second
	cfg.Timeout = time.Duration(15 * time.Second)
	cfg.CORS = NewCORS()
	cfg.router = nil // default create a nil instance of handler for error checking

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
