import * as React from 'react'
import { Slot } from '@radix-ui/react-slot'
import { cva, type VariantProps } from 'class-variance-authority'
import { cn } from './utils'

export const buttonVariants = cva(
  'inline-flex touch-manipulation items-center justify-center gap-2 whitespace-nowrap rounded-md text-sm font-medium transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[var(--ring)] focus-visible:ring-offset-2 disabled:pointer-events-none disabled:opacity-50 [&_svg]:pointer-events-none [&_svg]:shrink-0',
  {
    variants: {
      variant: {
        default: 'border border-[var(--primary)] bg-[var(--primary)] text-[var(--primary-foreground)] shadow-sm hover:opacity-90',
        destructive: 'border border-[var(--destructive)] bg-[var(--destructive)] text-[var(--destructive-foreground)] shadow-sm hover:opacity-90',
        outline: 'border border-[var(--input)] bg-[var(--background)] text-[var(--foreground)] shadow-sm hover:bg-[var(--ui-accent)] hover:text-[var(--ui-accent-foreground)]',
        secondary: 'border border-[var(--secondary)] bg-[var(--secondary)] text-[var(--secondary-foreground)] shadow-sm hover:bg-zinc-200',
        ghost: 'border border-transparent bg-transparent text-[var(--foreground)] hover:bg-[var(--ui-accent)] hover:text-[var(--ui-accent-foreground)]',
        link: 'h-auto border-0 bg-transparent p-0 text-[var(--foreground)] underline-offset-4 hover:underline',
      },
      size: {
        default: 'h-10 px-4 py-2',
        sm: 'h-9 rounded-md px-3 text-xs',
        lg: 'h-11 rounded-md px-6',
        icon: 'h-11 w-11 p-0',
        'icon-sm': 'h-9 w-9 p-0',
      },
    },
    defaultVariants: {
      variant: 'default',
      size: 'default',
    },
  },
)

export type ButtonVariant = NonNullable<VariantProps<typeof buttonVariants>['variant']>
export type ButtonSize = NonNullable<VariantProps<typeof buttonVariants>['size']>

export interface ButtonProps
  extends React.ButtonHTMLAttributes<HTMLButtonElement>,
    VariantProps<typeof buttonVariants> {
  asChild?: boolean
}

export const Button = React.forwardRef<HTMLButtonElement, ButtonProps>(function Button(
  { asChild = false, className, size, type = 'button', variant, ...props },
  ref,
) {
  const Comp = asChild ? Slot : 'button'
  return (
    <Comp
      ref={ref}
      type={asChild ? undefined : type}
      className={cn(buttonVariants({ className, size, variant }))}
      {...props}
    />
  )
})
