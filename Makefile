SHELL := /bin/bash
GO ?= go
BIN_DIR ?= bin
ROOT_DIR := $(dir $(abspath $(lastword $(MAKEFILE_LIST))))

AGENT_BIND ?= :9091
PORT ?=
API_PORT ?=
API_BIND ?= :8080
AGENT_URL ?=
AGENT_BASE ?= http://127.0.0.1:9091

ifneq ($(strip $(PORT)),)
API_BIND := :$(PORT)
endif
ifneq ($(strip $(API_PORT)),)
API_BIND := :$(API_PORT)
endif
ifneq ($(strip $(AGENT_URL)),)
AGENT_BASE :=
endif

API_MAIN_DIR := $(shell cd $(ROOT_DIR) && $(GO) list -f '{{if eq .Name "main"}}{{.ImportPath}} {{.Dir}}{{end}}' ./... | awk '$$1 ~ /\/api(\/|$$)/ {print $$2; exit}')
API_PORT_EFFECTIVE := $(if $(strip $(PORT)),$(PORT),$(if $(strip $(API_PORT)),$(API_PORT),8080))

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
	@if [ -z "$(API_MAIN_DIR)" ]; then echo "Unable to locate API main package"; exit 1; fi
	@echo "Starting API on port $(API_PORT_EFFECTIVE)"
	@echo "Running: API_BIND=$(API_BIND) AGENT_BASE=$(AGENT_BASE) AGENT_URL=$(AGENT_URL) $(GO) run $(API_MAIN_DIR)"
	@cd $(ROOT_DIR) && API_BIND=$(API_BIND) AGENT_BASE=$(AGENT_BASE) AGENT_URL=$(AGENT_URL) $(GO) run $(API_MAIN_DIR)

test:
	$(GO) test -C agent ./...
	$(GO) test -C api ./...
	$(GO) test -C pkg ./...

build:
	mkdir -p $(BIN_DIR)
	$(GO) build -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/agent ./agent
	$(GO) build -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/api ./api
