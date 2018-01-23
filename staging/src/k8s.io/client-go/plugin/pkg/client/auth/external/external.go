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

package external

import (
	"errors"
	"fmt"

	"github.com/golang/glog"

	"k8s.io/apimachinery/pkg/util/net"
	"k8s.io/client-go/plugin/pkg/client/auth"
	restclient "k8s.io/client-go/rest"
)

func init() {
	if err := resetclient.RegisterAuthProviderPlugin("external", newExternalAuthProvider); err != nil {
		glog.Fatalf("Failed to register external auth provider plugin: %v", err)
	}
}

func newExternalAuthProvider(_ string, cfg map[string]string, persister restclient.AuthProviderConfigPersister) (restclient.AuthProvider, error) {
	return &externalAuthProvider{}, nil
}

type externalAuthProvider struct {
	tokenSource auth.TokenSource
}

var _ restclient.AuthProvider = &externalAuthProvider{}

func (e *externalAuthProvider) WrapTransport(rt http.RoundTripper) http.RoundTripper {
	return &externalAuthRoundTripper{
		tokenSource:  e.tokenSource,
		roundTripper: rt,
	}
}
func (e *externalAuthProvider) Login() error {
	return errors.New("not implemented")
}

type externalAuthRoundTripper struct {
	tokenSource  auth.TokenSource
	roundTripper http.RoundTripper
}

var _ net.RoundTripperWrapper = &externalAuthRoundTripper{}

func (r *externalAuthRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	if len(req.Header.Get("Authorization")) != 0 {
		return r.roundTripper.RoundTrip(req)
	}

	token, err := r.tokenSource.Token()
	if err != nil {
		glog.Errorf("Failed to acquire a token: %v", err)
		return nil, fmt.Errorf("acquiring a token for authorization header: %v", err)
	}

	// clone Header to avoid modifying on original request, as per RoundTripper contract
	req2 := new(http.Request)
	*req2 = *req
	req2.Header = make(http.Header, len(req.Header))
	for k, s := range req.Header {
		req2.Header[k] = append([]string(nil), s...) // copy slice
	}

	req2.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token.Token))

	return r.roundTripper.RoundTrip(req2)
}

func (r *externalAuthRoundTripper) WrappedRoundTripper() http.RoundTripper { return r.roundTripper }
