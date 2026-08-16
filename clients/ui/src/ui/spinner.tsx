import * as React from 'react'
import { LoaderCircle } from 'lucide-react'
import { cn } from './utils'

export type SpinnerProps = React.ComponentPropsWithoutRef<typeof LoaderCircle>

export const Spinner = React.forwardRef<SVGSVGElement, SpinnerProps>(function Spinner(
  { className, ...props },
  ref,
) {
  return (
    <LoaderCircle
      ref={ref}
      className={cn('h-4 w-4 shrink-0 animate-spin motion-reduce:animate-none', className)}
      {...props}
    />
  )
})
