APP = rhtap-installer-cli

BIN_DIR ?= ./bin
BIN ?= $(BIN_DIR)/$(APP)

CMD ?= ./cmd/...
PKG ?= ./pkg/...

GOFLAGS ?= -v
GOFLAGS_TEST ?= -failfast -v -cover

.default: build

.PHONY: $(BIN)
$(BIN):
	@[ -d $(BIN_DIR) ] || mkdir -p $(BIN_DIR)
	go build -o $(BIN) $(CMD) $(ARGS)

.PHONY: build
build: $(BIN)

.PHONY: run
run:
	go run $(CMD) $(ARGS)

test: test-unit

.PHONY: test-unit
test-unit:
	go test $(GOFLAGS_TEST) $(CMD) $(PKG) $(ARGS)
