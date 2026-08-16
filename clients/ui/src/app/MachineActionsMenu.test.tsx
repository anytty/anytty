import { cleanup, render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { MachineActionsMenu } from './MachineActionsMenu'

describe('MachineActionsMenu', () => {
  afterEach(cleanup)

  it('keeps a failed authorization removal visible and retryable', async () => {
    const onForget = vi.fn(async () => { throw new Error('registry unavailable') })
    render(
      <MachineActionsMenu
        open
        labels={{
          trigger: 'More actions',
          details: 'Details',
          connection: 'Connection',
          disconnect: 'Disconnect',
          disconnecting: 'Disconnecting...',
          disconnectFailed: 'Could not disconnect.',
          forget: 'Remove authorization',
          forgetting: 'Removing authorization...',
          forgetFailed: 'Could not remove authorization.',
        }}
        canConfigure={false}
        canDisconnect={false}
        canForget
        onOpenChange={vi.fn()}
        onShowDetails={vi.fn()}
        onConfigure={vi.fn()}
        onDisconnect={vi.fn(async () => false)}
        onForget={onForget}
      />,
    )

    await userEvent.click(screen.getByRole('menuitem', { name: 'Remove authorization' }))

    expect(onForget).toHaveBeenCalledOnce()
    expect((await screen.findByRole('alert')).textContent).toContain('Could not remove authorization.')
    expect(screen.getByRole('menuitem', { name: 'Remove authorization' }).getAttribute('aria-disabled')).not.toBe('true')
  })
})
