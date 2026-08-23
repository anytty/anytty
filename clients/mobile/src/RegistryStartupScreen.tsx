import { useState } from 'react'
import { Button, Spinner, anyttyI18n } from '@anytty/ui'

export interface RegistryStartupScreenProps {
  error: string | null
  onRetry: () => Promise<void>
  onResetLocalPairings: () => Promise<void>
}

type RegistryStartupFailureCode =
  | 'registry_access_denied'
  | 'registry_decode_failed'
  | 'registry_integrity_failed'
  | 'registry_open_failed'
  | 'registry_read_timeout'
  | 'registry_unavailable'

type StartupAction = 'retry' | 'copy' | 'reset' | null
type CopyStatus = 'copied' | 'failed' | null

function classifyRegistryStartupFailure(error: string): RegistryStartupFailureCode {
  const normalized = error.slice(0, 512).toLowerCase()
  if (/\b(checksum|corrupt(?:ed|ion)?|integrity)\b/.test(normalized)) return 'registry_integrity_failed'
  if (/\b(denied|forbidden|permission|unauthori[sz]ed)\b/.test(normalized)) return 'registry_access_denied'
  if (/\b(deadline|timed out|timeout)\b/.test(normalized)) return 'registry_read_timeout'
  if (/\b(decode|malformed|parse|unmarshal)\b/.test(normalized)) return 'registry_decode_failed'
  if (/\b(missing|not exist|not found|unavailable)\b/.test(normalized)) return 'registry_unavailable'
  return 'registry_open_failed'
}

export function createRegistryStartupDiagnostic(error: string): string {
  return JSON.stringify({
    schema: 'anytty.mobile.registry-startup-diagnostic',
    version: 1,
    failure: {
      category: 'native_endpoint_registry',
      code: classifyRegistryStartupFailure(error),
    },
    recovery: {
      retry: true,
      reset_local_pairings: true,
    },
  })
}

