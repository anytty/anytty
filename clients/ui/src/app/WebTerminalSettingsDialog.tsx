import { Minus, Plus, X } from 'lucide-react'
import type { ReactNode } from 'react'
import { useTranslation } from 'react-i18next'
import { TERMINAL_FONT_OPTIONS, TERMINAL_THEME_OPTIONS, type TerminalSettings } from '../terminal/terminalSettings'
import { Button } from '../ui/button'
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle } from '../ui/dialog'
import { NativeSelect } from '../ui/native-select'
import { Switch } from '../ui/switch'

export function WebTerminalSettingsDialog({ open, settings, onChange, onOpenChange }: {
  open: boolean
  settings: TerminalSettings
  onChange: (patch: Partial<TerminalSettings>) => void
  onOpenChange: (open: boolean) => void
}) {
  const { t } = useTranslation()
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-h-[min(44rem,calc(100dvh-3rem))] max-w-2xl grid-rows-[auto_minmax(0,1fr)] gap-0 overflow-hidden p-0" hideClose>
        <DialogHeader className="relative border-b border-[var(--anytty-app-line)] px-6 py-5 pr-16">
          <DialogTitle>{t('common.settings')}</DialogTitle>
          <DialogDescription>{t('settings.terminal')}</DialogDescription>
          <Button
            aria-label={t('common.closeNamed', { name: t('common.settings') })}
            className="absolute right-3 top-3"
            onClick={() => onOpenChange(false)}
            size="icon"
            title={t('common.closeNamed', { name: t('common.settings') })}
            variant="ghost"
          >
            <X className="size-4" />
          </Button>
        </DialogHeader>

        <div className="min-h-0 overflow-y-auto">
          <SettingsBand title={t('settings.appearance')}>
            <SettingRow label={t('settings.fontSize')}>
              <div className="inline-flex h-9 items-center overflow-hidden rounded-md border border-[var(--anytty-app-line)] bg-[var(--anytty-app-bg)]">
                <Button
                  aria-label={t('settings.decreaseFont')}
                  className="size-9 rounded-none border-0 p-0 shadow-none"
                  disabled={settings.fontSize <= 8}
                  onClick={() => onChange({ fontSize: Math.max(8, settings.fontSize - 1) })}
                  size="icon-sm"
                  variant="ghost"
                >
                  <Minus className="size-3.5" />
                </Button>
                <output className="flex h-full min-w-12 items-center justify-center border-x border-[var(--anytty-app-line)] px-2 font-mono text-xs tabular-nums">
                  {settings.fontSize}
                </output>
                <Button
                  aria-label={t('settings.increaseFont')}
                  className="size-9 rounded-none border-0 p-0 shadow-none"
                  disabled={settings.fontSize >= 32}
                  onClick={() => onChange({ fontSize: Math.min(32, settings.fontSize + 1) })}
                  size="icon-sm"
                  variant="ghost"
                >
                  <Plus className="size-3.5" />
                </Button>
              </div>
            </SettingRow>
            <SettingRow label={t('settings.font')}>
              <NativeSelect
                aria-label={t('settings.font')}
                className="h-9 min-w-48 bg-[var(--anytty-app-bg)] text-xs"
                onChange={(event) => onChange({ fontFamily: event.currentTarget.value })}
                value={settings.fontFamily}
              >
                {TERMINAL_FONT_OPTIONS.map((font) => <option key={font.value} value={font.value}>{font.label}</option>)}
              </NativeSelect>
            </SettingRow>
            <SettingRow label={t('settings.renderer')}>
              <div className="inline-flex rounded-md border border-[var(--anytty-app-line)] bg-[var(--anytty-app-bg)] p-0.5">
                {(['auto', 'webgl', 'canvas', 'dom'] as const).map((renderer) => (
                  <button
                    aria-pressed={settings.renderer === renderer}
                    className="h-8 rounded px-3 text-xs font-medium text-[var(--anytty-app-muted)] outline-none hover:text-[var(--anytty-app-text)] focus-visible:ring-2 focus-visible:ring-[var(--anytty-app-accent)] aria-pressed:bg-[var(--anytty-app-surface-soft)] aria-pressed:text-[var(--anytty-app-text)]"
                    key={renderer}
                    onClick={() => onChange({ renderer })}
                    type="button"
                  >
                    {renderer === 'auto' ? t('settings.auto') : renderer === 'webgl' ? 'WebGL' : renderer === 'canvas' ? 'Canvas' : 'DOM'}
                  </button>
                ))}
              </div>
            </SettingRow>
            <SettingRow label={t('settings.cursorBlink')}>
              <Switch aria-label={t('settings.cursorBlink')} checked={settings.cursorBlink} onCheckedChange={(cursorBlink) => onChange({ cursorBlink })} />
            </SettingRow>
            <SettingRow label={t('settings.autoAcquireResizeOwner')}>
              <Switch aria-label={t('settings.autoAcquireResizeOwner')} checked={settings.autoAcquireResizeOwner} onCheckedChange={(autoAcquireResizeOwner) => onChange({ autoAcquireResizeOwner })} />
            </SettingRow>
          </SettingsBand>

          <SettingsBand title={t('settings.terminalTheme')}>
            <div className="grid grid-cols-2 gap-2 px-6 py-4 sm:grid-cols-3">
              {TERMINAL_THEME_OPTIONS.map((theme) => {
                const selected = settings.themeId === theme.id
                return (
                  <button
                    aria-pressed={selected}
                    className={`flex min-h-12 items-center gap-3 rounded-md border px-3 text-left text-xs font-medium outline-none transition-colors focus-visible:ring-2 focus-visible:ring-[var(--anytty-app-accent)] ${selected ? 'border-[var(--anytty-app-accent)] bg-[var(--anytty-app-surface-soft)] text-[var(--anytty-app-text)]' : 'border-[var(--anytty-app-line)] text-[var(--anytty-app-muted)] hover:bg-[var(--anytty-app-surface-soft)] hover:text-[var(--anytty-app-text)]'}`}
                    key={theme.id}
                    onClick={() => onChange({ themeId: theme.id })}
                    type="button"
                  >
                    <span className="flex size-7 shrink-0 overflow-hidden rounded border border-white/15" style={{ background: theme.theme.background }}>
                      <span className="h-full w-1/2" style={{ background: theme.theme.blue }} />
                      <span className="h-full w-1/2" style={{ background: theme.theme.foreground }} />
                    </span>
                    <span className="min-w-0 truncate">{theme.label}</span>
                  </button>
                )
              })}
            </div>
          </SettingsBand>
        </div>
      </DialogContent>
    </Dialog>
  )
}

function SettingsBand({ children, title }: { children: ReactNode; title: string }) {
  return (
    <section className="border-b border-[var(--anytty-app-line)] last:border-b-0">
      <h2 className="bg-[var(--anytty-app-surface-soft)] px-6 py-2 text-[11px] font-semibold uppercase text-[var(--anytty-app-muted)]">{title}</h2>
      {children}
    </section>
  )
}

function SettingRow({ children, label }: { children: ReactNode; label: string }) {
  return (
    <div className="flex min-h-14 items-center justify-between gap-4 border-b border-[var(--anytty-border-subtle)] px-6 py-2 last:border-b-0">
      <span className="text-sm font-medium text-[var(--anytty-app-text)]">{label}</span>
      <div className="shrink-0">{children}</div>
    </div>
  )
}
