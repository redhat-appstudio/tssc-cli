APP = rhtap-installer-cli

BIN_DIR ?= ./bin
BIN ?= $(BIN_DIR)/$(APP)

CMD ?= ./cmd/...
PKG ?= ./pkg/...

GOFLAGS ?= -v
GOFLAGS_TEST ?= -failfast -v -cover

.default: build

#
# Tools
#

# Installs golangci-lint.
tool-golangci-lint: GOFLAGS =
tool-golangci-lint:
	@which golangci-lint &>/dev/null || \
		go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest &>/dev/null

#
# Build and Run
#

.PHONY: $(BIN)
$(BIN):
	@[ -d $(BIN_DIR) ] || mkdir -p $(BIN_DIR)
	go build -o $(BIN) $(CMD) $(ARGS)

.PHONY: build
build: $(BIN)

# Runs the application with arbitrary ARGS.
.PHONY: run
run:
	go run $(CMD) $(ARGS)

#
# Test and Lint
#

test: test-unit

# Runs the unit tests.
.PHONY: test-unit
test-unit:
	go test $(GOFLAGS_TEST) $(CMD) $(PKG) $(ARGS)

# Uses golangci-lint to inspect the code base.
.PHONY: lint
lint: tool-golangci-lint
	golangci-lint run ./...