export function RegistryStartupScreen({
  error,
  onRetry,
  onResetLocalPairings,
}: RegistryStartupScreenProps) {
  const [action, setAction] = useState<StartupAction>(null)
  const [confirmReset, setConfirmReset] = useState(false)
  const [actionFailed, setActionFailed] = useState(false)
  const [copyStatus, setCopyStatus] = useState<CopyStatus>(null)

  const run = async (nextAction: Exclude<StartupAction, null>, callback: () => Promise<void>) => {
    setAction(nextAction)
    setActionFailed(false)
    setCopyStatus(null)
    try {
      await callback()
    } catch {
      setActionFailed(true)
    } finally {
      setAction(null)
    }
  }

  const copyDiagnostics = async () => {
    if (!error) return
    setAction('copy')
    setActionFailed(false)
    setCopyStatus(null)
    try {
      const snapshot = createRegistryStartupDiagnostic(error)
      const clipboard = navigator.clipboard
      if (!clipboard?.writeText) throw new Error('clipboard unavailable')
      await clipboard.writeText(snapshot)
      setCopyStatus('copied')
    } catch {
      setCopyStatus('failed')
    } finally {
      setAction(null)
    }
  }

  if (!error) {
    return (
      <section aria-live="polite" className="bg-[var(--anytty-app-bg)] text-[var(--anytty-app-text)] flex h-full w-full items-center justify-center bg-[var(--anytty-app-bg)]">
        <Spinner aria-label={anyttyI18n.t('common.loading')} className="h-7 w-7 text-[var(--anytty-app-accent)]" role="status" />
      </section>
    )
  }

  const failureCode = classifyRegistryStartupFailure(error)

  return (
    <main className="bg-[var(--anytty-app-bg)] text-[var(--anytty-app-text)] flex h-full w-full overflow-y-auto bg-[var(--anytty-app-bg)] pb-[calc(env(safe-area-inset-bottom)+1rem)] pl-[calc(env(safe-area-inset-left)+1rem)] pr-[calc(env(safe-area-inset-right)+1rem)] pt-[calc(env(safe-area-inset-top)+1rem)] sm:pb-[calc(env(safe-area-inset-bottom)+2rem)] sm:pt-[calc(env(safe-area-inset-top)+2rem)]">
      <section aria-labelledby="registry-startup-title" className="rounded-lg border border-[var(--anytty-app-line)] bg-[var(--anytty-app-surface)] shadow-sm my-auto min-w-0 w-full max-w-md p-5" data-testid="registry-startup-error">
        <h1 className="text-lg font-semibold text-zinc-950" id="registry-startup-title">{anyttyI18n.t('startup.registryTitle')}</h1>
        <p className="mt-2 text-sm leading-5 text-zinc-600" role="alert">{anyttyI18n.t('startup.registryCopy')}</p>
        <details className="mt-4 min-w-0 rounded-lg border border-[var(--anytty-app-line)] bg-[var(--anytty-app-soft)] px-3 py-2 text-xs text-zinc-600">
          <summary className="min-h-11 cursor-pointer py-3 font-semibold">{anyttyI18n.t('startup.technicalDetails')}</summary>
          <dl className="grid min-w-0 gap-1 pb-2 font-mono">
            <div className="grid min-w-0 grid-cols-[auto_minmax(0,1fr)] gap-2">
              <dt>{anyttyI18n.t('startup.diagnosticCategory')}</dt>
              <dd className="min-w-0 break-words text-right">{anyttyI18n.t('startup.registryDiagnosticCategory')}</dd>
            </div>
            <div className="grid min-w-0 grid-cols-[auto_minmax(0,1fr)] gap-2">
              <dt>{anyttyI18n.t('startup.diagnosticCode')}</dt>
              <dd className="min-w-0 break-all text-right">{failureCode}</dd>
            </div>
          </dl>
        </details>
        {actionFailed ? <p className="mt-3 text-sm font-medium text-red-600" role="alert">{anyttyI18n.t('startup.actionFailed')}</p> : null}
        {copyStatus === 'copied' ? <p className="mt-3 text-sm font-medium text-[var(--anytty-app-success)]" role="status">{anyttyI18n.t('startup.diagnosticsCopied')}</p> : null}
        {copyStatus === 'failed' ? <p className="mt-3 text-sm font-medium text-red-600" role="alert">{anyttyI18n.t('startup.diagnosticsCopyFailed')}</p> : null}
        <div className="mt-5 grid min-w-0 grid-cols-1 gap-2 sm:grid-cols-2">
          <Button className="min-h-11 px-4 font-semibold" disabled={action !== null} onClick={() => void run('retry', onRetry)}>
            {action === 'retry' ? anyttyI18n.t('startup.retrying') : anyttyI18n.t('startup.retry')}
          </Button>
          <Button aria-busy={action === 'copy'} className="min-h-11 px-4 font-semibold" disabled={action !== null} variant="secondary" onClick={() => void copyDiagnostics()}>
            {action === 'copy' ? anyttyI18n.t('startup.copyingDiagnostics') : anyttyI18n.t('startup.copyDiagnostics')}
          </Button>
        </div>
        <div className="mt-5 border-t border-[var(--anytty-app-line)] pt-4">
          {confirmReset ? (
            <div role="group" aria-label={anyttyI18n.t('startup.resetLocalPairings')}>
              <p className="text-sm leading-5 text-red-700">{anyttyI18n.t('startup.resetWarning')}</p>
              <div className="mt-3 grid min-w-0 grid-cols-1 gap-2 min-[360px]:grid-cols-2">
                <Button className="min-h-11 px-3 font-semibold" disabled={action !== null} variant="secondary" onClick={() => setConfirmReset(false)}>{anyttyI18n.t('common.cancel')}</Button>
                <Button className="min-h-11 px-3 font-semibold" disabled={action !== null} variant="destructive" onClick={() => void run('reset', onResetLocalPairings)}>
                  {action === 'reset' ? anyttyI18n.t('startup.resetting') : anyttyI18n.t('startup.confirmReset')}
                </Button>
              </div>
            </div>
          ) : (
            <Button className="min-h-11 w-full border-red-300 px-4 font-semibold text-red-700 hover:bg-red-50" disabled={action !== null} variant="outline" onClick={() => setConfirmReset(true)}>{anyttyI18n.t('startup.resetLocalPairings')}</Button>
          )}
        </div>
      </section>
    </main>
  )
}

export function UnsupportedWebPreview() {
  return (
    <main className="bg-[var(--anytty-app-bg)] text-[var(--anytty-app-text)] flex min-h-full w-full items-center justify-center bg-[var(--anytty-app-bg)] px-4 py-8">
      <section aria-labelledby="unsupported-preview-title" className="rounded-lg border border-[var(--anytty-app-line)] bg-[var(--anytty-app-surface)] shadow-sm w-full max-w-md p-5" data-testid="unsupported-web-preview">
        <h1 className="text-lg font-semibold text-zinc-950" id="unsupported-preview-title">{anyttyI18n.t('startup.previewTitle')}</h1>
        <p className="mt-2 text-sm leading-5 text-zinc-600">{anyttyI18n.t('startup.previewCopy')}</p>
      </section>
    </main>
  )
}
