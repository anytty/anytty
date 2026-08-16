import { act, cleanup, render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { useState } from 'react'
import { dispatchNativeBack } from '../platform/nativeBack'
import { FileTransferPanel } from './FileTransferPanel'
import type { TransferInfo } from './fileApi'

describe('FileTransferPanel', () => {
  afterEach(() => {
    cleanup()
  })

  it('shows source machine and transfer target details for downloads and uploads', async () => {
    render(
      <FileTransferPanel
        transfers={[
          transfer({
            id: 'download-1',
            machineId: 'machine-a',
            name: 'server.log',
            direction: 'download',
            totalSize: 1024,
            transferredSize: 1024,
            status: 'completed',
            filePath: '/var/log/server.log',
            savedPath: 'Downloads/AnyTTY/server.log',
          }),
          transfer({
            id: 'upload-1',
            machineId: 'machine-b',
            name: 'report.txt',
            direction: 'upload',
            totalSize: 512,
            transferredSize: 120,
            status: 'transferring',
            targetDir: '/srv/reports',
          }),
        ]}
        hasActiveTransfers
        resolveMachineLabel={(machineId) => machineId === 'machine-a' ? 'Build Runner' : machineId === 'machine-b' ? 'Office Mac' : machineId}
        onCancel={vi.fn()}
        onDismiss={vi.fn()}
        open
      />,
    )

    expect(screen.getByRole('dialog', { name: 'Data Transfer Center' })).toBeTruthy()
    expect(screen.getByText('From Build Runner')).toBeTruthy()
    expect(screen.getByText('/var/log/server.log -> Downloads/AnyTTY/server.log')).toBeTruthy()
    expect(screen.getByText('To Office Mac')).toBeTruthy()
    expect(screen.getByText('/srv/reports / report.txt')).toBeTruthy()
    await userEvent.click(screen.getByRole('button', { name: /close data transfer center/i }))
  })

  it('opens a completed native download from its filename or system-open action', async () => {
    const onOpenFile = vi.fn(async () => undefined)
    render(
      <FileTransferPanel
        transfers={[transfer({
          id: 'download-1',
          name: 'report.pdf',
          status: 'completed',
          transferredSize: 12,
          totalSize: 12,
          savedUri: 'content://media/external/downloads/42',
        })]}
        hasActiveTransfers={false}
        onCancel={vi.fn()}
        onDismiss={vi.fn()}
        onOpenFile={onOpenFile}
        open
      />,
    )

    await userEvent.click(screen.getByRole('button', { name: 'report.pdf' }))
    expect(onOpenFile).toHaveBeenCalledWith('download-1')

    await userEvent.click(screen.getByRole('button', { name: 'Open report.pdf' }))
    expect(onOpenFile).toHaveBeenCalledTimes(2)
  })

  it('keeps a completed download visible when the system cannot open it', async () => {
    render(
      <FileTransferPanel
        transfers={[transfer({
          id: 'download-1',
          name: 'unknown.data',
          status: 'completed',
          transferredSize: 12,
          totalSize: 12,
          savedUri: 'file:///Downloads/AnyTTY/unknown.data',
        })]}
        hasActiveTransfers={false}
        onCancel={vi.fn()}
        onDismiss={vi.fn()}
        onOpenFile={vi.fn(async () => { throw new Error('no compatible application') })}
        open
      />,
    )

    await userEvent.click(screen.getByRole('button', { name: 'Open unknown.data' }))
    expect((await screen.findByRole('alert')).textContent).toContain('This file could not be opened.')
    expect(screen.getByText('unknown.data')).toBeTruthy()
  })

  it('keeps local transfer controls available while remote resume actions are disabled', async () => {
    const onResume = vi.fn()
    const onCancel = vi.fn()
    render(
      <FileTransferPanel
        transfers={[transfer({ id: 'paused-1', name: 'paused.bin', status: 'paused' })]}
        hasActiveTransfers={false}
        onCancel={onCancel}
        onDismiss={vi.fn()}
        onResume={onResume}
        remoteActionsDisabled
        open
      />,
    )

    expect((screen.getByRole('button', { name: /resume paused\.bin/i }) as HTMLButtonElement).disabled).toBe(true)
    await userEvent.click(screen.getByRole('button', { name: /cancel paused\.bin/i }))
    expect(onCancel).toHaveBeenCalledWith('paused-1')
    expect(onResume).not.toHaveBeenCalled()
  })

  it('focuses the named transfer dialog and requests one close on Escape', async () => {
    const onOpenChange = vi.fn()
    render(
      <FileTransferPanel
        transfers={[transfer({ status: 'completed', transferredSize: 1 })]}
        hasActiveTransfers={false}
        onCancel={vi.fn()}
        onDismiss={vi.fn()}
        open
        onOpenChange={onOpenChange}
      />,
    )

    const dialog = screen.getByRole('dialog', { name: 'Data Transfer Center' })
    const close = screen.getByRole('button', { name: /close data transfer center/i })
    expect(dialog.getAttribute('aria-modal')).toBe('true')
    expect(document.activeElement).toBe(close)

    await userEvent.keyboard('{Escape}')
    expect(onOpenChange).toHaveBeenCalledTimes(1)
    expect(onOpenChange).toHaveBeenCalledWith(false)
  })

  it('keeps title and summary associations unique across two transfer centers', () => {
    render(
      <>
        <FileTransferPanel
          transfers={[transfer({ id: 'first', name: 'first.txt' })]}
          hasActiveTransfers
          onCancel={vi.fn()}
          onDismiss={vi.fn()}
          open
        />
        <FileTransferPanel
          transfers={[transfer({ id: 'second', name: 'second.txt' })]}
          hasActiveTransfers
          onCancel={vi.fn()}
          onDismiss={vi.fn()}
          open
        />
      </>,
    )

    const dialogs = Array.from(document.querySelectorAll<HTMLElement>('[role="dialog"]'))
    expect(dialogs).toHaveLength(2)
    const ids: string[] = []
    dialogs.forEach((dialog) => {
      const titleId = dialog.getAttribute('aria-labelledby')
      const summaryId = dialog.getAttribute('aria-describedby')
      expect(titleId).toBeTruthy()
      expect(summaryId).toBeTruthy()
      ids.push(titleId!, summaryId!)

      const title = document.getElementById(titleId!)
      const summary = document.getElementById(summaryId!)
      expect(dialog.contains(title)).toBe(true)
      expect(dialog.contains(summary)).toBe(true)
      expect(title?.textContent).toBe('Data Transfer Center')
      expect(summary?.textContent).not.toBe('')
    })
    expect(new Set(ids).size).toBe(ids.length)
  })

  it('sorts newest transfer tasks first by added time', () => {
    render(
      <FileTransferPanel
        transfers={[
          transfer({ id: 'old', name: 'old.log', startedAt: 1000, status: 'completed', transferredSize: 1 }),
          transfer({ id: 'new', name: 'new.log', startedAt: 3000, status: 'completed', transferredSize: 1 }),
          transfer({ id: 'middle', name: 'middle.log', startedAt: 2000, status: 'completed', transferredSize: 1 }),
        ]}
        hasActiveTransfers={false}
        onCancel={vi.fn()}
        onDismiss={vi.fn()}
        open
      />,
    )

    const items = screen.getAllByRole('listitem')
    expect(items).toHaveLength(3)
    const [first, second, third] = items
    expect(first?.textContent).toContain('new.log')
    expect(second?.textContent).toContain('middle.log')
    expect(third?.textContent).toContain('old.log')
  })

  it('runs bulk controls for selected, completed, and failed transfer tasks', async () => {
    const onPause = vi.fn()
    const onResume = vi.fn()
    const onDismiss = vi.fn()

    render(
      <FileTransferPanel
        transfers={[
          transfer({ id: 'active-1', name: 'active.bin', status: 'transferring', startedAt: 5000, transferredSize: 4, totalSize: 10 }),
          transfer({ id: 'paused-1', name: 'paused.bin', status: 'paused', startedAt: 4000, transferredSize: 4, totalSize: 10 }),
          transfer({ id: 'failed-1', name: 'failed.bin', status: 'failed', startedAt: 3000, error: 'network lost' }),
          transfer({ id: 'missing-1', name: 'missing.bin', status: 'missing', startedAt: 2000 }),
          transfer({ id: 'done-1', name: 'done.bin', status: 'completed', startedAt: 1000, transferredSize: 1 }),
        ]}
        hasActiveTransfers
        onCancel={vi.fn()}
        onDismiss={onDismiss}
        onPause={onPause}
        onResume={onResume}
        open
      />,
    )

    expect(screen.queryByRole('button', { name: /select all transfers/i })).toBeNull()

    await userEvent.click(screen.getByRole('button', { name: /select transfers/i }))
    await userEvent.click(screen.getByRole('button', { name: /select all transfers/i }))
    await userEvent.click(screen.getByRole('button', { name: /pause selected transfers/i }))
    await userEvent.click(screen.getByRole('button', { name: /start selected transfers/i }))
    await userEvent.click(screen.getByRole('button', { name: /^cancel$/i }))
    await userEvent.click(screen.getByRole('button', { name: /delete all completed transfers/i }))
    await userEvent.click(screen.getByRole('button', { name: /delete all failed transfers/i }))

    expect(onPause).toHaveBeenCalledTimes(1)
    expect(onPause).toHaveBeenCalledWith('active-1')
    expect(onResume).toHaveBeenCalledTimes(3)
    expect(onResume).toHaveBeenCalledWith('paused-1')
    expect(onResume).toHaveBeenCalledWith('failed-1')
    expect(onResume).toHaveBeenCalledWith('missing-1')
    expect(onDismiss).toHaveBeenCalledTimes(3)
    expect(onDismiss).toHaveBeenCalledWith('done-1')
    expect(onDismiss).toHaveBeenCalledWith('failed-1')
    expect(onDismiss).toHaveBeenCalledWith('missing-1')
  })

  it('puts both transfer centers in one LIFO Back stack and closes one surface per event', () => {
    const firstClosed = vi.fn()
    const secondClosed = vi.fn()
    const view = render(<TwoTransferCenters firstClosed={firstClosed} secondClosed={secondClosed} />)
    expect(document.querySelectorAll('[role="dialog"]')).toHaveLength(2)

    view.rerender(<TwoTransferCenters firstClosed={firstClosed} secondClosed={secondClosed} />)
    act(() => { expect(dispatchNativeBack()).toBe(true) })
    expect(secondClosed).toHaveBeenCalledOnce()
    expect(firstClosed).not.toHaveBeenCalled()
    expect(document.querySelectorAll('[role="dialog"]')).toHaveLength(1)
    expect(screen.getByText('first.txt')).toBeTruthy()

    act(() => { expect(dispatchNativeBack()).toBe(true) })
    expect(firstClosed).toHaveBeenCalledOnce()
    expect(document.querySelectorAll('[role="dialog"]')).toHaveLength(0)
    expect(dispatchNativeBack()).toBe(false)
  })

  it('exits selection and clears selected transfers before closing the center', async () => {
    const onClose = vi.fn()
    render(<SelectionTransferCenter onClose={onClose} />)

    await userEvent.click(screen.getByRole('button', { name: /select transfers/i }))
    await userEvent.click(screen.getByRole('button', { name: /select selected\.txt/i }))
    expect(screen.getByText('1 selected')).toBeTruthy()

    act(() => { expect(dispatchNativeBack()).toBe(true) })
    expect(onClose).not.toHaveBeenCalled()
    expect(screen.queryByText('1 selected')).toBeNull()
    expect(screen.getByRole('dialog', { name: 'Data Transfer Center' })).toBeTruthy()
    await userEvent.click(screen.getByRole('button', { name: /select transfers/i }))
    expect(screen.getByText('0 selected')).toBeTruthy()
    await userEvent.click(screen.getByRole('button', { name: /^cancel$/i }))

    act(() => { expect(dispatchNativeBack()).toBe(true) })
    expect(onClose).toHaveBeenCalledOnce()
    expect(screen.queryByRole('dialog', { name: 'Data Transfer Center' })).toBeNull()
  })
})

function SelectionTransferCenter({ onClose }: { onClose: () => void }) {
  const [open, setOpen] = useState(true)
  return (
    <FileTransferPanel
      transfers={[transfer({ id: 'selected', name: 'selected.txt' })]}
      hasActiveTransfers
      onCancel={vi.fn()}
      onDismiss={vi.fn()}
      open={open}
      onOpenChange={(nextOpen) => {
        setOpen(nextOpen)
        if (!nextOpen) onClose()
      }}
    />
  )
}

function TwoTransferCenters({
  firstClosed,
  secondClosed,
}: {
  firstClosed: () => void
  secondClosed: () => void
}) {
  const [firstOpen, setFirstOpen] = useState(true)
  const [secondOpen, setSecondOpen] = useState(true)
  return (
    <>
      <FileTransferPanel
        transfers={[transfer({ id: 'first', name: 'first.txt' })]}
        hasActiveTransfers
        onCancel={vi.fn()}
        onDismiss={vi.fn()}
        open={firstOpen}
        onOpenChange={(open) => {
          setFirstOpen(open)
          if (!open) firstClosed()
        }}
      />
      <FileTransferPanel
        transfers={[transfer({ id: 'second', name: 'second.txt' })]}
        hasActiveTransfers
        onCancel={vi.fn()}
        onDismiss={vi.fn()}
        open={secondOpen}
        onOpenChange={(open) => {
          setSecondOpen(open)
          if (!open) secondClosed()
        }}
      />
    </>
  )
}

function transfer(overrides: Partial<TransferInfo>): TransferInfo {
  return {
    id: overrides.id ?? 'transfer-1',
    name: overrides.name ?? 'file.txt',
    direction: overrides.direction ?? 'download',
    totalSize: overrides.totalSize ?? 1,
    transferredSize: overrides.transferredSize ?? 0,
    status: overrides.status ?? 'pending',
    startedAt: overrides.startedAt ?? Date.now(),
    ...(overrides.machineId !== undefined ? { machineId: overrides.machineId } : {}),
    ...(overrides.updatedAt !== undefined ? { updatedAt: overrides.updatedAt } : { updatedAt: Date.now() }),
    ...(overrides.bytesPerSecond !== undefined ? { bytesPerSecond: overrides.bytesPerSecond } : {}),
    ...(overrides.error !== undefined ? { error: overrides.error } : {}),
    ...(overrides.filePath !== undefined ? { filePath: overrides.filePath } : {}),
    ...(overrides.localUri !== undefined ? { localUri: overrides.localUri } : {}),
    ...(overrides.targetDir !== undefined ? { targetDir: overrides.targetDir } : {}),
    ...(overrides.savedPath !== undefined ? { savedPath: overrides.savedPath } : {}),
    ...(overrides.savedUri !== undefined ? { savedUri: overrides.savedUri } : {}),
  }
}
