GOMOBILE=gomobile
GOBIND=$(GOMOBILE) bind
XGOCMD=xgo
BUILDDIR=$(shell pwd)/build
IMPORT_PATH=github.com/Jigsaw-Code/outline-go-tun2socks
ELECTRON_PATH=$(IMPORT_PATH)/outline/electron
LDFLAGS='-s -w'
ANDROID_LDFLAGS='-w -extldflags=-Wl,-z,max-page-size=16384' # Don't strip Android debug symbols so we can upload them to crash reporting tools + 16KB page alignment
TUN2SOCKS_VERSION=v1.16.11
XGO_LDFLAGS='-s -w -X main.version=$(TUN2SOCKS_VERSION)'
ELECTRON_PKG=outline/electron


LINUX_BUILDDIR=$(BUILDDIR)/linux

ANDROID_BUILD_CMD=$(GOBIND) -androidapi=21 -a -ldflags $(ANDROID_LDFLAGS) -target=android -tags android -work -o $(ANDROID_ARTIFACT)
ANDROID_OUTLINE_BUILD_CMD="$(ANDROID_BUILD_CMD) $(IMPORT_PATH)/outline/android $(IMPORT_PATH)/outline/shadowsocks"
ANDROID_INTRA_BUILD_CMD="$(ANDROID_BUILD_CMD) $(IMPORT_PATH)/intra $(IMPORT_PATH)/intra/android $(IMPORT_PATH)/intra/doh $(IMPORT_PATH)/intra/split $(IMPORT_PATH)/intra/protect"
IOS_BUILD_CMD="$(GOBIND) -a -ldflags $(LDFLAGS) -bundleid org.outline.tun2socks -target=ios/arm64 -tags ios -o $(IOS_ARTIFACT) $(IMPORT_PATH)/outline/apple $(IMPORT_PATH)/outline/shadowsocks"
MACOS_BUILD_CMD="./tools/$(GOBIND) -a -ldflags $(LDFLAGS) -bundleid org.outline.tun2socks -target=ios/amd64 -tags ios -o $(MACOS_ARTIFACT) $(IMPORT_PATH)/outline/apple $(IMPORT_PATH)/outline/shadowsocks"
WINDOWS_BUILD_CMD="$(XGOCMD) -ldflags $(XGO_LDFLAGS) --targets=windows/386 -dest $(WINDOWS_BUILDDIR) $(ELECTRON_PATH)"
LINUX_BUILD_CMD="$(XGOCMD) -ldflags $(XGO_LDFLAGS) --targets=linux/amd64 -dest $(LINUX_BUILDDIR) $(ELECTRON_PATH)"

$(LINUX_BUILDDIR)/tun2socks: $(XGO)
	mkdir -p "$(LINUX_BUILDDIR)/$(IMPORT_PATH)"
	$(XGO) -ldflags $(XGO_LDFLAGS) --targets=linux/amd64 -dest "$(LINUX_BUILDDIR)" -pkg $(ELECTRON_PKG) .
	mv "$(LINUX_BUILDDIR)/$(IMPORT_PATH)-linux-amd64" "$@"
	rm -r "$(LINUX_BUILDDIR)/$(IMPORT_HOST)"


WINDOWS_BUILDDIR=$(BUILDDIR)/windows

windows: $(WINDOWS_BUILDDIR)/tun2socks.exe

$(WINDOWS_BUILDDIR)/tun2socks.exe: $(XGO)
	mkdir -p "$(WINDOWS_BUILDDIR)/$(IMPORT_PATH)"
	$(XGO) -ldflags $(XGO_LDFLAGS) --targets=windows/386 -dest "$(WINDOWS_BUILDDIR)" -pkg $(ELECTRON_PKG) .
	mv "$(WINDOWS_BUILDDIR)/$(IMPORT_PATH)-windows-386.exe" "$@"
	rm -r "$(WINDOWS_BUILDDIR)/$(IMPORT_HOST)"


$(GOMOBILE): go.mod
	env GOBIN="$(GOBIN)" go install golang.org/x/mobile/cmd/gomobile
	env GOBIN="$(GOBIN)" $(GOMOBILE) init

$(XGO): go.mod
	env GOBIN="$(GOBIN)" go install github.com/crazy-max/xgo

go.mod: tools.go
	go mod tidy
	touch go.mod

clean:
	rm -rf "$(BUILDDIR)"
	go clean

clean-all: clean
	rm -rf "$(GOBIN)"
