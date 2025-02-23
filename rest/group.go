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
	"fmt"
	"net/http"
)

type Group struct {
	base string
	g    *Handler
}

// NewGroup - create new instance of group
func NewGroup(base string, middlewares ...func(http.Handler) http.Handler) (group *Group) {
	group = new(Group)
	group.g = NewNewHandler()

	/*
		// add '/' at the start if none exists
		if !strings.HasPrefix(base, "/") {
			base = "/" + base
		}
		group.base = strings.TrimSuffix(base, "/") // remove any trailing slash
	*/
	group.base = formatBase(base)

	group.g.h.Use(middlewares...)

	return group
}

// Set - sets a handler to a given pattern
func (group *Group) Set(method Method, pattern string, hFn http.HandlerFunc, middlewares ...func(http.Handler) http.Handler) (err error) {

	err = group.g.Set(method, pattern, hFn, middlewares...)
	if nil != err {
		return fmt.Errorf("groupset: %w", err)
	}

	return
}
