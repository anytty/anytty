SHELL := /bin/bash
.SHELLFLAGS := -eu -o pipefail -c
.DEFAULT_GOAL := build

ARTIFACT_DIR := $(CURDIR)/.artifacts
ANYTTY_BIN := $(ARTIFACT_DIR)/bin/anytty

.PHONY: build test test-clients site-build site-check site-preview public-check clean

build:
	mkdir -p "$(dir $(ANYTTY_BIN))"
	GOWORK=off go build -trimpath -o "$(ANYTTY_BIN)" ./cmd/anytty

test:
	GOWORK=off go test ./... -count=1

test-clients:
	npm run test
	npm run typecheck
	npm run build

site-build:
	npm run site:build

site-check:
	npm run site:check

site-preview:
	npm run site:preview

public-check:
	npm run public:check

clean:
	rm -rf "$(ARTIFACT_DIR)" clients/ui/dist clients/mobile/dist site/.astro site/dist
