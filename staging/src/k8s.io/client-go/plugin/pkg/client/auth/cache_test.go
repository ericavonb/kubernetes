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
	"testing"
	"time"
)

type mockTokenSource struct {
	t   *Token
	err error
}

func (m mockTokenSource) Token() (*Token, error) {
	return m.t, m.err
}

func TestTokenCache(t *testing.T) {
	const (
		unexpiredTokenString = "mock-unexpired-token-string"
		expiredTokenString   = "mock-expired-token-string"
	)
	var (
		onceWasNow       = time.Date(2018, time.January, 19, 22, 0, 0, 0, time.UTC)
		onceWasTheFuture = time.Date(2018, time.January, 19, 23, 30, 0, 0, time.UTC)
		stillIsThePast   = time.Date(2018, time.January, 19, 21, 30, 0, 0, time.UTC)
		unexpiredToken   = &Token{
			Token:     unexpiredTokenString,
			ExpiresOn: onceWasTheFuture,
		}
		expiredToken = &Token{
			Token:     expiredTokenString,
			ExpiresOn: stillIsThePast,
		}
	)
	timeNow = func() time.Time { return onceWasNow }

	tests := []struct {
		description string

		cachedToken  *Token
		storedToken  *Token
		storageError error

		expectedTokenString string
		expectedError       error
	}{
		{
			description: "returns cached token when not expired",
			cachedToken: unexpiredToken,
			storedToken: expiredToken,

			expectedTokenString: unexpiredTokenString,
		},
		{
			description: "returns new token from storage when expired",
			cachedToken: expiredToken,
			storedToken: unexpiredToken,

			expectedTokenString: unexpiredTokenString,
		},
		{
			description:  "returns errors fetching new token from storage",
			cachedToken:  expiredToken,
			storageError: errors.New("token store error"),

			expectedError: errors.New("could not fetch new token: token store error"),
		},
		{
			description: "returns error when storage returns no token",
			cachedToken: expiredToken,
			storedToken: nil,

			expectedError: errors.New("nil token returned by source"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			store := &TokenStore{
				token:  tt.cachedToken,
				source: mockTokenSource{tt.storedToken, tt.storageError},
			}
			actualTokenStr, err := store.Token()
			if tt.expectedError != nil {
				if err == nil {
					t.Errorf("missing expected error: expected=%v", tt.expectedError)
				}
				if tt.expectedError.Error() != err.Error() {
					t.Errorf("unexpected error: expected=%v actual=%v", tt.expectedError, err)
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if actualTokenStr != tt.expectedTokenString {
				t.Errorf("got unexpected token from cache: expected='%s' actual='%s'",
					tt.expectedTokenString, actualTokenStr)
			}
			refetchedTokenString, err := store.Token()
			if refetchedTokenString != tt.expectedTokenString {
				t.Errorf("unexpected token refetching token: expected='%s' actual='%s'",
					tt.expectedTokenString, refetchedTokenString)
			}
		})
	}

}
