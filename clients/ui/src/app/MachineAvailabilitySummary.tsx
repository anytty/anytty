import { Cloud, RadioTower, Route, SquareTerminal, WifiOff, type LucideIcon } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import type { TFunction } from 'i18next'
import type { MachineConnectionSnapshot } from '../connection/machineConnectionSnapshot'
import { ConnectionSummary } from '../connection/ConnectionSummary'
import { connectionPhaseLabel } from '../connection/connectionState'
import {
  projectConnectionPresentation,
  type ConnectionPresentation,
  type ConnectionReachability,
} from '../connection/connectionPresentation'
import type { MachineAccessClass } from '../state/appMachine'
import { cn } from '../ui/utils'

export type MachineRouteReachability = 'checking' | 'online' | 'offline' | 'unknown'

export function MachineAvailabilitySummary({
  accessClass,
  cloud,
  cloudDetail,
  connection,
  local,
}: {
  accessClass: MachineAccessClass
  cloud: MachineRouteReachability
  cloudDetail?: string | undefined
  connection: MachineConnectionSnapshot
  local: MachineRouteReachability
}) {
  const { t } = useTranslation()
  const idleSnapshot = { ...connection, phase: 'idle' as const, connectionInfo: null, relayInUse: false, error: null }
  const paths = [
    ...(accessClass === 'cloud' ? [] : [{ kind: 'local' as const, state: local, icon: Route }]),
    ...(accessClass === 'local' ? [] : [{ kind: 'cloud' as const, state: cloud, icon: Cloud }]),
  ]
  const presentation = projectConnectionPresentation({
    phoneOnline: true,
    authAvailable: true,
    reachability: combinedReachability(accessClass, local, cloud),
    snapshot: connection,
  })
  const lifecyclePresentation = presentation.state === 'connecting' && presentation.reachability === 'unreachable'
    ? projectConnectionPresentation({
        phoneOnline: true,
        authAvailable: true,
        reachability: 'unreachable',
        snapshot: idleSnapshot,
      })
    : presentation
  const lifecycle = lifecycleStatus(connection, lifecyclePresentation, t)
  return (
    <span className="inline-flex min-w-0 flex-wrap items-center gap-x-1 gap-y-0.5">
      {paths.map(({ kind, state, icon }) => {
        const detail = reachabilityLabel(kind, state, t)
        const pathPresentation = projectConnectionPresentation({
          phoneOnline: true,
          authAvailable: true,
          reachability: mapReachability(state),
          snapshot: idleSnapshot,
        })
        return (
          <ConnectionSummary
            aria-label={detail}
            className={cn('gap-1 text-[10px] leading-none transition-colors duration-200 ease-out motion-reduce:transition-none [&>span:first-child]:size-4 [&>span:first-child]:rounded [&>span:first-child>svg]:size-2.5', toneClass(pathPresentation))}
            icon={icon}
            key={kind}
            label={kind === 'local' ? t('machines.path.localNetwork') : cloudDetail ? `${t('machines.path.cloud')} · ${cloudDetail}` : t('machines.path.cloud')}
            presentation={pathPresentation}
            title={detail}
          />
        )
      })}
      {lifecycle ? (
        <ConnectionSummary
          announce
          className={cn('gap-1 text-[10px] leading-none [&>span:first-child]:size-4 [&>span:first-child]:rounded [&>span:first-child>svg]:size-2.5', toneClass(lifecyclePresentation))}
          icon={lifecycle.icon}
          label={lifecycle.label}
          presentation={lifecyclePresentation}
          title={lifecycle.label}
        />
      ) : null}
    </span>
  )
}

function lifecycleStatus(connection: MachineConnectionSnapshot, presentation: ConnectionPresentation, t: TFunction) {
  if (presentation.state === 'idle') {
    return presentation.reachability === 'unreachable'
      ? { label: t('machines.notReachable'), icon: WifiOff }
      : null
  }
  if (presentation.state === 'ready') {
    if (presentation.observedPath === 'relay') return { label: t('workspace.connection.pathRelay'), icon: Cloud }
    if (presentation.observedPath === 'p2p' || presentation.route === 'direct' || presentation.route === 'local') {
      return { label: t('workspace.connection.pathDirect'), icon: RadioTower }
    }
    if (presentation.route === 'ssh') return { label: t('workspace.connection.routeSSH'), icon: SquareTerminal }
    return { label: t('machines.connected'), icon: undefined as LucideIcon | undefined }
  }
  if (presentation.state === 'connecting') return { label: connectionPhaseLabel(connection.phase, t) }
  if (presentation.state === 'waiting_network') return { label: connectionPhaseLabel('waiting_network', t) }
  return { label: t('machines.connectionFailed') }
}

function combinedReachability(accessClass: MachineAccessClass, local: MachineRouteReachability, cloud: MachineRouteReachability): ConnectionReachability {
  const states = [accessClass === 'cloud' ? null : local, accessClass === 'local' ? null : cloud].filter(Boolean)
  if (states.includes('online')) return 'reachable'
  if (states.includes('checking')) return 'checking'
  if (states.length > 0 && states.every((state) => state === 'offline')) return 'unreachable'
  return 'unknown'
}

function mapReachability(state: MachineRouteReachability): ConnectionReachability {
  return state === 'online' ? 'reachable' : state === 'offline' ? 'unreachable' : state
}

function reachabilityLabel(kind: 'local' | 'cloud', state: MachineRouteReachability, t: TFunction): string {
  const prefix = kind === 'local' ? 'local' : 'cloud'
  const suffix = state === 'online' ? 'Online' : state === 'offline' ? 'Offline' : state === 'checking' ? 'Checking' : 'Unknown'
  return t(`machines.reachability.${prefix}${suffix}`)
}

function toneClass(presentation: ConnectionPresentation): string {
  if (presentation.tone === 'positive') return 'text-[var(--anytty-app-success)]'
  if (presentation.tone === 'info') return 'text-[var(--anytty-app-accent)]'
  if (presentation.tone === 'critical' || presentation.reachability === 'unreachable') return 'text-[var(--destructive)] [&>span:first-child]:text-[var(--destructive)]'
  return 'text-[var(--muted-foreground)]'
}
