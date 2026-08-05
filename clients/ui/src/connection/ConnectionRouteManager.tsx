import { create } from '@bufbuild/protobuf'
import { ArrowDown, ArrowUp, CirclePlay, KeyRound, Plus, Save, Trash2, X } from 'lucide-react'
import { useEffect, useMemo, useState, type ReactNode } from 'react'
import { useTranslation } from 'react-i18next'
import type { SSHCredentialProvisionResult } from '../generated/bindingpb/client_binding_pb'
import {
  DirectWebRTCTCPRouteConfigSchema,
  EndpointCredentialDescriptorSchema,
  EndpointCredentialKind,
  EndpointConfigV1Schema,
  EndpointRouteConfigV1Schema,
  EndpointSource,
  SSHWebRTCTCPRouteConfigSchema,
  type EndpointConfigV1,
  type EndpointRouteConfigV1,
} from '../generated/remoteauthpb/remote_auth_pb'
import { hapticImpact, hapticSelection } from '../platform/haptics'

export interface ConnectionRouteManagementAdapter {
  load(signal?: AbortSignal): Promise<EndpointConfigV1>
  save(endpoint: EndpointConfigV1, signal?: AbortSignal): Promise<EndpointConfigV1>
  test(routeId: string, signal?: AbortSignal): Promise<void>
  provisionSSH(routeId: string, signal?: AbortSignal): Promise<SSHCredentialProvisionResult>
}

type RouteKind = 'direct' | 'ssh'
type RouteOperation = { kind: 'idle' | 'testing' | 'saving' | 'ready' | 'failed'; message?: string }

export function routeKind(route: EndpointRouteConfigV1): RouteKind | 'cloud' | 'local' | 'unknown' {
  switch (route.route.case) {
    case 'directWebrtcTcp': return 'direct'
    case 'sshWebrtcTcp': return 'ssh'
    case 'managedWebrtc': return 'cloud'
    case 'localUnix': return 'local'
    default: return 'unknown'
  }
}

export function orderedRoutes(endpoint: EndpointConfigV1): EndpointRouteConfigV1[] {
  return [...endpoint.routes].sort((left, right) => {
    const leftPriority = left.priority ?? Number.MAX_SAFE_INTEGER
    const rightPriority = right.priority ?? Number.MAX_SAFE_INTEGER
    return leftPriority - rightPriority || left.routeId.localeCompare(right.routeId)
  })
}

export function moveRoute(endpoint: EndpointConfigV1, routeId: string, direction: -1 | 1): EndpointConfigV1 {
  const routes = orderedRoutes(endpoint).map((route) => create(EndpointRouteConfigV1Schema, route))
  const current = routes.findIndex((route) => route.routeId === routeId)
  const target = current + direction
  if (current < 0 || target < 0 || target >= routes.length) return create(EndpointConfigV1Schema, endpoint)
  const currentRoute = routes[current]
  const targetRoute = routes[target]
  if (!currentRoute || !targetRoute) return create(EndpointConfigV1Schema, endpoint)
  routes[current] = targetRoute
  routes[target] = currentRoute
  routes.forEach((route, index) => {
    route.priority = (index + 1) * 10
    route.policySource = EndpointSource.USER
  })
  return create(EndpointConfigV1Schema, { ...endpoint, routes })
}

export function removeRoute(endpoint: EndpointConfigV1, routeId: string): EndpointConfigV1 {
  if (endpoint.routes.length <= 1) return create(EndpointConfigV1Schema, endpoint)
  return create(EndpointConfigV1Schema, { ...endpoint, routes: endpoint.routes.filter((route) => route.routeId !== routeId) })
}

export function replaceRoute(endpoint: EndpointConfigV1, route: EndpointRouteConfigV1): EndpointConfigV1 {
  const exists = endpoint.routes.some((candidate) => candidate.routeId === route.routeId)
  const routes = exists
    ? endpoint.routes.map((candidate) => candidate.routeId === route.routeId ? route : candidate)
    : [...endpoint.routes, route]
  if (!exists) {
    const ordered = routes.map((candidate) => create(EndpointRouteConfigV1Schema, candidate))
    ordered.forEach((candidate, index) => {
      candidate.priority = (index + 1) * 10
      candidate.policySource = EndpointSource.USER
    })
    return create(EndpointConfigV1Schema, { ...endpoint, routes: ordered })
  }
  return create(EndpointConfigV1Schema, { ...endpoint, routes })
}

