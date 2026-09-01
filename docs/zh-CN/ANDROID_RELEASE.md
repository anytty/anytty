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

本地执行 `make test-android` 会运行 Flutter 测试、构建 release 模式的 arm64 APK，并检查产物包含 Flutter 运行时、Go Client Engine 和终端输入库，同时不包含 WebView 或 JavaScript 运行时。未配置 upload key 环境变量时，Gradle 仅对本地构建使用 debug key。

Release 工作流通过独立命令构建 Flutter AAB、分 ABI APK 和 universal APK。随后检查每个 APK 的精确 ABI 集合、签名、原生 ELF 对齐和 Flutter 原生边界，同时确认签名 AAB 为 `armeabi-v7a`、`arm64-v8a` 和 `x86_64` 包含 Flutter 运行时、Go Client Engine 和终端输入库；全部通过后才发布 Android 资产。
