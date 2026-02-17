SHELL := /bin/bash
GO ?= go
BIN_DIR ?= bin
ROOT_DIR := $(dir $(abspath $(lastword $(MAKEFILE_LIST))))

AGENT_BIND ?= :9091
API_BIND ?= :8080
AGENT_BASE ?= http://127.0.0.1:9091

API_MAIN_DIR := $(shell cd $(ROOT_DIR) && rg -l --no-messages '^package main$$' -g '*.go' | rg 'api/' | sed -n '1{s|/[^/]*$$||;p;}')
API_MAIN_TARGET := ./$(API_MAIN_DIR)

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
	AGENT_BIND=$(AGENT_BIND) $(GO) run ./agent

run-api:
	@if [ -z "$(API_MAIN_DIR)" ]; then echo "Unable to locate API main package"; exit 1; fi
	@echo "Starting API with API_BIND=$${API_BIND:-:8080} $(PORT_ARG)"
	@echo "Running: AGENT_BASE=$(AGENT_BASE) API_BIND=$(API_BIND) $(GO) run $(API_MAIN_TARGET) $(PORT_ARG)"
	@cd $(ROOT_DIR) && AGENT_BASE=$(AGENT_BASE) API_BIND=$(API_BIND) $(GO) run $(API_MAIN_TARGET) $(PORT_ARG)

test:
	$(GO) test -C agent ./...
	$(GO) test -C api ./...
	$(GO) test -C pkg ./...

build:
	mkdir -p $(BIN_DIR)
	$(GO) build -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/agent ./agent
	$(GO) build -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/api ./api