export function ConnectionRouteManager({ adapter, endpointId }: { adapter: ConnectionRouteManagementAdapter; endpointId: string }) {
  const { t } = useTranslation()
  const [endpoint, setEndpoint] = useState<EndpointConfigV1 | null>(null)
  const [loading, setLoading] = useState(true)
  const [globalError, setGlobalError] = useState<string | null>(null)
  const [editing, setEditing] = useState<EndpointRouteConfigV1 | null>(null)
  const [addingKind, setAddingKind] = useState<RouteKind | null>(null)
  const [operations, setOperations] = useState<Record<string, RouteOperation>>({})
  const [sshResult, setSSHResult] = useState<SSHCredentialProvisionResult | null>(null)

  useEffect(() => {
    const controller = new AbortController()
    setLoading(true)
    void adapter.load(controller.signal).then((value) => {
      setEndpoint(value)
      setGlobalError(null)
    }).catch((error) => {
      if (!controller.signal.aborted) setGlobalError(routeErrorText(error, t))
    }).finally(() => {
      if (!controller.signal.aborted) setLoading(false)
    })
    return () => controller.abort()
  }, [adapter, endpointId, t])

  const routes = useMemo(() => endpoint ? orderedRoutes(endpoint).filter((route) => routeKind(route) !== 'cloud') : [], [endpoint])
  const persist = async (next: EndpointConfigV1, routeId: string) => {
    setOperations((current) => ({ ...current, [routeId]: { kind: 'saving' } }))
    try {
      const saved = await adapter.save(next)
      setEndpoint(saved)
      setEditing(null)
      setAddingKind(null)
      setOperations((current) => ({ ...current, [routeId]: { kind: 'ready', message: t('workspace.routeManager.saved') } }))
    } catch (error) {
      setOperations((current) => ({ ...current, [routeId]: { kind: 'failed', message: routeErrorText(error, t) } }))
    }
  }

  const test = async (routeId: string) => {
    setOperations((current) => ({ ...current, [routeId]: { kind: 'testing' } }))
    try {
      await adapter.test(routeId)
      setOperations((current) => ({ ...current, [routeId]: { kind: 'ready', message: t('workspace.routeManager.testReady') } }))
    } catch (error) {
      setOperations((current) => ({ ...current, [routeId]: { kind: 'failed', message: routeErrorText(error, t) } }))
    }
  }

  const provisionSSH = async (routeId: string) => {
    setOperations((current) => ({ ...current, [routeId]: { kind: 'saving' } }))
    try {
      const result = await adapter.provisionSSH(routeId)
      if (result.endpoint) setEndpoint(result.endpoint)
      setSSHResult(result)
      setOperations((current) => ({ ...current, [routeId]: { kind: 'ready', message: t('workspace.routeManager.sshReady') } }))
    } catch (error) {
      setOperations((current) => ({ ...current, [routeId]: { kind: 'failed', message: routeErrorText(error, t) } }))
    }
  }

  if (loading) return <p className="py-3 text-[13px] text-zinc-500">{t('common.loading')}</p>
  if (!endpoint) return <p className="py-3 text-[13px] text-red-700" role="alert">{globalError ?? t('errors.generic')}</p>

  return (
    <div className="space-y-3 pb-2">
      <div className="flex flex-wrap items-center justify-between gap-2">
        <p className="text-[12px] leading-5 text-zinc-600">{t('workspace.routeManager.description')}</p>
        <div className="flex flex-wrap gap-2">
          {(['direct', 'ssh'] as RouteKind[]).map((kind) => (
            <button key={kind} type="button" className="anytty-app-secondary-button min-h-12 px-3 text-[12px] font-semibold" onClick={() => { hapticSelection(); setEditing(null); setAddingKind(kind) }}>
              <Plus className="mr-1.5 h-4 w-4" aria-hidden="true" />{t(`workspace.routeManager.add.${kind}`)}
            </button>
          ))}
        </div>
      </div>

      <p className="border-l-2 border-amber-500 pl-3 text-[12px] leading-5 text-zinc-600" role="status">
        {t('workspace.connection.unavailableReason.cloud_unavailable')}
      </p>

      {addingKind ? (
        <RouteEditForm
          endpoint={endpoint}
          route={newRouteDraft(endpoint, addingKind)}
          isNew
          onCancel={() => setAddingKind(null)}
          onSave={(route) => void persist(replaceRoute(endpoint, route), route.routeId)}
        />
      ) : null}

      <div className="divide-y divide-[var(--anytty-app-line)] border-y border-[var(--anytty-app-line)]">
        {routes.map((route, index) => {
          const operation = operations[route.routeId] ?? { kind: 'idle' }
          const kind = routeKind(route)
          const enabledCount = endpoint.routes.filter((candidate) => candidate.enabled).length
          const canDisable = !route.enabled || enabledCount > 1
          return (
            <section key={route.routeId} className="py-3">
              <div className="flex min-w-0 items-start gap-3">
                <label className="flex min-h-12 min-w-0 flex-1 items-center gap-3 text-[14px]">
                  <input type="checkbox" className="h-5 w-5 shrink-0 accent-zinc-900" checked={route.enabled} disabled={!canDisable || operation.kind === 'saving'} onChange={() => {
                    const next = create(EndpointRouteConfigV1Schema, { ...route, enabled: !route.enabled, policySource: EndpointSource.USER })
                    void persist(replaceRoute(endpoint, next), route.routeId)
                  }} />
                  <span className="min-w-0">
                    <span className="block break-words font-semibold text-zinc-950">{route.displayName || route.routeId}</span>
                    <span className="block break-words text-[11px] leading-4 text-zinc-500">{routeKindLabel(kind, t)} · {route.routeId}</span>
                  </span>
                </label>
                <div className="flex shrink-0 flex-wrap justify-end gap-2">
                  <IconAction label={t('workspace.routeManager.moveUp')} disabled={index === 0 || operation.kind === 'saving'} onClick={() => void persist(moveRoute(endpoint, route.routeId, -1), route.routeId)}><ArrowUp /></IconAction>
                  <IconAction label={t('workspace.routeManager.moveDown')} disabled={index === routes.length - 1 || operation.kind === 'saving'} onClick={() => void persist(moveRoute(endpoint, route.routeId, 1), route.routeId)}><ArrowDown /></IconAction>
                  <IconAction label={t('workspace.routeManager.test')} disabled={!route.enabled || operation.kind === 'testing'} onClick={() => void test(route.routeId)}><CirclePlay /></IconAction>
                </div>
              </div>
              <div className="mt-2 flex flex-wrap gap-2 pl-8">
                <button type="button" className="anytty-app-secondary-button min-h-12 px-3 text-[12px] font-semibold" onClick={() => { setAddingKind(null); setEditing(create(EndpointRouteConfigV1Schema, route)) }}>{t('workspace.routeManager.edit')}</button>
                {kind === 'ssh' ? <button type="button" className="anytty-app-secondary-button min-h-12 px-3 text-[12px] font-semibold" onClick={() => void provisionSSH(route.routeId)}><KeyRound className="mr-1.5 h-4 w-4" />{t('workspace.routeManager.prepareSSH')}</button> : null}
                <button type="button" className="anytty-app-secondary-button min-h-12 px-3 text-[12px] font-semibold text-red-700" disabled={routes.length <= 1 || operation.kind === 'saving'} onClick={() => {
                  if (globalThis.confirm(t('workspace.routeManager.deleteConfirm', { name: route.displayName || route.routeId }))) void persist(removeRoute(endpoint, route.routeId), route.routeId)
                }}><Trash2 className="mr-1.5 h-4 w-4" />{t('workspace.routeManager.delete')}</button>
              </div>
              {operation.kind === 'testing' || operation.kind === 'saving' ? <p className="mt-2 pl-8 text-[12px] text-zinc-600" aria-live="polite">{operation.kind === 'testing' ? t('workspace.routeManager.testing') : t('workspace.routeManager.saving')}</p> : null}
              {operation.message ? <p className={`mt-2 pl-8 text-[12px] leading-5 ${operation.kind === 'failed' ? 'text-red-700' : 'text-emerald-700'}`} role={operation.kind === 'failed' ? 'alert' : 'status'}>{operation.message}</p> : null}
              {editing?.routeId === route.routeId ? <RouteEditForm endpoint={endpoint} route={editing} onCancel={() => setEditing(null)} onSave={(value) => void persist(replaceRoute(endpoint, value), value.routeId)} /> : null}
            </section>
          )
        })}
      </div>

      {sshResult?.authorizedKey ? (
        <section className="border-y border-[var(--anytty-app-line)] py-3">
          <div className="flex items-center justify-between gap-2">
            <h4 className="text-[13px] font-semibold text-zinc-950">{t('workspace.routeManager.sshPublicKey')}</h4>
            <IconAction label={t('workspace.routeManager.closeKey')} onClick={() => setSSHResult(null)}><X /></IconAction>
          </div>
          <p className="mt-1 break-all text-[11px] text-zinc-500">{sshResult.keyFingerprint}</p>
          <textarea className="anytty-app-input mt-2 min-h-24 w-full resize-y px-3 py-2 font-mono text-[12px]" readOnly value={sshResult.authorizedKey} aria-label={t('workspace.routeManager.sshPublicKey')} />
          <button type="button" className="anytty-app-secondary-button mt-2 min-h-12 px-3 text-[12px] font-semibold" onClick={() => void navigator.clipboard.writeText(sshResult.authorizedKey)}>{t('workspace.routeManager.copyKey')}</button>
        </section>
      ) : null}
    </div>
  )
}

