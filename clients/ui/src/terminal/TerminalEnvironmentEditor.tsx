import { useState } from 'react'
import { ClipboardPaste, Plus, Trash2, X } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { environmentEntry, parseEnvironmentBlock, parseEnvironmentEntry } from './terminalCreateForm'
import '../i18n'

/** TerminalEnvironmentEditorProps 传递 TerminalCreateSpec.env 的 UI 投影与受控更新。 */
export interface TerminalEnvironmentEditorProps {
  value: string[]
  onChange: (value: string[]) => void
}

/** TerminalEnvironmentEditor 只编辑 TerminalCreateSpec.env 的 UI 投影，不持久化或展示 daemon 环境。 */
export function TerminalEnvironmentEditor({ value, onChange }: TerminalEnvironmentEditorProps) {
  const { t } = useTranslation()
  const [pasteOpen, setPasteOpen] = useState(false)
  const [pasteValue, setPasteValue] = useState('')
  const [pasteError, setPasteError] = useState<string | null>(null)

  const update = (index: number, field: 'key' | 'value', next: string) => {
    const current = parseEnvironmentEntry(value[index] ?? '')
    onChange(value.map((entry, entryIndex) => entryIndex === index
      ? environmentEntry({ ...current, [field]: next })
      : entry))
  }

  const applyPaste = () => {
    try {
      const parsed = parseEnvironmentBlock(pasteValue)
      onChange(parsed)
      setPasteError(null)
      setPasteOpen(false)
      setPasteValue('')
    } catch (error) {
      setPasteError(error instanceof Error ? error.message : String(error))
    }
  }

  return (
    <fieldset className="flex flex-col gap-2">
      <legend className="mb-2 text-[14px] font-semibold text-zinc-700">{t('workspace.terminalForm.environment')}</legend>
      {value.map((entry, index) => {
        const variable = parseEnvironmentEntry(entry)
        return (
          <div className="grid grid-cols-[minmax(0,0.9fr)_minmax(0,1.1fr)_3rem] gap-2" key={index}>
            <input
              aria-label={t('workspace.terminalForm.environmentKey')}
              className="min-h-12 min-w-0 border border-[var(--anytty-app-line)] bg-zinc-50 px-3 text-[15px] text-zinc-900 outline-none"
              placeholder={t('workspace.terminalForm.environmentKey')}
              value={variable.key}
              onChange={(event) => update(index, 'key', event.currentTarget.value)}
            />
            <input
              aria-label={t('workspace.terminalForm.environmentValue')}
              className="min-h-12 min-w-0 border border-[var(--anytty-app-line)] bg-zinc-50 px-3 text-[15px] text-zinc-900 outline-none"
              placeholder={t('workspace.terminalForm.environmentValue')}
              value={variable.value}
              onChange={(event) => update(index, 'value', event.currentTarget.value)}
            />
            <button
              type="button"
              aria-label={t('workspace.terminalForm.removeEnvironment')}
              className="flex min-h-12 min-w-12 items-center justify-center border border-[var(--anytty-app-line)] text-zinc-500 active:bg-zinc-100"
              onClick={() => onChange(value.filter((_, entryIndex) => entryIndex !== index))}
            >
              <Trash2 className="h-4 w-4" />
            </button>
          </div>
        )
      })}
      <div className="grid grid-cols-2 gap-2">
        <button type="button" className="anytty-app-secondary-button min-h-12 gap-2 px-3 text-[13px] font-semibold" onClick={() => onChange([...value, '='])}>
          <Plus className="h-4 w-4" />
          {t('workspace.terminalForm.addEnvironment')}
        </button>
        <button type="button" className="anytty-app-secondary-button min-h-12 gap-2 px-3 text-[13px] font-semibold" onClick={() => { setPasteError(null); setPasteOpen(true) }}>
          <ClipboardPaste className="h-4 w-4" />
          {t('workspace.terminalForm.pasteEnvironment')}
        </button>
      </div>
      {pasteOpen ? (
        <div className="flex flex-col gap-2 border border-[var(--anytty-app-line)] bg-zinc-50 p-3">
          <div className="flex items-center justify-between gap-2">
            <span className="text-[13px] font-semibold text-zinc-700">{t('workspace.terminalForm.pasteEnvironment')}</span>
            <button type="button" aria-label={t('common.close')} className="flex h-11 w-11 items-center justify-center text-zinc-500" onClick={() => setPasteOpen(false)}>
              <X className="h-4 w-4" />
            </button>
          </div>
          <textarea
            aria-label={t('workspace.terminalForm.environmentPasteContent')}
            className="min-h-28 resize-y border border-[var(--anytty-app-line)] bg-white p-3 font-mono text-[14px] text-zinc-900 outline-none"
            placeholder={'KEY=value\nexport OTHER=value'}
            value={pasteValue}
            onChange={(event) => setPasteValue(event.currentTarget.value)}
          />
          {pasteError ? <p className="text-[13px] font-medium text-red-700" role="alert">{pasteError}</p> : null}
          <button type="button" className="anytty-app-primary-button min-h-12 px-4 text-[14px] font-semibold" onClick={applyPaste}>
            {t('workspace.terminalForm.applyEnvironment')}
          </button>
        </div>
      ) : null}
    </fieldset>
  )
}
