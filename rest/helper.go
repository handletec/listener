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
	"strings"
)

type Method uint8

const (
	MethodUnknown Method = iota
	MethodGet
	MethodPost
	MethodPut
	MethodPatch
	MethodDelete
	MethodHead
	MethodOptions
	MethodConnect
)

const (
	// PatternAll - helper constant for all
	PatternAll = "/*"
	// HealthEndpoint - default endpoint for healthchecks
	HealthEndpoint = "healthcheck"
)

/*
// OptionsHandler - helper function to handle Options method
func OptionsHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}
*/

// HealthCheckHandler - helper function to handle Options method
func HealthCheckHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func (method Method) String() (str string) {

	methodName := []string{"UNKNOWN", "GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS", "CONNECT"}
	methodInt := int(method)

	if methodInt < 0 || methodInt >= len(methodName) {
		methodInt = 0
	}

	return methodName[methodInt]
}

// formatBase - reformats the base pattern to have the correct formatting
func formatBase(input string) (output string) {

	// add '/' at the start if none exists
	if !strings.HasPrefix(input, "/") {
		input = "/" + input
	}

	return strings.TrimSuffix(input, "/") // remove any trailing slash
}
