import FileViewer, { type FileViewerHandle, type ViewerOptions, type ViewerState } from '@file-viewer/react'
import textRenderer from '@file-viewer/renderer-text'
import pdfRenderer from '@file-viewer/renderer-pdf'
import wordRenderer from '@file-viewer/renderer-word'
import spreadsheetRenderer from '@file-viewer/renderer-spreadsheet'
import mediaRenderer from '@file-viewer/renderer-media'
import modelRenderer from '@file-viewer/renderer-3d'
import { ChevronDown, ChevronUp, Eraser, Minus, Plus, Search } from 'lucide-react'
import { type FormEvent, type ReactNode, type TouchEvent, useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { useTranslation } from 'react-i18next'
import type { FilePreviewResponse, FilePreviewStreamOptions, FilePreviewStreamResult } from '../fileApi'
import '../../i18n'
import { Button } from '../../ui/button'
import { Input } from '../../ui/input'
import { Spinner } from '../../ui/spinner'

interface FileViewerPreviewProps {
  preview: FilePreviewResponse
  streamPreview(path: string, mimeType: string, options?: FilePreviewStreamOptions): Promise<FilePreviewStreamResult>
}

const previewRenderers = [textRenderer, pdfRenderer, wordRenderer, spreadsheetRenderer, mediaRenderer, modelRenderer] as unknown as NonNullable<ViewerOptions['renderers']>
const searchableDocumentExtensions = new Set([
  'csv', 'doc', 'docx', 'htm', 'html', 'md', 'markdown', 'odp', 'ods', 'odt', 'ofd',
  'pdf', 'ppt', 'pptx', 'rtf', 'tsv', 'txt', 'xls', 'xlsx', 'xml',
])
const pinchStepRatio = 1.12
// The viewer owns these nested nodes, so Tailwind utilities cannot target them directly.
const integrationStyle = `
  .fv-video-viewer {
    height: 100% !important;
    min-height: 0 !important;
    padding: 0 !important;
    align-items: stretch !important;
    background: #050505 !important;
  }
  .fv-video-shell {
    display: flex !important;
    width: 100% !important;
    height: 100% !important;
    min-height: 0 !important;
    flex-direction: column !important;
    border: 0 !important;
    border-radius: 0 !important;
    background: #050505 !important;
    box-shadow: none !important;
  }
  .fv-video-heading { display: none !important; }
  .fv-video-player {
    width: 100% !important;
    height: 100% !important;
    min-height: 0 !important;
    flex: 1 1 auto !important;
    aspect-ratio: auto !important;
    object-fit: contain !important;
  }
  .code-viewer:not(.code-viewer--virtual) {
    display: flex !important;
    width: 100% !important;
    height: 100% !important;
    min-width: 0 !important;
    min-height: 0 !important;
    flex-direction: column !important;
    overflow: hidden !important;
    background: var(--code-bg, #f4f4f5) !important;
  }
  .code-viewer:not(.code-viewer--virtual) .code-area {
    width: 100% !important;
    min-width: 0 !important;
    min-height: 0 !important;
    flex: 1 1 auto !important;
    overflow: auto !important;
    background: var(--code-bg, #f4f4f5) !important;
  }
  .code-viewer:not(.code-viewer--virtual) .code-area code {
    width: max-content !important;
    min-width: 100% !important;
  }
`

export function FileViewerPreview({ preview, streamPreview }: FileViewerPreviewProps) {
  const { i18n, t } = useTranslation()
  const isChinese = i18n.resolvedLanguage?.toLowerCase().startsWith('zh') ?? false
  const extension = preview.name.slice(preview.name.lastIndexOf('.') + 1).toLowerCase()
  const showSearch = preview.category === 'text' || searchableDocumentExtensions.has(extension)
  const [source, setSource] = useState<File | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [searchQuery, setSearchQuery] = useState('')
  const [viewerState, setViewerState] = useState<ViewerState | null>(null)
  const viewerRef = useRef<FileViewerHandle | null>(null)
  const searchInputRef = useRef<HTMLInputElement | null>(null)
  const pinchDistanceRef = useRef<number | null>(null)
  const unsupportedCopy = isChinese ? {
    description: '可以下载后使用系统中的其他应用打开。',
    message: '此文件类型暂不支持应用内预览。',
    title: '无法预览',
  } : {
    description: 'Download the file to open it in another app.',
    message: 'This file type is not available for in-app preview.',
    title: 'Preview unavailable',
  }
  const options = useMemo(() => ({
    rendererMode: 'extend' as const,
    autoRenderers: false,
    renderers: previewRenderers,
    theme: 'light' as const,
    locale: isChinese ? 'zh-CN' as const : 'en-US' as const,
    messages: {
      'state.unsupported.install.title': unsupportedCopy.title,
      'state.unsupported.install.message': unsupportedCopy.message,
      'state.unsupported.install.description': unsupportedCopy.description,
      'state.unsupported.title': unsupportedCopy.title,
      'state.unsupported.message': unsupportedCopy.message,
      'state.unsupported.description': unsupportedCopy.description,
    },
    toolbar: {
      position: 'bottom-right' as const,
      download: false,
      exportHtml: false,
      print: false,
      search: false,
      theme: false,
      zoom: false,
    },
    text: { toolbar: false },
    ui: { density: 'compact' as const, surfaceBackground: '#f4f4f5' },
    fit: { mode: 'width' as const, resize: 'until-interaction' as const, padding: 8 },
  }), [isChinese, unsupportedCopy.description, unsupportedCopy.message, unsupportedCopy.title])

  const searchLabels = isChinese ? {
    clear: '清除搜索',
    input: '搜索文档',
    next: '下一个匹配项',
    noMatches: '无匹配项',
    placeholder: '搜索',
    previous: '上一个匹配项',
  } : {
    clear: 'Clear search',
    input: 'Search document',
    next: 'Next match',
    noMatches: 'No matches',
    placeholder: 'Search',
    previous: 'Previous match',
  }
  const zoomLabels = isChinese ? {
    in: '放大',
    out: '缩小',
    reset: '重置缩放',
  } : {
    in: 'Zoom in',
    out: 'Zoom out',
    reset: 'Reset zoom',
  }

  const updateViewerState = useCallback((state: ViewerState) => {
    setViewerState(state)
  }, [])

  const runSearch = useCallback(async (event?: FormEvent) => {
    event?.preventDefault()
    const query = searchQuery.trim()
    if (query) await viewerRef.current?.searchDocument(query)
    else await viewerRef.current?.clearDocumentSearch()
    searchInputRef.current?.focus({ preventScroll: true })
  }, [searchQuery])

  const moveSearch = useCallback(async (direction: 'previous' | 'next') => {
    if (direction === 'previous') await viewerRef.current?.previousSearchResult()
    else await viewerRef.current?.nextSearchResult()
    searchInputRef.current?.focus({ preventScroll: true })
  }, [])

  const clearSearch = useCallback(async () => {
    setSearchQuery('')
    await viewerRef.current?.clearDocumentSearch()
    searchInputRef.current?.focus({ preventScroll: true })
  }, [])

  const handleTouchStart = useCallback((event: TouchEvent<HTMLDivElement>) => {
    pinchDistanceRef.current = event.touches.length === 2 ? touchDistance(event) : null
  }, [])

  const handleTouchMove = useCallback((event: TouchEvent<HTMLDivElement>) => {
    if (event.touches.length !== 2 || pinchDistanceRef.current === null) return
    const distance = touchDistance(event)
    const ratio = distance / pinchDistanceRef.current
    if (ratio >= pinchStepRatio) {
      event.preventDefault()
      pinchDistanceRef.current = distance
      void viewerRef.current?.zoomIn()
    } else if (ratio <= 1 / pinchStepRatio) {
      event.preventDefault()
      pinchDistanceRef.current = distance
      void viewerRef.current?.zoomOut()
    }
  }, [])

  const handleTouchEnd = useCallback((event: TouchEvent<HTMLDivElement>) => {
    if (event.touches.length < 2) pinchDistanceRef.current = null
  }, [])

  useEffect(() => {
    const controller = new AbortController()
    setSource(null)
    setError(null)
    setSearchQuery('')
    setViewerState(null)

    const load = async () => {
      if (preview.content !== undefined) {
        setSource(new File([preview.content], preview.name, { type: preview.mimeType }))
        return
      }
      const streamed = await streamPreview(preview.path, preview.mimeType, { signal: controller.signal })
      if (!controller.signal.aborted) {
        setSource(new File([streamed.blob], preview.name, {
          type: streamed.blob.type || preview.mimeType,
          lastModified: Date.now(),
        }))
      }
    }

    void load().catch((reason) => {
      if (!controller.signal.aborted) setError(reason instanceof Error ? reason.message : String(reason))
    })
    return () => {
      controller.abort()
    }
  }, [preview.content, preview.mimeType, preview.path, preview.size, streamPreview])

  useEffect(() => {
    if (!source) return
    const frame = window.requestAnimationFrame(() => {
      const shadowRoot = viewerRef.current?.getController?.()?.container.shadowRoot
      if (!shadowRoot || shadowRoot.querySelector('style[data-anytty-file-viewer]')) return
      const style = document.createElement('style')
      style.dataset.anyttyFileViewer = ''
      style.textContent = integrationStyle
      shadowRoot.append(style)
    })
    return () => window.cancelAnimationFrame(frame)
  }, [source])

  if (error) {
    return <div className="flex min-h-56 items-center justify-center px-6 text-center text-sm text-red-700" role="alert">{error}</div>
  }
  if (!source) {
    return (
      <div className="flex min-h-56 items-center justify-center gap-3 text-sm font-medium text-zinc-500" role="status">
        <Spinner className="h-5 w-5" aria-hidden="true" />
        {t('files.preview.loading')}
      </div>
    )
  }
  const searchState = viewerState?.search
  const zoomState = viewerState?.zoom
  const showToolbar = showSearch || zoomState !== null && zoomState !== undefined

  return (
    <div
      className="flex h-full min-h-full w-full flex-col overflow-hidden bg-[var(--background)]"
      onTouchStart={handleTouchStart}
      onTouchMove={handleTouchMove}
      onTouchEnd={handleTouchEnd}
      onTouchCancel={handleTouchEnd}
      style={{ touchAction: 'pan-x pan-y' }}
    >
      {showToolbar ? <form
        className="flex h-10 w-full shrink-0 items-center justify-center gap-0.5 border-b border-zinc-200 bg-[var(--background)] px-1.5"
        onSubmit={(event) => void runSearch(event)}
        role={showSearch ? 'search' : undefined}
      >
        {showSearch ? <>
          <Search className="ml-1 h-4 w-4 shrink-0 text-zinc-500" aria-hidden="true" />
          <Input
            ref={searchInputRef}
            type="search"
            value={searchQuery}
            onChange={(event) => setSearchQuery(event.target.value)}
            placeholder={searchLabels.placeholder}
            aria-label={searchLabels.input}
            className="h-8 min-w-12 w-full max-w-72 border-0 bg-transparent px-1 text-zinc-950 shadow-none placeholder:text-zinc-400"
          />
          {searchState && searchState.query ? (
            <span
              className="shrink-0 px-1 text-xs tabular-nums text-zinc-500"
              aria-label={searchState.total > 0 ? undefined : searchLabels.noMatches}
              aria-live="polite"
            >
              {searchState.total > 0 ? `${searchState.currentIndex + 1}/${searchState.total}` : '0/0'}
            </span>
          ) : null}
          <ToolbarButton label={searchLabels.previous} disabled={!searchState?.total} onClick={() => void moveSearch('previous')}>
            <ChevronUp className="h-4 w-4" />
          </ToolbarButton>
          <ToolbarButton label={searchLabels.next} disabled={!searchState?.total} onClick={() => void moveSearch('next')}>
            <ChevronDown className="h-4 w-4" />
          </ToolbarButton>
          <ToolbarButton label={searchLabels.clear} disabled={!searchQuery && !searchState?.query} onClick={() => void clearSearch()}>
            <Eraser className="h-4 w-4" />
          </ToolbarButton>
        </> : null}
        {showSearch && zoomState ? <span className="mx-0.5 h-5 w-px shrink-0 bg-zinc-200" aria-hidden="true" /> : null}
        {zoomState ? <div className="flex shrink-0 items-center gap-0.5">
          <ToolbarButton label={zoomLabels.out} disabled={!zoomState.canZoomOut} onClick={() => void viewerRef.current?.zoomOut()}>
            <Minus className="h-4 w-4" />
          </ToolbarButton>
          <Button variant="ghost"
            type="button"
            className="h-8 min-w-12 rounded-md px-1 text-xs font-medium tabular-nums text-zinc-600 hover:bg-zinc-100 disabled:opacity-40"
            aria-label={zoomLabels.reset}
            disabled={!zoomState.canReset}
            onClick={() => void viewerRef.current?.resetZoom()}
          >
            {zoomState.label}
          </Button>
          <ToolbarButton label={zoomLabels.in} disabled={!zoomState.canZoomIn} onClick={() => void viewerRef.current?.zoomIn()}>
            <Plus className="h-4 w-4" />
          </ToolbarButton>
        </div> : null}
      </form> : null}
      <FileViewer
        ref={viewerRef}
        file={source}
        className="min-h-0 w-full flex-1 bg-[var(--background)]"
        filename={preview.name}
        size={preview.size}
        options={options}
        onStateChange={updateViewerState}
        style={{ height: 'auto' }}
      />
    </div>
  )
}

function ToolbarButton({
  label,
  disabled,
  onClick,
  children,
}: {
  label: string
  disabled: boolean
  onClick(): void
  children: ReactNode
}) {
  return (
    <Button variant="ghost"
      type="button"
      className="inline-flex h-8 w-8 shrink-0 items-center justify-center rounded-md text-zinc-600 hover:bg-zinc-100 disabled:opacity-40"
      aria-label={label}
      disabled={disabled}
      onClick={onClick}
    >
      {children}
    </Button>
  )
}

function touchDistance(event: TouchEvent<HTMLElement>): number {
  const first = event.touches[0]
  const second = event.touches[1]
  if (!first || !second) return 0
  return Math.hypot(second.clientX - first.clientX, second.clientY - first.clientY)
}
