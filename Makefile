SHELL := /usr/bin/env bash
.DEFAULT_GOAL := help

GO := go
GOFLAGS ?=
# `./...` would walk web/node_modules where some npm packages embed Go files.
# Restrict the package list to our actual source dirs.
PKG := ./cmd/... ./internal/... ./pkg/...
VERSION ?= dev
LDFLAGS := -s -w -X main.Version=$(VERSION)

.PHONY: help
help:
	@grep -E '^[a-zA-Z0-9_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk -F':.*?## ' '{printf "  \033[36m%-22s\033[0m %s\n", $$1, $$2}'

.PHONY: tidy
tidy: ## go mod tidy
	$(GO) mod tidy

.PHONY: build
build: ## build the supremacy binary into ./bin
	mkdir -p bin
	$(GO) build $(GOFLAGS) -trimpath -ldflags='$(LDFLAGS)' -o bin/supremacy ./cmd/supremacy

.PHONY: vet
vet: ## go vet
	$(GO) vet $(PKG)

.PHONY: test
test: ## run unit tests with race detector
	$(GO) test -race -count=1 $(PKG)

.PHONY: test-short
test-short: ## fast unit tests (no race)
	$(GO) test -count=1 -short $(PKG)

.PHONY: lint
lint: ## golangci-lint (must be installed)
	@which golangci-lint > /dev/null || (echo "install: https://golangci-lint.run/" && exit 1)
	golangci-lint run

.PHONY: run-core-api
run-core-api: ## run core-api locally (assumes infra via 'make up')
	$(GO) run ./cmd/supremacy core-api

.PHONY: run-gateway
run-gateway:
	$(GO) run ./cmd/supremacy gateway

.PHONY: run-engine
run-engine:
	$(GO) run ./cmd/supremacy engine

.PHONY: run-worker
run-worker:
	$(GO) run ./cmd/supremacy worker

.PHONY: up
up: ## bring up Postgres, Redis and NATS for local dev
	docker compose up -d postgres redis nats

.PHONY: up-all
up-all: ## bring up the full stack (build images first)
	docker compose up -d --build

.PHONY: down
down: ## tear down everything
	docker compose down -v

.PHONY: logs
logs: ## tail the full stack
	docker compose logs -f

.PHONY: web-install
web-install: ## install web deps
	cd web && npm install

.PHONY: web-dev
web-dev: ## run web SPA in dev mode
	cd web && npm run dev

.PHONY: web-build
web-build:
	cd web && npm run build

.PHONY: web-lint
web-lint:
	cd web && npm run lint

.PHONY: web-format-check
web-format-check:
	cd web && npm run format:check

.PHONY: web-typecheck
web-typecheck:
	cd web && npm run typecheck

.PHONY: refactor-check
refactor-check: vet lint test web-lint web-format-check web-typecheck ## run every gate before opening a PR
	@echo "all gates passed"

.PHONY: tf-fmt
tf-fmt:
	terraform -chdir=terraform fmt -recursive

.PHONY: tf-validate
tf-validate:
	terraform -chdir=terraform/envs/dev init -backend=false
	terraform -chdir=terraform/envs/dev validate
