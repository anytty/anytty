import { StrictMode } from 'react'
import { createRoot, type Root } from 'react-dom/client'
import { resolveTerminalThemeOption, type TerminalThemeId } from '../terminal/terminalSettings'
import { appThemeCssVariables, readAppTheme, type AppTheme } from '../app/appTheme'

export interface RemoteControlEntryOptions {
  root?: HTMLElement | null | undefined
  appTheme?: AppTheme | undefined
  /** @deprecated Use appTheme. */
  themeId?: TerminalThemeId | undefined
}

export function mountRemoteControlApp(options: RemoteControlEntryOptions = {}): Root {
  const rootElement = options.root ?? document.getElementById('root')
  if (!rootElement) {
    throw new Error('remote app root element is required')
  }
  const root = createRoot(rootElement)
  const appTheme = options.appTheme
    ?? (options.themeId ? resolveTerminalThemeOption(options.themeId).group : readAppTheme())
  root.render(
    <StrictMode>
      <CloudUnavailableApp appTheme={appTheme} />
    </StrictMode>,
  )
  return root
}

function CloudUnavailableApp({ appTheme }: { appTheme: AppTheme }) {
  return (
    <section
      aria-describedby="anytty-cloud-unavailable-description"
      aria-labelledby="anytty-cloud-unavailable-title"
      className="flex h-[100dvh] w-screen items-center justify-center bg-[var(--anytty-app-bg)] px-6 text-[var(--anytty-app-text)] antialiased"
      data-testid="anytty-cloud-unavailable"
      role="alert"
      style={appThemeCssVariables(appTheme)}
    >
      <div className="w-full max-w-sm border-l-2 border-emerald-600 pl-5">
        <h1 className="text-lg font-semibold text-[var(--anytty-app-text)]" id="anytty-cloud-unavailable-title">AnyTTY Cloud 暂不可用</h1>
        <p className="mt-2 text-sm leading-6 text-[var(--anytty-app-text)]" id="anytty-cloud-unavailable-description">云端服务正在重构。Direct 和 SSH 客户端不受影响。</p>
      </div>
    </section>
  )
}
