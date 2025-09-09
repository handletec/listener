//go:build darwin

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

func checkPeerCred(c *net.UnixConn, allowUIDs, allowGIDs []uint32) (bool, error) {
	rc, err := c.SyscallConn()
	if err != nil {
		return false, err
	}
	var ok bool
	var sysErr error
	if err := rc.Control(func(fd uintptr) {
		xu, e := unix.GetsockoptXucred(int(fd), unix.SOL_LOCAL, unix.LOCAL_PEERCRED)
		if e != nil {
			sysErr = e
			return
		}
		// Primary group is Groups[0] when present
		var gid uint32
		if xu.Ngroups > 0 {
			gid = xu.Groups[0]
		}
		ok = inList(allowUIDs, xu.Uid) && inList(allowGIDs, gid)
	}); err != nil {
		return false, err
	}
	if sysErr != nil {
		return false, sysErr
	}
	return ok, nil
}