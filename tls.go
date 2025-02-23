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
	"strings"
)

// TLS - configure TLS information
type TLS struct {
	clientCAs         *x509.CertPool
	certFile, keyFile string
	at                TLSClientAuth
}

// NewTLS - create new instance of TLS
func NewTLS(useOSCA bool) (t *TLS, err error) {
	t = new(TLS)
	t.at = TLSClientAuthNone // default client auth type is none

	if useOSCA {
		t.clientCAs, err = x509.SystemCertPool()
		if nil != err {
			return nil, fmt.Errorf("new tls: %w", err)
		}

		if nil == t.clientCAs {
			t.clientCAs = x509.NewCertPool() // if the system CA pool is empty, create a new empty instance
		}
	} else {
		t.clientCAs = x509.NewCertPool() // create a new instance of cert pool
	}

	return
}

// AddCADir - adds a certificate authority from directory
func (t *TLS) AddCADir(cadirname string) (err error) {

	if len(cadirname) == 0 {
		return // if an empty CA dirname is given, return immediately
	}

	cadir, err := os.Open(cadirname)
	if nil != err {
		return fmt.Errorf("add CA directory: error opening CA directory '%s' -> %w", cadirname, err)
	}

	files, err := cadir.Readdir(-1)
	if nil != err {
		return fmt.Errorf("add CA directory: error reading CA directory '%s' -> %w", cadirname, err)
	}

	for _, file := range files {

		if file.Mode().IsRegular() {

			switch strings.ToLower(filepath.Ext(file.Name())) {
			case ".crt", ".pem": // only these 2 files are supporteds
				err = t.AddCAFile(filepath.Join(cadirname, file.Name()))
				if nil != err {
					return fmt.Errorf("add CA directory: reading contents of client CA file '%s' error -> %w", file, err)
				}
			default:
				// do nothing
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
	if !t.clientCAs.AppendCertsFromPEM(cabytes) {
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

// GetTLSconfig - returns istance of tls.Config for listeners
func (t *TLS) GetTLSconfig() (tlsConfig *tls.Config) {

	// if both the cert and key file are given, only then we specify the function to read them for every request
	if len(t.certFile) != 0 && len(t.keyFile) != 0 {
		tlsConfig = &tls.Config{
			ClientCAs:  t.clientCAs,      // server uses this CA Pool to verify client certificates
			MinVersion: tls.VersionTLS12, // this is the minimum version of TLS we support, anything else is discarded
			ClientAuth: t.at.AuthType(),
			GetCertificate: func(*tls.ClientHelloInfo) (*tls.Certificate, error) {
				cert, err := tls.LoadX509KeyPair(t.certFile, t.keyFile)
				if nil != err {
					return nil, fmt.Errorf("get tls certificate: %w", err)
				}
				return &cert, nil
			},
		}
	}

	return tlsConfig
}
