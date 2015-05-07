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
	"strings"
	"time"

	"code.google.com/p/goauth2/oauth"
	"github.com/docker/machine/log"
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

func newGCEService(authTokenPath string) (*compute.Service, error) {
	client, err := newOauthClient(authTokenPath)
	if err != nil {
		return nil, err
	}
	service, err := compute.New(client)
	return service, err
}

func newOauthClient(authTokenPath string) (*http.Client, error) {
	config := &oauth.Config{
		ClientId:     clientID,
		ClientSecret: clientSecret,
		Scope:        compute.ComputeScope,
		AuthURL:      authURL,
		TokenURL:     tokenURL,
	}

	token, err := getToken(authTokenPath, config)
	if err != nil {
		return nil, err
	}

	t := oauth.Transport{
		Token:     token,
		Config:    config,
		Transport: http.DefaultTransport,
	}
	return t.Client(), nil
}

func getToken(tokenPath string, config *oauth.Config) (*oauth.Token, error) {
	token, err := tokenFromCache(tokenPath)
	if err == nil {
		return token, nil
	}

	token, err = tokenFromWeb(config)
	if err != nil {
		return nil, err
	}

	saveToken(tokenPath, token)
	return token, nil
}

func tokenFromCache(tokenPath string) (*oauth.Token, error) {
	f, err := os.Open(tokenPath)
	if err != nil {
		return nil, err
	}
	token := new(oauth.Token)
	err = gob.NewDecoder(f).Decode(token)
	return token, err
}

func tokenFromWeb(config *oauth.Config) (*oauth.Token, error) {
	randState := fmt.Sprintf("st%d", time.Now().UnixNano())

	config.RedirectURL = redirectURI
	authURL := config.AuthCodeURL(randState)

	log.Info("Opening auth URL in browser.")
	log.Info(authURL)
	log.Info("If the URL doesn't open please open it manually and copy the code here.")
	openURL(authURL)
	code := getCodeFromStdin()

	log.Infof("Got code: %s", code)

	t := &oauth.Transport{
		Config:    config,
		Transport: http.DefaultTransport,
	}
	_, err := t.Exchange(code)
	if err != nil {
		return nil, err
	}
	return t.Token, nil
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

func saveToken(tokenPath string, token *oauth.Token) {
	log.Infof("Saving token in %v", tokenPath)
	f, err := os.Create(tokenPath)
	if err != nil {
		log.Infof("Warning: failed to cache oauth token: %v", err)
		return
	}
	defer f.Close()
	gob.NewEncoder(f).Encode(token)
}
