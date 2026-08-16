import * as React from 'react'
import { cn } from './utils'

export type CheckedState = boolean | 'indeterminate'

export interface CheckboxProps
  extends Omit<React.ButtonHTMLAttributes<HTMLButtonElement>, 'checked' | 'defaultChecked' | 'onChange' | 'value'> {
  checked?: CheckedState
  defaultChecked?: CheckedState
  onCheckedChange?: (checked: CheckedState) => void
  value?: string
}

export const Checkbox = React.forwardRef<HTMLButtonElement, CheckboxProps>(function Checkbox(
  { checked, className, defaultChecked = false, disabled, onCheckedChange, value = 'on', ...props },
  ref,
) {
  const controlled = checked !== undefined
  const [internalChecked, setInternalChecked] = React.useState<CheckedState>(defaultChecked)
  const state = controlled ? checked : internalChecked
  const active = state === true
  const mixed = state === 'indeterminate'

  function toggle() {
    if (disabled) return
    const next = active ? false : true
    if (!controlled) setInternalChecked(next)
    onCheckedChange?.(next)
  }

  return (
    <button
      {...props}
      aria-checked={mixed ? 'mixed' : active}
      data-state={mixed ? 'indeterminate' : active ? 'checked' : 'unchecked'}
      disabled={disabled}
      ref={ref}
      role="checkbox"
      type="button"
      value={value}
      className={cn(
        'peer flex h-5 w-5 shrink-0 items-center justify-center rounded-sm border border-[var(--input)] bg-[var(--background)] text-[var(--primary-foreground)] shadow-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[var(--ring)] focus-visible:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50 data-[state=checked]:border-[var(--primary)] data-[state=checked]:bg-[var(--primary)] data-[state=indeterminate]:border-[var(--primary)] data-[state=indeterminate]:bg-[var(--primary)]',
        className,
      )}
      onClick={(event) => {
        props.onClick?.(event)
        if (!event.defaultPrevented) toggle()
      }}
    >
      <span aria-hidden="true" className={cn('h-2.5 w-1.5 rotate-45 border-b-2 border-r-2 border-current', !active && 'hidden')} />
      <span aria-hidden="true" className={cn('h-0.5 w-2.5 bg-current', !mixed && 'hidden')} />
    </button>
  )
})
