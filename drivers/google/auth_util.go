// Copyright 2015 The Cockroach Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or
// implied. See the License for the specific language governing
// permissions and limitations under the License. See the AUTHORS file
// for names of contributors.
//
// Author: Marc Berhault (marc@cockroachlabs.com)

// This is a slight modification of: https://github.com/docker/machine/blob/master/drivers/google/auth_util.go
// The main difference is that we have a single path for tokens, whereas docker-machine
// has --google-auth-token and a default store-path.
// Original license follows:

// Copyright 2014 Docker, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or
// implied. See the License for the specific language governing
// permissions and limitations under the License. See the AUTHORS file
// for names of contributors.

package google

import (
	"encoding/gob"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"code.google.com/p/goauth2/oauth" // Deprecated. TODO: replace with golang.org/x/oauth2
	"github.com/cockroachdb/cockroach/util/log"
	compute "google.golang.org/api/compute/v1"
)

// OAuth logic. This initializes a GCE Service with a OAuth token.
// If the token (in Gob format) exists at 'authTokenPath', load it.
// Otherwise, redirect to the Google consent screen to get a code,
// generate a token from it, and save it in 'authTokenPath'.
//
// The token file format must be the same as that used by docker-machine.
const (
	authURL  = "https://accounts.google.com/o/oauth2/auth"
	tokenURL = "https://accounts.google.com/o/oauth2/token"
	// Cockroach client ID and secret.
	// TODO(marc): details show my personal email for now. We should have a more
	// generic user-facing one.
	clientID     = "962032490974-5avmqm15uklkgus98c7f862dk23u5mdk.apps.googleusercontent.com"
	clientSecret = "SSytmGLypTUPnj6a3PeV8LiR"
	redirectURI  = "urn:ietf:wg:oauth:2.0:oob"
)

// gobCache implements oauth.Cache.
// Its value is the full path name to the cache file.
// This is pretty much oauth.CacheFile, but with gob encoding.
type gobCache string

// Token returns the cached token value, or an error if none is found.
func (f gobCache) Token() (*oauth.Token, error) {
	file, err := os.Open(string(f))
	if err != nil {
		return nil, err
	}
	tok := &oauth.Token{}
	if err = gob.NewDecoder(file).Decode(tok); err != nil {
		return nil, err
	}
	return tok, file.Close()
}

// PutToken stores the given token in the cache.
// TODO(marc): we should write to a tmp file and rename in case we error out.
func (f gobCache) PutToken(tok *oauth.Token) error {
	filename := string(f)
	// Create the parent directory if necessary.
	parent := filepath.Dir(filename)
	err := os.MkdirAll(parent, 0700)
	if err != nil {
		return err
	}

	file, err := os.OpenFile(filename, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	encErr := gob.NewEncoder(file).Encode(tok)
	clErr := file.Close()

	if encErr != nil {
		return encErr
	}

	if clErr != nil {
		return clErr
	}

	return nil
}

func newOauthClient(authTokenPath string) (*http.Client, error) {
	config := &oauth.Config{
		ClientId:     clientID,
		ClientSecret: clientSecret,
		Scope:        compute.ComputeScope,
		AuthURL:      authURL,
		TokenURL:     tokenURL,
		RedirectURL:  redirectURI,
		TokenCache:   gobCache(authTokenPath),
		// Needed for refresh tokens:
		AccessType:     "offline",
		ApprovalPrompt: "force",
	}

	transport := &oauth.Transport{
		Config:    config,
		Transport: http.DefaultTransport,
	}

	err := initTransport(transport)
	if err != nil {
		return nil, err
	}
	return transport.Client(), nil
}

func initTransport(transport *oauth.Transport) error {
	// First: check the cache.
	if token, err := transport.Config.TokenCache.Token(); err == nil {
		// We have a token, refresh it. The lifetime is 1h, so we always
		// refresh to ensure lengthy commands do not time out.
		transport.Token = token
		err := transport.Refresh()
		if err == nil {
			return nil
		}
		log.Infof("token refresh failed, requesting new one")
	}

	// Get a new token. Pops up a browser window (hopefully).
	randState := fmt.Sprintf("st%d", time.Now().UnixNano())
	authURL := transport.Config.AuthCodeURL(randState)
	log.Infof("Opening auth URL in browser: %s", authURL)
	log.Infof("If the URL doesn't open please open it manually and copy the code here.")
	openURL(authURL)
	code := getCodeFromStdin()

	_, err := transport.Exchange(code)
	if err != nil {
		log.Infof("problem exchanging code: %v", err)
		return err
	}

	return nil
}

func getCodeFromStdin() string {
	fmt.Print("Enter code: ")
	var code string
	fmt.Scanln(&code)
	return strings.Trim(code, "\n")
}

func openURL(url string) {
	try := []string{"xdg-open", "google-chrome", "open"}
	for _, bin := range try {
		err := exec.Command(bin, url).Run()
		if err == nil {
			return
		}
	}
}
