import { useState, type DragEvent } from 'react'
import { useTranslation } from 'react-i18next'
import type { TerminalPaneKey } from './terminalSplitLayout'

export const ANYTTY_TERMINAL_DRAG_TYPE = 'application/x-anytty-terminal-id'

export type WebPaneDropTarget = 'left' | 'right' | 'top' | 'bottom'

export function WebTerminalDropOverlay({ canSplit, draggedTerminalId, onDrop }: {
  canSplit: boolean
  draggedTerminalId: string
  onDrop: (terminalId: string, paneKey: TerminalPaneKey, target: WebPaneDropTarget) => void
}) {
  const { t } = useTranslation()
  const [preview, setPreview] = useState<WebPaneDropPreview | null>(null)
  if (!canSplit) return null
  const targetLabel = preview ? {
    left: t('workspace.splitLeft'),
    right: t('workspace.splitRight'),
    top: t('workspace.splitAbove'),
    bottom: t('workspace.splitBelow'),
  }[preview.target] : undefined

  const updatePreview = (event: DragEvent<HTMLDivElement>) => {
    event.preventDefault()
    event.dataTransfer.dropEffect = 'move'
    const next = paneDropPreviewFromEvent(event, draggedTerminalId)
    setPreview((current) => samePaneDropPreview(current, next) ? current : next)
  }

  return (
    <div
      className="absolute inset-0 z-40 hidden md:block"
      data-preview-pane-key={preview?.paneKey}
      data-preview-target={preview?.target}
      data-testid="anytty-web-terminal-drop-overlay"
      onDragEnter={updatePreview}
      onDragLeave={(event) => {
        if (event.relatedTarget instanceof Node && event.currentTarget.contains(event.relatedTarget)) return
        const bounds = event.currentTarget.getBoundingClientRect()
        if (event.clientX >= bounds.left && event.clientX <= bounds.right && event.clientY >= bounds.top && event.clientY <= bounds.bottom) return
        setPreview(null)
      }}
      onDragOver={updatePreview}
      onDrop={(event) => {
        event.preventDefault()
        const next = paneDropPreviewFromEvent(event, draggedTerminalId)
        setPreview(null)
        if (!next) return
        onDrop(terminalIdFromDrag(event) || draggedTerminalId, next.paneKey, next.target)
      }}
    >
      {preview ? <PaneDropPreview preview={preview} targetLabel={targetLabel} /> : null}
    </div>
  )
}

export function terminalIdFromDrag(event: DragEvent<HTMLElement>): string {
  return event.dataTransfer.getData(ANYTTY_TERMINAL_DRAG_TYPE) || event.dataTransfer.getData('text/plain')
}

interface WebPaneDropPreview {
  paneKey: TerminalPaneKey
  target: WebPaneDropTarget
  bounds: { left: number; top: number; width: number; height: number }
}

function PaneDropPreview({ preview, targetLabel }: { preview: WebPaneDropPreview; targetLabel: string | undefined }) {
  const before = preview.target === 'left' || preview.target === 'top'
  const vertical = preview.target === 'top' || preview.target === 'bottom'
  return (
    <div
      aria-label={targetLabel}
      className="absolute bg-black/35 backdrop-blur-[1px]"
      role="status"
      style={preview.bounds}
    >
      <div
        aria-hidden="true"
        className={`absolute inset-3 flex min-h-0 min-w-0 gap-1.5 overflow-hidden ${vertical ? 'flex-col' : 'flex-row'}`}
        data-preview-layout={vertical ? 'rows' : 'columns'}
        data-testid="anytty-web-terminal-drop-preview"
      >
        {before ? <DropPreviewPane incoming /> : <DropPreviewPane />}
        {before ? <DropPreviewPane /> : <DropPreviewPane incoming />}
      </div>
    </div>
  )
}

