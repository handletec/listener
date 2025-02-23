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

	"github.com/go-chi/chi/v5"
)

// Router - specific path handler
type Router struct {
	base    string
	r       chi.Router
	handler *Handler
}

// NewRouter - create new instance of router under the given base pattern
func NewRouter(base string, middlewares ...func(http.Handler) http.Handler) (router *Router) {
	router = new(Router)
	router.r = chi.NewRouter()

	/*
		// add '/' at the start if none exists
		if !strings.HasPrefix(base, "/") {
			base = "/" + base
		}
		router.base = strings.TrimSuffix(base, "/") // remove any trailing slash
	*/
	router.base = formatBase(base)

	router.r.Use(middlewares...)

	return router
}

// SetHandler - set given handlers for this router instance
func (router *Router) SetHandler(h *Handler) {
	//router.handler = h
	router.handler = h
}

// AddGoup - adds given group to the base of the router, with custom middlewares for an entire group (optional) with group middlewares being executed before handler middlewares
func (router *Router) AddGoup(group *Group) {
	router.r.Mount(router.base+group.base, group.g.h)
	//router.r.With(group.g.h.Middlewares()...).Mount(router.base+group.base, group.g.h)
}

// AddHealthCheck - creates a healthcheck endpoint at the root of the router using the GET verb, without using `base` but still respecting all middlewares for the router only
func (router *Router) AddHealthCheck(rootPath string, hFn http.HandlerFunc) {
	router.r.Mount(formatBase(rootPath), hFn)
}

// mount - makes all handlers not already available under a group to the given defined base pattern
func (router *Router) mount() (err error) {

	if router.handler == nil {
		return errors.New("mount: no handlers configured")
	}

	router.r.Mount(router.base, router.handler.h)
	return
}

// NewChi - set raw `chi.Router` for specific need and greater flexibility
func NewChi(cr *chi.Mux) (router *Router) {
	router = new(Router)
	router.r = cr
	return
}
