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

// UIDProvider resolves app ownership of network connections.
// This interface must be implemented on the Android/Kotlin side.
//
// GetUID should use ConnectivityManager.getConnectionOwnerUid() (API 29+).
// GetUIDForPackage should use PackageManager.getPackageUid().
type UIDProvider interface {
	// GetUID returns the UID of the app that owns a network socket.
	// protocol is the IP protocol number: 6 for TCP, 17 for UDP.
	// localAddr and remoteAddr are in "host:port" format.
	// Returns -1 if the UID cannot be determined.
	GetUID(protocol int, localAddr, remoteAddr string) int32

	// GetUIDForPackage returns the UID for a given package name.
	// Returns -1 if the package is not installed.
	GetUIDForPackage(packageName string) int32
}
