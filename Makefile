APP = tssc

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

# GoReleaser executable and version.
GORELEASER_BIN ?= goreleaser
GORELEASER_VERSION ?= v2.13.1

# GitHub action current ref name, provided by the action context environment
# variables, and credentials needed to push the release.
GITHUB_REF_NAME ?= ${GITHUB_REF_NAME:-}
GITHUB_TOKEN ?= ${GITHUB_TOKEN:-}

# Container registry credentials.
IMAGE_REPO_USERNAME ?=
IMAGE_REPO_PASSWORD ?=

# Container registry repository, the hostname of the registry, or empty for
# default registry.
IMAGE_REPO ?= ghcr.io
# Container image namespace, usually the organization or user name.
IMAGE_NAMESPACE ?= redhat-appstudio
# Container image tag.
IMAGE_TAG ?= latest
# Fully qualified container image name.
IMAGE_FQN ?= $(IMAGE_REPO)/$(IMAGE_NAMESPACE)/$(APP):$(IMAGE_TAG)

# Determine the appropriate tar command based on the operating system.
UNAME_S := $(shell uname -s)
ifeq ($(UNAME_S),Darwin)
	TAR := gtar
else
	TAR := tar
endif

# Directory with the installer resources, scripts, Helm Charts, etc.
INSTALLER_DIR ?= ./installer
# Tarball with the installer resources.
INSTALLER_TARBALL ?= $(INSTALLER_DIR)/installer.tar
# Data to include in the tarball.
INSTALLER_TARBALL_DATA ?= $(shell find -L $(INSTALLER_DIR) -type f \
	! -path "$(INSTALLER_TARBALL)" \
	! -name embed.go \
)

# Version will be set at build time via git describe
VERSION ?= $(shell \
	if [ -n "$(GITHUB_REF_NAME)" ]; then echo "${GITHUB_REF_NAME}"; \
	else git describe --tags --always || echo "v0.0.0-SNAPSHOT"; \
	fi)

# Commit will be set at build time via git commit hash
COMMIT_ID ?= $(shell git rev-parse HEAD)

.EXPORT_ALL_VARIABLES:

.default: build

#
# Build and Run
#

# Builds the application executable with installer resources embedded.
.PHONY: $(BIN)
$(BIN): installer-tarball
$(BIN):  
	@echo "# Building '$(BIN)'"
	@[ -d $(BIN_DIR) ] || mkdir -p $(BIN_DIR)
	go build -ldflags "-X main.version=$(VERSION) -X main.commitID=$(COMMIT_ID)" -o $(BIN) $(CMD)

.PHONY: build
build: $(BIN)

# Builds the application executable with debugging enabled.
.PHONY: debug
debug: GOFLAGS = "-gcflags=all=-N -l"
debug: $(BIN)

# Uses goreleaser to create a snapshot build.
.PHONY: goreleaser-snapshot
goreleaser-snapshot: installer-tarball
goreleaser-snapshot: tool-goreleaser
	$(GORELEASER_BIN) build --clean --snapshot $(ARGS)

snapshot: goreleaser-snapshot

# Runs the application with arbitrary ARGS.
.PHONY: run
run: installer-tarball
	go run $(CMD) $(ARGS)

#
# Installer Tarball
#

# Creates a tarball with all resources required for the installation process.
.PHONY: installer-tarball
installer-tarball: $(INSTALLER_TARBALL)
$(INSTALLER_TARBALL): $(INSTALLER_TARBALL_DATA)
	@echo "# Generating '$(INSTALLER_TARBALL)'"
	@test -f "$(INSTALLER_TARBALL)" && rm -f "$(INSTALLER_TARBALL)" || true
	@$(TAR) -C "$(INSTALLER_DIR)" -cpf "$(INSTALLER_TARBALL)" \
	$(shell echo "$(INSTALLER_TARBALL_DATA)" | sed "s:\./installer/:./:g")

#
# Container Image
#

# By default builds the container image using Podman.
image: image-podman

# Builds the container image with Podman.
image-podman:
	@echo "# Building '$(IMAGE_FQN)'..."
	podman build --build-arg COMMIT_ID=$(COMMIT_ID) --build-arg VERSION_ID=$(VERSION) --tag="$(IMAGE_FQN)" .

# Logins into the container registry.
login-buildah:
	@echo "# Login into '$(IMAGE_REPO)' with user '$(IMAGE_REPO_USERNAME)'"
	@buildah login \
		--username="$(IMAGE_REPO_USERNAME)" \
		--password="$(IMAGE_REPO_PASSWORD)" \
		$(IMAGE_REPO)

# Builds the container image with Buildah.
image-buildah:
	@echo "# Building '$(IMAGE_FQN)'..."
	buildah bud --build-arg COMMIT_ID=$(COMMIT_ID) --build-arg VERSION_ID=$(VERSION) --tag="$(IMAGE_FQN)" .

# Tags the container image with the provided arguments as tag.
image-buildah-tag: NEW_IMAGE_FQN = $(IMAGE_REPO)/$(IMAGE_NAMESPACE)/$(APP):$(ARGS)
image-buildah-tag:
	@echo "# Tagging '$(IMAGE_FQN)' with $(ARGS)..."
	buildah tag $(IMAGE_FQN) $(NEW_IMAGE_FQN)

# Pushes the container image to the registry.
image-buildah-push:
	@echo "# Pushing '$(IMAGE_FQN)'..."
	buildah push $(IMAGE_FQN)

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
	@which $(GORELEASER_BIN) >/dev/null 2>&1 || \
		go install github.com/goreleaser/goreleaser/v2@$(GORELEASER_VERSION)

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
	$(GORELEASER_BIN) release --clean --fail-fast $(ARGS)

# Releases the GITHUB_REF_NAME.
github-release: \
	github-preflight \
	github-release-create \
	goreleaser-release
