# AnyTTY Native

Flutter presentation and platform bridge for the AnyTTY Go client engine and
the pinned libghostty-vt input adapter. The first targets are iOS and Android;
the TypeScript application remains the Web client.

## Toolchain

The checked build environment uses Flutter 3.47.2, Dart 3.13.2, Go 1.26.7,
Zig 0.16.0, Android NDK 28.2.13676358, and Xcode 26.3. On macOS, install the
command-line dependencies with Homebrew and install the full Xcode application
and Android SDK separately:

```sh
brew install go zig cmake ninja cocoapods protobuf
brew install --cask flutter
```

Do not change the host-wide `xcode-select` value. The iOS build script selects
`/Applications/Xcode.app` through `DEVELOPER_DIR`.

## Verify

From this directory:

```sh
flutter analyze
flutter test
flutter build apk --debug
flutter build apk --release --target-platform android-arm64 --split-per-abi
../../scripts/verify-flutter-android-apk-boundary.sh \
  build/app/outputs/flutter-apk/app-arm64-v8a-release.apk
DEVELOPER_DIR=/Applications/Xcode.app/Contents/Developer \
  flutter build ios --simulator --debug
DEVELOPER_DIR=/Applications/Xcode.app/Contents/Developer \
  flutter build ios --debug --no-codesign
```

Native libraries are built by the Flutter platform build hooks through
`scripts/build-flutter-android-native.sh` and
`scripts/build-flutter-ios-native.sh` at the repository root.
