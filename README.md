# outline-go-tun2socks

> [!WARNING]  
> This repository is no longer being maintained. The tun2socks source is now maintained in the [outline-apps repository](https://github.com/Jigsaw-Code/outline-apps/tree/master) and [Intra repository](https://github.com/Jigsaw-Code/Intra/tree/master/Android/app/src/go).

Go package for building [go-tun2socks](https://github.com/eycorsican/go-tun2socks)-based clients for [Outline](https://getoutline.org) and [Intra](https://getintra.org) (now with support for [Choir](https://github.com/Jigsaw-Code/choir) metrics).  For macOS, iOS, and Android, the output is a library; for Linux and Windows it is a command-line executable.

## Prerequisites

- macOS host (iOS, macOS)
- make
- Go >= 1.18
- A C compiler (e.g.: clang, gcc)

## Android

### Set up

- [sdkmanager](https://developer.android.com/studio/command-line/sdkmanager)
  1. Download the command line tools from https://developer.android.com/studio.
  1. Unzip the pacakge as `~/Android/Sdk/cmdline-tools/latest/`. Make sure `sdkmanager` is located at `~/Android/Sdk/cmdline-tools/latest/bin/sdkmanager`
- Android NDK 23
  1. Install the NDK with `~/Android/Sdk/cmdline-tools/latest/bin/sdkmanager "platforms;android-30" "ndk;23.1.7779620"` (platform from [outline-client](https://github.com/Jigsaw-Code/outline-client#building-the-android-app), exact NDK 23 version obtained from `sdkmanager --list`)
  1. Set up the environment variables:
     ```
     export ANDROID_NDK_HOME=~/Android/Sdk/ndk/23.1.7779620 ANDROID_HOME=~/Android/Sdk
     ```
- [gomobile](https://pkg.go.dev/golang.org/x/mobile/cmd/gobind) (installed as needed by `make`)

### Build

```bash
make clean && make android
```
This will create `build/android/{tun2socks.aar,tun2socks-sources.jar}`

If needed, you can extract the jni files into `build/android/jni` with:
```bash
unzip build/android/tun2socks.aar 'jni/*' -d build/android
```

## Apple (iOS and macOS)

### Set up

- Xcode
- [gomobile](https://pkg.go.dev/golang.org/x/mobile/cmd/gobind) (installed as needed by `make`)


### Build
```
make clean && make apple
```
This will create `build/apple/Tun2socks.xcframework`.

## Linux and Windows

We build binaries for Linux and Windows from source without any custom integrations. `xgo` and Docker are required to support cross-compilation.

### Set up

- [Docker](https://docs.docker.com/get-docker/) (for xgo)
- [xgo](https://github.com/crazy-max/xgo) (installed as needed by `make`)
- [ghcr.io/crazy-max/xgo Docker image](https://github.com/crazy-max/xgo/pkgs/container/xgo). This is pulled automatically by xgo and takes ~6.8 GB of disk space.

## Build

For Linux:
```
make clean && make linux
```
This will create `build/linux/tun2socks`.

For Windows:
```
make clean && make windows
```
This will create `build/windows/tun2socks.exe`.

## Intra (Android)

Same set up as for the Outline Android library.

Build with:

```bash
make clean && make intra
```
This will create `build/intra/{tun2socks.aar,tun2socks-sources.jar}`

## Per-App DNS Blocking (Intra / Android)

This fork adds **per-app DNS blocking**, allowing a managing component (e.g. a parental-control or digital-wellbeing app) to selectively block DNS resolution for specific Android applications. When an app is blocked, all its DNS queries — both TCP and UDP — are intercepted inside the tunnel and answered with `SERVFAIL`, effectively preventing the app from reaching the network while leaving all other apps unaffected.

### How It Works

```
┌──────────────┐
│  Android App │
│ (e.g. TikTok)│
└──────┬───────┘
       │ DNS query (UDP :53 or TCP :53)
       ▼
┌──────────────────────────────────┐
│         TUN device               │
└──────────────┬───────────────────┘
               ▼
┌──────────────────────────────────┐
│  intraPacketProxy (UDP)          │
│  intraStreamDialer (TCP)         │
│                                  │
│  1. Is destination the fake DNS? │
│  2. UIDProvider → get caller UID │
│  3. DNSBlocker.IsBlocked(uid)?   │
│     YES → return SERVFAIL / error│
│     NO  → forward to DoH server │
└──────────────────────────────────┘
```

1. **`UIDProvider`** (interface, implemented in Kotlin) — resolves which Android app owns a given network socket and maps package names to UIDs.
   - `GetUID(protocol, localAddr, remoteAddr)` — returns the UID of the socket owner (uses `ConnectivityManager.getConnectionOwnerUid()`, API 29+).
   - `GetUIDForPackage(packageName)` — resolves a package name to its UID (uses `PackageManager.getPackageUid()`).

2. **`DNSBlocker`** — a thread-safe, in-memory blocklist keyed by UID.
   - `SetBlockedApps(apps, provider)` — accepts a **comma-separated** list of package names (e.g. `"com.youtube,com.tiktok"`). Each call **fully replaces** the previous blocklist; apps not in the new list are automatically unblocked. An empty string clears the blocklist entirely.
   - `IsBlocked(uid)` — O(1) map lookup, called on every DNS packet.

3. **Interception points** — both the TCP stream dialer and the UDP packet proxy check the blocklist before forwarding DNS queries:
   - **UDP** (`packet_proxy.go`): returns a `SERVFAIL` DNS response via `doh.Servfail()`.
   - **TCP** (`stream_dialer.go`): returns an error, refusing the connection.

### API

#### Go side

The `Tunnel` type exposes a single method:

```go
// SetBlockedApps replaces the entire DNS blocklist.
// apps: comma-separated package names, e.g. "com.youtube,com.tiktok"
// An empty string unblocks all apps.
tunnel.SetBlockedApps(apps string)
```

`ConnectIntraTunnel` now requires a `UIDProvider` parameter:

```go
func ConnectIntraTunnel(
    fd int, fakedns string, dohdns doh.Transport,
    protector protect.Protector, eventListener intra.Listener,
    uidProvider intra.UIDProvider,       // ← new
) (*intra.Tunnel, error)
```

#### Kotlin / Android side

Implement the `UIDProvider` interface and pass it when connecting the tunnel:

```kotlin
class AppUIDProvider(private val context: Context) : UIDProvider {
    override fun getUID(protocol: Int, localAddr: String, remoteAddr: String): Int {
        val cm = context.getSystemService(ConnectivityManager::class.java)
        // parse localAddr / remoteAddr into InetSocketAddress, then:
        return cm.getConnectionOwnerUid(protocol, local, remote)
    }

    override fun getUIDForPackage(packageName: String): Int {
        return try {
            context.packageManager.getPackageUid(packageName, 0)
        } catch (_: PackageManager.NameNotFoundException) {
            -1
        }
    }
}
```

Then call `SetBlockedApps` at any time to update the blocklist:

```kotlin
tunnel.setBlockedApps("com.youtube,com.tiktok")  // block these apps
tunnel.setBlockedApps("")                         // unblock all
```

### Key Files

| File | Purpose |
|---|---|
| `intra/uidprovider.go` | `UIDProvider` interface definition |
| `intra/dnsblocker.go` | `DNSBlocker` blocklist manager |
| `intra/dnsblocker_test.go` | Unit tests (replacement semantics, concurrency, whitespace handling) |
| `intra/tunnel.go` | `Tunnel.SetBlockedApps()` — public entry point |
| `intra/stream_dialer.go` | TCP DNS blocking interception |
| `intra/packet_proxy.go` | UDP DNS blocking interception |
| `intra/android/tun2socks.go` | `ConnectIntraTunnel` entry point (accepts `UIDProvider`) |

### Design Notes

- **Full replacement semantics** — every `SetBlockedApps` call atomically swaps the entire blocklist; there is no additive/subtractive API. This keeps the contract simple and avoids stale state.
- **Thread safety** — `DNSBlocker` uses `sync.RWMutex`; reads (`IsBlocked`) take a read lock, writes (`SetBlockedApps`) take a write lock.
- **Graceful degradation** — if `UIDProvider` or `DNSBlocker` is `nil`, blocking is silently skipped and the tunnel behaves as before.
- **Unknown packages** — packages that cannot be resolved to a UID are silently skipped (logged at WARN level).
