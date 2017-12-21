/*
Copyright 2017 The Kubernetes Authors.

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

package auth

import (
	"errors"
	"fmt"
	"sync"
	"time"
)

// alias so tests can use stub for deterministic results
var timeNow = time.Now

// Token groups the token string and expiration for caching
type Token struct {
	Token     string
	ExpiresOn time.Time
}

func (t *Token) isExpired() bool {
	return timeNow().After(t.ExpiresOn)
}

// TokenSource should be implemented by auth providers
type TokenSource interface {
	Token() (*Token, error)
}

// TokenStore is a simple token cache to use with whichever auth provider
type TokenStore struct {
	lock   sync.Mutex
	token  *Token
	source TokenSource
}

func newTokenStore(source TokenSource) *TokenStore {
	return &TokenStore{source: source}
}

// Token returns a token from source, cached and refetched when expired
func (t *TokenStore) Token() (string, error) {
	t.lock.Lock()
	defer t.lock.Unlock()

	// check for cached token
	if t.token != nil && !t.token.isExpired() {
		return t.token.Token, nil
	}
	// fetch a new token
	token, err := t.source.Token()
	if err != nil {
		return "", fmt.Errorf("could not fetch new token: %v", err)
	}
	if token == nil {
		return "", errors.New("nil token returned by source")
	}
	// update cache
	t.token = token
	return token.Token, nil
}
