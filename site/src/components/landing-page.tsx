import { ArrowRight, Cable, Cloud, Command, FileStack, GitBranch, MonitorSmartphone, Network, QrCode, TerminalSquare } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { buttonVariants } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import { localizedPath, sitePath, type Locale } from "@/lib/site";

const copy = {
  en: {
    eyebrow: "Open-source remote terminals",
    title: "One terminal workflow, wherever you connect.",
    lead: "Run the daemon on your machine. Work through the TUI, CLI, Android, or iOS without handing process ownership to a web dashboard.",
    source: "Get started", repo: "View on GitHub", maturity: "Beta 0.0.1 · downloads available",
    routesTitle: "Reach the same daemon your way.",
    routesLead: "Start locally, reuse SSH access you already operate, or connect directly. AnyTTY Cloud is an optional managed discovery and relay route.",
    routes: [["Local", "Current-user socket"], ["SSH", "OpenSSH tunnel"], ["Direct", "WebRTC / ICE-TCP"], ["Cloud", "Optional P2P or relay"]],
    workflowEyebrow: "CLI + TUI", workflowTitle: "Terminal work stays terminal-shaped.",
    workflowLead: "Create, attach, detach, review retained output, and manage approved file roots without changing context to administer a browser control plane.",
    steps: [["01", "Start the daemon", "A user-owned daemon remains the authority for processes, history, files, and client access."], ["02", "Open the TUI", "Scan terminals, inspect connection state, and move between sessions from one keyboard-first surface."], ["03", "Drop to commands", "Use composable CLI commands for scripts, automation, and direct terminal attachment."]],
    terminalTitle: "anytty · terminals", terminalRows: [["dev-shell", "attached", "local"], ["release", "detached", "ssh: builder"], ["logs", "running", "direct"]],
    appEyebrow: "Android + iOS", appTitle: "Your terminal and project files, in your pocket.", appLead: "Reconnect to running sessions, use a terminal-ready extra-key bar, and browse the same workspace without moving your processes away from their machine.",
    appPoints: ["Pair by QR or pasted claim", "Attach to running terminal sessions", "Preview, upload, and download files"], pairing: "Read pairing docs",
    docsTitle: "The implementation and the operating model are documented together.", docsLead: "Start with the CLI and TUI workflow, then follow the connection, security, mobile, and contribution guides.", docs: "Open documentation",
  },
  "zh-CN": {
    eyebrow: "开源远程终端",
    title: "一套终端工作流，连接不受地点限制。",
    lead: "daemon 运行在你的机器上。通过 TUI、CLI、Android 或 iOS 工作，无需把进程所有权交给网页控制台。",
    source: "快速开始", repo: "查看 GitHub", maturity: "Beta 0.0.1 · 已提供下载",
    routesTitle: "用适合你的方式连接同一个 daemon。",
    routesLead: "从本机开始，复用已有的 SSH，或建立 Direct 连接。AnyTTY Cloud 只是可选的托管发现与 Relay 路径。",
    routes: [["Local", "当前用户 socket"], ["SSH", "OpenSSH 隧道"], ["Direct", "WebRTC / ICE-TCP"], ["Cloud", "可选 P2P 或 Relay"]],
    workflowEyebrow: "CLI + TUI", workflowTitle: "终端工作，保留终端应有的形态。",
    workflowLead: "创建、附加、分离、查看保留输出和管理授权文件根目录，不必切换到浏览器控制平面。",
    steps: [["01", "启动 daemon", "用户拥有的 daemon 始终掌管进程、历史、文件和客户端访问。"], ["02", "打开 TUI", "用一套键盘优先的界面查看终端、连接状态并在会话间切换。"], ["03", "随时回到命令", "用可组合的 CLI 命令完成脚本、自动化和直接终端附加。"]],
    terminalTitle: "anytty · 终端", terminalRows: [["dev-shell", "已附加", "本机"], ["release", "已分离", "ssh: builder"], ["logs", "运行中", "direct"]],
    appEyebrow: "Android + iOS", appTitle: "把终端和项目文件带在身边。", appLead: "随时回到正在运行的会话，使用面向终端的扩展键盘，并浏览同一工作区；进程始终留在原来的机器上。",
    appPoints: ["扫描二维码或粘贴 claim 配对", "连接正在运行的终端会话", "预览、上传和下载文件"], pairing: "阅读配对文档",
    docsTitle: "实现细节与运行模型放在同一套文档里。", docsLead: "先了解 CLI 与 TUI 工作流，再查看连接、安全、移动端和贡献指南。", docs: "打开文档",
  },
} as const;

