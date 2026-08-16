export type PrivacyPolicyLanguage = 'en' | 'zh-CN'

type PrivacyPolicyCopy = {
  title: string
  updated: string
  introduction: string
  sections: Array<{ title: string; paragraphs?: string[]; items?: string[] }>
  contact: string
}

const privacyPolicyCopy: Record<PrivacyPolicyLanguage, PrivacyPolicyCopy> = {
  en: {
    title: 'AnyTTY Privacy Policy',
    updated: 'Last updated: August 15, 2026',
    introduction: 'This policy explains how the AnyTTY mobile app and its optional Cloud connection service process information. The AnyTTY app does not require an account.',
    sections: [
      {
        title: 'Information stored on your device',
        items: [
          'Paired endpoint configurations, display names, icons, app preferences, and transfer history are stored locally.',
          'Pairing credentials and SSH private keys are stored using operating-system protected storage when available.',
          'Downloaded files remain in locations you select or in app-scoped storage. AnyTTY does not upload this local information merely because it is stored by the app.',
        ],
      },
      {
        title: 'Camera and QR pairing',
        paragraphs: ['Camera access is used only when you choose to scan a pairing QR code. Camera frames are processed on the device for code recognition and are not uploaded or retained by AnyTTY.'],
      },
      {
        title: 'Remote connections and Cloud routing',
        items: [
          'Commands, terminal output, and files are transmitted to or from the remote device you explicitly paired. Connection content is encrypted in transit.',
          'For Cloud-routed connections, AnyTTY Cloud processes routing grants, pseudonymous endpoint, device, and Edge identifiers, source IP addresses, connection timing and status, and encrypted traffic volume needed to route, secure, operate, and account for the service.',
          'Relay infrastructure forwards encrypted terminal and file traffic and is not designed to decrypt its contents. Direct and SSH connections communicate with the selected remote device without sending session content to AnyTTY Cloud.',
        ],
      },
      {
        title: 'Diagnostics, analytics, and advertising',
        paragraphs: ['The app does not contain advertising SDKs or third-party analytics SDKs. Operational diagnostics use redacted categories, states, and timing values. Diagnostic logs are not uploaded automatically.'],
      },
      {
        title: 'How information is used and shared',
        paragraphs: ['Information is used to establish requested connections, prevent abuse, maintain reliability, enforce service limits, and support billing for optional Cloud service. AnyTTY does not sell personal information. Information may be handled by infrastructure providers acting on our behalf, or disclosed when required by law or necessary to protect users and the service.'],
      },
      {
        title: 'Retention and deletion',
        paragraphs: ['Local app data remains until you remove a pairing, reset local pairings, delete the corresponding transfer record or file, or uninstall the app. Cloud routing, security, and usage records are retained only as needed to operate the service, prevent abuse, meet accounting or legal obligations, and resolve disputes. You may request deletion of data associated with you by contacting us.'],
      },
      {
        title: 'Security and children',
        paragraphs: ['AnyTTY uses encrypted transport, scoped pairing credentials, and platform-protected credential storage. No system can guarantee absolute security. The service is intended for people who are authorized to access the paired computers and is not directed to children under 13.'],
      },
      {
        title: 'Changes',
        paragraphs: ['We may update this policy when the app, service, or legal requirements change. The current version and its effective date will remain available on this page.'],
      },
    ],
    contact: 'Privacy questions and deletion requests: privacy@anytty.com',
  },
  'zh-CN': {
    title: 'AnyTTY 隐私政策',
    updated: '最后更新：2026 年 8 月 15 日',
    introduction: '本政策说明 AnyTTY 移动应用及其可选 Cloud 连接服务如何处理信息。AnyTTY App 本身无需注册账号。',
    sections: [
      {
        title: '保存在设备上的信息',
        items: [
          '已配对端点配置、显示名称、图标、应用偏好和传输记录保存在本机。',
          '配对凭据和 SSH 私钥会在系统支持时保存在操作系统保护的安全存储中。',
          '下载文件保存在你选择的位置或应用专属目录中。AnyTTY 不会仅因为这些信息保存在 App 内就将其上传。',
        ],
      },
      {
        title: '摄像头与二维码配对',
        paragraphs: ['只有在你主动扫描配对二维码时才会使用摄像头。画面仅在设备上用于识别二维码，不会上传，也不会由 AnyTTY 保存。'],
      },
      {
        title: '远程连接与 Cloud 路由',
        items: [
          '命令、终端输出和文件只会传输到你明确配对的远程设备或从该设备传回，连接内容在传输过程中加密。',
          '使用 Cloud 路由时，AnyTTY Cloud 会处理路由授权、经过假名化的端点、设备和 Edge 标识、来源 IP、连接时间与状态，以及提供路由、安全、运营和用量结算所必需的加密流量统计。',
          'Relay 基础设施只转发加密的终端和文件流量，其设计不能解密这些内容。Direct 和 SSH 连接直接与所选远程设备通信，不会把会话内容发送给 AnyTTY Cloud。',
        ],
      },
      {
        title: '诊断、分析与广告',
        paragraphs: ['App 不包含广告 SDK 或第三方分析 SDK。运行诊断仅使用已脱敏的类别、状态和耗时信息，诊断日志不会自动上传。'],
      },
      {
        title: '信息用途与共享',
        paragraphs: ['信息仅用于建立用户请求的连接、防止滥用、维持可靠性、执行服务额度及支持可选 Cloud 服务计费。AnyTTY 不出售个人信息。信息可能由代表我们提供基础设施的服务商处理，或在法律要求以及保护用户和服务所必需时披露。'],
      },
      {
        title: '保留与删除',
        paragraphs: ['本地数据会保留到你移除配对、重置本地配对、删除相应传输记录或文件，或卸载 App。Cloud 路由、安全和用量记录只会在运营服务、防止滥用、履行财务或法律义务及解决争议所需的期限内保留。你可以联系我们，请求删除与你相关的数据。'],
      },
      {
        title: '安全与未成年人',
        paragraphs: ['AnyTTY 使用加密传输、限定范围的配对凭据和系统保护的凭据存储，但任何系统都无法承诺绝对安全。本服务仅供获授权访问已配对电脑的用户使用，不面向 13 岁以下儿童。'],
      },
      {
        title: '政策变更',
        paragraphs: ['当 App、服务或法律要求发生变化时，我们可能更新本政策。当前版本及其生效日期会持续在本页面提供。'],
      },
    ],
    contact: '隐私问题及删除请求：privacy@anytty.com',
  },
}

