//go:build linux

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

	"golang.org/x/sys/unix"
)

// checkPeerCred verifies the peer's credentials on Linux via SO_PEERCRED.
//
// Platform semantics (Linux):
//   - UCred.Uid is the peer's real UID (not effective UID).
//   - UCred.Gid is the peer's real GID (not effective GID).
//   - Supplementary group membership is NOT checked; only the primary real GID
//     is compared against allowGIDs.
//
// See Config.AllowUIDs / Config.AllowGIDs for gate logic and empty-list behaviour.
func checkPeerCred(c *net.UnixConn, allowUIDs, allowGIDs []uint32) (bool, error) {
	rc, err := c.SyscallConn()
	if err != nil {
		return false, err
	}
	var ok bool
	var sysErr error
	if err := rc.Control(func(fd uintptr) {
		ucred, e := unix.GetsockoptUcred(int(fd), unix.SOL_SOCKET, unix.SO_PEERCRED)
		if e != nil {
			sysErr = e
			return
		}
		ok = inList(allowUIDs, ucred.Uid) && inList(allowGIDs, ucred.Gid)
	}); err != nil {
		return false, err
	}
	if sysErr != nil {
		return false, sysErr
	}
	return ok, nil
}