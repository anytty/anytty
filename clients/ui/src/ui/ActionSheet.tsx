import { useId, useRef, type ReactNode } from 'react'
import { X } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import '../i18n'
import { hapticImpact, hapticSelection } from '../platform/haptics'
import { ModalSurface } from './ModalSurface'
import { Button } from './button'

export interface ActionSheetItem {
  label: string
  ariaLabel?: string | undefined
  subtitle?: string | undefined
  icon: ReactNode
  onClick: () => void
  danger?: boolean
  closeOnClick?: boolean | undefined
  secondaryAction?: ActionSheetSecondaryAction | undefined
}

export interface ActionSheetSecondaryAction {
  label: string
  icon: ReactNode
  onClick: () => void
  danger?: boolean | undefined
  closeOnClick?: boolean | undefined
}

export interface ActionSheetProps {
  isOpen: boolean
  onClose: () => void
  title?: string | undefined
  subtitle?: string | undefined
  actions: ActionSheetItem[]
}

export function ActionSheet({ isOpen, onClose, title, subtitle, actions }: ActionSheetProps) {
  const { t } = useTranslation()
  const titleId = useId()
  const subtitleId = useId()
  const firstActionRef = useRef<HTMLButtonElement>(null)
  if (!isOpen) return null
  const closeWithHaptic = () => {
    hapticSelection()
    onClose()
  }

  return (
    <div
      className="fixed inset-0 z-[100] flex items-end justify-center bg-black/40 backdrop-blur-[2px] md:items-center"
      onClick={closeWithHaptic}
      data-testid="action-sheet-backdrop"
    >
      <ModalSurface
        aria-label={title ? undefined : subtitle || t('common.actions')}
        aria-labelledby={title ? titleId : undefined}
        aria-describedby={title && subtitle ? subtitleId : undefined}
        className="w-full max-w-xl animate-slide-up rounded-t-xl border-t border-[var(--anytty-app-line)] bg-[var(--anytty-app-surface)] pb-[env(safe-area-inset-bottom,20px)] shadow-xl md:rounded-xl md:border md:pb-4"
        initialFocusRef={firstActionRef}
        onRequestClose={closeWithHaptic}
        onClick={(e) => e.stopPropagation()}
      >
        <div className="flex flex-col">
          <div className="mx-auto mt-3 h-1 w-10 rounded-full bg-[var(--anytty-app-line-strong)] md:hidden" />

          <div className="flex items-center justify-between px-5 pt-4 pb-2">
            <div className="flex flex-col">
              {title && <h3 id={titleId} className="text-[17px] font-bold text-zinc-900">{title}</h3>}
              {subtitle && <p id={subtitleId} className="text-[13px] font-medium text-zinc-500">{subtitle}</p>}
            </div>
            <Button
              aria-label={t('common.close')}
              size="icon"
              variant="ghost"
              onClick={closeWithHaptic}
            >
              <X className="h-4 w-4" />
            </Button>
          </div>

          <div className="mt-2 grid grid-cols-1 divide-y divide-zinc-100">
            {actions.map((action, index) => {
              if (action.secondaryAction) {
                return (
                  <div
                    key={index}
                    className="flex w-full items-stretch gap-2 px-5 py-2 transition-colors hover:bg-zinc-50 active:bg-zinc-50"
                    data-testid="action-sheet-item"
                  >
                    <Button
                      ref={index === 0 ? firstActionRef : undefined}
                      aria-label={action.ariaLabel ?? action.label}
                      variant="ghost"
                      className={`min-h-11 min-w-0 flex-1 justify-start gap-4 px-0 py-2 text-left hover:bg-transparent ${
                        action.danger ? 'text-red-600' : 'text-zinc-700'
                      }`}
                      onClick={() => runSheetAction(action, onClose)}
                    >
                      <ActionIcon danger={action.danger}>{action.icon}</ActionIcon>
                      <ActionText action={action} />
                    </Button>
                    <Button
                      aria-label={action.secondaryAction.label}
                      title={action.secondaryAction.label}
                      size="icon"
                      variant="outline"
                      className={`my-auto shrink-0 ${
                        action.secondaryAction.danger
                          ? 'border-red-200 bg-red-50 text-red-600 hover:bg-red-100 active:bg-red-100'
                          : 'border-[var(--anytty-app-line)] bg-zinc-50 text-zinc-500 hover:bg-zinc-100 active:bg-zinc-100'
                      }`}
                      onClick={(event) => {
                        event.stopPropagation()
                        runSheetAction(action.secondaryAction!, onClose)
                      }}
                    >
                      {action.secondaryAction.icon}
                    </Button>
                  </div>
                )
              }
              return (
                <Button
                  ref={index === 0 ? firstActionRef : undefined}
                  key={index}
                  aria-label={action.ariaLabel ?? action.label}
                  data-testid="action-sheet-item"
                  variant="ghost"
                  className={`min-h-12 w-full justify-start gap-3 rounded-none px-5 py-2 text-left hover:bg-zinc-50 active:bg-[var(--anytty-app-soft)] ${
                    action.danger ? 'text-red-600' : 'text-zinc-700'
                  }`}
                  onClick={() => runSheetAction(action, onClose)}
                >
                  <ActionIcon danger={action.danger}>{action.icon}</ActionIcon>
                  <ActionText action={action} />
                </Button>
              )
            })}
          </div>
        </div>
      </ModalSurface>
    </div>
  )
}

function runSheetAction(
  action: {
    onClick: () => void
    danger?: boolean | undefined
    closeOnClick?: boolean | undefined
  },
  onClose: () => void,
) {
  if (action.danger) hapticImpact()
  else hapticSelection()
  action.onClick()
  if (action.closeOnClick !== false) onClose()
}

function ActionIcon({ children, danger }: { children: ReactNode; danger?: boolean | undefined }) {
  return (
    <div className={`flex h-9 w-9 shrink-0 items-center justify-center rounded-md border ${
      danger ? 'border-red-200 bg-red-50' : 'border-[var(--anytty-app-line)] bg-zinc-50'
    }`}>
      <span className={danger ? 'text-red-600' : 'text-zinc-500'}>
        {children}
      </span>
    </div>
  )
}

function ActionText({ action }: { action: Pick<ActionSheetItem, 'label' | 'subtitle'> }) {
  return (
    <span className="min-w-0">
      <span className="block truncate text-[16px] font-semibold">{action.label}</span>
      {action.subtitle ? (
        <span className="mt-0.5 block truncate text-[12px] font-medium text-zinc-500">{action.subtitle}</span>
      ) : null}
    </span>
  )
}
