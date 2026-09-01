import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useId,
  useMemo,
  useState,
  type ReactNode,
} from 'react'
import { createPortal } from 'react-dom'
import { CircleAlert, LoaderCircle, RefreshCw, WifiOff } from 'lucide-react'
import { Button } from '../ui/button'

export type ConnectionRecoveryOverlayKind = 'recovering' | 'offline' | 'failed'

export interface ConnectionRecoveryOverlayAction {
  label: string
  onClick: () => void | Promise<void>
  pending?: boolean | undefined
  testId?: string | undefined
}

/** A layer publishes only the user-facing outcome it needs; the root owns placement and priority. */
export interface ConnectionRecoveryOverlayIntent {
  kind: ConnectionRecoveryOverlayKind
  title: string
  description?: string | undefined
  action?: ConnectionRecoveryOverlayAction | undefined
}

type ConnectionRecoveryOverlayPublisher = (
  sourceId: string,
  intent: ConnectionRecoveryOverlayIntent | null,
) => void

type ConnectionRecoveryOverlayHostPublisher = (
  hostId: string,
  host: HTMLElement | null,
) => void

interface ConnectionRecoveryOverlayContextValue {
  publish: ConnectionRecoveryOverlayPublisher
  registerHost: ConnectionRecoveryOverlayHostPublisher
}

const ConnectionRecoveryOverlayContext = createContext<ConnectionRecoveryOverlayContextValue | null>(null)

export function ConnectionRecoveryOverlayProvider({
  appIntent,
  children,
}: {
  appIntent: ConnectionRecoveryOverlayIntent | null
  children: ReactNode
}) {
  const [sourceIntents, setSourceIntents] = useState<Map<string, ConnectionRecoveryOverlayIntent>>(() => new Map())
  const [hosts, setHosts] = useState<Map<string, HTMLElement>>(() => new Map())
  const publish = useCallback<ConnectionRecoveryOverlayPublisher>((sourceId, intent) => {
    setSourceIntents((current) => {
      if (intent === null && !current.has(sourceId)) return current
      if (intent !== null && current.get(sourceId) === intent) return current
      const next = new Map(current)
      if (intent) next.set(sourceId, intent)
      else next.delete(sourceId)
      return next
    })
  }, [])
  const registerHost = useCallback<ConnectionRecoveryOverlayHostPublisher>((hostId, host) => {
    setHosts((current) => {
      if (host === null && !current.has(hostId)) return current
      if (host !== null && current.get(hostId) === host) return current
      const next = new Map(current)
      next.delete(hostId)
      if (host) next.set(hostId, host)
      return next
    })
  }, [])
  const visibleIntent = useMemo(
    () => appIntent ?? highestPriorityIntent(sourceIntents.values()),
    [appIntent, sourceIntents],
  )
  const activeHost = useMemo(() => {
    let selected: HTMLElement | null = null
    for (const host of hosts.values()) selected = host
    return selected
  }, [hosts])
  const contextValue = useMemo<ConnectionRecoveryOverlayContextValue>(
    () => ({ publish, registerHost }),
    [publish, registerHost],
  )
  const overlay = <ConnectionRecoveryOverlay intent={visibleIntent} />

  return (
    <ConnectionRecoveryOverlayContext.Provider value={contextValue}>
      {children}
      {activeHost ? createPortal(overlay, activeHost) : null}
    </ConnectionRecoveryOverlayContext.Provider>
  )
}

/** Returns false when a component is rendered outside the shared recovery provider. */
export function useConnectionRecoveryOverlay(intent: ConnectionRecoveryOverlayIntent | null): boolean {
  const publish = useContext(ConnectionRecoveryOverlayContext)?.publish
  const sourceId = useId()

  useEffect(() => {
    if (!publish) return
    publish(sourceId, intent)
    return () => publish(sourceId, null)
  }, [intent, publish, sourceId])

  return Boolean(publish)
}

