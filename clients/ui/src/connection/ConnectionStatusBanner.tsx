import { Loader2 } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import type { RtcConnectionPhase } from '../core/transport'
import {
  connectionPhaseLabel,
  connectionStatusIsSettled,
} from './connectionState'

export interface ConnectionStatusBannerProps {
  status: string | null
  phase?: RtcConnectionPhase | null | undefined
  tone?: 'light' | 'dark' | undefined
  className?: string | undefined
}

/**
 * ConnectionStatusBanner 只展示 Go Client Engine 已发布的连接阶段。
 * 组件不解释自由文本、不拥有重试或 session 生命周期，已知 phase 始终经过本地化投影。
 */
export function ConnectionStatusBanner({
  status,
  phase,
  tone = 'light',
  className = '',
}: ConnectionStatusBannerProps) {
  const { t } = useTranslation()
  if (!status || connectionStatusIsSettled(phase)) return null
  const dark = tone === 'dark'
  const label = phase ? connectionPhaseLabel(phase, t) : status
  return (
    <div
      className={`mx-3 mt-3 flex shrink-0 items-center gap-2 rounded-md border px-3 py-2 text-[12px] font-semibold shadow-sm ${
        dark
          ? 'absolute left-0 right-0 top-10 z-40 border-zinc-700 bg-zinc-950/90 text-zinc-100 backdrop-blur md:top-3'
          : 'border-blue-100 bg-blue-50 text-blue-800'
      } ${className}`}
      role="status"
      aria-live="polite"
    >
      <Loader2 className="h-3.5 w-3.5 shrink-0 animate-spin" />
      <span className="min-w-0 truncate">{label}</span>
    </div>
  )
}
