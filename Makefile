ifndef GOOS
UNAME_S := $(shell uname -s)
ifeq ($(UNAME_S),Darwin)
	GOOS := darwin
else ifeq ($(UNAME_S),Linux)
	GOOS := linux
else
$(error "$$GOOS is not defined. If you are using Windows, try to re-make using 'GOOS=windows make ...' ")
endif
endif

ifndef ARCH
UNAME_M := $(shell uname -m)
ifeq ($(UNAME_M),x86_64)
	ARCH := amd64
else ifeq ($(UNAME_M),i386)
	ARCH := 386
else
$(error "$$ARCH is not defined. If you are using x86_64, try to re-make using 'ARCH=x86_64 make ...' ")
endif
endif

PACKAGES    := $(shell go list ./... | grep -v '/vendor/' | grep -v '/crypto/ed25519/chainkd')

BUILD_FLAGS := -ldflags "-X github.com/doslink/doslink/version.GitCommit=`git rev-parse HEAD`"

CHAIN_NAME := doslink

MINER_BINARY := miner-$(GOOS)_$(ARCH)

SERVER_BINARY := server-$(GOOS)_$(ARCH)

CLIENT_BINARY := client-$(GOOS)_$(ARCH)

VERSION := $(shell awk -F= '/Version =/ {print $$2}' version/version.go | tr -d "\" ")

MINER_RELEASE := miner-$(VERSION)-$(GOOS)_$(ARCH)

SERVER_RELEASE := server-$(VERSION)-$(GOOS)_$(ARCH)

CLIENT_RELEASE := client-$(VERSION)-$(GOOS)_$(ARCH)

CHAIN_RELEASE := $(CHAIN_NAME)-$(VERSION)-$(GOOS)_$(ARCH)

all: test target release-all

server:
	@echo "Building server to cmd/server/server"
	@go build $(BUILD_FLAGS) -o cmd/server/server cmd/server/main.go

client:
	@echo "Building client to cmd/client/client"
	@go build $(BUILD_FLAGS) -o cmd/client/client cmd/client/main.go

target:
	mkdir -p $@

binary: target/$(SERVER_BINARY) target/$(CLIENT_BINARY) target/$(MINER_BINARY)

ifeq ($(GOOS),windows)
release: binary
	cd target && cp -f $(MINER_BINARY) $(MINER_BINARY).exe
	cd target && cp -f $(SERVER_BINARY) $(SERVER_BINARY).exe
	cd target && cp -f $(CLIENT_BINARY) $(CLIENT_BINARY).exe
	cd target && md5sum $(MINER_BINARY).exe $(SERVER_BINARY).exe $(CLIENT_BINARY).exe >$(CHAIN_RELEASE).md5
	cd target && zip $(CHAIN_RELEASE).zip $(MINER_BINARY).exe $(SERVER_BINARY).exe $(CLIENT_BINARY).exe $(CHAIN_RELEASE).md5
	cd target && rm -f $(MINER_BINARY) $(SERVER_BINARY) $(CLIENT_BINARY) $(MINER_BINARY).exe $(SERVER_BINARY).exe $(CLIENT_BINARY).exe $(CHAIN_RELEASE).md5
else
release: binary
	cd target && md5sum $(MINER_BINARY) $(SERVER_BINARY) $(CLIENT_BINARY) >$(CHAIN_RELEASE).md5
	cd target && tar -czf $(CHAIN_RELEASE).tgz $(MINER_BINARY) $(SERVER_BINARY) $(CLIENT_BINARY) $(CHAIN_RELEASE).md5
	cd target && rm -f $(MINER_BINARY) $(SERVER_BINARY) $(CLIENT_BINARY) $(CHAIN_RELEASE).md5
endif

release-all: clean
	GOOS=darwin  ARCH=amd64 make release
	GOOS=darwin  ARCH=386   make release
	GOOS=linux   ARCH=amd64 make release
	GOOS=linux   ARCH=386   make release
	GOOS=windows ARCH=amd64 make release
	GOOS=windows ARCH=386   make release

clean:
	@echo "Cleaning binaries built"
	@rm -rf cmd/server/server
	@rm -rf cmd/client/client
	@rm -rf cmd/miner/miner
	@rm -rf target
	@echo "Cleaning temp test data..."
	@rm -rf test/pseudo_hsm*
	@rm -rf blockchain/pseudohsm/testdata/pseudo/
	@echo "Cleaning sm2 pem files..."
	@rm -rf crypto/sm2/*.pem
	@echo "Done."

target/$(SERVER_BINARY):
	CGO_ENABLED=0 GOARCH=$(ARCH) go build $(BUILD_FLAGS) -o $@ cmd/server/main.go

target/$(CLIENT_BINARY):
	CGO_ENABLED=0 GOARCH=$(ARCH) go build $(BUILD_FLAGS) -o $@ cmd/client/main.go

target/$(MINER_BINARY):
	CGO_ENABLED=0 GOARCH=$(ARCH) go build $(BUILD_FLAGS) -o $@ cmd/miner/main.go

test:
	@echo "====> Running go test"
	@go test -tags "network" $(PACKAGES)

benchmark:
	@go test -bench $(PACKAGES)

functional-tests:
	@go test -v -timeout=5m -tags="functional" ./test 

ci: test functional-tests

.PHONY: all target release-all clean test benchmark