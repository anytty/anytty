import { lazy, Suspense, useId } from 'react'
import { createPortal } from 'react-dom'
import { WifiOff, X } from 'lucide-react'
import { hapticSelection } from '../../platform/haptics'
import { ModalSurface } from '../../ui/ModalSurface'
import { Button } from '../../ui/button'
import { Spinner } from '../../ui/spinner'
import type { FilePreviewResponse, FilePreviewStreamOptions, FilePreviewStreamResult } from '../fileApi'
import { basename, formatBytes } from '../fileUtils'
import { PreviewNotice } from './PreviewNotice'
import { useTranslation } from 'react-i18next'
import '../../i18n'

const FileViewerPreview = lazy(() => import('./FileViewerPreview').then((module) => ({ default: module.FileViewerPreview })))

interface FilePreviewSheetProps {
  path: string
  preview: FilePreviewResponse | null
  loading: boolean
  error: string | null
  remoteAvailable?: boolean | undefined
  unavailableLabel?: string | undefined
  streamPreview(path: string, mimeType: string, options?: FilePreviewStreamOptions): Promise<FilePreviewStreamResult>
  onClose(): void
}

export function FilePreviewSheet({ path, preview, loading, error, remoteAvailable = true, unavailableLabel, streamPreview, onClose }: FilePreviewSheetProps) {
  const { t } = useTranslation()
  const titleId = useId()
  const subtitleId = useId()
  const title = preview?.name ?? basename(path)
  const subtitle = preview ? `${formatBytes(preview.size)} · ${preview.mimeType}` : path

  const sheet = (
    <ModalSurface
      className="fixed inset-0 z-[80] flex h-[100dvh] flex-col bg-[var(--background)]"
      data-testid="anytty-file-preview"
      aria-labelledby={titleId}
      aria-describedby={subtitleId}
      onRequestClose={onClose}
    >
      <header className="border-[var(--anytty-app-line)] bg-[var(--anytty-app-bg)] flex min-h-11 shrink-0 items-center gap-2 border-b px-2.5 pb-1.5 pt-[calc(env(safe-area-inset-top)+0.375rem)] md:h-11 md:py-0">
        <div className="flex min-w-0 flex-1 items-baseline gap-2">
          <h2 id={titleId} className="min-w-0 truncate text-[15px] font-semibold text-zinc-950">{title}</h2>
          <p id={subtitleId} className="min-w-0 truncate text-[11px] font-medium text-zinc-500">{subtitle}</p>
        </div>
        <Button
          className="shrink-0"
          size="icon"
          variant="ghost"
          aria-label={t('files.preview.close')}
          onClick={() => { hapticSelection(); onClose() }}
        >
          <X className="h-4 w-4" />
        </Button>
      </header>
      <div className="min-h-0 flex-1 overflow-hidden bg-zinc-100 pb-[env(safe-area-inset-bottom)]">
        {!remoteAvailable ? (
          <div className="flex h-56 flex-col items-center justify-center gap-3 px-6 text-center text-[14px] font-medium text-zinc-500" role="status" aria-live="polite">
            <WifiOff className="h-6 w-6" aria-hidden="true" />
            {unavailableLabel ?? t('workspace.connection.phase.waiting_network')}
          </div>
        ) : loading ? (
          <div className="flex h-56 flex-col items-center justify-center gap-3 text-[14px] font-medium text-zinc-500">
            <Spinner className="h-6 w-6 text-zinc-500" aria-hidden="true" />
            {t('files.preview.loading')}
          </div>
        ) : error ? (
          <PreviewNotice title={t('files.preview.error')} message={error} />
        ) : preview ? (
          <PreviewContent preview={preview} streamPreview={streamPreview} />
        ) : null}
      </div>
    </ModalSurface>
  )

  return typeof document === 'undefined' ? sheet : createPortal(sheet, document.body)
}

function PreviewContent({
  preview,
  streamPreview,
}: {
  preview: FilePreviewResponse
  streamPreview(path: string, mimeType: string, options?: FilePreviewStreamOptions): Promise<FilePreviewStreamResult>
}) {
  const { t } = useTranslation()
  return (
    <Suspense fallback={<PreviewNotice title={t('files.preview.loading')} message="" />}>
      <FileViewerPreview preview={preview} streamPreview={streamPreview} />
    </Suspense>
  )
}
