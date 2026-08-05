import { hapticImpact, hapticSelection } from '../platform/haptics'
import { useTranslation } from 'react-i18next'
import '../i18n'
import { ModalSurface } from '../ui/ModalSurface'

export interface PasteConfirmDialogProps {
  text: string
  onCancel: () => void
  onConfirm: () => void
}

export function PasteConfirmDialog({ text, onCancel, onConfirm }: PasteConfirmDialogProps) {
  const { t } = useTranslation()
  const lineCount = text.split(/\r\n|\r|\n/).length
  const preview = text.length > 600 ? `${text.slice(0, 600)}...` : text

  return (
    <div className="absolute inset-0 z-50 flex items-end bg-black/60 backdrop-blur-sm md:items-center md:justify-center" data-testid="anytty-paste-confirm" onClick={() => { hapticSelection(); onCancel() }}>
      <ModalSurface
        aria-labelledby="anytty-paste-confirm-title"
        className="w-full overflow-hidden border-y border-zinc-700 bg-zinc-950 text-zinc-100 md:max-w-md md:border"
        onRequestClose={onCancel}
        onClick={(event) => event.stopPropagation()}
      >
        <header className="border-b border-zinc-800 px-4 py-3">
          <h2 className="text-[16px] font-bold" id="anytty-paste-confirm-title">{t('terminal.paste.title')}</h2>
          <p className="mt-1 text-[12px] font-medium text-zinc-500">
            {t('terminal.paste.summary', { lines: lineCount, characters: text.length })}
          </p>
        </header>
        <pre className="max-h-64 overflow-auto whitespace-pre-wrap break-words bg-black px-4 py-3 font-mono text-[12px] leading-5 text-zinc-300">
          {preview}
        </pre>
        <div className="grid grid-cols-2 gap-3 border-t border-zinc-800 p-3">
          <button
            type="button"
            className="h-11 border border-zinc-700 bg-zinc-800 text-[14px] font-semibold text-zinc-200 hover:bg-zinc-700/80 active:bg-zinc-700"
            onClick={() => { hapticSelection(); onCancel() }}
          >
            {t('common.cancel')}
          </button>
          <button
            type="button"
            className="h-11 bg-blue-600 text-[14px] font-semibold text-white hover:bg-blue-600/90 active:bg-blue-500"
            onClick={() => { hapticImpact(); onConfirm() }}
          >
            {t('terminal.paste.confirm')}
          </button>
        </div>
      </ModalSurface>
    </div>
  )
}
