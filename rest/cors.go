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

// CORS - Cross Origin Resource Sharing configuration
type CORS struct {
	AllowedOrigins   []string
	AllowedMethods   []string
	AllowedHeaders   []string
	MaxAge           int  // maximum number of seconds the results can be cached
	Debug            bool // debug CORS detauls
	AllowCredentials bool // indicates if this REST service should release the response to cross-origin requests (typically JavaScript code); cross-origin are those that don't match the defined origins
}

// NewCORS - create new instance for stroring CORS information
func NewCORS() (c *CORS) {
	c = new(CORS)

	// set default CORS
	c.SetMethods([]string{"OPTIONS", "GET", "POST", "PUT", "DELETE", "HEAD", "CONNECT"})
	c.SetHeaders([]string{"Accept", "Content-Type", "Authorization", "Access-Control-Request-Method", "Access-Control-Request-Headers"})
	c.SetOrigins([]string{"https://*", "http://*"})

	return
}

// SetOrigins - sets origins allowed for this REST service
func (c *CORS) SetOrigins(v []string) {
	c.AllowedOrigins = v
}

// SetMethods - sets HTTP methods allowed for this REST service
func (c *CORS) SetMethods(v []string) {
	c.AllowedMethods = v
}

// SetHeaders - sets HTTP headers allowed for this REST service
func (c *CORS) SetHeaders(v []string) {
	c.AllowedHeaders = v
}

func (l *Listener) CORS(c *CORS) {
	l.config.CORS = c
}
