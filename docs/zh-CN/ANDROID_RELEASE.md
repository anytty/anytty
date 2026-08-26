# Android 发布流程

AnyTTY 支持以下 Android ABI：

| ABI | 设备类型 | Release 资产 |
| --- | --- | --- |
| `armeabi-v7a` | 32 位 ARM 设备 | `anytty-VERSION-android-armeabi-v7a.apk` |
| `arm64-v8a` | 64 位 ARM 设备 | `anytty-VERSION-android-arm64-v8a.apk` |
| `x86_64` | x86_64 设备与模拟器 | `anytty-VERSION-android-x86_64.apk` |

GitHub Releases 还会提供包含全部受支持 ABI 的 `anytty-VERSION-android-universal.apk`。Google Play 只接收一个 `anytty-VERSION-android-play.aab`，再由 Play 按设备生成对应的 APK split。

所有公开 APK 和 Play bundle 必须使用同一个长期保存的 Android upload key 签名。签名凭据不可用时，release 工作流会直接失败，不会发布未签名 Android 包。需要配置以下 GitHub Actions secrets：

- `ANYTTY_ANDROID_UPLOAD_KEYSTORE_BASE64`
- `ANYTTY_ANDROID_UPLOAD_STORE_PASSWORD`
- `ANYTTY_ANDROID_UPLOAD_KEY_ALIAS`
- `ANYTTY_ANDROID_UPLOAD_KEY_PASSWORD`

keystore 和密码必须保存在仓库之外。更换密钥需要明确的迁移方案，因为侧载 APK 只有与已安装版本使用同一密钥签名才能直接升级。

本地执行 `make test-android` 会构建未签名 universal APK 并运行 APK 边界检查。`scripts/build-android-aab.sh` 会创建或使用已被忽略的本地 upload keystore，构建签名 Play bundle，并验证其签名与 ABI 内容。

Android Gradle Plugin 不支持在启用 APK splits 的同一次调用中同时生成 AAB，因此 release 工作流会先构建 AAB，再启用 splits 构建 APK。随后工作流会检查每个 APK 的精确 ABI 集合、签名、生产资源与原生 ELF 对齐，同时确认签名 AAB 包含 `armeabi-v7a`、`arm64-v8a` 和 `x86_64` 原生库；全部通过后才发布 Android 资产。
