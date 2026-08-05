import { StrictMode } from 'react'
import { createRoot, type Root } from 'react-dom/client'
import { readTerminalSettings, terminalThemeCssVariables, type TerminalThemeId } from '../terminal/terminalSettings'

export interface RemoteControlEntryOptions {
  root?: HTMLElement | null | undefined
  themeId?: TerminalThemeId | undefined
}

export function mountRemoteControlApp(options: RemoteControlEntryOptions = {}): Root {
  const rootElement = options.root ?? document.getElementById('root')
  if (!rootElement) {
    throw new Error('remote app root element is required')
  }
  const root = createRoot(rootElement)
  const themeId = options.themeId ?? readTerminalSettings().themeId
  root.render(
    <StrictMode>
      <CloudUnavailableApp themeId={themeId} />
    </StrictMode>,
  )
  return root
}

function CloudUnavailableApp({ themeId }: { themeId?: TerminalThemeId | undefined }) {
  return (
    <section
      aria-describedby="anytty-cloud-unavailable-description"
      aria-labelledby="anytty-cloud-unavailable-title"
      className="flex h-[100dvh] w-screen items-center justify-center bg-[var(--anytty-surface)] px-6 text-[var(--anytty-text)] antialiased"
      data-testid="anytty-cloud-unavailable"
      role="alert"
      style={terminalThemeCssVariables(themeId)}
    >
      <div className="w-full max-w-sm border-l-2 border-emerald-600 pl-5">
        <h1 className="text-lg font-semibold text-[var(--anytty-text)]" id="anytty-cloud-unavailable-title">AnyTTY Cloud 暂不可用</h1>
        <p className="mt-2 text-sm leading-6 text-[var(--anytty-text)]" id="anytty-cloud-unavailable-description">云端服务正在重构。Direct 和 SSH 客户端不受影响。</p>
      </div>
    </section>
  )
}
