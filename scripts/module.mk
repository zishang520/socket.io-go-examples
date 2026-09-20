# Included from each example's module directory; no parent checkout required.
.DEFAULT_GOAL := help
SHELL := /bin/bash
.SHELLFLAGS := -eu -o pipefail -c
MAKEFLAGS += --no-print-directory
export GOPROXY ?= https://proxy.golang.org,direct
export TEST_TIMEOUT ?= 60s

# Respect the active workspace rather than just the presence of a local go.work.
define DEPS
workspace=$$(go env GOWORK); \
go mod tidy; \
if [ -n "$$workspace" ] && [ "$$workspace" != off ]; then \
	go work sync; \
	go work vendor; \
else \
	go mod vendor; \
fi
endef

.PHONY: all help env deps get update build fmt vet lint clean test check-module
all: help

env deps get update build fmt clean lint: check-module
check-module:
	@[ -f go.mod ] || { printf '%s\n' '[Error] Run this command from an example module.' >&2; exit 1; }

help:
	@printf '%s\n' \
		'Usage: make [env | deps | get | update | build | fmt | vet | lint | clean | test]' \
		'deps refreshes tidy and vendor, syncing the active workspace when enabled.' \
		'test, vet and lint refresh deps first; update runs go get -u first.' \
		'Options: FIX=1 for lint; GOPROXY, TEST_TIMEOUT (default: 60s).'

env:
	@go env

deps:
	@$(DEPS)

get:
	go get ./...

update:
	go get -u -v ./...
	@$(DEPS)

build:
	go build ./...

fmt:
	go fmt ./...

clean:
	go clean -v -r ./...

vet: deps
	go vet ./...

lint:
	@command -v golangci-lint >/dev/null 2>&1 || { printf '%s\n' '[Error] golangci-lint is not installed.' >&2; exit 1; }
	@$(DEPS)
	golangci-lint run --timeout=5m $(if $(filter 1,$(FIX)),--fix) ./... </dev/null

test: deps
	go test -count=1 -timeout="$$TEST_TIMEOUT" -race -cover -covermode=atomic ./... </dev/null
