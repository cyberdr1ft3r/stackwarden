SHELL := /bin/bash
GO ?= go
BIN_DIR ?= bin
ROOT_DIR := $(dir $(abspath $(lastword $(MAKEFILE_LIST))))

AGENT_SOCKET ?= /run/stackwarden/agent.sock
API_BIND ?= 127.0.0.1:8080

# Keep Linux runtime requirements minimal (Go + Make only):
# avoid rg-based discovery; Windows users can run `go run ./api` directly.
API_MAIN_TARGET ?= ./api

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

PORT_ARG :=
ifneq ($(origin PORT), undefined)
ifneq ($(strip $(PORT)),)
PORT_ARG := --port $(PORT)
endif
endif

run-agent:
	AGENT_SOCKET=$(AGENT_SOCKET) $(GO) run ./agent

run-api:
	@echo "Starting API with API_BIND=$${API_BIND:-127.0.0.1:8080} $(PORT_ARG)"
	@echo "Running: AGENT_SOCKET=$(AGENT_SOCKET) API_BIND=$(API_BIND) $(GO) run $(API_MAIN_TARGET) $(PORT_ARG)"
	@cd $(ROOT_DIR) && AGENT_SOCKET=$(AGENT_SOCKET) API_BIND=$(API_BIND) $(GO) run $(API_MAIN_TARGET) $(PORT_ARG)

test:
	$(GO) test -C agent ./...
	$(GO) test -C api ./...
	$(GO) test -C pkg ./...

build:
	mkdir -p $(BIN_DIR)
	$(GO) build -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/agent ./agent
	$(GO) build -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/api ./api
