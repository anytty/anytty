import {
  ArrowDownToLine,
  ArrowRight,
  Clock3,
  Copy,
  Eye,
  FileStack,
  GitFork,
  History,
  Laptop,
  MonitorSmartphone,
  PanelsTopLeft,
  Radio,
  ShieldCheck,
  Smartphone,
  TerminalSquare,
} from "lucide-react";
import { localizedPath, sitePath, type Locale } from "@/lib/site";

const copy = {
  en: {
    eyebrow: "Open source · macOS · Linux · Windows · Android · iOS",
    promise: "Keep terminals running. Take control from anywhere.",
    lead: "AnyTTY keeps terminal sessions alive on machines you own, then lets the TUI, CLI, WebView, and mobile apps observe and control them without owning their lifetime.",
    quickStart: "Quick start",
    github: "GitHub",
    release: "Download beta",
    installLabel: "Install on macOS or Linux",
    installCommand: "curl -fsSL https://raw.githubusercontent.com/anytty/anytty/main/install.sh | sh",
    copyCommand: "Copy install command",
    copied: "Copied",
    previewLabel: "A real AnyTTY workspace",
    previewMeta: "local + remote · 5 terminals · one workspace",
    previewAlt: "AnyTTY TUI showing local and remote terminals in split and floating panels",
    observerEyebrow: "Terminal is not a panel",
    observerTitle: "Your layout is a view. The terminal keeps living behind it.",
    observerLead: "Workspaces and panels organize how you look at terminals; they do not own or stay bound to them. Switch the terminal shown by a panel without rebuilding the layout or restarting the task.",
    runtime: "terminal",
    runtimeState: "running on your machine",
    snapshots: "screen snapshots",
    views: ["TUI panel", "CLI", "WebView", "Mobile"],
    observerPoints: [
      ["Pick any terminal", "The Terminal Picker changes what the current panel observes while every split, float, and workspace stays exactly where it is."],
      ["Slow views stay isolated", "Every consumer reads snapshots at its own pace. A slow phone or network never makes the process inside the terminal wait."],
    ],
    workbenchEyebrow: "One workbench",
    workbenchTitle: "Local and remote terminals belong together.",
    workbenchLead: "Windows is a first-class platform, not a compatibility afterthought. Local, SSH, and Direct terminals all enter the same picker and behave like part of one workspace.",
    windowsAlt: "AnyTTY TUI running on Windows with multiple terminals",
    workbenchPoints: [
      ["Native Windows", "CLI, TUI, and daemon builds for x64 and ARM64, with no WSL requirement for core workflows."],
      ["Every endpoint", "Operate a workstation, build machine, or remote AI agent exactly as you operate a local terminal."],
      ["Recent activity", "See which terminal stopped producing output and find stalled tasks or agents without opening every session."],
    ],
    historyEyebrow: "History that outlives the screen",
    historyTitle: "Long-running work should not consume long-growing memory.",
    historyLead: "Output is written to files while the live path stays bounded. AnyTTY sets no fixed history line limit: practical capacity is limited by available disk space, so history can keep growing as long as disk capacity does.",
    historyStats: [
      ["Live memory", "Bounded"],
      ["History", "Disk-limited"],
      ["Reconnect", "Full context"],
    ],
    historyNote: "Useful after a disconnect, an accidental exit, or an unattended agent run that produced hours of output.",
    mobileEyebrow: "Android + iOS",
    mobileTitle: "Know what is alive. Step in when it matters.",
    mobileLead: "The mobile app is built for terminal work, not a desktop page squeezed onto a phone. Find recent activity, attach with a touch-optimized extra-key row, and manage files over the same connection.",
    mobilePoints: [
      ["Terminal activity", "Scan running sessions and time since last output."],
      ["Touch-ready control", "Use Esc, Ctrl, arrows, paging, and common terminal actions."],
      ["Files and previews", "Browse, upload, download, and preview remote files."],
    ],
    mobileAlts: ["AnyTTY mobile terminal list with recent activity", "AnyTTY mobile terminal with an extra-key row", "AnyTTY mobile remote file manager"],
    tabletEyebrow: "Tablet workflows",
    tabletTitle: "A full working surface when a phone is not enough.",
    tabletLead: "Use split-screen agent workflows or run terminal-native tools directly from the tablet client.",
    tabletCaptions: ["Codex and OpenCode in a vertical split", "The TTT editor running inside a terminal"],
    tabletAlts: ["Tablet split screen running Codex and OpenCode", "TTT editor running inside an AnyTTY terminal on a tablet"],
    installEyebrow: "Beta 0.0.1",
    installTitle: "Run it on the machine you already use.",
    installLead: "The release includes binaries for macOS, Linux, and Windows, plus an unsigned Android beta APK. iOS is supported; public distribution is waiting for the App Store. Google Play and App Store submissions are in progress.",
    unix: "macOS / Linux",
    windows: "Windows PowerShell",
    windowsCommand: "irm https://raw.githubusercontent.com/anytty/anytty/main/install.ps1 -OutFile install.ps1; ./install.ps1",
    docs: "Read the documentation",
    releases: "Open GitHub Releases",
    platforms: ["macOS", "Linux", "Windows", "Android", "iOS"],
    openSource: "Apache-2.0 · source, clients, and release artifacts on GitHub",
  },
  "zh-CN": {
    eyebrow: "开源 · macOS · Linux · Windows · Android · iOS",
    promise: "让终端持续运行，随时随地接管现场。",
    lead: "AnyTTY 让终端会话持续运行在你自己的机器上，再由 TUI、CLI、WebView 和手机 App 观察与控制；任何界面都不拥有终端的生命周期。",
    quickStart: "快速开始",
    github: "GitHub",
    release: "下载 Beta",
    installLabel: "安装到 macOS 或 Linux",
    installCommand: "curl -fsSL https://raw.githubusercontent.com/anytty/anytty/main/install.sh | sh",
    copyCommand: "复制安装命令",
    copied: "已复制",
    previewLabel: "真实的 AnyTTY 工作台",
    previewMeta: "本地 + 远程 · 5 个终端 · 一个工作区",
    previewAlt: "AnyTTY TUI 通过分屏和浮动 Panel 同时展示本地与远程终端",
    observerEyebrow: "Terminal 不等于 Panel",
    observerTitle: "布局只是观察窗口，Terminal 在它背后持续运行。",
    observerLead: "Workspace 和 Panel 只负责组织你如何观察终端，并不拥有、也不绑定某个 Terminal。切换当前 Panel 所观察的 Terminal，不需要重建布局，更不需要重启任务。",
    runtime: "terminal",
    runtimeState: "持续运行在你的机器上",
    snapshots: "画面快照",
    views: ["TUI Panel", "CLI", "WebView", "手机 App"],
    observerPoints: [
      ["随时选择任意 Terminal", "Terminal Picker 只改变当前 Panel 观察的对象，已有分屏、浮动窗和 Workspace 布局保持原样。"],
      ["慢消费者互不影响", "每个消费者都按自己的节奏读取快照。手机或网络再慢，也不会让 Terminal 中运行的程序等待。"],
    ],
    workbenchEyebrow: "一个工作台",
    workbenchTitle: "本地与远程 Terminal，本来就该在一起。",
    workbenchLead: "Windows 是默认支持的平台，而不是兼容性补丁。Local、SSH 和 Direct Terminal 都进入同一个 Picker，像同一工作区里的本地任务一样操作。",
    windowsAlt: "AnyTTY TUI 在 Windows 上管理多个终端",
    workbenchPoints: [
      ["原生支持 Windows", "CLI、TUI 和 daemon 均提供 x64 与 ARM64 构建，核心工作流无需依赖 WSL。"],
      ["统一操作远程对象", "工作站、构建机或远程 AI Agent，都可以像本地 Terminal 一样直接进入工作流。"],
      ["查看近期活动", "一眼发现长时间没有新输出的 Terminal，无需逐个打开就能定位停滞的任务或 Agent。"],
    ],
    historyEyebrow: "不随屏幕消失的历史",
    historyTitle: "长时间运行，不应该换来持续增长的内存。",
    historyLead: "输出持续写入文件，实时链路只保留有界数据。AnyTTY 不预设历史行数上限，实际容量只受可用磁盘空间限制；只要磁盘容量继续增长，历史记录就可以继续增长。",
    historyStats: [
      ["实时内存", "有界"],
      ["历史容量", "取决于磁盘"],
      ["重新连接", "完整上下文"],
    ],
    historyNote: "连接中断、程序意外退出，或 AI Agent 挂机执行数小时后，都能快速回到完整现场。",
    mobileEyebrow: "Android + iOS",
    mobileTitle: "先知道什么还活着，再在需要时直接接管。",
    mobileLead: "手机 App 是为终端工作专门设计的，不是把桌面网页塞进小屏幕。查看近期活动，通过触控优化的扩展按键接管，再用同一条连接管理远程文件。",
    mobilePoints: [
      ["Terminal 活动", "集中查看运行状态与距离上次输出的时间。"],
      ["触控接管", "直接使用 Esc、Ctrl、方向键、翻页键和常用终端操作。"],
      ["文件与预览", "浏览、上传、下载并在线预览远程文件。"],
    ],
    mobileAlts: ["AnyTTY 手机 App 展示终端列表与近期活动", "AnyTTY 手机 App 使用扩展按键操作终端", "AnyTTY 手机 App 的远程文件管理器"],
    tabletEyebrow: "平板工作流",
    tabletTitle: "当手机屏幕不够时，直接展开完整工作现场。",
    tabletLead: "上下分屏运行不同 Agent，或在平板客户端中直接使用终端原生工具。",
    tabletCaptions: ["上下分屏同时运行 Codex 与 OpenCode", "在终端内运行 TTT 编辑器"],
    tabletAlts: ["平板上下分屏运行 Codex 与 OpenCode", "在平板 AnyTTY 终端内运行 TTT 编辑器"],
    installEyebrow: "Beta 0.0.1",
    installTitle: "在你已经使用的机器上直接运行。",
    installLead: "Release 已提供 macOS、Linux、Windows 二进制和未签名的 Android Beta APK。手机 App 同时支持 iOS，但公开安装仍需等待 App Store；Google Play 与 App Store 的上架工作都在进行中。",
    unix: "macOS / Linux",
    windows: "Windows PowerShell",
    windowsCommand: "irm https://raw.githubusercontent.com/anytty/anytty/main/install.ps1 -OutFile install.ps1; ./install.ps1",
    docs: "阅读使用文档",
    releases: "打开 GitHub Releases",
    platforms: ["macOS", "Linux", "Windows", "Android", "iOS"],
    openSource: "Apache-2.0 · 源码、客户端与构建产物均在 GitHub",
  },
} as const;