function RouteEditForm({ endpoint, route, isNew = false, onCancel, onSave }: { endpoint: EndpointConfigV1; route: EndpointRouteConfigV1; isNew?: boolean; onCancel(): void; onSave(route: EndpointRouteConfigV1): void }) {
  const { t } = useTranslation()
  const [draft, setDraft] = useState(() => create(EndpointRouteConfigV1Schema, route))
  const [error, setError] = useState<string | null>(null)
  const kind = routeKind(draft)
  const direct = draft.route.case === 'directWebrtcTcp' ? draft.route.value : null
  const ssh = draft.route.case === 'sshWebrtcTcp' ? draft.route.value : null
  const setRoute = (value: EndpointRouteConfigV1['route']) => setDraft((current) => create(EndpointRouteConfigV1Schema, { ...current, route: value }))
  const submit = () => {
    const validation = validateRouteDraft(endpoint, draft, isNew, t)
    if (validation) { setError(validation); return }
    onSave(create(EndpointRouteConfigV1Schema, { ...draft, source: EndpointSource.USER, policySource: EndpointSource.USER }))
  }
  return (
    <div className="mt-3 space-y-3 border-y border-[var(--anytty-app-line)] bg-zinc-50 px-3 py-3">
      <div className="grid gap-3 sm:grid-cols-2">
        <RouteField label={t('workspace.routeManager.routeId')} value={draft.routeId} disabled={!isNew} onChange={(value) => setDraft((current) => create(EndpointRouteConfigV1Schema, { ...current, routeId: value.trim() }))} />
        <RouteField label={t('workspace.routeManager.displayName')} value={draft.displayName} onChange={(displayName) => setDraft((current) => create(EndpointRouteConfigV1Schema, { ...current, displayName }))} />
      </div>
      {kind === 'direct' && direct ? (
        <>
          <RouteField label={t('workspace.routeManager.signalingAddresses')} multiline value={direct.signalingAddresses.join('\n')} onChange={(value) => setRoute({ case: 'directWebrtcTcp', value: create(DirectWebRTCTCPRouteConfigSchema, { ...direct, signalingAddresses: splitLines(value) }) })} />
          <RouteField label={t('workspace.routeManager.iceAddresses')} multiline value={direct.iceTcpAddresses.join('\n')} onChange={(value) => setRoute({ case: 'directWebrtcTcp', value: create(DirectWebRTCTCPRouteConfigSchema, { ...direct, iceTcpAddresses: splitLines(value) }) })} />
          <RouteField label={t('workspace.routeManager.serverName')} value={direct.serverName} onChange={(serverName) => setRoute({ case: 'directWebrtcTcp', value: create(DirectWebRTCTCPRouteConfigSchema, { ...direct, serverName }) })} />
        </>
      ) : null}
      {kind === 'ssh' && ssh ? (
        <>
          <div className="grid gap-3 sm:grid-cols-3">
            <RouteField label={t('workspace.routeManager.sshHost')} value={ssh.host} onChange={(host) => setRoute({ case: 'sshWebrtcTcp', value: create(SSHWebRTCTCPRouteConfigSchema, { ...ssh, host }) })} />
            <RouteField label={t('workspace.routeManager.sshPort')} inputMode="numeric" value={String(ssh.port || 22)} onChange={(value) => setRoute({ case: 'sshWebrtcTcp', value: create(SSHWebRTCTCPRouteConfigSchema, { ...ssh, port: Number.parseInt(value, 10) || 0 }) })} />
            <RouteField label={t('workspace.routeManager.sshUser')} value={ssh.user} onChange={(user) => setRoute({ case: 'sshWebrtcTcp', value: create(SSHWebRTCTCPRouteConfigSchema, { ...ssh, user }) })} />
          </div>
          <RouteField label={t('workspace.routeManager.hostKeys')} multiline value={ssh.hostKeyFingerprints.join('\n')} onChange={(value) => setRoute({ case: 'sshWebrtcTcp', value: create(SSHWebRTCTCPRouteConfigSchema, { ...ssh, hostKeyFingerprints: splitLines(value) }) })} />
          <div className="grid gap-3 sm:grid-cols-2">
            <RouteField label={t('workspace.routeManager.remoteSignaling')} value={ssh.remoteSignalingAddress} onChange={(remoteSignalingAddress) => setRoute({ case: 'sshWebrtcTcp', value: create(SSHWebRTCTCPRouteConfigSchema, { ...ssh, remoteSignalingAddress }) })} />
            <RouteField label={t('workspace.routeManager.remoteICE')} value={ssh.remoteIceTcpAddress} onChange={(remoteIceTcpAddress) => setRoute({ case: 'sshWebrtcTcp', value: create(SSHWebRTCTCPRouteConfigSchema, { ...ssh, remoteIceTcpAddress }) })} />
          </div>
        </>
      ) : null}
      {kind === 'cloud' ? <p className="text-[12px] leading-5 text-zinc-600">{t('workspace.routeManager.cloudManaged')}</p> : null}
      {error ? <p className="text-[12px] text-red-700" role="alert">{error}</p> : null}
      <div className="flex flex-wrap justify-end gap-2">
        <button type="button" className="anytty-app-secondary-button min-h-12 px-3 text-[12px] font-semibold" onClick={onCancel}><X className="mr-1.5 h-4 w-4" />{t('common.cancel')}</button>
        <button type="button" className="anytty-app-primary-button min-h-12 px-3 text-[12px] font-semibold" onClick={() => { hapticImpact(); submit() }}><Save className="mr-1.5 h-4 w-4" />{t('workspace.routeManager.save')}</button>
      </div>
    </div>
  )
}

