# AnyTTY iOS

这是独立新增的 iOS 客户端工程。它复用现有 React 界面、Protobuf 协议和 Go 客户端核心；iOS 与 Android 共用移动端界面和配对输入逻辑，原生工程彼此独立。

## 环境

- macOS 与完整 Xcode（当前工程最低支持 iOS 15）
- Go 1.26 或兼容版本
- Node.js 与仓库根目录已安装的 npm 依赖
- 只有重新生成 Swift Protobuf 文件时才需要 `protoc` 和 `protoc-gen-swift`

默认使用 `/Applications/Xcode.app/Contents/Developer`。Xcode 位于其他目录时，通过 `DEVELOPER_DIR` 指定。

## 构建模拟器版本

在仓库根目录执行：

```sh
clients/mobile/ios/scripts/build-ios.sh
```

该命令依次构建现有移动端 Web UI、同步 Web 资源、生成真机与模拟器通用的 Go XCFramework，然后用 Xcode 构建 App。输出位于：

```text
clients/mobile/ios/DerivedData/Build/Products/Debug-iphonesimulator/App.app
```

## 构建真机版本

无签名编译检查：

```sh
ANYTTY_IOS_SDK=iphoneos \
ANYTTY_IOS_DERIVED_DATA=clients/mobile/ios/DerivedData-device \
clients/mobile/ios/scripts/build-ios.sh
```

需要安装到真机或发布时，在 Xcode 中打开 `clients/mobile/ios/App/App.xcodeproj`，为 `App` target 设置开发团队与签名证书后运行或 Archive。

## 单独更新生成内容

```sh
clients/mobile/ios/scripts/sync-web-assets.sh
clients/mobile/ios/scripts/build-go-xcframework.sh
clients/mobile/ios/scripts/generate-swift-protos.sh
```

`generate-swift-protos.sh` 仅在仓库的 `.proto` 定义变化后运行。Go XCFramework、Web 产物和 Xcode DerivedData 都是忽略的构建产物，不会进入源码提交。

## 原生能力

- 本机回环 WebSocket 与 Go 客户端核心通信
- iOS Keychain 保存远端访问凭据
- Secure Enclave（模拟器回退为 Keychain）保存 SSH 签名密钥
- iOS 文档选择器、上传流、断点下载与 SHA-256 校验
- 前后台切换、网络变化、触觉反馈和桥接代次刷新
