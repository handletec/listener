//go:build !linux && !darwin

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
	"errors"
	"net"
)

// Peer credential checking is not supported on this platform; deny the connection.
func checkPeerCred(_ *net.UnixConn, _ []uint32, _ []uint32) (bool, error) {
	return false, errors.New("peer credentials not supported on this platform")
}
