# Cloud daemon 生命周期

本文描述当前 EdgeControl v10 的 daemon 状态与 Edge 重选协议。状态真值在 Controller PostgreSQL，Edge 保留最后一次可信的内存投影，控制链短暂断开时继续执行该状态。

## 1. 状态

| 状态 | 可恢复 | Agent 控制连接 | 新 Cloud client | 现有 Cloud session | enrollment |
| --- | --- | --- | --- | --- | --- |
| `ACTIVE` | - | 保持 | daemon ACK 后允许 | 保持 | 保留 |
| `BLOCKED` | 是 | 控制链正常时保持 | 拒绝 | 关闭 | 保留 |
| `DELETED` | 否 | cleanup ACK 后关闭 | 拒绝 | 关闭 | daemon 删除 |

Local、SSH、Direct、DeviceIdentity、AccessStore、本地 terminal 和 history 不受这三个 Cloud 状态影响。

## 2. Controller 写入

管理员 mutation 带 `expected_revision`。Controller 在同一 PostgreSQL 事务中：

1. 锁定当前 daemon state。
2. 校验允许的状态转换和 revision。
3. 写新 state 与递增 `state_revision`。
4. 提交后向已连接的 EdgeControl 广播包含完整 daemon record 的 delta。

广播失败不能回滚已提交数据库状态。Edge 重连后由完整 snapshot 收敛。

`DELETED` 是终态。重新使用 Cloud 必须创建新的 enrollment 和 daemon record，不能恢复旧 binding 或旧 generation。

## 3. Edge 状态表

Edge 内存保存 `daemon_id -> DaemonStateRecord`：

- 首个 EdgeControl snapshot 完成前，受管准入 fail closed。
- snapshot 原子替换整个状态表。
- delta 是单个 daemon 的完整 replacement；比当前 revision 小的记录忽略，相同 revision 必须内容相同，任意更大的 revision 都可替换。
- 是否丢失 Controller 消息由 EdgeControl `stream_seq` 连续性保证，不要求每个 daemon 的 `state_revision` 逐一连续。
- EdgeControl 断开会清空状态表、关闭 Agent，并排空 Edge 仍跟踪的 Cloud session；不会把旧策略留作离线准入。
- Edge 重启不从本地数据库恢复 policy，必须重连 Controller 获取 snapshot。

Controller 不持久化在线 Agent connection 或当前连接归属。binding 自身指定允许连接的 Edge；Agent connection ownership 是该 Edge 的内存事实。daemon 本地保存当前 binding 和 locator；Edge 位置失效时，用 DeviceIdentity 向 Controller 换取重新签名的完整 binding 和当前 locator。

## 4. BLOCKED 与恢复

Edge 收到 `BLOCKED` 时：

1. state actor 先移除业务 Presence，阻止新的 ClientGateway、pairing、signaling 与 Relay admission。
2. 排空该 daemon 的 pending signaling、现有 Cloud client session 和 Relay group。
3. 保留当前 Agent writer，并发送绑定 state revision 与 agent generation 的 lifecycle command。
4. daemon 关闭自己的 Cloud peer map 并确认已应用状态。

恢复 `ACTIVE` 时，Edge 先向当前 Agent 发送目标状态。只有 daemon ACK 当前 record 和 agent generation 后，Edge 才重新发布业务 Presence；迟到 ACK 或旧 generation 不能恢复准入。这个 ACK 门只作用于该 daemon 的业务 Presence，不影响 Edge 进程的全局 `/readyz`。

## 5. DELETED

Edge 收到 `DELETED` 时：

1. 移除业务 Presence并关闭 pending、ClientGateway、P2P 和 Relay。
2. 如果绑定到该 Edge 的 Agent 在线，发送 cleanup-only lifecycle command。
3. daemon 原子删除 Cloud enrollment record 和 Cloud issuer material。
4. daemon ACK 后停止 Cloud reconnect，Edge 关闭 AgentGateway。

如果 daemon 删除时离线，绑定 Edge 仍按 policy 拒绝旧 credential。daemon 后续能到达原 Edge时只执行 cleanup；原 locator 已失效时，daemon 向 Controller 证明 DeviceIdentity 并读取最新 `DELETED` 状态，不取得新路由材料，随后删除本机 Cloud enrollment。

## 6. 重连与 ready

### Edge 重连 Controller

