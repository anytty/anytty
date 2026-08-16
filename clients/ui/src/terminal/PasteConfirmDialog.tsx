import { hapticImpact, hapticSelection } from '../platform/haptics'
import { useTranslation } from 'react-i18next'
import '../i18n'
import { ModalSurface } from '../ui/ModalSurface'
import { Button } from '../ui/button'

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
        className="w-full overflow-hidden rounded-t-xl border-y border-[#3f3f46] bg-[#09090b] text-[#f4f4f5] md:max-w-md md:rounded-xl md:border"
        onRequestClose={onCancel}
        onClick={(event) => event.stopPropagation()}
      >
        <header className="border-b border-[#27272a] px-4 py-3">
          <h2 className="text-[16px] font-bold" id="anytty-paste-confirm-title">{t('terminal.paste.title')}</h2>
          <p className="mt-1 text-[12px] font-medium text-[#a1a1aa]">
            {t('terminal.paste.summary', { lines: lineCount, characters: text.length })}
          </p>
        </header>
        <pre className="max-h-64 overflow-auto whitespace-pre-wrap break-words bg-black px-4 py-3 font-mono text-[12px] leading-5 text-[#d4d4d8]">
          {preview}
        </pre>
        <div className="grid grid-cols-2 gap-3 border-t border-[#27272a] p-3">
          <Button
            variant="secondary"
            className="h-11 border-[#3f3f46] bg-[#27272a] font-semibold text-[#e4e4e7] hover:bg-[#3f3f46] active:bg-[#3f3f46]"
            onClick={() => { hapticSelection(); onCancel() }}
          >
            {t('common.cancel')}
          </Button>
          <Button
            className="h-11 border-blue-600 bg-blue-600 font-semibold text-white hover:bg-blue-600/90 active:bg-blue-500"
            onClick={() => { hapticImpact(); onConfirm() }}
          >
            {t('terminal.paste.confirm')}
          </Button>
        </div>
      </ModalSurface>
    </div>
  )
}
