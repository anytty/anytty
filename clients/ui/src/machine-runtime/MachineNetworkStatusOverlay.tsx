import { useTranslation } from 'react-i18next'
import { connectionPhaseLabel } from '../connection/connectionState'
import type { RtcConnectionPhase } from '../core/transport'

export interface MachineNetworkStatusOverlayProps {
  phase: RtcConnectionPhase | null
  status: string | null
}

/**
 * MachineNetworkStatusOverlay 是 workspace 的暂态连接遮罩，只消费稳定 phase 与 locale。
 * Endpoint/Route/session 真值仍属于 Go Client Engine，底层 statusText 不能覆盖已知阶段文案。
 */
export function MachineNetworkStatusOverlay({
  phase,
  status,
}: MachineNetworkStatusOverlayProps) {
  const { t } = useTranslation()
  // phase 是跨语言稳定真值；只有缺少 phase 的旧调用方才允许使用自由文本。
  const label = phase ? connectionPhaseLabel(phase, t) : status?.trim() || connectionPhaseLabel(phase, t)
  return (
    <div
      className="pointer-events-none absolute inset-0 z-[70] flex items-center justify-center bg-[var(--anytty-overlay)] px-5 backdrop-blur-[5px]"
      data-testid="anytty-machine-network-overlay"
      role="status"
      aria-live="polite"
      aria-busy="true"
    >
      <div className="w-full max-w-sm border border-[var(--anytty-border)] bg-[var(--anytty-surface)] px-4 py-3 text-[var(--anytty-text)] backdrop-blur-xl">
        <div className="flex items-center gap-3">
          <span className="anytty-square-spinner h-5 w-5 text-[var(--anytty-accent)]" aria-hidden="true" />
          <div className="min-w-0">
            <p className="text-[13px] font-semibold leading-5">{t('workspace.connection.progressTitle')}</p>
            <p className="mt-0.5 break-words text-[12px] font-medium leading-5 text-[var(--anytty-muted)]">{label}</p>
          </div>
        </div>
      </div>
    </div>
  )
}
