import { useState } from 'react'
import { Info, MoreHorizontal, Route, Trash2, Unplug } from 'lucide-react'
import { Button } from '../ui/button'
import { Spinner } from '../ui/spinner'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '../ui/dropdown-menu'

export interface MachineActionsMenuProps {
  open: boolean
  labels: {
    trigger: string
    details: string
    connection: string
    disconnect: string
    disconnecting: string
    disconnectFailed: string
    forget: string
    forgetting: string
    forgetFailed: string
  }
  canConfigure: boolean
  canDisconnect: boolean
  canForget: boolean
  onOpenChange: (open: boolean) => void
  onShowDetails: () => void
  onConfigure: () => void
  onDisconnect: () => Promise<boolean>
  onForget: () => Promise<void>
}

export function MachineActionsMenu({
  open,
  labels,
  canConfigure,
  canDisconnect,
  canForget,
  onOpenChange,
  onShowDetails,
  onConfigure,
  onDisconnect,
  onForget,
}: MachineActionsMenuProps) {
  const [disconnecting, setDisconnecting] = useState(false)
  const [disconnectError, setDisconnectError] = useState(false)
  const [forgetting, setForgetting] = useState(false)
  const [forgetError, setForgetError] = useState(false)

  const disconnect = async () => {
    setDisconnecting(true)
    setDisconnectError(false)
    try {
      if (await onDisconnect()) onOpenChange(false)
    } catch {
      setDisconnectError(true)
    } finally {
      setDisconnecting(false)
    }
  }

  const forget = async () => {
    setForgetting(true)
    setForgetError(false)
    try {
      await onForget()
      onOpenChange(false)
    } catch {
      setForgetError(true)
    } finally {
      setForgetting(false)
    }
  }

  return (
    <DropdownMenu open={open} onOpenChange={onOpenChange}>
      <DropdownMenuTrigger asChild>
        <Button aria-label={labels.trigger} className="text-zinc-500 lg:h-10 lg:w-10" size="icon" variant="ghost">
          <MoreHorizontal className="h-4 w-4" />
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end">
        <DropdownMenuItem className="text-xs font-semibold text-zinc-700" onSelect={onShowDetails}>
          <Info />
          {labels.details}
        </DropdownMenuItem>
        {canConfigure ? (
          <DropdownMenuItem className="text-xs font-semibold text-zinc-700" onSelect={onConfigure}>
            <Route />
            {labels.connection}
          </DropdownMenuItem>
        ) : null}
        {canDisconnect || canForget ? <DropdownMenuSeparator /> : null}
        {canDisconnect ? (
          <DropdownMenuItem
            aria-label={labels.disconnect}
            className="text-xs font-semibold text-red-600 focus:bg-red-50 focus:text-red-700"
            disabled={disconnecting || forgetting}
            onSelect={(event) => { event.preventDefault(); void disconnect() }}
          >
            {disconnecting ? <Spinner aria-hidden="true" /> : <Unplug />}
            {disconnecting ? labels.disconnecting : labels.disconnect}
          </DropdownMenuItem>
        ) : null}
        {disconnectError ? <p className="px-2.5 py-2 text-xs font-medium text-red-600" role="alert">{labels.disconnectFailed}</p> : null}
        {canForget ? (
          <DropdownMenuItem
            aria-label={labels.forget}
            className="text-xs font-semibold text-red-600 focus:bg-red-50 focus:text-red-700"
            disabled={forgetting || disconnecting}
            onSelect={(event) => { event.preventDefault(); void forget() }}
          >
            {forgetting ? <Spinner aria-hidden="true" /> : <Trash2 />}
            {forgetting ? labels.forgetting : labels.forget}
          </DropdownMenuItem>
        ) : null}
        {forgetError ? <p className="px-2.5 py-2 text-xs font-medium text-red-600" role="alert">{labels.forgetFailed}</p> : null}
      </DropdownMenuContent>
    </DropdownMenu>
  )
}