function RouteField({ label, value, disabled = false, multiline = false, inputMode, onChange }: { label: string; value: string; disabled?: boolean; multiline?: boolean; inputMode?: 'numeric'; onChange(value: string): void }) {
  return <label className="block text-[12px] font-semibold text-zinc-700"><span className="mb-1 block">{label}</span>{multiline
    ? <textarea className="anytty-app-input min-h-20 w-full resize-y px-3 py-2 text-[14px] font-normal" value={value} disabled={disabled} onChange={(event) => onChange(event.target.value)} />
    : <input className="anytty-app-input min-h-12 w-full px-3 text-[14px] font-normal" value={value} disabled={disabled} inputMode={inputMode} onChange={(event) => onChange(event.target.value)} />}</label>
}

function IconAction({ label, disabled = false, onClick, children }: { label: string; disabled?: boolean; onClick(): void; children: ReactNode }) {
  return <button type="button" title={label} aria-label={label} disabled={disabled} className="anytty-app-icon-button [&_svg]:h-4 [&_svg]:w-4 disabled:opacity-40" onClick={() => { hapticSelection(); onClick() }}>{children}</button>
}

function newRouteDraft(endpoint: EndpointConfigV1, kind: RouteKind): EndpointRouteConfigV1 {
  const credentialRef = endpoint.routes.find((route) => route.credentialRef.trim())?.credentialRef ?? ''
  const common = { schemaVersion: 1, routeId: uniqueRouteId(endpoint, kind), displayName: '', enabled: true, source: EndpointSource.USER, policySource: EndpointSource.USER, credentialRef }
  if (kind === 'direct') return create(EndpointRouteConfigV1Schema, { ...common, route: { case: 'directWebrtcTcp', value: create(DirectWebRTCTCPRouteConfigSchema) } })
  if (kind === 'ssh') return create(EndpointRouteConfigV1Schema, { ...common, route: { case: 'sshWebrtcTcp', value: create(SSHWebRTCTCPRouteConfigSchema, {
    port: 22, remoteSignalingAddress: '127.0.0.1:41120', remoteIceTcpAddress: '127.0.0.1:41121',
    credentialDescriptor: create(EndpointCredentialDescriptorSchema, { descriptorId: `${common.routeId}-key`, kind: EndpointCredentialKind.SSH_PRIVATE_KEY }),
  }) } })
  throw new Error(`unsupported route kind: ${kind satisfies never}`)
}

