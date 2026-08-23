# 发布检查清单

本清单供维护者从 `anytty/anytty` 准备 AnyTTY 公开发布时使用。

## 源码与边界

- [ ] 从公开的 `anytty/anytty` 仓库选择已审核提交，并确认 `VERSION` 与目标 tag 一致。
- [ ] 确认仓库只包含经过审核的用户设备源码与公开文档，不含私有服务实现、生产配置、凭据或私有历史。
- [ ] 在公开仓库执行 `npm run public:check`。
- [ ] 检查生成文件、截图、fixture 和声明中没有秘密或个人数据。

## 质量

- [ ] 运行 `make test`、`make test-clients`、Android 验证和支持的 iOS 构建检查。
- [ ] 构建 CLI/TUI，并确认官方 Cloud 客户端能力仍包含在二进制中。
- [ ] 生成 Pages 产物并检查桌面、手机、键盘和 reduced-motion。
- [ ] 验证 README、文档、issue 表单和安全链接。
- [ ] 审查变更记录、平台状态、已知限制和升级说明。

## 法律与供应链

- [ ] 检查 Apache-2.0、NOTICE、DCO、商标政策和版权年份。
- [ ] 重新生成并审核 Go、npm、Android、iOS、字体与固定第三方声明。
- [ ] 审查 Dependabot 与 CI，处理已知漏洞或记录接受的风险。
- [ ] 为实际发布的每个产物生成 checksum 与 provenance。

## GitHub 与发布

- [ ] 启用私密漏洞报告并验证维护者通知路径。
- [ ] Pages source 设为 GitHub Actions，并验证 `/anytty/` 基址。
- [ ] 检查分支保护、必需检查、CODEOWNERS、Discussions、issue 与 Actions 权限。
- [ ] 审查现有公开历史是否适合发布，之后才决定是否初始化干净历史。
- [ ] 创建签名 tag；确认 Release workflow 只发布到 `anytty/anytty`，并产生完整 checksum 与 attestation。
- [ ] 由第二名维护者或指定审核人确认公开边界后再发布。