1. 完成 mTLS EdgeControl v8 handshake。
2. 接收并验证 KeyBundle、desired config 和完整 daemon state snapshot。
3. 原子替换 policy table，并向当前 Agent 发送各自目标状态。
4. Controller generation 已同步且持久 KeyBundle 当前可用时，Edge `/readyz` 返回成功。

全局 ready 不等待所有 daemon lifecycle ACK；单个 `ACTIVE` daemon 在 ACK 前仍不发布业务 Presence。

### daemon 重连绑定 Edge

1. Edge 完成 challenge-first AgentGateway 身份验证。
2. Edge 查询当前内存 policy；policy table 不 ready 时拒绝。
3. `ACTIVE` 在 lifecycle ACK 后发布业务 Presence。
4. `BLOCKED` 保留控制连接但不发布业务 Presence。
5. `DELETED` 只执行 cleanup，完成后断开。

### binding 与 locator 刷新

1. daemon 启动时先使用稳定的 Controller 地址对账；后续当前 Edge 连接失败时再次发起一次性 challenge。
2. daemon 用本机 DeviceIdentity 私钥签名 challenge；Controller 从 PostgreSQL 读取身份和最新生命周期，不接受客户端提交账号或公钥。
3. daemon 可提交从自身网络位置测得的 TLS/gRPC 连接耗时和连接失败率；Controller 将新测量与在线状态、容量、负载和软偏好一起评分。
4. `ACTIVE`/`BLOCKED` 返回当前最佳 Edge locator、完整候选投影，以及绑定该 locator 摘要的新 binding。偏好 Edge 不可用时自动回退。
5. daemon 校验响应身份和 binding/locator 一致性，先原子保存 enrollment record，再切换运行时路由并重连。
6. `DELETED` 只返回终态，不返回 binding 或 locator。

网页或本机 CLI 发起立即重选时，Controller 经当前 Edge 和精确 Agent generation 下发命令。daemon 完成测速和原子落盘后只重建 AgentGateway；daemon 进程、terminal 和本地服务不重启，当前 Cloud 会话会随旧连接关闭并重新建立。

Controller 持久化软偏好和带时间戳的测量汇总，不持久化 daemon 当前归属 Edge。连接失败率来自 TCP/TLS/gRPC 健康探测，不等同于 UDP 丢包率。

## 7. Web 状态

当前 Cloud Web 展示业务状态、daemon 是否在线、当前 Edge 与软偏好，并允许查看候选和立即重选：

- `ACTIVE` 可执行阻断。
- `BLOCKED` 可恢复，也可删除。
- `DELETED` 不出现在普通用户和运营 daemon 列表，数据库终态记录仍保留。
- 候选列表展示 daemon 最近上报的连接耗时、连接失败率、节点负载和可用状态。
- 偏好使用独立 revision 做并发更新；网页显示“命令已送达”不等于迁移已经完成，当前 Edge 仍以 Directory 在线投影为准。

当前页面不展示 lifecycle command 的 pending/applied revision；不能把 Edge `/readyz` 当作每台 daemon 都已 ACK 的证明。

## 8. 安全与并发不变量

- `state_revision` 对同一 daemon 单调；EdgeControl `stream_seq` 对整个控制流严格连续。
- lifecycle command 绑定 daemon ID、完整 record 和 Agent generation。
- admission mutation、closer 注册和 session 清理在同一 Edge actor 线性化。
- 先关闭准入，再关闭已有连接；不能出现 Blocked 后加入的新 session。
- 旧连接、迟到 command/ACK 和重复 cleanup 不能影响新 generation。
- daemon 只有在当前 AgentGateway 已 ready、状态为 `ACTIVE` 且 lifecycle ACK 已成功发送后，才可生成含 Cloud route 的二维码或 grant。
- state payload 不包含 terminal 权限、ClientAccess 私钥或 terminal 内容。

## 9. 验收场景

- Active -> Blocked：新连接立即拒绝，现有 Cloud client/Relay 关闭，控制 Agent 保留。
- Blocked -> Active：daemon ACK 后恢复业务 Presence，不要求用户手工重连。
- Active/Blocked -> Deleted：enrollment 删除，Agent 断开，旧 credential 永久失败。
- Controller/Edge/daemon 分别在变更中断线，重连后由 snapshot + targeted command 收敛。
- snapshot 缺失、Controller stream sequence gap、旧 generation ACK 和 Controller link 断开全部 fail closed。
