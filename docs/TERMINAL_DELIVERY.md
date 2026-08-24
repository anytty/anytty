# 终端实时画面与历史协议

本文描述 daemon、TUI，以及 Android/iOS App 和 `anytty web` 本地浏览器入口通过共享 React UI 使用的终端交付模型。普通 Vite 浏览器预览仍不连接 daemon；正式本地 Web 入口由 daemon 仅在 IPv4 回环地址上按需提供。

## 1. 目标

- 高频输出不形成无界网络、Go heap 或 JavaScript 队列。
- 客户端尽快显示最新可见画面，不依赖固定 25ms 或 40 FPS 窗口。
- 增量丢失、客户端恢复和尺寸变化可以确定性回到全量画面。
- Live 与历史切换保持视觉连续，历史分页不阻塞新输出落盘。

## 2. PTY 输出所有权

每个 terminal generation 只有一个 `terminalOutputBuffer`。每个主 payload node 只拥有一份 byte slice，Live 和 History 通过独立 cursor 消费。raw PTY stream 为每个订阅者维护最多 16 个 chunk 的有界复制队列；它不属于 `resident_budget_bytes`，所以总内存还随活跃订阅者数量线性增长。

```text
PTY read -> bounded shared payload nodes -> Live consumer -> screen projector
                                    \----> History consumer -> history store
```

主 PTY payload 的单 terminal `capacity_bytes` 和 daemon 级 `resident_budget_bytes` 都是硬边界；raw-stream 队列由订阅者数和固定队列深度单独约束。

- `block`：没有容量时等待 consumer 释放空间，使 PTY 上游自然减速。
- `drop`：不等待，删除旧 payload，并给受影响 consumer 产生带 sequence/byte count 的 gap。
- History consumer 失败会暴露错误并释放自己的 cursor，不能永久钉住 Live payload。
- Seal、Flush、Close 和 consumer 退出都必须唤醒 waiter 并释放 budget。

默认值和允许范围见 [tui-v3.example.yaml](../tui/docs/tui-v3.example.yaml)。

## 3. Live revision

daemon 的 screen projector 将 PTY 字节解析为终端画面，并为可交付状态分配严格递增 `live_revision`。revision 只表达同一 terminal generation 和 parser epoch 内的顺序。

发生以下情况时旧增量基线失效：

- output gap。
- terminal generation 变化。
- cols/rows 变化导致画面几何不兼容。
- 客户端基线超过短期 TTL 或被容量清理。
- 客户端报告的 `observed_revision` 不属于当前 protocol session 的 baseline cache。

失效后必须发送全量，不能猜测拼接或跨 gap 比较。

## 4. 客户端基线与拉取

每条 core protocol session 都有独立的短期 baseline map，key 是 terminal ID。session 边界已经隔离不同客户端，不在 Live 请求中再传一份 client key。map 保存 confirmed/offered 画面并由两秒 TTL、单 session 条目/字节限制和 server 总字节限制清理。

请求：

```text
TerminalRef + observed_revision
```

响应只有两种：

- Full：完整当前 screen、最新 revision 和 geometry。
- Delta：相对客户端 `observed_revision` 的 row copy/replace 增量及最新 revision。

处理顺序：

1. 客户端把当前 revision 选入唯一 renderer submission。
2. submission 建立后立即携带该 `observed_revision` 重挂下一次 long-poll，用网络等待与物理渲染重叠。
3. daemon 若已有更新立即返回；否则等待更新或 context cancel。
4. 客户端把返回结果合并到 canonical screen；前一 submission 未完成时只保留最新可提交 damage，不并发写 renderer。
5. 前一 submission 完成后提交已合并的新画面。任何 revision mismatch、gap 或 apply 失败都请求 Full。

这不是“每帧写完才往返”。没有更新时只有一个挂起请求；有连续输出时 long-poll 与当前渲染重叠。慢 renderer 只提交合并后的最新 damage，不会排队一长串已过时画面。

