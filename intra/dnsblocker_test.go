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

package intra

import (
	"sync"
	"testing"
)

// mockUIDProvider is a test UIDProvider that maps package names to UIDs.
type mockUIDProvider struct {
	packages map[string]int32
}

func (m *mockUIDProvider) GetUID(protocol int, localAddr, remoteAddr string) int32 {
	return -1 // not used in DNSBlocker tests
}

func (m *mockUIDProvider) GetUIDForPackage(packageName string) int32 {
	uid, ok := m.packages[packageName]
	if !ok {
		return -1
	}
	return uid
}

func newMockProvider() *mockUIDProvider {
	return &mockUIDProvider{
		packages: map[string]int32{
			"com.youtube":   10045,
			"com.tiktok":    10102,
			"com.instagram": 10200,
			"com.snapchat":  10301,
		},
	}
}

func TestSetBlockedApps_FullReplacement(t *testing.T) {
	blocker := NewDNSBlocker()
	provider := newMockProvider()

	// Block YouTube and TikTok
	blocker.SetBlockedApps("com.youtube,com.tiktok", provider)

	if !blocker.IsBlocked(10045) {
		t.Error("YouTube (10045) should be blocked")
	}
	if !blocker.IsBlocked(10102) {
		t.Error("TikTok (10102) should be blocked")
	}
	if blocker.IsBlocked(10200) {
		t.Error("Instagram (10200) should NOT be blocked")
	}

	// Update: block only Instagram
	// YouTube and TikTok should be automatically unblocked
	blocker.SetBlockedApps("com.instagram", provider)

	if blocker.IsBlocked(10045) {
		t.Error("YouTube (10045) should NOT be blocked after update")
	}
	if blocker.IsBlocked(10102) {
		t.Error("TikTok (10102) should NOT be blocked after update")
	}
	if !blocker.IsBlocked(10200) {
		t.Error("Instagram (10200) should be blocked after update")
	}

	// Clear all blocks
	blocker.SetBlockedApps("", provider)

	if blocker.IsBlocked(10200) {
		t.Error("Instagram (10200) should NOT be blocked after clearing")
	}
	if blocker.BlockedCount() != 0 {
		t.Errorf("Expected 0 blocked apps, got %d", blocker.BlockedCount())
	}
}

func TestSetBlockedApps_UnknownPackage(t *testing.T) {
	blocker := NewDNSBlocker()
	provider := newMockProvider()

	// Include a package that doesn't exist
	blocker.SetBlockedApps("com.youtube,com.nonexistent.app", provider)

	if !blocker.IsBlocked(10045) {
		t.Error("YouTube should be blocked")
	}
	if blocker.BlockedCount() != 1 {
		t.Errorf("Expected 1 blocked app (unknown skipped), got %d", blocker.BlockedCount())
	}
}

func TestSetBlockedApps_WhitespaceHandling(t *testing.T) {
	blocker := NewDNSBlocker()
	provider := newMockProvider()

	blocker.SetBlockedApps(" com.youtube , com.tiktok , ", provider)

	if !blocker.IsBlocked(10045) {
		t.Error("YouTube should be blocked despite whitespace")
	}
	if !blocker.IsBlocked(10102) {
		t.Error("TikTok should be blocked despite whitespace")
	}
	if blocker.BlockedCount() != 2 {
		t.Errorf("Expected 2 blocked apps, got %d", blocker.BlockedCount())
	}
}

func TestIsBlocked_ConcurrentAccess(t *testing.T) {
	blocker := NewDNSBlocker()
	provider := newMockProvider()

	blocker.SetBlockedApps("com.youtube", provider)

	var wg sync.WaitGroup
	// Concurrent reads
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			blocker.IsBlocked(10045)
		}()
	}
	// Concurrent writes
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			blocker.SetBlockedApps("com.tiktok", provider)
		}()
	}
	wg.Wait()
}
