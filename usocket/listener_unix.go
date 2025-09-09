//go:build unix

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
	"crypto/tls"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"os"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"golang.org/x/sys/unix"
)

// Listener implements a hardened UNIX-domain-socket listener.
type Listener struct {
	logger    *slog.Logger
	path      string
	tlsConfig *tls.Config // ignored (parent interface parity)

	cfg Config

	l       net.Listener
	wg      sync.WaitGroup
	quit    chan struct{}
	mu      sync.Mutex
	started bool
	sem     chan struct{} // connection limiter
}

// New constructs a UNIX-socket listener.
func New() *Listener { return &Listener{quit: make(chan struct{})} }

func (u *Listener) Name() string { return "unix" }

// Init sets socket path and (ignored) tls config. Port is ignored for UDS.
func (u *Listener) Init(logger *slog.Logger, socketpath string, _ int, tlsConfig *tls.Config) error {
	if socketpath == "" {
		return errors.New("unix: empty socket path")
	}
	u.logger = logger
	u.path = socketpath
	u.tlsConfig = tlsConfig
	if tlsConfig != nil {
		u.logger.Warn("unix: tlsConfig ignored for UDS")
	}
	return nil
}

func (u *Listener) SetConfig(config any) error {
	cfg, ok := config.(Config)
	if !ok {
		return fmt.Errorf("unix: expected Config, got %T", config)
	}
	if cfg.Handler == nil {
		return errors.New("unix: Config.Handler must be set")
	}
	if cfg.Perm == 0 {
		cfg.Perm = 0o660
	}
	if cfg.AcceptBackoff <= 0 {
		cfg.AcceptBackoff = 100 * time.Millisecond
	}
	if cfg.OwnerUID == 0 && cfg.GroupGID == 0 {
		cfg.OwnerUID, cfg.GroupGID = -1, -1
	}
	u.cfg = cfg
	if cfg.MaxConns > 0 && u.sem == nil {
		u.sem = make(chan struct{}, cfg.MaxConns)
	}
	return nil
}

func (u *Listener) Start() error {
	u.mu.Lock()
	defer u.mu.Unlock()
	if u.started {
		return errors.New("unix: already started")
	}

	if err := os.MkdirAll(filepath.Dir(u.path), 0o755); err != nil {
		return fmt.Errorf("unix: mkdir %s: %w", filepath.Dir(u.path), err)
	}

	if u.cfg.RemoveStale {
		if err := u.isSafeToRemoveStaleSocket(u.path); err != nil {
			return err // be explicit instead of deleting something unexpected
		}
	}

	// Atomic initial perms with umask: final = 0777 &^ umask.
	want := int(u.cfg.Perm & 0o777)
	um := 0o777 &^ want
	prev := unix.Umask(um)
	ln, listenErr := net.Listen("unix", u.path)
	unix.Umask(prev)
	if listenErr != nil {
		return fmt.Errorf("unix: listen %s: %w", u.path, listenErr)
	}
	u.l = ln

	// Enforce final perms/ownership.
	if err := os.Chmod(u.path, u.cfg.Perm); err != nil {
		_ = u.l.Close()
		return fmt.Errorf("unix: chmod %s: %w", u.path, err)
	}
	if u.cfg.OwnerUID >= 0 || u.cfg.GroupGID >= 0 {
		uid := u.cfg.OwnerUID
		gid := u.cfg.GroupGID
		if uid < 0 {
			uid = -1
		}
		if gid < 0 {
			gid = -1
		}
		if err := os.Chown(u.path, uid, gid); err != nil {
			_ = u.l.Close()
			return fmt.Errorf("unix: chown %s: %w", u.path, err)
		}
	}

	u.started = true
	u.wg.Add(1)
	go u.acceptLoop()

	u.logger.Info("unix listener started",
		"path", u.path, "perm", fmt.Sprintf("%#o", u.cfg.Perm),
		"owner", u.cfg.OwnerUID, "group", u.cfg.GroupGID,
		"maxConns", u.cfg.MaxConns)
	return nil
}

func (u *Listener) Close() error {
	u.mu.Lock()
	if !u.started {
		u.mu.Unlock()
		return nil
	}
	close(u.quit)
	_ = u.l.Close() // unblock Accept
	u.mu.Unlock()

	u.wg.Wait()

	if err := os.Remove(u.path); err != nil && !errors.Is(err, os.ErrNotExist) {
		u.logger.Warn("unix: remove socket on close", "path", u.path, "err", err)
	}
	return nil
}

