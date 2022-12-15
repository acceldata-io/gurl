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

import (
	"crypto/tls"
	"errors"
	"fmt"
	"net/http"
	"os"
)

type httpMethod string

const (
	httpGET     httpMethod = "GET"
	httpPOST    httpMethod = "POST"
	httpPATCH   httpMethod = "PATCH"
	httpDELETE  httpMethod = "DELETE"
	httpPUT     httpMethod = "PUT"
	httpOPTIONS httpMethod = "OPTIONS"
	httpHEAD    httpMethod = "HEAD"
)

func (m httpMethod) ToString() (string, error) {
	return methodToString(m)
}

func stringToMethod(reqType string) (httpMethod, error) {
	// Validate the HTTP Method
	switch reqType {
	case "GET":
		return httpGET, nil
	case "POST":
		return httpPOST, nil
	case "PATCH":
		return httpPATCH, nil
	case "DELETE":
		return httpDELETE, nil
	case "PUT":
		return httpPUT, nil
	case "OPTIONS":
		return httpOPTIONS, nil
	case "HEAD":
		return httpHEAD, nil
	default:
		return "", errors.New("invalid http method")
	}
}

func methodToString(reqType httpMethod) (string, error) {
	switch reqType {
	case "DELETE", "GET", "POST", "PATCH", "PUT", "OPTIONS", "HEAD":
		return string(reqType), nil
	default:
		return string(reqType), errors.New("invalid http method")
	}
}

func makeRequest(requestType httpMethod, url string) ([]byte, int, error) {
	//
	clientTransport := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: !enforceTLSVerify},
	}

	// Default HTTP Client
	client := &http.Client{
		Transport: clientTransport,
	}

	// If required
	// Create the HTTP Client for Kerberos
	if isKerberized {
		client = &http.Client{Transport: &spnegoTransport{
			// This technically copies a mutex, but since we've just created
			// the object, we know that this mutex is unlocked.
			Transport: *clientTransport,
		}}
	}

	reqType, err := methodToString(requestType)
	if err != nil {
		return []byte{}, 0, err
	}

	req, err := http.NewRequest(reqType, url, nil)
	if err != nil {
		return []byte{}, 400, fmt.Errorf("cannot build the '%s' request for the URL: '%s'. Because: %w", requestType, url, err)
	}

	if isBasicAuth != "" {
		req.SetBasicAuth(basicAuthUser, basicAuthPassword)
	}

	// Setting the User-Agent to 'curl' will let the server know that the client is not-redirectable to a login page.
	// NOTE: If this user agent is not set, Knox server will redirect the request to the SSO login page.
	if clientUserAgent != "" {
		req.Header.Add("User-Agent", clientUserAgent)
	}

	resp, err := client.Do(req)
	if err != nil {
		return []byte{}, 400, fmt.Errorf("unable to make the '%s' request for the URL: '%s'. Because: %w", reqType, url, err)
	}

	defer resp.Body.Close()

	switch rcode := resp.StatusCode; {
	case rcode <= 299:
		if outputFile != "" {

			if _, err := os.Stat(outputFile); err == nil {
				if err := os.Remove(outputFile); err != nil {
					return []byte{}, 400, fmt.Errorf("unable to tuncate the existing file: '%s'. Because: %w", outputFile, err)
				}
			} else if !os.IsNotExist(err) {
				return []byte{}, 400, fmt.Errorf("unable to access the existing file: '%s'. Because: %w", outputFile, err)
			}

			f, e := os.OpenFile(outputFile, os.O_CREATE|os.O_WRONLY, 0o644)
			if e != nil {
				return []byte{}, 400, fmt.Errorf("unable to create the output file at: '%s'. Because: %w", outputFile, err)
			}
			defer f.Close()

			_, err = f.ReadFrom(resp.Body)
			if err != nil {
				//
				return []byte{}, resp.StatusCode, err
			}
			fmt.Println("INFO: Output redirected the file: '" + outputFile + "'")
		} else {
			body := []byte{}
			_, readErr := resp.Body.Read(body)
			if readErr != nil {
				fmt.Println("ERROR: Unabled to read the response body")
				return []byte{}, resp.StatusCode, readErr
			}
			fmt.Println("HEADERS: ")
			for k, v := range resp.Header {
				fmt.Printf("'%s' : '%s'\n", k, v)
			}
			fmt.Println("BODY: \n", string(body))
		}
	case rcode >= 300:
		fmt.Println("ERROR: Server returned status: ", resp.Status)
		body := []byte{}
		_, readErr := resp.Body.Read(body)
		if readErr != nil {
			fmt.Println("ERROR: Unable to read the response body")
			return []byte{}, resp.StatusCode, readErr
		}
		fmt.Println("HEADERS: ")
		for k, v := range resp.Header {
			fmt.Printf("'%s' : '%s'\n", k, v)
		}
		fmt.Println("BODY: \n", string(body))
	}

	return []byte{}, resp.StatusCode, nil
}