/** Mounts the single merged recovery surface inside the current page's content region. */
export function ConnectionRecoveryOverlayHost() {
  const registerHost = useContext(ConnectionRecoveryOverlayContext)?.registerHost
  const hostId = useId()
  const attachHost = useCallback((host: HTMLDivElement | null) => {
    registerHost?.(hostId, host)
  }, [hostId, registerHost])

  useEffect(() => () => registerHost?.(hostId, null), [hostId, registerHost])

  return (
    <div
      className="pointer-events-none absolute inset-0 z-40 min-h-0 overflow-hidden"
      data-anytty-connection-overlay-host=""
      ref={attachHost}
    />
  )
}

export function ConnectionRecoveryOverlay({ intent }: { intent: ConnectionRecoveryOverlayIntent | null }) {
  if (!intent) return null

  const failed = intent.kind === 'failed'
  const offline = intent.kind === 'offline'

  return (
    <div
      className="pointer-events-auto absolute inset-0 flex min-h-0 items-center justify-center overflow-y-auto bg-[var(--anytty-app-bg)]/55 px-5 py-8 backdrop-blur-[6px] backdrop-saturate-150 animate-in fade-in duration-200 motion-reduce:animate-none sm:px-8"
      data-anytty-connection-overlay-root=""
    >
      <section
        aria-atomic="true"
        aria-busy={!failed || intent.action?.pending || undefined}
        aria-live={failed ? 'assertive' : 'polite'}
        className="flex max-h-full w-full max-w-sm min-w-0 flex-col items-center justify-center text-center text-[var(--anytty-app-text)]"
        data-connection-overlay-kind={intent.kind}
        data-connection-overlay-surface="unframed"
        data-testid="anytty-connection-recovery-overlay"
        role={failed ? 'alert' : 'status'}
      >
        {intent.kind === 'recovering' ? (
          <ConnectionSignalLoader />
        ) : (
          <span
            aria-hidden="true"
            className={`grid h-12 w-12 shrink-0 place-items-center ${
              failed ? 'text-red-400' : 'text-amber-300'
            }`}
          >
            {offline ? <WifiOff className="size-7" /> : <CircleAlert className="size-7" />}
          </span>
        )}
        <div className="mt-5 min-w-0 max-w-sm">
          <p className="break-words text-base font-semibold leading-6">{intent.title}</p>
          {intent.description ? (
            <p className="mt-2 break-words text-sm leading-5 text-[var(--anytty-app-muted)]">{intent.description}</p>
          ) : null}
        </div>
        {intent.action ? (
          <Button
            aria-busy={intent.action.pending || undefined}
            className="mt-5 min-h-11 w-full max-w-56 shrink-0 gap-2 px-4 text-sm"
            data-testid={intent.action.testId}
            disabled={intent.action.pending}
            onClick={intent.action.onClick}
            variant="secondary"
          >
            {intent.action.pending
              ? <LoaderCircle className="size-4 animate-spin motion-reduce:animate-none" aria-hidden="true" />
              : <RefreshCw className="size-4" aria-hidden="true" />}
            {intent.action.label}
          </Button>
        ) : null}
      </section>
    </div>
  )
}

function ConnectionSignalLoader() {
  return (
    <div
      aria-hidden="true"
      className="flex h-12 shrink-0 items-center text-[var(--anytty-app-text)]"
      data-connection-recovery-loader="signal"
    >
      <span className="h-px w-7 bg-current opacity-25" />
      <span className="mx-2 flex h-8 items-center gap-1.5" data-connection-signal-bars="">
        {Array.from({ length: 5 }, (_, index) => (
          <span
            className="anytty-connection-signal-bar h-7 w-0.5 bg-current"
            data-connection-signal-bar={index + 1}
            key={index}
          />
        ))}
      </span>
      <span className="h-px w-7 bg-current opacity-25" />
    </div>
  )
}

function highestPriorityIntent(intents: Iterable<ConnectionRecoveryOverlayIntent>): ConnectionRecoveryOverlayIntent | null {
  let selected: ConnectionRecoveryOverlayIntent | null = null
  for (const intent of intents) {
    if (!selected || connectionOverlayPriority[intent.kind] > connectionOverlayPriority[selected.kind]) selected = intent
  }
  return selected
}

const connectionOverlayPriority: Record<ConnectionRecoveryOverlayKind, number> = {
  recovering: 1,
  offline: 2,
  failed: 3,
}
