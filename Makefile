SHELL := /bin/bash
.SHELLFLAGS := -eu -o pipefail -c
.DEFAULT_GOAL := build

ARTIFACT_DIR := $(CURDIR)/.artifacts
ANYTTY_BIN := $(ARTIFACT_DIR)/bin/anytty
ANDROID_DIR := $(CURDIR)/clients/mobile/android
ANDROID_ARTIFACT_DIR := $(ARTIFACT_DIR)/android
RELEASE_VERSION ?=

.PHONY: build release test test-clients test-android local-web-bundle public-check clean

build:
	mkdir -p "$(dir $(ANYTTY_BIN))"
	GOWORK=off go build -trimpath -o "$(ANYTTY_BIN)" ./cmd/anytty

release:
	@test -n "$(RELEASE_VERSION)" || { echo 'RELEASE_VERSION is required' >&2; exit 2; }
	scripts/build-release-artifacts.sh "$(RELEASE_VERSION)"

test:
	scripts/test-install.sh
	GOWORK=off go test ./... -count=1

test-clients:
	npm run test
	npm run typecheck
	npm run build
	GOWORK=off go run ./internal/cmd/localwebbundle --check

local-web-bundle:
	npm run build --workspace @anytty/mobile
	GOWORK=off go run ./internal/cmd/localwebbundle

test-android:
	npm run cap:build
	mkdir -p "$(ANDROID_ARTIFACT_DIR)"
	cd "$(ANDROID_DIR)" && ./gradlew clean testDebugUnitTest assembleRelease
	cp "$(ANDROID_DIR)/app/build/outputs/apk/release/app-release-unsigned.apk" "$(ANDROID_ARTIFACT_DIR)/app-release-unsigned.apk"
	scripts/verify-android-apk-boundary.sh "$(ANDROID_ARTIFACT_DIR)/app-release-unsigned.apk"

public-check:
	npm run public:check

clean:
	rm -rf "$(ARTIFACT_DIR)" clients/ui/dist clients/mobile/dist
