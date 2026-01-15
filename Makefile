APP = helmet

# Primary source code directories.
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
# Build
#

# Build the application
.PHONY: build
build: installer-tarball
	go build $(GOFLAGS) -ldflags "-X main.Version=$(VERSION) -X main.Commit=$(COMMIT_ID)" ./...

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

#
# Test and Lint
#

test: test-unit

# Runs the unit tests.
.PHONY: test-unit
test-unit: installer-tarball
	go test $(GOFLAGS_TEST) $(PKG) $(ARGS)

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

# Releases the GITHUB_REF_NAME.
github-release: \
	github-preflight \
	github-release-create