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
	"slices"
	"syscall"
)

func inList(list []uint32, v uint32) bool {
	if len(list) == 0 {
		return true
	}
	return slices.Contains(list, v)
}

// isRetryableAcceptErr returns true for transient accept errors worth retrying.
func isRetryableAcceptErr(err error) bool {
	return errors.Is(err, syscall.EINTR) || // interrupted system call
		errors.Is(err, syscall.EAGAIN) || // try again / would block
		errors.Is(err, syscall.ECONNABORTED) // client aborted before accept
}
