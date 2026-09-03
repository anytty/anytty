# Web Session 冷切换

## 目标

浏览器的页面由手机本地 WebView 渲染，网络请求、DNS 解析和目标连接通过当前 endpoint 的远程 session 完成。一个 endpoint 对应一个 Web session，但手机上始终只保留一个活跃 WebView。

切换 session 时不保留后台运行的 WebView：先保存现场并移除当前实例、清理共享网页数据、关闭旧 lease，再建立目标 session 的代理 lease，最后创建新的 WebView。这样页面中的 JavaScript、定时器、WebSocket 和未完成请求都不能继续使用另一台设备的路由。

## Flutter 侧已实现

Flutter 侧的实现位于 `features/browser`：

- session 快照按 `endpointId` 独立保存；元数据进入 `SharedPreferences`，页面 PNG 进入应用 support 目录，每个 session 最多一张。
- 快照保存 URL、标题、滚动位置、route id、route generation 和本地图片路径，不保存不可控的大型页面对象。它是视觉预览和恢复入口，不等同于完整浏览器 profile。
- 任意切换操作串行执行，并带 operation token。旧 WebView 的迟到回调不能更新新 session 的地址栏、标题或状态。
- 新 WebView 只有在 `BrowserProxyLease` 成功返回后才会创建和加载 URL。代理 channel 不可用时，页面保持快照或显示等待状态，不会退回手机直连。
- 每次创建目标 WebView 前都会调用 native `clearData`。Android 清理 Cookie、WebStorage 和 HTTP auth；iOS 清理默认 `WKWebsiteDataStore` 的 website data。当前策略不恢复旧 session 的 cookie/storage，避免共享 store 串扰。
- WebView 使用 unrestricted JavaScript，这是正常浏览器页面所需的能力；最终安全边界必须由 session-bound route 和 native 冷销毁保证。

## 切换时序

```text
active A
  -> capture URL/title/scroll/PNG
  -> remove WebView from PlatformView tree
  -> wait one frame and request native view teardown
  -> close A proxy lease and its sockets
  -> clear shared WebView data
  -> open B proxy lease
  -> create WebView and bind B route
  -> load B snapshot URL
  -> show live page after navigation finishes
```

不能把 `WebView` 仅仅隐藏或放到 `Offstage`。隐藏不会停止 JavaScript，也不能证明网络连接已经关闭。Flutter 侧会移除 widget、丢弃 controller 并等待一帧；要达到可证明的即时释放，native platform view 还必须显式停止加载、断开代理 lease 并销毁 WebView。不能把 Dart 对象进入 GC 当成安全边界。

## Native proxy channel

channel 名称：`com.anytty.app/browser-proxy`

### `open`

请求：

```json
{
  "sessionId":"endpoint-a",
  "endpointId":"endpoint-a",
  "proxyHost":"127.0.0.1",
  "proxyPort":41237,
  "routeId":"direct-or-cloud-route",
  "routeGeneration":12
}
```

成功返回：

```json
{
  "leaseId":"lease-...",
  "sessionId":"endpoint-a",
  "endpointId":"endpoint-a",
  "routeId":"direct-or-cloud-route",
  "routeGeneration":12,
  "dnsProxied":true
}
```

native 实现必须在返回前完成本地代理或 WebView 的 per-instance route 配置。`sessionId`、`endpointId` 和 `routeId` 不匹配时 Flutter 会拒绝 lease。

Android 通过 AndroidX WebKit `ProxyController` 把当前 WebView provider 的所有 scheme 指向 Flutter 在 loopback 上启动的 HTTP proxy，并移除 implicit rules。iOS 17 及以上通过默认 `WKWebsiteDataStore` 的 HTTP CONNECT proxy configuration 完成同样的绑定。WebView 在代理配置成功后才创建，因此本地负责渲染，目标连接和 DNS 由 Go daemon 通过当前 endpoint 的 browser resource 完成。

### `close`

请求：

```json
{"leaseId":"lease-..."}
```

native 实现必须关闭该 lease 产生的 listener、socket、DNS resolver 和远程 stream，并保证重复 close 幂等。

Go client/daemon 已增加 `BROWSER_PROXY` resource：Flutter loopback proxy 每接收一个 HTTP CONNECT 或 HTTP 请求，就为该 TCP 连接打开一个独立 resource；数据以有界的 binary stream 双向转发，daemon 在远程 session 所在机器上执行目标 TCP dial。目标 hostname 不在手机上解析，HTTP CONNECT 对 HTTPS、WebSocket 和其他 TLS 流量保持端到端字节转发，因此 DNS 和 TCP 都留在远程 session。Android WebView 显式移除 localhost、`127.0.0.0/8` 和 link-local 地址的隐式 bypass，所以 `127.0.0.1` 指向远程 session 机器的 loopback，而不是手机本地 loopback。

当前实现限制为一个可见浏览器实例和一个 active native proxy lease。多 session 通过冷切换隔离：目标设备先完成 session provider、代理 listener 和快照准备，准备失败时当前 WebView 保持不变；准备成功后才关闭旧连接、从 widget tree 移除并销毁旧 WebView、清理 shared website data，再创建新实例。最后一步代理绑定发生竞态失败时，会自动排队恢复原 session。Android 需要 WebView provider 支持 `PROXY_OVERRIDE`；iOS 需要 iOS 17 或更高版本。能力不满足时会 fail closed，页面不会退回手机直连。

冷恢复只承诺 URL、滚动位置和 PNG 快照，不恢复 cookie、localStorage、sessionStorage 或 service worker。Android 的清理覆盖 Cookie、WebStorage 和 HTTP auth，平台缓存/service worker 的完整清除仍需要 native 集成测试；如果产品最终需要保留登录态，则必须设计加密、按 session 分区的 browser-state adapter，不能把共享 WebView store 当作 profile。

另外，当前 iOS deployment target 是 15.0。WebKit 的 `WKWebsiteDataStore.proxyConfigurations` 从 iOS 17 才可用，因此 iOS 端不能直接照搬 Android 的全局 ProxyController：需要提高最低系统版本，或者为 iOS 15 单独实现本地代理/自定义 WebView platform view，并在能力不可用时继续保持“浏览器不启动”的安全行为。

## UI

终端工作区顶部增加 Web 图标，进入 `/browser/:endpointId`。浏览器页面提供：

- 地址栏、后退、前进、打开和刷新；窄屏将后退/前进收进溢出菜单。
- 设备按钮打开 Web session 列表；切换在同一个页面状态机内完成，不依靠路由替换绕过冷停顺序。目标准备期间仍显示当前设备，失败时可继续使用当前页面。
- route / DNS 状态收纳在设备状态点、tooltip 和溢出菜单中，不再单独占用一层工具栏。
- 地址栏获得焦点时会收缩设备选择器并平滑展开，方便查看和编辑完整 URL。
- 冷停期间显示本地快照，并标记为暂停预览；恢复完成后才替换为 live WebView。

## 后续验收

1. A 页面访问 `127.0.0.1`，切到 B 后 A 的 JavaScript、WebSocket 和请求全部停止。
2. B 访问相同地址时只能得到 B 机器的响应。
3. 切回 A 后恢复 A 的 URL、滚动位置和快照；native browser-state adapter 完成后，再验收 cookie/data store 不读取 B 的状态。
4. route generation 变化时旧 lease 不能复用，DNS 必须和 TCP/CONNECT 使用同一远程 session。
5. Android renderer 被系统回收、iOS WebContent process 终止或代理断线时，都只能回到快照/重试状态，不能静默直连。
