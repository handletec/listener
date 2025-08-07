/*
Copyright © 2024 Vicknesh Suppramaniam <vicknesh@handletec.my>

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
provided under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package listener

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/fsnotify/fsnotify"
)

// TLSConfigBuilder - builds and manages tls.Config instances for both server and client.
type TLSConfigBuilder struct {
	ca         *x509.CertPool
	certFile   string
	keyFile    string
	clientAuth TLSClientAuth
	insecure   bool
	cert       atomic.Value // stores *tls.Certificate
	watcher    *fsnotify.Watcher
	done       chan struct{}
	mu         sync.Mutex // protects CA mutation
}

// NewTLSConfigBuilder - creates a new TLSConfigBuilder. If useSystemCA is true, it loads system root CAs.
func NewTLSConfigBuilder(useSystemCA bool) (*TLSConfigBuilder, error) {
	t := &TLSConfigBuilder{
		clientAuth: TLSClientAuthNone,
		done:       make(chan struct{}),
	}

	var err error
	if useSystemCA {
		t.ca, err = x509.SystemCertPool()
		if err != nil {
			return nil, fmt.Errorf("system cert pool: %w", err)
		}
		if t.ca == nil {
			t.ca = x509.NewCertPool()
		}
	} else {
		t.ca = x509.NewCertPool()
	}

	runtime.SetFinalizer(t, func(obj *TLSConfigBuilder) {
		obj.Close()
	})

	return t, nil
}

// SetInsecureSkipVerify - enables or disables skipping TLS verification (for testing).
func (t *TLSConfigBuilder) SetInsecureSkipVerify(skip bool) {
	t.insecure = skip
}

// AddCAFile - loads a CA certificate from file and adds it to the pool.
func (t *TLSConfigBuilder) AddCAFile(path string) error {
	if path == "" {
		return nil
	}
	pemBytes, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read CA file '%s': %w", path, err)
	}
	return t.AddCABytes(pemBytes)
}

// AddCADir - loads all .crt/.pem files in a directory into the CA pool.
func (t *TLSConfigBuilder) AddCADir(dir string) error {
	if dir == "" {
		return nil
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("read CA dir '%s': %w", dir, err)
	}
	for _, entry := range entries {
		if !entry.Type().IsRegular() {
			continue
		}
		ext := strings.ToLower(filepath.Ext(entry.Name()))
		if ext == ".crt" || ext == ".pem" {
			path := filepath.Join(dir, entry.Name())
			if err := t.AddCAFile(path); err != nil {
				return fmt.Errorf("load CA '%s': %w", path, err)
			}
		}
	}
	return nil
}

// AddCABytes - adds PEM-encoded certificates to the CA pool.
func (t *TLSConfigBuilder) AddCABytes(pemData []byte) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	for {
		block, rest := pem.Decode(pemData)
		if block == nil {
			break
		}
		if block.Type != "CERTIFICATE" {
			pemData = rest
			continue
		}
		cert, err := x509.ParseCertificate(block.Bytes)
		if err != nil {
			return fmt.Errorf("parse cert: %w", err)
		}
		t.ca.AddCert(cert)
		pemData = rest
	}
	return nil
}

// SetCertKeyFile - sets the cert and key files.
func (t *TLSConfigBuilder) SetCertKeyFile(certPath, keyPath string) error {
	if err := t.FileExists(certPath); err != nil {
		return err
	}
	if err := t.FileExists(keyPath); err != nil {
		return err
	}
	t.certFile = certPath
	t.keyFile = keyPath
	return nil
}

// FileExists - checks if the given path exists and is a regular file.
func (t *TLSConfigBuilder) FileExists(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("file check '%s': %w", path, err)
	}
	if info.IsDir() {
		return fmt.Errorf("'%s' is a directory, expected file", path)
	}
	return nil
}

// SetCertKeyFromBytes - sets the cert and key directly from memory.
func (t *TLSConfigBuilder) SetCertKeyFromBytes(certPEM, keyPEM []byte) error {
	cert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		return fmt.Errorf("set cert/key from bytes: %w", err)
	}
	t.cert.Store(&cert)
	return nil
}

// SetClientAuth - sets the desired client auth level.
func (t *TLSConfigBuilder) SetClientAuth(auth TLSClientAuth) {
	t.clientAuth = auth
}

// ForServer - returns a configured *tls.Config for server usage.
func (t *TLSConfigBuilder) ForServer() *tls.Config {
	tlsCfg := &tls.Config{
		ClientAuth: t.clientAuth.AuthType(),
		ClientCAs:  t.ca, // verifies client certificate
		MinVersion: tls.VersionTLS12,
	}
	t.injectServerCert(tlsCfg)
	return tlsCfg
}

// ForClient - returns a configured *tls.Config for client usage.
func (t *TLSConfigBuilder) ForClient() *tls.Config {
	tlsCfg := &tls.Config{
		RootCAs:            t.ca, // verifies server certificate
		MinVersion:         tls.VersionTLS12,
		InsecureSkipVerify: t.insecure,
	}
	t.injectClientCert(tlsCfg)
	return tlsCfg
}

// injectServerCert - sets up the server certificate and starts the file watcher.
func (t *TLSConfigBuilder) injectServerCert(cfg *tls.Config) {
	if t.cert.Load() == nil && t.certFile != "" && t.keyFile != "" {
		if err := t.reloadCert(); err != nil {
			panic(fmt.Errorf("server cert load error: %w", err))
		}
	}
	if cert, ok := t.cert.Load().(*tls.Certificate); ok {
		cfg.Certificates = []tls.Certificate{*cert}
		t.startWatcher()
	}
}

// injectClientCert - sets a static client certificate if configured.
func (t *TLSConfigBuilder) injectClientCert(cfg *tls.Config) {
	if cert, ok := t.cert.Load().(*tls.Certificate); ok {
		cfg.Certificates = []tls.Certificate{*cert}
	}
}

// reloadCert - loads the TLS certificate from configured cert and key files.
func (t *TLSConfigBuilder) reloadCert() error {
	cert, err := tls.LoadX509KeyPair(t.certFile, t.keyFile)
	if err != nil {
		return err
	}
	t.cert.Store(&cert)
	return nil
}

// startWatcher - initializes a file watcher to monitor changes to cert and key files.
func (t *TLSConfigBuilder) startWatcher() {
	if t.watcher != nil {
		return
	}

	w, err := fsnotify.NewWatcher()
	if err != nil {
		fmt.Fprintf(os.Stderr, "watcher init error: %v\n", err)
		return
	}
	t.watcher = w
	dirs := map[string]struct{}{
		filepath.Dir(t.certFile): {},
		filepath.Dir(t.keyFile):  {},
	}
	for dir := range dirs {
		_ = w.Add(dir) // ignore add errors
	}
	go func() {
		defer w.Close()
		for {
			select {
			case ev := <-w.Events:
				if ev.Op&(fsnotify.Write|fsnotify.Create|fsnotify.Rename) != 0 &&
					(ev.Name == t.certFile || ev.Name == t.keyFile) {
					time.Sleep(100 * time.Millisecond)
					if err := t.reloadCert(); err != nil {
						fmt.Fprintf(os.Stderr, "cert reload error: %v\n", err)
					}
				}
			case err := <-w.Errors:
				fmt.Fprintf(os.Stderr, "watcher error: %v\n", err)
			case <-t.done:
				return
			}
		}
	}()
}

// Close - stops file watching.
func (t *TLSConfigBuilder) Close() {
	if t.watcher != nil {
		select {
		case <-t.done:
			// already closed
		default:
			close(t.done)
		}
		t.watcher = nil
	}
}

// VerifyCertTrusted - checks if a given PEM cert is trusted by the internal CA pool.
func (t *TLSConfigBuilder) VerifyCertTrusted(certPEM []byte) error {
	certs, err := x509.ParseCertificates(certPEM)
	if err != nil {
		return fmt.Errorf("parse cert: %w", err)
	}
	for _, cert := range certs {
		_, err := cert.Verify(x509.VerifyOptions{Roots: t.ca})
		if err != nil {
			return fmt.Errorf("cert not trusted: %w", err)
		}
	}
	return nil
}
