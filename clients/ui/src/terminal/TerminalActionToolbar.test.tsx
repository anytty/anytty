import { cleanup, fireEvent, render, screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { TerminalActionToolbar } from './TerminalActionToolbar'

describe('TerminalActionToolbar', () => {
  afterEach(() => {
    cleanup()
  })

  it('keeps resize control clicks inside the toolbar', () => {
    const onReleaseResizeOwner = vi.fn()
    const onToggleSizeLock = vi.fn()
    const onOuterPointerDown = vi.fn()
    const onOuterClick = vi.fn()

    render(
      <div onPointerDown={onOuterPointerDown} onClick={onOuterClick}>
        <TerminalActionToolbar
          mode="default"
          hasSelection={false}
          resizeControl={{ canResize: true, reason: 'owner' }}
          onModeChange={vi.fn()}
          onSelectAll={vi.fn()}
          onSelectVisible={vi.fn()}
          onCopy={vi.fn()}
          onPaste={vi.fn()}
          onOpenClipboardHistory={vi.fn()}
          onOpenSnippets={vi.fn()}
          onReleaseResizeOwner={onReleaseResizeOwner}
          onToggleSizeLock={onToggleSizeLock}
        />
      </div>,
    )

    const resizeGroup = screen.getByRole('group', { name: 'Resize control' })
    expect(within(resizeGroup).getAllByRole('button')).toHaveLength(2)
    const ownerButton = within(resizeGroup).getByRole('button', { name: /release owner permission/i })
    const sizeLockButton = within(resizeGroup).getByRole('button', { name: /lock terminal size/i })
    fireEvent.pointerDown(ownerButton)
    fireEvent.click(ownerButton)
    fireEvent.pointerDown(sizeLockButton)
    fireEvent.click(sizeLockButton)

    expect(onReleaseResizeOwner).toHaveBeenCalledTimes(1)
    expect(onToggleSizeLock).toHaveBeenCalledTimes(1)
    expect(onOuterPointerDown).not.toHaveBeenCalled()
    expect(onOuterClick).not.toHaveBeenCalled()
  })

  it('requires owner permission before the size can be locked', async () => {
    const onToggleSizeLock = vi.fn()
    render(
      <TerminalActionToolbar
        mode="default"
        hasSelection={false}
        resizeControl={{ canResize: false, reason: 'follower' }}
        onModeChange={vi.fn()}
        onSelectAll={vi.fn()}
        onSelectVisible={vi.fn()}
        onCopy={vi.fn()}
        onPaste={vi.fn()}
        onOpenSnippets={vi.fn()}
        onToggleSizeLock={onToggleSizeLock}
      />,
    )

    const lockButton = screen.getByRole('button', { name: /lock terminal size/i })
    expect(lockButton.hasAttribute('disabled')).toBe(true)
    await userEvent.click(lockButton)
    expect(onToggleSizeLock).not.toHaveBeenCalled()
  })

  it('allows owner transfer while the terminal size remains locked', async () => {
    const onAcquireResizeOwner = vi.fn()
    const onToggleSizeLock = vi.fn()
    render(
      <TerminalActionToolbar
        mode="default"
        hasSelection={false}
        resizeControl={{ canResize: false, reason: 'follower', sizeLocked: true }}
        sizeLocked
        onModeChange={vi.fn()}
        onSelectAll={vi.fn()}
        onSelectVisible={vi.fn()}
        onCopy={vi.fn()}
        onPaste={vi.fn()}
        onOpenSnippets={vi.fn()}
        onAcquireResizeOwner={onAcquireResizeOwner}
        onToggleSizeLock={onToggleSizeLock}
      />,
    )

    const ownerButton = screen.getByRole('button', { name: /acquire owner permission/i })
    const lockButton = screen.getByRole('button', { name: /unlock terminal size/i })
    expect(ownerButton.hasAttribute('disabled')).toBe(false)
    expect(ownerButton.textContent).toContain('Get Owner')
    expect(lockButton.hasAttribute('disabled')).toBe(true)
    await userEvent.click(ownerButton)

    expect(onAcquireResizeOwner).toHaveBeenCalledTimes(1)
    expect(onToggleSizeLock).not.toHaveBeenCalled()
  })

  it('executes settings actions through keyboard activation', async () => {
    const user = userEvent.setup()
    const onFontSizeChange = vi.fn()
    const onRendererChange = vi.fn()
    const onAcquireResizeOwner = vi.fn()

    render(
      <TerminalActionToolbar
        mode="default"
        hasSelection={false}
        fontSize={14}
        renderer="auto"
        resizeControl={{ canResize: false, reason: 'follower' }}
        onModeChange={vi.fn()}
        onSelectAll={vi.fn()}
        onSelectVisible={vi.fn()}
        onCopy={vi.fn()}
        onPaste={vi.fn()}
        onOpenClipboardHistory={vi.fn()}
        onOpenSnippets={vi.fn()}
        onFontSizeChange={onFontSizeChange}
        onRendererChange={onRendererChange}
        onAcquireResizeOwner={onAcquireResizeOwner}
      />,
    )

    screen.getByRole('button', { name: /decrease terminal font size/i }).focus()
    await user.keyboard('{Enter}')
    screen.getByRole('button', { name: /increase terminal font size/i }).focus()
    await user.keyboard('{Enter}')
    screen.getByRole('button', { name: /renderer: auto/i }).focus()
    await user.keyboard('{Enter}')
    screen.getByRole('button', { name: /acquire owner permission/i }).focus()
    await user.keyboard('{Enter}')

    expect(onFontSizeChange).toHaveBeenNthCalledWith(1, 13)
    expect(onFontSizeChange).toHaveBeenNthCalledWith(2, 15)
    expect(onRendererChange).toHaveBeenCalledWith('webgl')
    expect(onAcquireResizeOwner).toHaveBeenCalledTimes(1)
  })

  it('changes the terminal keyboard mode independently from resize ownership', async () => {
    const onKeyboardModeChange = vi.fn()
    const onReleaseResizeOwner = vi.fn()
    render(
      <TerminalActionToolbar
        mode="default"
        hasSelection={false}
        keyboardMode="shift"
        resizeControl={{ canResize: false, reason: 'size_locked', sizeLocked: true }}
        sizeLocked
        onModeChange={vi.fn()}
        onSelectAll={vi.fn()}
        onSelectVisible={vi.fn()}
        onCopy={vi.fn()}
        onPaste={vi.fn()}
        onOpenSnippets={vi.fn()}
        onKeyboardModeChange={onKeyboardModeChange}
        onReleaseResizeOwner={onReleaseResizeOwner}
      />,
    )

    expect(screen.getByRole('button', { name: 'Always Shift' }).getAttribute('aria-pressed')).toBe('true')
    expect(screen.getByRole('button', { name: /release owner permission/i }).getAttribute('aria-pressed')).toBe('true')
    expect(screen.getByRole('button', { name: /release owner permission/i }).hasAttribute('disabled')).toBe(false)
    expect(screen.getByRole('button', { name: /unlock terminal size/i }).getAttribute('aria-pressed')).toBe('true')
    await userEvent.click(screen.getByRole('button', { name: /release owner permission/i }))
    await userEvent.click(screen.getByRole('button', { name: 'Always Resize' }))
    await userEvent.click(screen.getByRole('button', { name: 'Auto' }))
    expect(onReleaseResizeOwner).toHaveBeenCalledTimes(1)
    expect(onKeyboardModeChange.mock.calls).toEqual([['resize'], ['auto']])
  })

  it('keeps connection and split actions in the settings toolbar', async () => {
    const onOpenConnectionInfo = vi.fn()
    const onToggleSyncSplitInput = vi.fn()
    const onCloseSplitTerminal = vi.fn()

    render(
      <TerminalActionToolbar
        mode="default"
        hasSelection={false}
        splitTerminalOpen
        onModeChange={vi.fn()}
        onSelectAll={vi.fn()}
        onSelectVisible={vi.fn()}
        onCopy={vi.fn()}
        onPaste={vi.fn()}
        onOpenClipboardHistory={vi.fn()}
        onOpenSnippets={vi.fn()}
        onOpenConnectionInfo={onOpenConnectionInfo}
        onToggleSyncSplitInput={onToggleSyncSplitInput}
        onCloseSplitTerminal={onCloseSplitTerminal}
      />,
    )

    await userEvent.click(screen.getByRole('button', { name: 'Connection' }))
    await userEvent.click(screen.getByRole('button', { name: 'Sync input' }))
    await userEvent.click(screen.getByRole('button', { name: 'Close split' }))

    expect(onOpenConnectionInfo).toHaveBeenCalledTimes(1)
    expect(onToggleSyncSplitInput).toHaveBeenCalledTimes(1)
    expect(onCloseSplitTerminal).toHaveBeenCalledTimes(1)
  })

  it('splits the active mobile pane in all four directions', async () => {
    const onSplitTerminal = vi.fn()

    render(
      <TerminalActionToolbar
        mode="default"
        hasSelection={false}
        canSplitTerminal
        onModeChange={vi.fn()}
        onSelectAll={vi.fn()}
        onSelectVisible={vi.fn()}
        onCopy={vi.fn()}
        onPaste={vi.fn()}
        onOpenSnippets={vi.fn()}
        onSplitTerminal={onSplitTerminal}
      />,
    )

    const splitGroup = screen.getByRole('group', { name: 'Split terminal' })
    await userEvent.click(within(splitGroup).getByRole('button', { name: 'Split left' }))
    await userEvent.click(within(splitGroup).getByRole('button', { name: 'Split right' }))
    await userEvent.click(within(splitGroup).getByRole('button', { name: 'Split above' }))
    await userEvent.click(within(splitGroup).getByRole('button', { name: 'Split below' }))

    expect(onSplitTerminal.mock.calls).toEqual([
      ['left'],
      ['right'],
      ['top'],
      ['bottom'],
    ])
  })

  it('keeps all default tools reachable in a short terminal viewport', () => {
    render(
      <TerminalActionToolbar
        mode="default"
        hasSelection={false}
        onModeChange={vi.fn()}
        onSelectAll={vi.fn()}
        onSelectVisible={vi.fn()}
        onCopy={vi.fn()}
        onPaste={vi.fn()}
        onOpenClipboardHistory={vi.fn()}
        onOpenSnippets={vi.fn()}
      />,
    )

    const toolbar = screen.getByTestId('anytty-terminal-action-toolbar')
    expect(toolbar.classList.contains('max-h-[calc(100%_-_1rem)]')).toBe(true)
    expect(toolbar.classList.contains('overflow-y-auto')).toBe(true)
  })

  it('opens history search from the default tools surface', async () => {
    const onOpenHistorySearch = vi.fn()
    render(
      <TerminalActionToolbar
        mode="default"
        hasSelection={false}
        onModeChange={vi.fn()}
        onSelectAll={vi.fn()}
        onSelectVisible={vi.fn()}
        onCopy={vi.fn()}
        onPaste={vi.fn()}
        onOpenHistorySearch={onOpenHistorySearch}
        onOpenClipboardHistory={vi.fn()}
        onOpenSnippets={vi.fn()}
      />,
    )

    await userEvent.click(screen.getByRole('button', { name: 'Search history' }))
    expect(onOpenHistorySearch).toHaveBeenCalledOnce()
  })

  it('keeps both toolbar modes visible on wide single-pane devices', () => {
    const props = {
      hasSelection: true,
      wideViewportVisible: true,
      onModeChange: vi.fn(),
      onSelectAll: vi.fn(),
      onSelectVisible: vi.fn(),
      onCopy: vi.fn(),
      onPaste: vi.fn(),
      onOpenClipboardHistory: vi.fn(),
      onOpenSnippets: vi.fn(),
    }
    const view = render(<TerminalActionToolbar {...props} mode="default" />)

    expect(screen.getByTestId('anytty-terminal-action-toolbar').classList.contains('md:hidden')).toBe(false)

    view.rerender(<TerminalActionToolbar {...props} mode="selection" />)
    expect(screen.getByTestId('anytty-terminal-action-toolbar').classList.contains('md:hidden')).toBe(false)
  })

  it('keeps local selection and display settings available while remote tools are disabled', async () => {
    const onModeChange = vi.fn()
    const onFontSizeChange = vi.fn()
    const onPaste = vi.fn()
    render(
      <TerminalActionToolbar
        mode="default"
        hasSelection={false}
        remoteActionsDisabled
        onModeChange={onModeChange}
        onSelectAll={vi.fn()}
        onSelectVisible={vi.fn()}
        onCopy={vi.fn()}
        onPaste={onPaste}
        onOpenClipboardHistory={vi.fn()}
        onOpenSnippets={vi.fn()}
        onFontSizeChange={onFontSizeChange}
      />,
    )

    await userEvent.click(screen.getByRole('button', { name: 'Select' }))
    await userEvent.click(screen.getByRole('button', { name: /increase terminal font size/i }))
    expect(onModeChange).toHaveBeenCalledWith('selection')
    expect(onFontSizeChange).toHaveBeenCalled()
    expect((screen.getByRole('button', { name: /paste/i }) as HTMLButtonElement).disabled).toBe(true)
    expect((screen.getByRole('button', { name: /clipboard/i }) as HTMLButtonElement).disabled).toBe(true)
    expect(onPaste).not.toHaveBeenCalled()
  })
})
