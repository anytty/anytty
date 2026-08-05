# 扫码配对协议

## 1. 产品规则

- 移动 App 不登录、不关联 Cloud 账号、不自动发现设备。
- 每台移动客户端必须由用户主动扫描目标服务生成的一次性 pairing offer；App 不提供文本导入入口。
- Cloud 账号不能把 daemon 自动加入 App，也不能替代 pairing。
- pairing 只授予目标 daemon 明确签发的能力，不授予系统账号或任意终端访问。

## 2. Offer

`anytty pair create` 由目标 daemon 生成短期一次性 claim。移动 App 使用二维码；文本和命令输出只用于 CLI 工作流。

offer 包含：

- daemon identity 和展示 label。
- 一次性 claim、过期时间和目标限制。
- 客户端可尝试的 SSH、Direct 或 Cloud route hint。
- Cloud route 所需 Edge endpoint、server name 和 CA fingerprint。

offer 不包含 daemon 私钥、ClientAccessIdentity 私钥、长期 CapabilityGrant 或账号密码。

默认 claim TTL 为十分钟。claim 在 daemon 内原子消费；其他客户端身份不能重放。同一 bundle 和同一客户端 key 可在成功后的 24 小时 delivery grace 内幂等取回原结果，用于恢复响应丢失，不会签发第二份 grant。

## 3. 创建

以下示例假设 `anytty` 已安装到 `PATH`；从仓库构建时可替换为 `./.artifacts/bin/anytty`。

最小二维码：

```sh
anytty pair create --qr-file ./anytty-pair.png
```

默认二维码携带 Direct route。Cloud 配对必须显式指定 route，并且 daemon 当前已连接 Edge、完成 `ACTIVE` 生命周期确认：

```sh
anytty pair create --route cloud --qr-file ./anytty-cloud-pair.png
```

只限制到一个 terminal：

```sh
TERMINAL_ID=REPLACE_WITH_TERMINAL_ID
anytty pair create --terminal "$TERMINAL_ID" --qr-file ./anytty-pair.png
```

携带 SSH route：

```sh
anytty pair create \
  --route ssh \
  --ssh-host server.example.com \
  --ssh-user alice \
  --ssh-host-key SHA256:REPLACE_WITH_PINNED_HOST_KEY \
  --qr-file ./anytty-pair.png
```

完整选项以 `anytty pair create --help` 为准。

## 4. 兑换

1. 客户端严格解析 URI、版本、过期时间、identity 和 route。
2. 客户端创建自己的 ClientAccessIdentity key pair。
3. 客户端按 offer 中的 route 到达目标 daemon。
4. SSH 校验 host key；Direct/Cloud 校验 daemon identity 和相应 TLS/Edge 信任锚。
5. 客户端与 daemon 在 DTLS DataChannel 内提交一次性 claim 和 client public key。
6. daemon 原子消费 claim，创建 client-bound grant。
7. 客户端验证 daemon 返回的 CapabilityGrant、route credential 和 identity。
8. 只有全部通过后，客户端才原子写入 credential 与 endpoint registry。

Cloud pairing 时，Edge 只做准入、关联在线 Agent 和信令。claim 本体与最终权限交换仍在端到端通道内。

## 5. 存储

daemon 保存：

- DeviceIdentity。
- pairing ticket、消费状态、目标客户端摘要和短期 delivery recovery 信息。
- 已签发 ClientAccess grant 与撤销状态。

客户端 secure store 保存：

- ClientAccessIdentity 私钥。
- CapabilityGrant。
- route credential、locator 和 daemon identity pin。

客户端普通 endpoint registry 只保存非秘密的展示和 route metadata。日志不得记录完整 URI、claim、私钥或 signed envelope。

## 6. 失败语义

| 错误 | 行为 |
| --- | --- |
| URI 无效或字段未知 | 拒绝，不写 registry |
| claim 过期，或被其他客户端身份消费 | 拒绝，要求目标服务重新生成 |
| 同一客户端未收到兑换响应 | 24 小时 delivery grace 内返回原结果；超时后拒绝 |
| route 不可达 | 保留当前设备列表，可重试仍有效的 offer |
| daemon 未连接 Edge，或 Cloud 状态不是 `ACTIVE` | 拒绝生成含 Cloud route 的 offer；Direct/SSH 配对不受影响 |
| SSH host key / daemon identity 不匹配 | 拒绝，不 fallback 到较弱校验 |
| Edge/Controller 不可达 | Cloud route 失败；不能公开按 daemon ID 猜测位置 |
| credential 写入失败 | 关闭新 session，不留下半个 endpoint |
| App 重启或 WebView 重载 | 原 generation 取消；只使用已原子提交的 credential |

## 7. 生命周期

- daemon 可通过 `anytty access list` 查看 client-bound grant。
- `anytty access revoke REPLACE_WITH_GRANT_ID` 使对应客户端后续授权失败。
- 删除 App endpoint 只移除当前设备的 registry/credential，不删除 daemon。
- Cloud daemon `BLOCKED` 只阻断 Cloud 数据面；Local、SSH 和 Direct pairing 仍按 daemon 本地策略工作。
- Cloud daemon `DELETED` 清除 Cloud enrollment，不清除 DeviceIdentity、AccessStore、本地历史或非 Cloud route。
