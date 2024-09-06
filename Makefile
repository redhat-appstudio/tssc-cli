APP = rhtap-cli

BIN_DIR ?= ./bin
BIN ?= $(BIN_DIR)/$(APP)

# Primary source code directories.
CMD ?= ./cmd/...
PKG ?= ./pkg/...

# Golang general flags for build and testing.
GOFLAGS ?= -v
GOFLAGS_TEST ?= -failfast -v -cover
CGO_ENABLED ?= 0
CGO_LDFLAGS ?= 

# GitHub action current ref name, provided by the action context environment
# variables, and credentials needed to push the release.
GITHUB_REF_NAME ?= ${GITHUB_REF_NAME:-}
GITHUB_TOKEN ?= ${GITHUB_TOKEN:-}

# Coordinates for the container image build.
IMAGE_REPO ?= ghcr.io/redhat-appstudio/rhtap-cli
IMAGE_TAG ?= latest

# Directory with the installer resources, scripts, Helm Charts, etc.
INSTALLER_DIR ?= ./installer
# Tarball with the installer resources.
INSTALLER_TARBALL ?= $(INSTALLER_DIR)/installer.tar
# Data to include in the tarball.
INSTALLER_TARBALL_DATA ?= charts config.yaml scripts

.EXPORT_ALL_VARIABLES:

.default: build

#
# Build and Run
#

# Builds the application executable with installer resources embedded.
.PHONY: $(BIN)
$(BIN): installer-tarball
	@[ -d $(BIN_DIR) ] || mkdir -p $(BIN_DIR)
	go build -o $(BIN) $(CMD) $(ARGS)

.PHONY: build
build: $(BIN)

# Uses goreleaser to create a snapshot build.
.PHONY: goreleaser-snapshot
goreleaser-snapshot: installer-tarball
goreleaser-snapshot: tool-goreleaser
	goreleaser build --clean --snapshot --single-target -o $(BIN) $(ARGS)

snapshot: goreleaser-snapshot

# Runs the application with arbitrary ARGS.
.PHONY: run
run:
	go run $(CMD) $(ARGS)

#
# Installer Tarball
#

# Creates a tarball with all resources required for the installation process.
.PHONY: installer-tarball
installer-tarball:
	@test -f "$(INSTALLER_TARBALL)" && rm -f "$(INSTALLER_TARBALL)" || true
	tar -C "$(INSTALLER_DIR)" -cpf "$(INSTALLER_TARBALL)" \
		$(INSTALLER_TARBALL_DATA)

#
# Container Image
#

image:
	podman build --tag="$(IMAGE_REPO)/$(APP):$(IMAGE_TAG)" .

image-run:
	podman run \
		--rm \
		--interactive \
		--tty \
		$(IMAGE_REPO)/$(APP):$(IMAGE_TAG) \
		$(ARGS)

#
# Tools
#

# Installs golangci-lint.
tool-golangci-lint: GOFLAGS =
tool-golangci-lint:
	@which golangci-lint &>/dev/null || \
		go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest &>/dev/null

# Installs GitHub CLI ("gh").
tool-gh: GOFLAGS =
tool-gh:
	@which gh >/dev/null 2>&1 || \
		go install github.com/cli/cli/v2/cmd/gh@latest >/dev/null 2>&1

# Installs GoReleaser.
tool-goreleaser: GOFLAGS =
tool-goreleaser:
	@which goreleaser >/dev/null 2>&1 || \
		go install github.com/goreleaser/goreleaser@latest >/dev/null 2>&1

#
# Test and Lint
#

test: test-unit

# Runs the unit tests.
.PHONY: test-unit
test-unit: installer-tarball
	go test $(GOFLAGS_TEST) $(CMD) $(PKG) $(ARGS)

# Uses golangci-lint to inspect the code base.
.PHONY: lint
lint: tool-golangci-lint
	golangci-lint run ./...

#
# GitHub Release
#

# Asserts the required environment variables are set and the target release
# version starts with "v".
github-preflight:
ifeq ($(strip $(GITHUB_REF_NAME)),)
	$(error variable GITHUB_REF_NAME is not set)
endif
ifeq ($(shell echo ${GITHUB_REF_NAME} |grep -v -E '^v'),)
	@echo GITHUB_REF_NAME=\"${GITHUB_REF_NAME}\"
else
	$(error invalid GITHUB_REF_NAME, it must start with "v")
endif
ifeq ($(strip $(GITHUB_TOKEN)),)
	$(error variable GITHUB_TOKEN is not set)
endif

# Creates a new GitHub release with GITHUB_REF_NAME.
.PHONY: github-release-create
github-release-create: tool-gh
	gh release view $(GITHUB_REF_NAME) >/dev/null 2>&1 || \
		gh release create --generate-notes $(GITHUB_REF_NAME)

# Runs "goreleaser" to build the artifacts and upload them into the current
# release payload, it amends the release in progress with the application
# executables.
.PHONY: goreleaser-release
goreleaser-release: installer-tarball
goreleaser-release: tool-goreleaser
goreleaser-release: CGO_ENABLED = 0
goreleaser-release: GOFLAGS = -a
goreleaser-release:
	goreleaser release --clean --fail-fast $(ARGS)

# Releases the GITHUB_REF_NAME.
github-release: \
	github-preflight \
	github-release-create \
	goreleaser-release
