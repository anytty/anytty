# 贡献指南

AnyTTY 欢迎范围明确的修复、测试、文档、无障碍改进，以及保持现有安全边界的功能建议。项目当前处于早期 Beta，大型变更请先创建 issue 讨论。

修改行为前请阅读 `README.md`、`ARCHITECTURE.md` 和相关协议文档。同一变更中的 schema、生成代码、实现、测试和文档必须保持一致。

```sh
npm ci
make test
make test-clients
npm run public:check
```

Android release 验证还需要 Java 21 和 Android SDK。不要提交 `.artifacts/`、凭据、pairing claim、enrollment code、终端内容、私有服务源码、生产配置或平台构建目录。

pull request 应只包含一个可审查的行为范围，并说明用户可见变化、安全影响、验证命令以及 UI 变化的截图。依赖或打包资产变化时同步更新第三方声明。

所有提交须按根目录 `DCO` 使用 `git commit -s` 添加 `Signed-off-by`。贡献按 Apache-2.0 授权。安全问题必须按[安全策略](SECURITY.md)私密报告，所有项目空间均适用[行为准则](CODE_OF_CONDUCT.md)。
