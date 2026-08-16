import { cleanup, fireEvent, render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { TerminalActionToolbar } from './TerminalActionToolbar'

describe('TerminalActionToolbar', () => {
  afterEach(() => {
    cleanup()
  })

  it('keeps resize control clicks inside the toolbar', () => {
    const onAcquireResizeOwner = vi.fn()
    const onOuterPointerDown = vi.fn()
    const onOuterClick = vi.fn()

    render(
      <div onPointerDown={onOuterPointerDown} onClick={onOuterClick}>
        <TerminalActionToolbar
          mode="default"
          hasSelection={false}
          resizeControl={{ canResize: false, reason: 'follower' }}
          onModeChange={vi.fn()}
          onSelectAll={vi.fn()}
          onSelectVisible={vi.fn()}
          onCopy={vi.fn()}
          onPaste={vi.fn()}
          onOpenClipboardHistory={vi.fn()}
          onOpenSnippets={vi.fn()}
          onAcquireResizeOwner={onAcquireResizeOwner}
        />
      </div>,
    )

    const resizeButton = screen.getByRole('button', { name: /acquire resize control/i })
    fireEvent.pointerDown(resizeButton)
    fireEvent.click(resizeButton)

    expect(onAcquireResizeOwner).toHaveBeenCalledTimes(1)
    expect(onOuterPointerDown).not.toHaveBeenCalled()
    expect(onOuterClick).not.toHaveBeenCalled()
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
    screen.getByRole('button', { name: /acquire resize control/i }).focus()
    await user.keyboard('{Enter}')

    expect(onFontSizeChange).toHaveBeenNthCalledWith(1, 13)
    expect(onFontSizeChange).toHaveBeenNthCalledWith(2, 15)
    expect(onRendererChange).toHaveBeenCalledWith('webgl')
    expect(onAcquireResizeOwner).toHaveBeenCalledTimes(1)
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