## 5. 渲染边界

- 同一个渲染周期内只提交一次可见更新。
- 渲染中到达的更新保留最新可应用状态，不能并发写 xterm/canvas。
- Delta 不能触发 terminal reset、全屏遮罩或重新创建 renderer。
- Full 只替换终端内容，不重建与当前 terminal/session 无关的 UI。
- 用户离开底部后取消 bottom anchor；回到底部时恢复。
- reconnect 和 generation 切换必须先撤销旧 long-poll，再建立新 baseline。

## 6. 历史数据

history store 保存逻辑行、时间信息和终端重放所需内容。客户端首次以 `latest + limit + cols` 获取冻结 token、history generation、边界和首尾 cursor；随后用 `before_cursor` 向旧内容翻页、用 `after_cursor` 向新内容翻页。协议没有数字 offset，也不一次加载全部历史。

客户端维护：

- 已加载逻辑范围。
- 当前 viewport 在已加载逻辑行中的位置。
- 进入历史时的 Live tail anchor。
- 当前行、总行数与可用时间信息。
- 搜索结果的逻辑行位置。
- 复制 start/end range。

daemon 对每个响应返回连续范围、token/generation、cursor、边界和总量。客户端只能沿已确认 cursor 请求下一页，不能用当前 DOM 行数或自增 offset 推导服务端位置。

## 7. Live 到历史的连续性

进入历史模式：

1. 保留服务端返回的 `HistoryViewportAnchor`（`top_line_id`、`top_cell_offset`、`at_end`）和当前已应用的 Live revision。
2. 首个历史页在当前 Live frame 后方 staging。
3. 第一次真实向旧内容移动时移除 frame hold，并按保存的 viewport anchor 恢复同一视觉位置。
4. 新 Live 输出继续写入 history store，但不修改冻结视口。

退出历史模式：

1. 用户滚动到逻辑尾部时自动退出。
2. 丢弃冻结 anchor 和历史搜索/选择的瞬时 viewport 状态。
3. 使用最后已应用的 Live revision 请求最新画面；基线不可用则 Full resync。
4. 只恢复一次，不能重复拉取 latest 历史页。

## 8. 搜索与复制

- 搜索按逻辑行工作，返回行号和匹配范围，不扫描渲染 canvas。
- 客户端只保留当前结果窗口；跳转到未加载结果时按逻辑位置加载页面。
- 复制选择保存逻辑 start/end，而不是预先拼成巨大字符串。
- 用户确认复制时按范围分块读取；当前 Web 客户端会暂存全部分块并 `join` 成完整字符串后写 clipboard，因此确认阶段内存随复制文本增长。
- selection 跨越仍在 Live、尚未进入历史块的数据时，daemon 使用同一逻辑行空间解析。
- clipboard 与 copy mode 分离；普通粘贴不要求进入历史或复制模式。

## 9. 恢复规则

| 情况 | 恢复 |
| --- | --- |
| Live revision 不连续或 Delta apply 失败 | 请求 Full |
| daemon 没有 client baseline | 返回 Full 并建立新 baseline |
| output gap | parser epoch 变化，返回 Full |
| App 进后台 | 取消请求；前台用最后已应用的 Live revision 恢复 |
| WebView/session generation 变化 | 旧请求结果全部失效 |
| 历史页请求失败 | 保留已加载页面和 viewport，显示可重试错误 |
| 滚动到历史底部 | 自动退出历史并拉最新 Live |

## 10. 验收

- 100000 行输出时内存受配置上限约束。
- 慢客户端不导致 daemon 或客户端待处理帧数持续增长。
- Delta/Full/gap/reconnect/resize 均有确定性测试。
- TUI 与 Android 都覆盖首个历史页、连续向旧分页、Live 冻结和回到底部。
- 终端 canvas 非空，历史切换不闪白、不重建页面、不改变前后行的视觉位置。
