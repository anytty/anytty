# Android release process

AnyTTY supports these Android ABIs:

| ABI | Device class | Release asset |
| --- | --- | --- |
| `armeabi-v7a` | 32-bit ARM devices | `anytty-VERSION-android-armeabi-v7a.apk` |
| `arm64-v8a` | 64-bit ARM devices | `anytty-VERSION-android-arm64-v8a.apk` |
| `x86_64` | x86_64 devices and emulators | `anytty-VERSION-android-x86_64.apk` |

GitHub Releases also contain `anytty-VERSION-android-universal.apk`, which includes all supported ABIs. Google Play receives one `anytty-VERSION-android-play.aab`; Play generates the device-specific APK splits from that bundle.

All public APKs and the Play bundle must be signed with the same persistent Android upload key. The release workflow fails instead of publishing unsigned Android packages when signing credentials are unavailable. Configure these GitHub Actions secrets:

- `ANYTTY_ANDROID_UPLOAD_KEYSTORE_BASE64`
- `ANYTTY_ANDROID_UPLOAD_STORE_PASSWORD`
- `ANYTTY_ANDROID_UPLOAD_KEY_ALIAS`
- `ANYTTY_ANDROID_UPLOAD_KEY_PASSWORD`

Keep the keystore and passwords outside the repository. Rotating the key requires an explicit migration plan because sideloaded APK updates must be signed by the same key as the installed app.

For local validation, `make test-android` runs the Flutter tests, builds a release-mode arm64 APK, and checks that the artifact contains the Flutter runtime, Go client engine, and terminal input library without a WebView or JavaScript runtime. Without upload-key environment variables, Gradle uses the debug key for this local build only.

The release workflow builds the Flutter AAB, split APKs, and universal APK as separate invocations. It verifies every APK's exact ABI set, signature, native ELF alignment, and Flutter-native boundary. It also verifies that the signed AAB contains the Flutter runtime, Go client engine, and terminal input library for `armeabi-v7a`, `arm64-v8a`, and `x86_64` before publishing any Android asset.
