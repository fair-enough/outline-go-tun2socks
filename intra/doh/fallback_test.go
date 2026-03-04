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
	"bytes"
	"errors"
	"testing"
)

// mockTransport is a simple Transport for testing fallback behavior.
type mockTransport struct {
	url      string
	response []byte
	err      error
	called   bool
}

func (t *mockTransport) Query(q []byte) ([]byte, error) {
	t.called = true
	if t.err != nil {
		return nil, t.err
	}
	return t.response, nil
}

func (t *mockTransport) GetURL() string {
	return t.url
}

// Test that the fallback is not used when the primary succeeds.
func TestFallbackPrimarySucceeds(t *testing.T) {
	primary := &mockTransport{
		url:      "https://primary.example.com/dns-query",
		response: []byte{0xbe, 0xef, 1, 2, 3},
	}
	fallback := &mockTransport{
		url:      "https://fallback.example.com/dns-query",
		response: []byte{0xde, 0xad, 4, 5, 6},
	}

	ft := NewFallbackTransport(primary, fallback)
	query := []byte{0xbe, 0xef, 1, 0, 0, 1, 0, 0, 0, 0, 0, 0}
	resp, err := ft.Query(query)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if !bytes.Equal(resp, primary.response) {
		t.Errorf("Expected primary response %v, got %v", primary.response, resp)
	}
	if !primary.called {
		t.Error("Primary was not called")
	}
	if fallback.called {
		t.Error("Fallback should not have been called when primary succeeds")
	}
}

// Test that the fallback is used when the primary fails.
func TestFallbackPrimaryFails(t *testing.T) {
	primary := &mockTransport{
		url: "https://primary.example.com/dns-query",
		err: errors.New("primary server unavailable"),
	}
	fallback := &mockTransport{
		url:      "https://fallback.example.com/dns-query",
		response: []byte{0xde, 0xad, 4, 5, 6},
	}

	ft := NewFallbackTransport(primary, fallback)
	query := []byte{0xbe, 0xef, 1, 0, 0, 1, 0, 0, 0, 0, 0, 0}
	resp, err := ft.Query(query)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if !bytes.Equal(resp, fallback.response) {
		t.Errorf("Expected fallback response %v, got %v", fallback.response, resp)
	}
	if !primary.called {
		t.Error("Primary was not called")
	}
	if !fallback.called {
		t.Error("Fallback should have been called after primary failure")
	}
}

// Test that an error is returned when both primary and fallback fail.
func TestFallbackBothFail(t *testing.T) {
	primary := &mockTransport{
		url: "https://primary.example.com/dns-query",
		err: errors.New("primary server unavailable"),
	}
	fallback := &mockTransport{
		url: "https://fallback.example.com/dns-query",
		err: errors.New("fallback server unavailable"),
	}

	ft := NewFallbackTransport(primary, fallback)
	query := []byte{0xbe, 0xef, 1, 0, 0, 1, 0, 0, 0, 0, 0, 0}
	_, err := ft.Query(query)
	if err == nil {
		t.Fatal("Expected error when both transports fail")
	}
	if !primary.called {
		t.Error("Primary was not called")
	}
	if !fallback.called {
		t.Error("Fallback should have been called after primary failure")
	}
}

// Test that GetURL returns the primary URL.
func TestFallbackGetURL(t *testing.T) {
	primary := &mockTransport{url: "https://primary.example.com/dns-query"}
	fallback := &mockTransport{url: "https://fallback.example.com/dns-query"}

	ft := NewFallbackTransport(primary, fallback)
	if ft.GetURL() != primary.url {
		t.Errorf("Expected GetURL() = %q, got %q", primary.url, ft.GetURL())
	}
}

// Test that the fallback receives the original query bytes, even if the primary mutates them.
func TestFallbackQueryNotMutated(t *testing.T) {
	originalQuery := []byte{0xbe, 0xef, 1, 0, 0, 1, 0, 0, 0, 0, 0, 0}

	primary := &mockTransport{
		url: "https://primary.example.com/dns-query",
		err: errors.New("fail"),
	}
	var fallbackReceivedQuery []byte
	fallback := &mockTransport{
		url:      "https://fallback.example.com/dns-query",
		response: []byte{0, 0, 1, 2, 3},
	}
	// Override Query to capture what fallback actually receives
	ft := &fallbackTransport{primary: primary, fallback: &queryCapture{
		Transport:     fallback,
		capturedQuery: &fallbackReceivedQuery,
	}}

	query := make([]byte, len(originalQuery))
	copy(query, originalQuery)
	ft.Query(query)

	if !bytes.Equal(fallbackReceivedQuery, originalQuery) {
		t.Errorf("Fallback received mutated query.\nExpected: %v\nGot:      %v", originalQuery, fallbackReceivedQuery)
	}
}

// queryCapture wraps a Transport to capture the query bytes it receives.
type queryCapture struct {
	Transport
	capturedQuery *[]byte
}

func (c *queryCapture) Query(q []byte) ([]byte, error) {
	captured := make([]byte, len(q))
	copy(captured, q)
	*c.capturedQuery = captured
	return c.Transport.Query(q)
}
