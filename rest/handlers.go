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
	"errors"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
)

// Handler - method, path and function to run
type Handler struct {
	h *chi.Mux
}

// NewNewHandler - create new instance of handler
func NewNewHandler() (handler *Handler) {
	handler = new(Handler)
	handler.h = chi.NewRouter()

	return handler
}

// Set - sets a handler to a given pattern
func (handler *Handler) Set(method Method, pattern string, hFn http.HandlerFunc, middlewares ...func(http.Handler) http.Handler) (err error) {

	methodStr := method.String()

	if methodStr == "UNKNOWN" {
		return errors.New("REST sethandler: unknown method type")
	}

	if len(pattern) == 0 {
		return errors.New("REST sethandler: pattern cannot be left blank")
	}

	// add custom middlewares for this endpoint, could be authentication, authorization, etc
	handler.h.With(middlewares...).MethodFunc(methodStr, pattern, hFn)

	return
}

// optionsHandler - automatically respond to OPTIONS
func optionsHandler(cors *CORS) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Set CORS headers
		w.Header().Set("Access-Control-Allow-Origin", strings.Join(cors.AllowedOrigins, ","))
		w.Header().Set("Access-Control-Allow-Methods", strings.Join(cors.AllowedMethods, ","))
		w.Header().Set("Access-Control-Allow-Headers", strings.Join(cors.AllowedHeaders, ","))
	}
}
