# AnyTTY 开源与私有实现边界

AnyTTY 采用“用户设备代码开源、官方服务器实现私有”的产品边界。私有
monorepo 是开发和生产发布的权威来源；公开仓库由允许列表生成，不继承
monorepo 中包含私有实现的 Git 历史。

## 公开内容

公开快照使用 Apache License 2.0，包含所有交付到用户设备的代码：

- Core daemon、CLI、TUI、共享 UI、Android App 和后续桌面 GUI；
- Local、SSH、Direct、WebRTC、AnyTTY Cloud client 和 Cloud daemon agent；
- pairing、端到端身份认证、授权、ticket 验证和客户端所需 Cloud wire 协议；
- 构建上述客户端所需的生成代码、测试、资源和第三方 notice。

从公开源码构建的 `anytty` 具备连接官方 AnyTTY Cloud 的能力。Local、SSH 和
Direct 不依赖 Cloud 账号或订阅。

## 私有内容

只在 AnyTTY 官方基础设施运行的实现保留在私有 monorepo：

- Cloud Controller、Edge 和 Cloud Web；
- 账号、运营、计费、用量、Relay reservation 和数据库迁移；
- Edge bootstrap、证书签发服务、生产部署、监控和恢复流程；
- 服务端集成测试、商业分析以及生产配置。

私有实现依赖公开协议和客户端安全包；公开代码不得导入
`cloud/controller`、`cloud/edge`、`cloud/web`、`cloud/deploy`、
`cloud/integration` 或 Cloud 服务进程入口。

## 发布方式

`public-source.paths` 是唯一允许列表。新增文件和目录默认保持私有，只有经过
明确审查加入该文件后才会进入公开快照。

```sh
scripts/export-public-source.sh /path/to/new-empty-directory
make check-public-source
```

导出器使用公开仓库专用 README、Makefile、CI、Apache-2.0 LICENSE、NOTICE 和
DCO。`check-public-source` 会从当前工作树生成临时快照，确认私有路径和 import
没有泄漏，再独立执行 Go、CLI、UI 和 Mobile 构建测试。

首次公开发布必须创建全新的 Git 仓库和初始提交。不能把当前 monorepo 改成
Public，也不能仅删除当前版本的私有目录；历史提交仍包含 Controller、Edge、
Cloud Web、迁移和部署实现。

公开仓库初始化后，后续版本从私有 monorepo 单向同步，并保留公开仓库自己的
Git 历史：

```sh
make sync-public-source PUBLIC_DIR=/path/to/public-repository
```

同步目标必须是 Git 仓库，并且包含导出器生成的 `.anytty-public-source` 标记。
同步会让公开工作树严格匹配当前允许列表，因此应先提交或暂存公开仓库中的独立
修改。

## 商标与兼容性

Apache-2.0 允许修改、再发布和商业使用公开代码，但不授予 AnyTTY 名称和标识。
第三方兼容 Controller 或 Edge 不属于官方 AnyTTY Cloud，除非取得明确授权。
官方服务的安全依赖 CA、签名密钥、认证和服务端强制策略，不依赖协议保密。
