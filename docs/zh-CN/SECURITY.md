# 安全策略

AnyTTY 尚未发布，目前没有受支持版本、漏洞奖励或保证响应时间。

安全问题请使用 [GitHub 私密漏洞报告](https://github.com/lozzo/anytty/security/advisories/new)。不要创建包含利用细节、凭据、配对材料、终端内容或用户隐私的公开 issue。

daemon 是终端、文件、设备身份和客户端权限的最终权威。Cloud 服务提供发现、信令、策略和 Relay 传输，但不能授予终端或文件 capability。完整信任边界见 `ARCHITECTURE.md` 和 `docs/PAIRING_PROTOCOL.md`。

报告应包含受影响提交、前置条件、使用新生成测试凭据的最小复现、实际影响和已知缓解措施。不要附带真实私钥、claim、grant、终端内容或个人数据。

维护者会在能力范围内确认和分级，私下协调修复，并与报告者商定披露时间。维护者明确确认前，不应假定修复或披露日期。普通使用和服务账号问题请按[支持文档](SUPPORT.md)处理。
