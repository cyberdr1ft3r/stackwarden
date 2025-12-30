SHELL := /bin/bash
GO ?= go
BIN_DIR ?= bin

AGENT_BIND ?= :9091
API_BIND ?= :8080
AGENT_BASE ?= http://127.0.0.1:9091

VERSION ?= dev
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "")
BUILT_AT ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)

LDFLAGS := -X main.version=$(VERSION)
ifneq ($(strip $(COMMIT)),)
LDFLAGS += -X main.commit=$(COMMIT)
endif
ifneq ($(strip $(BUILT_AT)),)
LDFLAGS += -X main.builtAt=$(BUILT_AT)
endif

.PHONY: run-agent run-api test build

run-agent:
	AGENT_BIND=$(AGENT_BIND) $(GO) run ./agent

run-api:
	API_BIND=$(API_BIND) AGENT_BASE=$(AGENT_BASE) $(GO) run ./api

test:
	$(GO) test -C agent ./...
	$(GO) test -C api ./...
	$(GO) test -C pkg ./...

build:
	mkdir -p $(BIN_DIR)
	$(GO) build -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/agent ./agent
	$(GO) build -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/api ./api
