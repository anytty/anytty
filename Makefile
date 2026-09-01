SHELL := /bin/bash
.SHELLFLAGS := -eu -o pipefail -c
.DEFAULT_GOAL := build

ARTIFACT_DIR := $(CURDIR)/.artifacts
ANYTTY_BIN := $(ARTIFACT_DIR)/bin/anytty
FLUTTER_DIR := $(CURDIR)/clients/flutter
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
	npm run build --workspace @anytty/web
	GOWORK=off go run ./internal/cmd/localwebbundle

test-android:
	cd "$(FLUTTER_DIR)" && flutter pub get
	cd "$(FLUTTER_DIR)" && flutter test
	cd "$(FLUTTER_DIR)" && flutter build apk --release --target-platform android-arm64 --split-per-abi
	mkdir -p "$(ANDROID_ARTIFACT_DIR)"
	cp "$(FLUTTER_DIR)/build/app/outputs/flutter-apk/app-arm64-v8a-release.apk" "$(ANDROID_ARTIFACT_DIR)/anytty-arm64-v8a-release.apk"
	ANYTTY_ANDROID_EXPECTED_ABIS=arm64-v8a scripts/verify-flutter-android-apk-boundary.sh "$(ANDROID_ARTIFACT_DIR)/anytty-arm64-v8a-release.apk"

public-check:
	npm run public:check

clean:
	rm -rf "$(ARTIFACT_DIR)" clients/ui/dist clients/web/dist clients/flutter/build
