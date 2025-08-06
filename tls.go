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
package listener

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync/atomic"
	"time"

	"github.com/fsnotify/fsnotify"
)

// TLS - configure TLS information
type TLS struct {
	ca                *x509.CertPool
	certFile, keyFile string
	at                TLSClientAuth

	cert    atomic.Value      // holds *tls.Certificate
	watcher *fsnotify.Watcher // internal fsnotify watcher
	done    chan struct{}     // signals watcher shutdown
}

// NewTLS - create new instance of TLS
func NewTLS(useOSCA bool) (t *TLS, err error) {
	t = new(TLS)
	t.at = TLSClientAuthNone // default client auth type is none

	if useOSCA {
		t.ca, err = x509.SystemCertPool()
		if nil != err {
			return nil, fmt.Errorf("new tls: %w", err)
		}

		if nil == t.ca {
			t.ca = x509.NewCertPool() // if the system CA pool is empty, create a new empty instance
		}
	} else {
		t.ca = x509.NewCertPool() // create a new instance of cert pool
	}

	// register finalizer for automatic cleanup
	runtime.SetFinalizer(t, func(obj *TLS) {
		obj.Close()
	})

	return
}

// AddCADir - adds a certificate authority from directory
func (t *TLS) AddCADir(cadirname string) (err error) {

	if len(cadirname) == 0 {
		return // if an empty CA dirname is given, return immediately
	}

	entries, err := os.ReadDir(cadirname)
	if err != nil {
		return fmt.Errorf("add CA directory: error reading '%s' -> %w", cadirname, err)
	}

	for _, entry := range entries {

		if entry.Type().IsRegular() {
			switch strings.ToLower(filepath.Ext(entry.Name())) {
			case ".crt", ".pem":
				path := filepath.Join(cadirname, entry.Name())
				if err := t.AddCAFile(path); err != nil {
					return fmt.Errorf("add CA directory: reading contents of client CA file '%s' error -> %w", entry.Name(), err)
				}
			}
		}
	}

	return
}

// AddCAFile - adds a certificate authority from file
func (t *TLS) AddCAFile(cafile string) (err error) {

	if len(cafile) == 0 {
		return // if an empty CA filename is given, return immediately
	}

	cabytes, err := os.ReadFile(cafile)
	if nil != err {
		return fmt.Errorf("add CA file: error reading CA file '%s' -> %w", cafile, err)
	}

	err = t.AddCABytes(cabytes)
	if nil != err {
		return fmt.Errorf("add CA file '%s': %w", cafile, err)
	}

	return
}

// AddCABytes - adds a certificate authority from bytes
func (t *TLS) AddCABytes(cabytes []byte) (err error) {
	if !t.ca.AppendCertsFromPEM(cabytes) {
		return fmt.Errorf("add CA bytes: unable to append client CA bytes")
	}

	return
}

/*
// SetCertBytes - sets cert bytes for the listener
func (t *TLS) SetCertBytes(certbytes []byte) (err error) {

	return
}

// SetKeyBytes - sets key bytes for the listener
func (t *TLS) SetKeyBytes(keybytes []byte) (err error) {

	return
}
*/

// FileExists - check if file exists
func (t *TLS) FileExists(filename string) (err error) {
	f, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return fmt.Errorf("'%s' does not exist", filename)
	} else if err != nil {
		return fmt.Errorf("'%s' file error: %w", filename, err)
	}

	if f.IsDir() {
		return fmt.Errorf("'%s' file expected, directory given", filename)
	}

	return nil
}

// SetCertFile - sets a cert file for the listener
func (t *TLS) SetCertFile(certfile string) (err error) {
	err = t.FileExists(certfile)
	if nil != err {
		return fmt.Errorf("setcertfile '%s' error: %w", certfile, err)
	}

	t.certFile = certfile
	return
}

// SetKeyFile - sets a key file for the listener
func (t *TLS) SetKeyFile(keyfile string) (err error) {
	err = t.FileExists(keyfile)
	if nil != err {
		return fmt.Errorf("setkeyfile '%s' error: %w", keyfile, err)
	}

	t.keyFile = keyfile
	return
}

// SetClientAuthType - sets preferred client auth type
func (t *TLS) SetClientAuthType(at TLSClientAuth) (err error) {
	t.at = at
	return
}

// GetTLSConfig - returns istance of tls.Config for listeners
func (t *TLS) GetTLSConfig() (tlsConfig *tls.Config) {

	// if both the cert and key file are given, only then we specify the function to read them for every request
	if len(t.certFile) != 0 && len(t.keyFile) != 0 {
		tlsConfig = &tls.Config{
			RootCAs:    t.ca,             // server uses this CA Pool to verify outgoing certificates (useful when we reuse this code on the client end)
			ClientCAs:  t.ca,             // server uses this CA Pool to verify incoming client certificates
			MinVersion: tls.VersionTLS12, // this is the minimum version of TLS we support, anything else is discarded
			ClientAuth: t.at.AuthType(),
		}

		// Initial load if needed
		if t.cert.Load() == nil {
			if err := t.reloadCert(); err != nil {
				panic(fmt.Errorf("getTLSConfig: initial cert load failed -> %w", err))
			}
		}

		// Zero‑overhead handshake path
		cert := t.cert.Load().(*tls.Certificate)
		tlsConfig.Certificates = []tls.Certificate{*cert}

		// Start watcher once
		if t.watcher == nil {
			t.startWatcher()
		}
	} else {
		tlsConfig = new(tls.Config) // if no certificate is given, we return a new instance of `tls.Config` so it won't be nil
	}

	return tlsConfig
}

// reloadCert reads the cert/key pair from disk and updates the atomic cache.
func (t *TLS) reloadCert() error {
	pair, err := tls.LoadX509KeyPair(t.certFile, t.keyFile)
	if err != nil {
		return fmt.Errorf("reload cert: %w", err)
	}
	t.cert.Store(&pair)
	return nil
}

// startWatcher initializes fsnotify to watch certFile and keyFile directories.
func (t *TLS) startWatcher() {
	w, err := fsnotify.NewWatcher()
	if err != nil {
		fmt.Fprintf(os.Stderr, "tls watcher init error: %v\n", err)
		return
	}
	t.watcher = w
	t.done = make(chan struct{})

	dirs := map[string]struct{}{
		filepath.Dir(t.certFile): {},
		filepath.Dir(t.keyFile):  {},
	}
	for d := range dirs {
		if err := w.Add(d); err != nil {
			fmt.Fprintf(os.Stderr, "tls watcher watch %s error: %v\n", d, err)
		}
	}

	go func() {
		defer w.Close()
		for {
			select {
			case ev := <-w.Events:
				if ev.Op&(fsnotify.Write|fsnotify.Create|fsnotify.Rename) != 0 &&
					(ev.Name == t.certFile || ev.Name == t.keyFile) {
					time.Sleep(100 * time.Millisecond) // debounce
					if err := t.reloadCert(); err != nil {
						fmt.Fprintf(os.Stderr, "tls hot reload error: %v\n", err)
					}
				}
			case err := <-w.Errors:
				fmt.Fprintf(os.Stderr, "tls watcher error: %v\n", err)
			case <-t.done:
				return
			}
		}
	}()
}

// Close stops the internal file watcher (call on shutdown).
func (t *TLS) Close() {
	if t.watcher == nil {
		return
	}
	select {
	case <-t.done:
		// already closed
	default:
		close(t.done)
	}
	t.watcher = nil
}
