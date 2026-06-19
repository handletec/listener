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
	ca          *x509.CertPool
	certFile    string
	keyFile     string
	clientAuth  TLSClientAuth
	insecure    bool
	cert        atomic.Value // stores *tls.Certificate
	watcher     *fsnotify.Watcher
	watcherOnce sync.Once
	done        chan struct{}
	mu          sync.Mutex // protects ca, certFile, keyFile, clientAuth, insecure, watcher
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

// SetInsecureSkipVerify - enables or disables skipping TLS verification.
// WARNING: for testing only; never set true in production.
func (t *TLSConfigBuilder) SetInsecureSkipVerify(skip bool) {
	t.mu.Lock()
	t.insecure = skip
	t.mu.Unlock()
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
// Returns an error if the input is empty, contains no CERTIFICATE blocks,
// or any CERTIFICATE block contains malformed DER.
func (t *TLSConfigBuilder) AddCABytes(pemData []byte) error {
	if len(pemData) == 0 {
		return fmt.Errorf("CA data is empty")
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	found := false
	for {
		block, rest := pem.Decode(pemData)
		if block == nil {
			break
		}
		pemData = rest
		if block.Type != "CERTIFICATE" {
			continue
		}
		cert, err := x509.ParseCertificate(block.Bytes)
		if err != nil {
			return fmt.Errorf("parse certificate: %w", err)
		}
		t.ca.AddCert(cert)
		found = true
	}

	if !found {
		return fmt.Errorf("no certificate found in PEM data")
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
	t.mu.Lock()
	t.certFile = certPath
	t.keyFile = keyPath
	t.mu.Unlock()
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
	t.mu.Lock()
	t.clientAuth = auth
	t.mu.Unlock()
}

// ForServer - returns a configured *tls.Config for server usage.
// The returned config holds a snapshot of the CA pool at the time of the call.
func (t *TLSConfigBuilder) ForServer() (*tls.Config, error) {
	t.mu.Lock()
	caClone := t.ca.Clone()
	clientAuth := t.clientAuth
	t.mu.Unlock()

	tlsCfg := &tls.Config{
		ClientAuth: clientAuth.AuthType(),
		ClientCAs:  caClone,
		MinVersion: tls.VersionTLS12,
	}
	if err := t.injectServerCert(tlsCfg); err != nil {
		return nil, err
	}
	return tlsCfg, nil
}

// ForClient - returns a configured *tls.Config for client usage.
// Returns an error if SetCertKeyFile was used and the certificate files cannot be loaded.
// The returned config holds a snapshot of the CA pool at the time of the call.
func (t *TLSConfigBuilder) ForClient() (*tls.Config, error) {
	t.mu.Lock()
	caClone := t.ca.Clone()
	insecure := t.insecure
	t.mu.Unlock()

	tlsCfg := &tls.Config{
		RootCAs:            caClone,
		MinVersion:         tls.VersionTLS12,
		InsecureSkipVerify: insecure, //nolint:gosec // controlled by caller via SetInsecureSkipVerify
	}
	if err := t.injectClientCert(tlsCfg); err != nil {
		return nil, err
	}
	return tlsCfg, nil
}

// injectServerCert - sets up the server certificate and starts the file watcher.
func (t *TLSConfigBuilder) injectServerCert(cfg *tls.Config) error {
	t.mu.Lock()
	hasPaths := t.certFile != "" && t.keyFile != ""
	t.mu.Unlock()

	if t.cert.Load() == nil && hasPaths {
		if err := t.reloadCert(); err != nil {
			return fmt.Errorf("server cert load: %w", err)
		}
	}
	if cert, ok := t.cert.Load().(*tls.Certificate); ok {
		cfg.Certificates = []tls.Certificate{*cert}
		t.startWatcher()
	}
	return nil
}

// injectClientCert - loads and injects a client certificate if configured.
// Loads from files if SetCertKeyFile was used and the cert has not yet been loaded.
// Does not start the file watcher; client-side hot reload is out of scope.
func (t *TLSConfigBuilder) injectClientCert(cfg *tls.Config) error {
	t.mu.Lock()
	hasPaths := t.certFile != "" && t.keyFile != ""
	t.mu.Unlock()

	if t.cert.Load() == nil && hasPaths {
		if err := t.reloadCert(); err != nil {
			return fmt.Errorf("client cert load: %w", err)
		}
	}
	if cert, ok := t.cert.Load().(*tls.Certificate); ok {
		cfg.Certificates = []tls.Certificate{*cert}
	}
	return nil
}

// reloadCert - loads the TLS certificate from configured cert and key files.
func (t *TLSConfigBuilder) reloadCert() error {
	t.mu.Lock()
	certFile, keyFile := t.certFile, t.keyFile
	t.mu.Unlock()

	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return err
	}
	t.cert.Store(&cert)
	return nil
}

// startWatcher - initializes a file watcher to monitor changes to cert and key files.
// Safe to call concurrently; the watcher is started at most once.
func (t *TLSConfigBuilder) startWatcher() {
	t.watcherOnce.Do(func() {
		select {
		case <-t.done:
			return
		default:
		}

		t.mu.Lock()
		certFile := t.certFile
		keyFile := t.keyFile
		t.mu.Unlock()

		w, err := fsnotify.NewWatcher()
		if err != nil {
			fmt.Fprintf(os.Stderr, "watcher init error: %v\n", err)
			return
		}

		t.mu.Lock()
		t.watcher = w
		t.mu.Unlock()

		dirs := map[string]struct{}{
			filepath.Dir(certFile): {},
			filepath.Dir(keyFile):  {},
		}
		for dir := range dirs {
			if err := w.Add(dir); err != nil {
				fmt.Fprintf(os.Stderr, "watcher: failed to watch dir %s: %v\n", dir, err)
			}
		}

		go func() {
			defer w.Close()
			for {
				select {
				case ev := <-w.Events:
					if ev.Op&(fsnotify.Write|fsnotify.Create|fsnotify.Rename) != 0 &&
						(ev.Name == certFile || ev.Name == keyFile) {
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
	})
}

// Close - stops file watching. Safe to call multiple times.
func (t *TLSConfigBuilder) Close() {
	t.mu.Lock()
	defer t.mu.Unlock()

	select {
	case <-t.done:
		// already closed
	default:
		close(t.done)
	}
	t.watcher = nil
}

// VerifyCertTrusted - checks if a PEM-encoded certificate chain is trusted by the builder's CA pool.
// The first certificate in the chain is the leaf; any remaining certificates are intermediates.
// Verification uses a snapshot of the current CA pool and requires ExtKeyUsageServerAuth.
func (t *TLSConfigBuilder) VerifyCertTrusted(certPEM []byte) error {
	var certs []*x509.Certificate
	rest := certPEM
	for len(rest) > 0 {
		var block *pem.Block
		block, rest = pem.Decode(rest)
		if block == nil {
			break
		}
		if block.Type != "CERTIFICATE" {
			continue
		}
		cert, err := x509.ParseCertificate(block.Bytes)
		if err != nil {
			return fmt.Errorf("parse certificate: %w", err)
		}
		certs = append(certs, cert)
	}

	if len(certs) == 0 {
		return fmt.Errorf("no certificates found")
	}

	leaf := certs[0]
	intermediates := x509.NewCertPool()
	for _, c := range certs[1:] {
		intermediates.AddCert(c)
	}

	t.mu.Lock()
	roots := t.ca.Clone()
	t.mu.Unlock()

	_, err := leaf.Verify(x509.VerifyOptions{
		Roots:         roots,
		Intermediates: intermediates,
		KeyUsages:     []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	})
	if err != nil {
		return fmt.Errorf("certificate not trusted: %w", err)
	}

	return nil
}
