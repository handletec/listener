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

import "crypto/tls"

// TLSClientAuth - TLS client authentication type
type TLSClientAuth tls.ClientAuthType

const (
	// TLSClientAuthNone - there is no need to verify clients
	TLSClientAuthNone TLSClientAuth = TLSClientAuth(tls.NoClientCert)

	// TLSClientAuthRequest - server may request client cert but clients are not obligated to send it
	TLSClientAuthRequest TLSClientAuth = TLSClientAuth(tls.RequestClientCert)

	// TLSClientAuthRequire - clients should send a certificate however the cert does not need to be valid
	TLSClientAuthRequire TLSClientAuth = TLSClientAuth(tls.RequireAnyClientCert)

	// TLSClientAuthVerify - server may request client cert and if client responds, the cert must be valid
	TLSClientAuthVerify TLSClientAuth = TLSClientAuth(tls.VerifyClientCertIfGiven)

	// TLSClientAuthRequireVerify - server requests client cert and the client **MUST** respond with a valid certificate
	TLSClientAuthRequireVerify TLSClientAuth = TLSClientAuth(tls.RequireAndVerifyClientCert)
)

func (tca TLSClientAuth) AuthType() (at tls.ClientAuthType) {
	tcaInt := int(tca)
	if tcaInt < 0 || tcaInt > int(tls.RequireAndVerifyClientCert) {
		tcaInt = 0 // if some other unknown client auth type, set it to 'none'
	}

	return tls.ClientAuthType(tcaInt)
}

func (tca TLSClientAuth) String() (str string) {

	tcaName := []string{"none", "request", "require", "verify", "requireverify"}
	tcaInt := int(tca)

	if tcaInt < 0 || tcaInt >= len(tcaName) {
		tcaInt = 0
	}

	return tcaName[tcaInt]
}