const releaseUrl = "https://github.com/anytty/anytty/releases/tag/v0.0.1-beta.0";
const repoUrl = "https://github.com/anytty/anytty";

export function LandingPage({ locale }: { locale: Locale }) {
  const t = copy[locale];
  const docsPath = sitePath(localizedPath(locale, "/docs/quick-start"));

  return (
    <main id="main" className="landing-page">
      <section className="product-hero" aria-labelledby="product-title">
        <div className="site-container hero-intro">
          <img className="hero-mark" src={sitePath("/assets/app-icon.png")} alt="" width="72" height="72" />
          <p className="hero-eyebrow">{t.eyebrow}</p>
          <h1 id="product-title">AnyTTY</h1>
          <p className="hero-promise">{t.promise}</p>
          <p className="hero-lead">{t.lead}</p>
          <div className="hero-actions">
            <a className="landing-button landing-button-primary" href={docsPath}>{t.quickStart}<ArrowRight aria-hidden="true" /></a>
            <a className="landing-button landing-button-secondary" href={repoUrl}><GitFork aria-hidden="true" />{t.github}</a>
            <a className="landing-link" href={releaseUrl}><ArrowDownToLine aria-hidden="true" />{t.release}</a>
          </div>
          <InstallCommand
            command={t.installCommand}
            label={t.installLabel}
            copyLabel={t.copyCommand}
            copied={t.copied}
          />
        </div>

        <div className="site-container hero-product">
          <div className="product-frame-meta">
            <span><Radio aria-hidden="true" />{t.previewLabel}</span>
            <span>{t.previewMeta}</span>
          </div>
          <img
            className="hero-screenshot"
            src={sitePath("/assets/product/mac-tui.webp")}
            alt={t.previewAlt}
            width="1800"
            height="1125"
          />
          <ObserverStrip t={t} />
        </div>
      </section>

      <section id="features" className="observer-section" aria-labelledby="observer-title">
        <div className="site-container section-split-heading">
          <p className="section-kicker">{t.observerEyebrow}</p>
          <div>
            <h2 id="observer-title">{t.observerTitle}</h2>
            <p>{t.observerLead}</p>
          </div>
        </div>
        <div className="site-container observer-diagram" aria-label={t.observerTitle}>
          <div className="runtime-node">
            <TerminalSquare aria-hidden="true" />
            <strong>{t.runtime}</strong>
            <span>{t.runtimeState}</span>
          </div>
          <div className="snapshot-line"><span>{t.snapshots}</span></div>
          <div className="consumer-list">
            {t.views.map((view, index) => {
              const Icon = [PanelsTopLeft, TerminalSquare, Laptop, Smartphone][index];
              return <div key={view}><Icon aria-hidden="true" /><span>{view}</span></div>;
            })}
          </div>
        </div>
        <div className="site-container observer-points">
          {t.observerPoints.map(([title, body], index) => {
            const Icon = index === 0 ? Eye : ShieldCheck;
            return <article key={title}><Icon aria-hidden="true" /><div><h3>{title}</h3><p>{body}</p></div></article>;
          })}
        </div>
      </section>

      <section className="workbench-section" aria-labelledby="workbench-title">
        <div className="site-container dark-heading">
          <p className="section-kicker">{t.workbenchEyebrow}</p>
          <div><h2 id="workbench-title">{t.workbenchTitle}</h2><p>{t.workbenchLead}</p></div>
        </div>
        <div className="site-container workbench-media">
          <img src={sitePath("/assets/product/win-tui.webp")} alt={t.windowsAlt} width="1800" height="955" loading="lazy" />
        </div>
        <div className="site-container proof-grid">
          {t.workbenchPoints.map(([title, body], index) => {
            const Icon = [MonitorSmartphone, Laptop, Clock3][index];
            return <article key={title}><Icon aria-hidden="true" /><h3>{title}</h3><p>{body}</p></article>;
          })}
        </div>
      </section>

      <section className="history-section" aria-labelledby="history-title">
        <div className="site-container history-grid">
          <div className="history-copy">
            <p className="section-kicker">{t.historyEyebrow}</p>
            <h2 id="history-title">{t.historyTitle}</h2>
            <p>{t.historyLead}</p>
            <p className="history-note"><History aria-hidden="true" />{t.historyNote}</p>
          </div>
          <dl className="history-stats">
            {t.historyStats.map(([term, detail]) => <div key={term}><dt>{term}</dt><dd>{detail}</dd></div>)}
          </dl>
        </div>
      </section>

      <section id="screens" className="mobile-section" aria-labelledby="mobile-title">
        <div className="site-container mobile-heading">
          <div>
            <p className="section-kicker">{t.mobileEyebrow}</p>
            <h2 id="mobile-title">{t.mobileTitle}</h2>
            <p>{t.mobileLead}</p>
          </div>
          <div className="mobile-points">
            {t.mobilePoints.map(([title, body], index) => {
              const Icon = [Clock3, TerminalSquare, FileStack][index];
              return <article key={title}><Icon aria-hidden="true" /><div><h3>{title}</h3><p>{body}</p></div></article>;
            })}
          </div>
        </div>
        <div className="site-container phone-gallery">
          {["mobile-terminals.webp", "mobile-terminal.webp", "mobile-files.webp"].map((image, index) => (
            <figure key={image}><img src={sitePath(`/assets/product/${image}`)} alt={t.mobileAlts[index]} width="480" height="1068" loading="lazy" /></figure>
          ))}
        </div>
      </section>

      <section className="tablet-section" aria-labelledby="tablet-title">
        <div className="site-container tablet-heading">
          <p className="section-kicker">{t.tabletEyebrow}</p>
          <div><h2 id="tablet-title">{t.tabletTitle}</h2><p>{t.tabletLead}</p></div>
        </div>
        <div className="site-container tablet-gallery">
          {["pad-agents.webp", "pad-ttt-editor.webp"].map((image, index) => (
            <figure key={image}>
              <img src={sitePath(`/assets/product/${image}`)} alt={t.tabletAlts[index]} width="900" height={index === 0 ? "1179" : "1146"} loading="lazy" />
              <figcaption>{t.tabletCaptions[index]}</figcaption>
            </figure>
          ))}
        </div>
      </section>

      <section id="install" className="install-section" aria-labelledby="install-title">
        <div className="site-container install-heading">
          <p className="section-kicker">{t.installEyebrow}</p>
          <div><h2 id="install-title">{t.installTitle}</h2><p>{t.installLead}</p></div>
        </div>
        <div className="site-container platform-list" aria-label={t.platforms.join(", ")}>
          {t.platforms.map((platform) => <span key={platform}>{platform}</span>)}
        </div>
        <div className="site-container install-grid">
          <InstallCommand command={t.installCommand} label={t.unix} copyLabel={t.copyCommand} copied={t.copied} />
          <InstallCommand command={t.windowsCommand} label={t.windows} copyLabel={t.copyCommand} copied={t.copied} />
        </div>
        <div className="site-container install-footer">
          <p>{t.openSource}</p>
          <div>
            <a className="landing-button landing-button-primary" href={docsPath}>{t.docs}<ArrowRight aria-hidden="true" /></a>
            <a className="landing-button landing-button-secondary" href={releaseUrl}>{t.releases}<ArrowDownToLine aria-hidden="true" /></a>
          </div>
        </div>
      </section>
    </main>
  );
}

function ObserverStrip({ t }: { t: (typeof copy)[Locale] }) {
  return (
    <div className="observer-strip" aria-hidden="true">
      <span className="observer-source"><TerminalSquare />{t.runtime}</span>
      <span className="observer-track" />
      {t.views.map((view, index) => <span key={view} className={index === 0 ? "is-active" : ""}>{view}</span>)}
    </div>
  );
}

function InstallCommand({ command, label, copyLabel, copied }: { command: string; label: string; copyLabel: string; copied: string }) {
  return (
    <div className="install-command">
      <span className="install-command-label">{label}</span>
      <div>
        <code><span aria-hidden="true">$ </span>{command}</code>
        <button type="button" data-copy-command={command} aria-label={copyLabel} title={copyLabel}><Copy aria-hidden="true" /></button>
      </div>
      <span className="copy-feedback" aria-live="polite">{copied}</span>
    </div>
  );
}