function DropPreviewPane({ incoming = false }: { incoming?: boolean }) {
  return (
    <div
      className={`min-h-0 min-w-0 flex-1 transition-colors duration-150 motion-reduce:transition-none ${incoming ? 'border-2 border-[var(--anytty-accent)] bg-[var(--anytty-accent)]/16' : 'border border-[var(--anytty-border)] bg-black/15'}`}
      data-preview-pane={incoming ? 'incoming' : 'existing'}
    />
  )
}

function paneDropPreviewFromEvent(event: DragEvent<HTMLElement>, fallbackTerminalId: string): WebPaneDropPreview | null {
  const overlayBounds = event.currentTarget.getBoundingClientRect()
  const draggedTerminalId = terminalIdFromDrag(event) || fallbackTerminalId
  const candidate = nearestTerminalPane(
    event.currentTarget.parentElement?.querySelectorAll<HTMLElement>('[data-pane-key][data-pane-terminal-id]') ?? [],
    event.clientX,
    event.clientY,
    draggedTerminalId,
  )
  if (!candidate) return null
  const x = Math.max(0, Math.min(candidate.bounds.width, event.clientX - candidate.bounds.left))
  const y = Math.max(0, Math.min(candidate.bounds.height, event.clientY - candidate.bounds.top))
  return {
    paneKey: candidate.paneKey,
    target: nearestDropTarget(x, y, candidate.bounds.width, candidate.bounds.height),
    bounds: {
      left: candidate.bounds.left - overlayBounds.left,
      top: candidate.bounds.top - overlayBounds.top,
      width: candidate.bounds.width,
      height: candidate.bounds.height,
    },
  }
}

function nearestTerminalPane(elements: Iterable<HTMLElement>, x: number, y: number, draggedTerminalId: string): { paneKey: TerminalPaneKey; bounds: DOMRect } | null {
  let nearest: { paneKey: TerminalPaneKey; terminalId: string; bounds: DOMRect } | null = null
  let nearestDistance = Number.POSITIVE_INFINITY
  for (const element of elements) {
    const paneKey = element.dataset.paneKey
    const terminalId = element.dataset.paneTerminalId
    if (!isTerminalPaneKey(paneKey) || !terminalId) continue
    const bounds = element.getBoundingClientRect()
    if (bounds.width <= 0 || bounds.height <= 0) continue
    const distance = distanceToBounds(x, y, bounds)
    if (distance >= nearestDistance) continue
    nearest = { paneKey, terminalId, bounds }
    nearestDistance = distance
  }
  if (!nearest || nearest.terminalId === draggedTerminalId) return null
  return { paneKey: nearest.paneKey, bounds: nearest.bounds }
}

function distanceToBounds(x: number, y: number, bounds: DOMRect): number {
  const dx = x < bounds.left ? bounds.left - x : x > bounds.right ? x - bounds.right : 0
  const dy = y < bounds.top ? bounds.top - y : y > bounds.bottom ? y - bounds.bottom : 0
  return (dx * dx) + (dy * dy)
}

function isTerminalPaneKey(value: string | undefined): value is TerminalPaneKey {
  return value === 'primary' || Boolean(value?.startsWith('terminal:'))
}

function samePaneDropPreview(current: WebPaneDropPreview | null, next: WebPaneDropPreview | null): boolean {
  if (!current || !next) return current === next
  return current.paneKey === next.paneKey
    && current.target === next.target
    && current.bounds.left === next.bounds.left
    && current.bounds.top === next.bounds.top
    && current.bounds.width === next.bounds.width
    && current.bounds.height === next.bounds.height
}

function nearestDropTarget(x: number, y: number, width: number, height: number): WebPaneDropTarget {
  if (width <= 0 || height <= 0) return 'bottom'
  const horizontal = (x / width) - 0.5
  const vertical = (y / height) - 0.5
  if (Math.abs(horizontal) > Math.abs(vertical)) return horizontal < 0 ? 'left' : 'right'
  return vertical < 0 ? 'top' : 'bottom'
}
