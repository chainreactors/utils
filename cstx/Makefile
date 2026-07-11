CSTX_CORE ?= $(realpath ../../asm/mapping/cstx/cstx-core)
CSTX_REPO ?= chainreactors/cstx
CSTX_VERSION ?= v0.0.1

UNAME_S := $(shell uname -s | tr A-Z a-z)
UNAME_M := $(shell uname -m)
ifeq ($(UNAME_M),x86_64)
  ARCH := amd64
else ifeq ($(UNAME_M),aarch64)
  ARCH := arm64
else ifeq ($(UNAME_M),arm64)
  ARCH := arm64
else
  ARCH := $(UNAME_M)
endif
PLATFORM := $(UNAME_S)_$(ARCH)

.PHONY: lib lib-local test clean download download-all

# Download from GitHub Release (current platform)
download:
	@./scripts/download-ffi.sh $(CSTX_VERSION) $(PLATFORM)

# Download all platforms
download-all:
	@./scripts/download-ffi.sh $(CSTX_VERSION) linux_amd64
	@./scripts/download-ffi.sh $(CSTX_VERSION) linux_arm64
	@./scripts/download-ffi.sh $(CSTX_VERSION) darwin_amd64
	@./scripts/download-ffi.sh $(CSTX_VERSION) darwin_arm64
	@./scripts/download-ffi.sh $(CSTX_VERSION) windows_amd64

# Build from local cstx-core source
lib-local:
	cd $(CSTX_CORE) && cargo build --release -p cstx-ffi
	mkdir -p lib/$(PLATFORM)
	cp $(CSTX_CORE)/target/release/libcstx_ffi.a lib/$(PLATFORM)/

# Alias
lib: download

test:
	CGO_ENABLED=1 go test -tags cstx_native -v ./...

clean:
	rm -rf lib/*/libcstx_ffi.a lib/*/libcstx_ffi.so lib/*/libcstx_ffi.dylib