function uniqueRouteId(endpoint: EndpointConfigV1, kind: RouteKind): string {
  const used = new Set(endpoint.routes.map((route) => route.routeId))
  if (!used.has(kind)) return kind
  let suffix = 2
  while (used.has(`${kind}-${suffix}`)) suffix++
  return `${kind}-${suffix}`
}

function validateRouteDraft(endpoint: EndpointConfigV1, route: EndpointRouteConfigV1, isNew: boolean, t: ReturnType<typeof useTranslation>['t']): string | null {
  if (!/^[A-Za-z0-9._-]+$/.test(route.routeId)) return t('workspace.routeManager.validation.routeId')
  if (isNew && endpoint.routes.some((candidate) => candidate.routeId === route.routeId)) return t('workspace.routeManager.validation.duplicate')
  if (route.route.case === 'directWebrtcTcp' && (route.route.value.signalingAddresses.length === 0 || route.route.value.iceTcpAddresses.length === 0)) return t('workspace.routeManager.validation.direct')
  if (route.route.case === 'sshWebrtcTcp') {
    const value = route.route.value
    if (!value.host.trim() || !value.user.trim() || value.port < 1 || value.port > 65535 || value.hostKeyFingerprints.length === 0) return t('workspace.routeManager.validation.ssh')
  }
  return null
}

