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
	"strings"
	"sync"

	"github.com/eycorsican/go-tun2socks/common/log"
)

// DNSBlocker manages a set of blocked app UIDs for DNS filtering.
// It is safe for concurrent use.
//
// Each call to SetBlockedApps fully replaces the blocked set:
// only apps in the most recent call are blocked; all previously
// blocked apps that are not in the new list are unblocked.
type DNSBlocker struct {
	mu          sync.RWMutex
	blockedUIDs map[int32]string // UID -> package name (for logging)
}

// NewDNSBlocker creates a new DNSBlocker with an empty blocklist.
func NewDNSBlocker() *DNSBlocker {
	return &DNSBlocker{
		blockedUIDs: make(map[int32]string),
	}
}

// SetBlockedApps replaces the entire blocklist with the given apps.
// apps is a comma-separated list of package names (e.g. "com.youtube,com.tiktok").
// An empty string unblocks all apps.
// provider is used to resolve each package name to its UID.
// Packages that cannot be resolved (UID == -1) are silently skipped.
func (b *DNSBlocker) SetBlockedApps(apps string, provider UIDProvider) {
	newBlocked := make(map[int32]string)

	if apps != "" {
		for _, pkg := range strings.Split(apps, ",") {
			pkg = strings.TrimSpace(pkg)
			if pkg == "" {
				continue
			}
			uid := provider.GetUIDForPackage(pkg)
			if uid < 0 {
				log.Warnf("DNSBlocker: cannot resolve package %q to UID, skipping", pkg)
				continue
			}
			newBlocked[uid] = pkg
			log.Infof("DNSBlocker: blocking DNS for %s (UID %d)", pkg, uid)
		}
	}

	b.mu.Lock()
	b.blockedUIDs = newBlocked
	b.mu.Unlock()

	log.Infof("DNSBlocker: blocklist updated, %d app(s) blocked", len(newBlocked))
}

// IsBlocked returns true if the given UID is in the current blocklist.
func (b *DNSBlocker) IsBlocked(uid int32) bool {
	b.mu.RLock()
	defer b.mu.RUnlock()
	_, blocked := b.blockedUIDs[uid]
	return blocked
}

// BlockedCount returns the number of currently blocked apps.
func (b *DNSBlocker) BlockedCount() int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return len(b.blockedUIDs)
}
