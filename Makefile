OS_TYPE=$(shell echo `uname`| tr '[A-Z]' '[a-z]')
ifeq ($(OS_TYPE),darwin)
	OS := osx
else
	OS := linux
endif

# version 11.4 or later of go
HAS_REQUIRED_GO := $(shell go version | grep -E 'go[2-9]|go1.1[2-9]|go1.12.[4-9]')

PACKAGE_NAME = github.com/lyraproj/lyra-operator
LDFLAGS += -X "$(PACKAGE_NAME)/pkg/version.BuildTime=$(shell date -u '+%Y-%m-%d %I:%M:%S %Z')"
LDFLAGS += -X "$(PACKAGE_NAME)/pkg/version.BuildTag=$(shell git describe --all --exact-match `git rev-parse HEAD` | grep tags | sed 's/tags\///')"
LDFLAGS += -X "$(PACKAGE_NAME)/pkg/version.BuildSHA=$(shell git rev-parse --short HEAD)"
BUILDARGS =

PHONY+= default
default: LINTFLAGS = --fast
default: everything

PHONY+= all
all: LDFLAGS += -s -w # Strip debug information
all: TESTFLAGS = --race
all: everything

PHONY+= everything
everything: clean lint test lyra smoke-test

PHONY+= docker-build
docker-build: BUILDARGS += CGO_ENABLED=0 GOOS=linux
docker-build: LDFLAGS += -extldflags "-static"
docker-build: lyra plugins

PHONY+= shrink
shrink:
	for f in build/*; do \
		upx $$f; \
	done;

PHONY+= lyra
lyra:
	@$(call build,bin/lyra,cmd/lyra/main.go)

PHONY+= test
test:
	@echo "🔘 Running unit tests... (`date '+%H:%M:%S'`)"
	$(BUILDARGS) go test $(TESTFLAGS) github.com/lyraproj/lyra-operator/...

PHONY+= clean
clean:
	@echo "🔘 Cleaning build dir..."
	@rm -rf build

PHONY+= lint
lint: $(GOPATH)/bin/golangci-lint
	@$(call checklint,pkg/...)
	@$(call checklint,cmd/lyra/...)

PHONY+= dist-release
dist-release:
	@if [ "$(OS)" != "linux" ]; \
	then \
		echo ""; \
		echo "🔴 dist-release target only supported on linux (Travis CI)"; \
		exit 1; \
	fi

	echo "🔘 Deploying release to Github..."
	for f in build/*; do \
		echo "  - $$f"; \
		tar czf $$f.tar.gz $$f; \
		sha256sum $$f.tar.gz | awk '{ print $1 }' > $$f.tar.gz.sha256 ; \
	done;

PHONY+= smoke-test
smoke-test: lyra

define build
	echo "🔘 Building - $(1) (`date '+%H:%M:%S'`)"
	mkdir -p build/
	$(BUILDARGS) go build -ldflags '$(LDFLAGS)' -o build/$(1) $(2)
	echo "✅ Build complete - $(1) (`date '+%H:%M:%S'`)"
endef

define checklint
	echo "🔘 Linting $(1) (`date '+%H:%M:%S'`)"
	lint=`$(BUILDARGS) golangci-lint run $(LINTFLAGS) $(1)`; \
	if [ "$$lint" != "" ]; \
	then echo "🔴 Lint found"; echo "$$lint"; exit 1;\
	else echo "✅ Lint-free (`date '+%H:%M:%S'`)"; \
	fi
endef

$(GOPATH)/bin/golangci-lint:
	@echo "🔘 Installing golangci-lint... (`date '+%H:%M:%S'`)"
	@GO111MODULE=off go get github.com/golangci/golangci-lint/cmd/golangci-lint

.PHONY: $(PHONY)