func (u *Listener) acceptLoop() {
	defer u.wg.Done()
	backoff := u.cfg.AcceptBackoff

	for {
		conn, err := u.l.Accept()
		if err != nil {
			// Shutdown path: listener closed
			if errors.Is(err, net.ErrClosed) {
				return
			}
			// External shutdown signal
			select {
			case <-u.quit:
				return
			default:
			}

			// Accept deadline exceeded (only if you set a deadline on the listener)
			var ne net.Error
			if errors.As(err, &ne) && ne.Timeout() {
				// Timeout isn't fatal; continue without backoff
				backoff = u.cfg.AcceptBackoff
				continue
			}

			// File descriptor exhaustion — brief pause
			if errors.Is(err, syscall.EMFILE) || errors.Is(err, syscall.ENFILE) {
				u.logger.Error("unix accept EMFILE/ENFILE", "err", err)
				time.Sleep(500 * time.Millisecond)
				continue
			}

			// Other retryable short-term conditions
			if isRetryableAcceptErr(err) {
				u.logger.Warn("unix accept transient", "err", err)
				time.Sleep(backoff)
				if backoff < 2*time.Second {
					backoff *= 2
				}
				continue
			}

			// Anything else: treat as fatal
			u.logger.Error("unix accept fatal", "err", err)
			return
		}

		// Reset backoff after a successful accept
		backoff = u.cfg.AcceptBackoff

		if u.sem != nil {
			select {
			case u.sem <- struct{}{}:
			case <-u.quit:
				_ = conn.Close()
				return
			}
		}

		u.wg.Add(1)
		go func(c net.Conn) {
			defer u.wg.Done()
			defer func() {
				if u.sem != nil {
					<-u.sem
				}
				_ = c.Close()
			}()

			// Peer-cred allowlist (Linux/macOS). If configured, require UnixConn.
			if len(u.cfg.AllowUIDs) > 0 || len(u.cfg.AllowGIDs) > 0 {
				uc, ok := c.(*net.UnixConn)
				if !ok {
					u.logger.Warn("peercred required but conn is not UnixConn")
					return
				}
				ok2, perr := checkPeerCred(uc, u.cfg.AllowUIDs, u.cfg.AllowGIDs)
				if perr != nil {
					u.logger.Warn("peercred check failed", "err", perr)
					return
				}
				if !ok2 {
					u.logger.Warn("peercred rejected")
					return
				}
			}

			// Optional per-conn deadlines.
			if u.cfg.ReadTimeout > 0 {
				_ = c.SetReadDeadline(time.Now().Add(u.cfg.ReadTimeout))
			}
			if u.cfg.WriteTimeout > 0 {
				_ = c.SetWriteDeadline(time.Now().Add(u.cfg.WriteTimeout))
			}

			// Run the user handler.
			if err := u.cfg.Handler(c); err != nil {
				u.logger.Debug("handler error", "err", err)
			}

		}(conn)
	}
}

// isSafeToRemoveStaleSocket verifies the path is a UNIX socket we own and it's not active.
func (u *Listener) isSafeToRemoveStaleSocket(path string) error {
	fi, err := os.Lstat(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil // nothing to do
		}
		return fmt.Errorf("lstat %s: %w", path, err)
	}

	// Refuse to touch symlinks or non-sockets.
	if fi.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("refusing to remove symlink: %s", path)
	}
	if fi.Mode()&os.ModeSocket == 0 {
		return fmt.Errorf("existing path is not a unix socket: %s (mode=%v)", path, fi.Mode())
	}

	// Refuse to remove if owned by someone else (unless root).
	if st, ok := fi.Sys().(*syscall.Stat_t); ok {
		euid := uint32(os.Geteuid())
		if euid != 0 && st.Uid != euid {
			return fmt.Errorf("socket owned by uid=%d; refusing to remove as euid=%d", st.Uid, euid)
		}
	}

	// Probe: if we can connect, someone is listening → do NOT remove.
	if c, err := net.DialTimeout("unix", path, 200*time.Millisecond); err == nil {
		_ = c.Close()
		return fmt.Errorf("socket appears active; refusing to remove: %s", path)
	} else {
		// Remove only on clear stale signals.
		switch {
		case errors.Is(err, syscall.ECONNREFUSED):
			// stale socket file; safe to remove
		case errors.Is(err, os.ErrNotExist):
			// vanished between lstat and dial; nothing to do
			return nil
		default:
			// Permission denied / timeout / other ambiguous errors → be cautious.
			var ne net.Error
			if errors.As(err, &ne) && ne.Timeout() {
				return fmt.Errorf("connect probe timeout; refusing to remove: %w", err)
			}
			return fmt.Errorf("connect probe ambiguous; refusing to remove: %w", err)
		}
	}

	// Remove the stale socket file.
	if rmErr := os.Remove(path); rmErr != nil && !errors.Is(rmErr, os.ErrNotExist) {
		return fmt.Errorf("remove stale socket %s: %w", path, rmErr)
	}
	u.logger.Info("removed stale unix socket", "path", path)
	return nil
}
