import { useCallback, useEffect, useMemo, useState, type CSSProperties } from 'react'
import { anyttyI18n, appThemeCssVariables, readAppTheme } from '@anytty/ui'
import { LocalWebAnyTTYApp, parseLocalWebBootstrap, type LocalWebBootstrap } from './LocalWebAnyTTYApp'
import { LocalWebLogin } from './LocalWebLogin'

export function AnyTTYApp() {
  const initialAppThemeStyle = useMemo(
    () => appThemeCssVariables(readAppTheme()) as CSSProperties,
    [],
  )
  return <BrowserAnyTTYApp initialAppThemeStyle={initialAppThemeStyle} />
}

function BrowserAnyTTYApp({ initialAppThemeStyle }: { initialAppThemeStyle: CSSProperties }) {
  const [bootstrap, setBootstrap] = useState<LocalWebBootstrap | null | undefined>(undefined)
  const [authenticationRequired, setAuthenticationRequired] = useState(false)
  const loadBootstrap = useCallback(async (signal?: AbortSignal) => {
    const response = await fetch('/api/bootstrap', { cache: 'no-store', signal })
    if (response.status === 401) {
      setBootstrap(null)
      setAuthenticationRequired(true)
      return false
    }
    if (!response.ok || !response.headers.get('content-type')?.includes('application/json')) {
      setBootstrap(null)
      setAuthenticationRequired(false)
      return false
    }
    const value = parseLocalWebBootstrap(await response.json())
    setBootstrap(value)
    setAuthenticationRequired(false)
    return value !== null
  }, [])

  useEffect(() => {
    const controller = new AbortController()
    void loadBootstrap(controller.signal).catch(() => {
      if (!controller.signal.aborted) setBootstrap(null)
    })
    return () => controller.abort()
  }, [loadBootstrap])

  if (bootstrap === undefined) return <div className="h-full" style={initialAppThemeStyle} />
  if (authenticationRequired) {
    return (
      <LocalWebLogin
        initialAppThemeStyle={initialAppThemeStyle}
        onAuthenticated={async () => {
          if (!await loadBootstrap()) throw new Error('local Web bootstrap is unavailable')
        }}
      />
    )
  }
  if (bootstrap === null) return <UnsupportedWebPreview initialAppThemeStyle={initialAppThemeStyle} />
  return <LocalWebAnyTTYApp bootstrap={bootstrap} initialAppThemeStyle={initialAppThemeStyle} />
}

function UnsupportedWebPreview({ initialAppThemeStyle }: { initialAppThemeStyle: CSSProperties }) {
  return (
    <main
      className="flex min-h-full w-full items-center justify-center bg-[var(--anytty-app-bg)] px-4 py-8 text-[var(--anytty-app-text)]"
      style={initialAppThemeStyle}
    >
      <section
        aria-labelledby="unsupported-preview-title"
        className="w-full max-w-md rounded-lg border border-[var(--anytty-app-line)] bg-[var(--anytty-app-surface)] p-5 shadow-sm"
      >
        <h1 className="text-lg font-semibold" id="unsupported-preview-title">
          {anyttyI18n.t('startup.previewTitle')}
        </h1>
        <p className="mt-2 text-sm leading-5 text-[var(--anytty-app-muted)]">
          {anyttyI18n.t('startup.previewCopy')}
        </p>
      </section>
    </main>
  )
}