const routeIcons = [Cable, Network, MonitorSmartphone, Cloud];

export function LandingPage({ locale }: { locale: Locale }) {
  const t = copy[locale];
  const docsPath = localizedPath(locale, "/docs/quick-start");
  const pairingPath = localizedPath(locale, "/docs/pairing");
  return <>
    <main id="main">
      <section className="hero-section">
        <div className="site-container hero-grid">
          <div className="hero-copy">
            <Badge className="border-primary/20 bg-primary/5 text-primary">{t.eyebrow}</Badge>
            <h1>{t.title}</h1>
            <p>{t.lead}</p>
            <div className="flex flex-wrap gap-3">
              <a className={cn(buttonVariants({ size: "lg" }))} href={sitePath(docsPath)}>{t.source}<ArrowRight className="size-4" /></a>
              <a className={cn(buttonVariants({ variant: "outline", size: "lg" }))} href="https://github.com/anytty/anytty"><GitBranch className="size-4" />{t.repo}</a>
            </div>
            <p className="maturity"><span aria-hidden="true" />{t.maturity}</p>
          </div>
          <TerminalSurface title={t.terminalTitle} rows={t.terminalRows} />
        </div>
      </section>

      <section className="route-section" aria-labelledby="routes-heading">
        <div className="site-container">
          <div className="section-heading"><div><h2 id="routes-heading">{t.routesTitle}</h2><p>{t.routesLead}</p></div></div>
          <div className="route-grid">
            {t.routes.map(([name, detail], index) => { const Icon = routeIcons[index]; return <div className="route-item" key={name}><Icon className="size-5" aria-hidden="true" /><div><h3>{name}</h3><p>{detail}</p></div></div>; })}
          </div>
        </div>
      </section>

      <section className="workflow-section" aria-labelledby="workflow-heading">
        <div className="site-container workflow-grid">
          <div className="workflow-copy"><p className="section-label">{t.workflowEyebrow}</p><h2 id="workflow-heading">{t.workflowTitle}</h2><p>{t.workflowLead}</p></div>
          <div className="workflow-steps">
            {t.steps.map(([number, title, body]) => <article key={number}><span>{number}</span><div><h3>{title}</h3><p>{body}</p></div></article>)}
          </div>
        </div>
      </section>

      <section className="app-section" aria-labelledby="app-heading">
        <div className="site-container app-grid">
          <div className="app-media"><img src={sitePath("/assets/mobile-terminal.png")} alt={locale === "en" ? "AnyTTY Android remote terminal" : "AnyTTY Android 远程终端"} width="1080" height="2400" /></div>
          <div className="app-copy"><p className="section-label">{t.appEyebrow}</p><h2 id="app-heading">{t.appTitle}</h2><p>{t.appLead}</p><ul>{t.appPoints.map((item, index) => { const Icon = [QrCode, TerminalSquare, FileStack][index]; return <li key={item}><Icon className="size-5" /><span>{item}</span></li>; })}</ul><a href={sitePath(pairingPath)}>{t.pairing}<ArrowRight className="size-4" /></a></div>
        </div>
      </section>

      <section className="docs-cta">
        <div className="site-container"><div><Command className="size-6" /><h2>{t.docsTitle}</h2><p>{t.docsLead}</p></div><a className={cn(buttonVariants({ size: "lg" }))} href={sitePath(docsPath)}>{t.docs}<ArrowRight className="size-4" /></a></div>
      </section>
    </main>
  </>;
}

function TerminalSurface({ title, rows }: { title: string; rows: readonly (readonly [string, string, string])[] }) {
  return <div className="terminal-surface" aria-label={title}>
    <div className="terminal-bar"><span>{title}</span><span>?</span></div>
    <div className="terminal-command"><span>$</span> anytty terminal list</div>
    <div className="terminal-table"><div className="terminal-head"><span>NAME</span><span>STATE</span><span>ENDPOINT</span></div>{rows.map((row, index) => <div className={cn("terminal-row", index === 0 && "selected")} key={row[0]}><span>{index === 0 ? "› " : "  "}{row[0]}</span><span>{row[1]}</span><span>{row[2]}</span></div>)}</div>
    <div className="terminal-help"><span><kbd>enter</kbd> attach</span><span><kbd>n</kbd> new</span><span><kbd>q</kbd> quit</span></div>
  </div>;
}