function splitLines(value: string): string[] { return value.split(/[\n,]/).map((item) => item.trim()).filter(Boolean) }

function routeKindLabel(kind: ReturnType<typeof routeKind>, t: ReturnType<typeof useTranslation>['t']): string {
  if (kind === 'direct') return t('workspace.connection.routeDirect')
  if (kind === 'ssh') return t('workspace.connection.routeSSH')
  if (kind === 'cloud') return t('workspace.connection.routeCloud')
  if (kind === 'local') return t('workspace.connection.routeLocal')
  return t('workspace.connection.notProvided')
}

function routeErrorText(error: unknown, t: ReturnType<typeof useTranslation>['t']): string {
  const code = String((error as { code?: string } | null)?.code ?? '').toLowerCase()
  if (code.includes('credential')) return t('workspace.routeManager.errors.credential')
  if (code.includes('authorization') || code === 'unauthenticated' || code === 'forbidden') return t('workspace.routeManager.errors.authorization')
  if (code.includes('entitlement')) return t('workspace.routeManager.errors.entitlement')
  if (code.includes('quota')) return t('workspace.routeManager.errors.quota')
  if (code.includes('identity')) return t('workspace.routeManager.errors.identity')
  if (code.includes('route') || code.includes('config') || code === 'invalid_request' || code === 'not_found' || code === 'conflict') return t('workspace.routeManager.errors.config')
  if (code === 'unavailable' || code === 'stale_session') return t('workspace.routeManager.errors.unavailable')
  if (code === 'cancelled') return t('workspace.routeManager.errors.cancelled')
  if (code === 'internal' || code === 'unsupported_version' || code === 'unsupported_capability') return t('workspace.routeManager.errors.internal')
  // Route 错误只允许投影稳定分类，bridge/provider 的原始文本可能包含地址或传输细节。
  return t('workspace.routeManager.errors.internal')
}
