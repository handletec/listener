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
package rest

import "net/http"

// Header - sets custom header to be returned for the request
type Header map[string]string

func NewHeader() (header *Header) {
	h := make(Header)
	return &h
}

// Add - adds a header
func (header *Header) Add(key, value string) {
	(*header)[key] = value
}

// Has - checks if a given header exists
func (header *Header) Has(key string) (exist bool) {
	_, exist = (*header)[key]
	return exist
}

// SetCustomHeaders - sets custom headers to be injected to all requests
func (l *Listener) SetCustomHeaders(header *Header) {
	l.header = header
}

// headerMiddleware - inject the headers specified automatically into all requests
func headerMiddleware(headers *Header) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			for k, v := range *headers {
				w.Header().Set(k, v)
			}
			next.ServeHTTP(w, r)
		})
	}
}
