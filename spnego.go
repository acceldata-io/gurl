//go:build !windows
// +build !windows

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

// Unix implementation

package main

import (
	"net"
	"net/http"
	"os"
	"os/user"
	"strings"

	"github.com/jcmturner/gokrb5/v8/client"
	"github.com/jcmturner/gokrb5/v8/config"
	"github.com/jcmturner/gokrb5/v8/credentials"
	"github.com/jcmturner/gokrb5/v8/spnego"
)

// Provider is the interface that wraps OS agnostic functions for handling SPNEGO communication
type Provider interface {
	SetSPNEGOHeader(*http.Request) error
}

type krb5 struct {
	cfg *config.Config
	cl  client.Client
}

// New constructs OS specific implementation of spnego.Provider interface
func New() Provider {
	return &krb5{}
}

func (k *krb5) makeCfg() error {
	if k.cfg != nil {
		return nil
	}

	cfgPath := os.Getenv("KRB5_CONFIG")
	if _, err := os.Stat(cfgPath); os.IsNotExist(err) {
		// TODO: Macs and Windows have different path
		cfgPath = defaultKRBConfig
		if _, err := os.Stat(defaultKRBConfig); err != nil {
			// TODO: Need handle if the secondary config is also not found
			if _, err := os.Stat(secondaryKRBConfig); err == nil {
				cfgPath = secondaryKRBConfig
			}
		}
	}

	cfg, err := config.Load(cfgPath)
	if err != nil {
		return err
	}

	k.cfg = cfg
	return nil
}

func (k *krb5) makeClient() error {
	u, err := user.Current()
	if err != nil {
		return err
	}

	ccpath := "/tmp/krb5cc_" + u.Uid

	ccname := os.Getenv("KRB5CCNAME")
	if strings.HasPrefix(ccname, "FILE:") {
		ccpath = strings.SplitN(ccname, ":", 2)[1]
	}

	ccache, err := credentials.LoadCCache(ccpath)
	if err != nil {
		return err
	}

	client.DisablePAFXFAST(true)

	// create the client from the loaded cache
	cl, err := client.NewFromCCache(ccache, k.cfg)
	if err != nil {
		return err
	}

	//
	// TODO: Instead create a client with keytab & password to avoid the kinit
	// client.NewWithKeytab(username string, realm string, kt *keytab.Keytab, krb5conf *config.Config, settings ...func(*Settings))

	//
	k.cl = *cl
	return nil
}

func (k *krb5) SetSPNEGOHeader(req *http.Request) error {
	h, err := canonicalizeHostname(req.URL.Hostname())
	if err != nil {
		return err
	}

	if err := k.makeCfg(); err != nil {
		return err
	}

	if err := k.makeClient(); err != nil {
		return err
	}

	err = spnego.SetSPNEGOHeader(&k.cl, req, "HTTP/"+h)
	if err != nil {
		return err
	}

	return err
}

func canonicalizeHostname(hostname string) (string, error) {
	addrs, err := net.LookupHost(hostname)
	if err != nil {
		return "", err
	}
	if len(addrs) < 1 {
		return hostname, nil
	}

	names, err := net.LookupAddr(addrs[0])
	if err != nil {
		return "", err
	}
	if len(names) < 1 {
		return hostname, nil
	}

	return strings.TrimRight(names[0], "."), nil
}
