import * as React from 'react'
import { cva, type VariantProps } from 'class-variance-authority'
import { cn } from './utils'

export const alertVariants = cva(
  'relative w-full rounded-lg border border-[var(--border)] bg-[var(--card)] p-4 text-[var(--card-foreground)]',
  {
    variants: {
      variant: {
        default: '',
        destructive: 'border-[var(--destructive)] text-[var(--foreground)]',
      },
    },
    defaultVariants: { variant: 'default' },
  },
)

export interface AlertProps
  extends React.HTMLAttributes<HTMLDivElement>,
    VariantProps<typeof alertVariants> {}

export const Alert = React.forwardRef<HTMLDivElement, AlertProps>(function Alert(
  { className, variant, ...props },
  ref,
) {
  return <div ref={ref} className={cn(alertVariants({ variant }), className)} {...props} />
})

export const AlertTitle = React.forwardRef<HTMLHeadingElement, React.HTMLAttributes<HTMLHeadingElement>>(
  function AlertTitle({ className, ...props }, ref) {
    return <h3 ref={ref} className={cn('font-semibold leading-none tracking-normal', className)} {...props} />
  },
)

export const AlertDescription = React.forwardRef<HTMLDivElement, React.HTMLAttributes<HTMLDivElement>>(
  function AlertDescription({ className, ...props }, ref) {
    return <div ref={ref} className={cn('text-sm leading-5 text-[var(--muted-foreground)]', className)} {...props} />
  },
)
