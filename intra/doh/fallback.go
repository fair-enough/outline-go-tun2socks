// Copyright 2024 The Outline Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package doh

import (
	"github.com/eycorsican/go-tun2socks/common/log"
)

// fallbackTransport wraps a primary and fallback Transport.
// It always tries the primary first; if the primary query fails,
// it retries the same query on the fallback transport.
type fallbackTransport struct {
	primary  Transport
	fallback Transport
}

// NewFallbackTransport creates a Transport that tries `primary` first,
// and falls back to `fallback` on any error.
// Both transports must be non-nil.
func NewFallbackTransport(primary, fallback Transport) Transport {
	return &fallbackTransport{
		primary:  primary,
		fallback: fallback,
	}
}

func (t *fallbackTransport) Query(q []byte) ([]byte, error) {
	// Make a copy of the query for the fallback attempt, since the primary
	// transport may modify the slice in-place (e.g. zeroing the query ID).
	qCopy := make([]byte, len(q))
	copy(qCopy, q)

	resp, err := t.primary.Query(q)
	if err == nil {
		return resp, nil
	}

	log.Infof("Primary DoH server (%s) failed: %v, trying fallback (%s)", t.primary.GetURL(), err, t.fallback.GetURL())
	return t.fallback.Query(qCopy)
}

func (t *fallbackTransport) GetURL() string {
	return t.primary.GetURL()
}