export function PrivacyPolicyContent({ language, className = '' }: { language: PrivacyPolicyLanguage; className?: string | undefined }) {
  const copy = privacyPolicyCopy[language]
  return (
    <article className={`text-[var(--foreground)] ${className}`} lang={language}>
      <header>
        <h1 className="text-2xl font-semibold leading-tight">{copy.title}</h1>
        <p className="mt-2 text-xs text-[var(--muted-foreground)]">{copy.updated}</p>
        <p className="mt-4 text-sm leading-6 text-[var(--muted-foreground)]">{copy.introduction}</p>
      </header>
      <div className="mt-8 grid gap-7">
        {copy.sections.map((section) => (
          <section key={section.title}>
            <h2 className="text-base font-semibold">{section.title}</h2>
            {section.paragraphs?.map((paragraph) => <p className="mt-2 text-sm leading-6 text-[var(--muted-foreground)]" key={paragraph}>{paragraph}</p>)}
            {section.items ? <ul className="mt-2 list-disc space-y-2 pl-5 text-sm leading-6 text-[var(--muted-foreground)]">{section.items.map((item) => <li key={item}>{item}</li>)}</ul> : null}
          </section>
        ))}
      </div>
      <p className="mt-8 border-t border-[var(--border)] pt-5 text-sm font-medium">{copy.contact}</p>
    </article>
  )
}
