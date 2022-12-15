// Acceldata Inc. and its affiliates.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// 	Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import "net/http"

// spnegoTransport extends the native http.Transport to provide SPNEGO communication
type spnegoTransport struct {
	http.Transport
	spnego Provider
}

// Error is used to distinguish errors from underlying libraries (gokrb5 or sspi).
type Error struct {
	Err error
}

// Error implements the error interface
func (e *Error) Error() string {
	return e.Err.Error()
}

// RoundTrip implements the RoundTripper interface.
func (t *spnegoTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if t.spnego == nil {
		t.spnego = New()
	}

	if err := t.spnego.SetSPNEGOHeader(req); err != nil {
		return nil, &Error{Err: err}
	}

	return t.Transport.RoundTrip(req)
	// ToDo: process negotiate token from response
}
