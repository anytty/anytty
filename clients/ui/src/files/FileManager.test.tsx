import { act, cleanup, fireEvent, render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { useState } from 'react'
import { addNativeBackHandler, dispatchNativeBack, NATIVE_BACK_PRIORITY } from '../platform/nativeBack'
import type { ProtoClientSession } from '../core/protoClientSession'
import { FileTransferPanel } from './FileTransferPanel'
import type { FileTransferContext } from './fileApi'
import type { UseFileManagerResult } from './useFileManager'
import { FileManager } from './FileManager'

const useFileManagerMock = vi.hoisted(() => vi.fn())

vi.mock('./useFileManager', async (importOriginal) => {
  const actual = await importOriginal<typeof import('./useFileManager')>()
  return { ...actual, useFileManager: useFileManagerMock }
})

describe('FileManager overlays', () => {
  beforeEach(() => {
    useFileManagerMock.mockReset()
  })

  afterEach(cleanup)

  it('keeps the full-width path above bottom path actions', () => {
    useFileManagerMock.mockReturnValue(createManager({ currentPath: '/Users/ada' }))
    renderFileManager()

    const toolbar = screen.getByTestId('anytty-file-path-toolbar')
    const pathBar = within(toolbar).getByTestId('anytty-file-pathbar')
    const actions = screen.getByTestId('anytty-file-toolbar')
    const bookmarks = within(actions).getByRole('button', { name: 'Path bookmarks' })
    const copyPath = within(actions).getByRole('button', { name: 'Copy current directory path' })
    expect(toolbar.className).toContain('bg-zinc-50')
    expect(pathBar.className).toContain('bg-transparent')
    expect(toolbar.contains(bookmarks)).toBe(false)
    expect(toolbar.contains(copyPath)).toBe(false)
    expect(screen.queryByTestId('anytty-file-path-actions')).toBeNull()
  })

  it('makes the complete file manager inert while its device connection is blocked', () => {
    useFileManagerMock.mockReturnValue(createManager({ currentPath: '/Users/ada' }))
    render(
      <FileManager
        active={false}
        machineId="machine-1"
        session={{} as ProtoClientSession}
      />,
    )

    const manager = screen.getByTestId('anytty-file-manager')
    expect(manager.inert).toBe(true)
    expect(manager.getAttribute('aria-disabled')).toBe('true')
  })

  it('keeps new-directory confirm and cancel controls wide enough for touch', async () => {
    useFileManagerMock.mockImplementation(() => {
      const [newDirName, setNewDirName] = useState('release-assets')
      return createManager({ newDirName, setNewDirName })
    })
    renderFileManager()

    await userEvent.click(screen.getByRole('button', { name: 'New directory' }))
    const create = screen.getByRole('button', { name: 'Create directory' })
    const cancel = screen.getByRole('button', { name: 'Cancel new directory' })
    for (const button of [create, cancel]) {
      expect(button.className).toContain('h-11')
      expect(button.className).toContain('w-12')
      expect(button.className).toContain('shrink-0')
    }
  })

  it('opens a linked entry through the manager target resolver', async () => {
    const openEntry = vi.fn(async () => undefined)
    const linkedEntry = {
      name: 'docs',
      type: 'symlink',
      size: 0,
      linkTarget: '../shared/docs',
    }
    useFileManagerMock.mockReturnValue(createManager({
      currentPath: '/workspace',
      entries: [linkedEntry],
      visibleEntries: [linkedEntry],
      total: 1,
      openEntry,
    }))
    renderFileManager()

    await userEvent.click(screen.getByRole('button', { name: 'Open docs' }))

    expect(openEntry).toHaveBeenCalledWith('/workspace/docs', linkedEntry)
  })

  it('places monochrome paste controls in the bottom clipboard toolbar', async () => {
    const paste = vi.fn(async () => undefined)
    const setClipboard = vi.fn()
    useFileManagerMock.mockReturnValue(createManager({
      currentPath: '/Users/ada',
      clipboard: { mode: 'copy', paths: ['/tmp/notes.txt'] },
      paste,
      setClipboard,
    }))
    renderFileManager()

    const actions = screen.getByTestId('anytty-file-clipboard-toolbar')
    const pasteButton = within(actions).getByRole('button', { name: 'Paste' })
    expect(pasteButton.className).toContain('bg-[var(--primary)]')
    expect(pasteButton.className).toContain('text-[var(--primary-foreground)]')
    await userEvent.click(pasteButton)
    expect(paste).toHaveBeenCalledOnce()
    await userEvent.click(within(actions).getByRole('button', { name: 'Cancel' }))
    expect(setClipboard).toHaveBeenCalledWith(null)
  })

  it('keeps the file action sheet open under a named delete confirmation and deletes once', async () => {
    const user = userEvent.setup()
    const deleteEntry = vi.fn(async () => undefined)
    useFileManagerMock.mockReturnValue(createManager({
      currentPath: '/tmp',
      entries: [fileEntry],
      visibleEntries: [fileEntry],
      total: 1,
      deleteEntry,
    }))
    renderFileManager()

    const trigger = screen.getByRole('button', { name: 'More actions for notes.txt' })
    await user.click(trigger)
    const actions = screen.getByRole('dialog', { name: 'notes.txt' })
    await user.click(within(actions).getByRole('button', { name: 'Delete' }))

    const confirmation = screen.getByRole('dialog', { name: 'Delete this entry?' })
    expect(confirmation.parentElement?.parentElement).toBe(document.body)
    expect(confirmation.getAttribute('aria-modal')).toBe('true')
    expectAssociation(confirmation, 'aria-describedby', '/tmp/notes.txt')
    expect(actions.closest('[inert]')).toBeTruthy()
    expect(document.activeElement).toBe(within(confirmation).getByRole('button', { name: 'Cancel' }))

    await user.keyboard('{Escape}')
    expect(screen.queryByRole('dialog', { name: 'Delete this entry?' })).toBeNull()
    expect(screen.getByRole('dialog', { name: 'notes.txt' })).toBeTruthy()
    expect(deleteEntry).not.toHaveBeenCalled()

    await user.click(within(screen.getByRole('dialog', { name: 'notes.txt' })).getByRole('button', { name: 'Delete' }))
    const reopenedConfirmation = screen.getByRole('dialog', { name: 'Delete this entry?' })
    await user.click(within(reopenedConfirmation).getByRole('button', { name: 'Delete' }))

    expect(deleteEntry).toHaveBeenCalledTimes(1)
    expect(deleteEntry).toHaveBeenCalledWith('/tmp/notes.txt')
    expect(screen.queryByRole('dialog')).toBeNull()
    expect(document.activeElement).toBe(trigger)
  })

  it('uses a named bookmark editor and dispatches edit and remove once each', async () => {
    const user = userEvent.setup()
    const updatePathBookmark = vi.fn(async () => undefined)
    const removePathBookmark = vi.fn(async () => undefined)
    useFileManagerMock.mockReturnValue(createManager({
      pathBookmarks: [{
        id: 'bookmark-1',
        path: '/srv/app',
        label: 'Production',
        createdAt: '2026-01-01T00:00:00Z',
        updatedAt: '2026-01-01T00:00:00Z',
        version: 1,
      }],
      updatePathBookmark,
      removePathBookmark,
    }))
    renderFileManager()

    await user.click(screen.getByRole('button', { name: 'Path bookmarks' }))
    await user.click(screen.getByRole('button', { name: 'Edit bookmark Production' }))

    const editor = screen.getByRole('dialog', { name: 'Edit bookmark' })
    const alias = within(editor).getByRole('textbox', { name: 'Alias' })
    expect(editor.parentElement?.parentElement).toBe(document.body)
    expect(editor.getAttribute('aria-modal')).toBe('true')
    expectAssociation(editor, 'aria-describedby', '/srv/app')
    expect(document.activeElement).toBe(alias)
    await user.clear(alias)
    await user.type(alias, 'Production servers')
    await user.click(within(editor).getByRole('button', { name: 'Save' }))

    await waitFor(() => expect(updatePathBookmark).toHaveBeenCalledTimes(1))
    expect(updatePathBookmark).toHaveBeenCalledWith('bookmark-1', { label: 'Production servers' })
    await waitFor(() => expect(screen.queryByRole('dialog', { name: 'Edit bookmark' })).toBeNull())

    await user.click(screen.getByRole('button', { name: 'Edit bookmark Production' }))
    const reopenedEditor = screen.getByRole('dialog', { name: 'Edit bookmark' })
    await user.click(within(reopenedEditor).getByRole('button', { name: 'Remove bookmark' }))

    expect(removePathBookmark).toHaveBeenCalledTimes(1)
    expect(removePathBookmark).toHaveBeenCalledWith('bookmark-1')
    expect(screen.queryByRole('dialog', { name: 'Edit bookmark' })).toBeNull()
  })

  it('closes the bookmark editor before its bookmarks sheet', async () => {
    useFileManagerMock.mockReturnValue(createManager({
      pathBookmarks: [{
        id: 'bookmark-1',
        path: '/srv/app',
        label: 'Production',
        createdAt: '2026-01-01T00:00:00Z',
        updatedAt: '2026-01-01T00:00:00Z',
        version: 1,
      }],
    }))
    renderFileManager()

    await userEvent.click(screen.getByRole('button', { name: 'Path bookmarks' }))
    await userEvent.click(screen.getByRole('button', { name: 'Edit bookmark Production' }))
    expect(screen.getByRole('dialog', { name: 'Edit bookmark' })).toBeTruthy()

    act(() => { expect(dispatchNativeBack()).toBe(true) })
    expect(screen.queryByRole('dialog', { name: 'Edit bookmark' })).toBeNull()
    expect(screen.getByRole('dialog', { name: 'Path bookmarks' })).toBeTruthy()

    act(() => { expect(dispatchNativeBack()).toBe(true) })
    expect(screen.queryByRole('dialog', { name: 'Path bookmarks' })).toBeNull()
  })

  it('closes transfer before navigating /tmp, then keeps file navigation ahead of leaving files', () => {
    const navigate = vi.fn(async () => undefined)
    const leaveFiles = vi.fn()
    useFileManagerMock.mockReturnValue(createManager({ currentPath: '/tmp', navigate }))
    const unregisterLeaveFiles = addNativeBackHandler(leaveFiles, NATIVE_BACK_PRIORITY.WORKSPACE)
    render(<FileManagerWithTransfer />)

    act(() => { expect(dispatchNativeBack()).toBe(true) })
    expect(screen.queryByRole('dialog', { name: 'Data Transfer Center' })).toBeNull()
    expect(navigate).not.toHaveBeenCalled()
    expect(leaveFiles).not.toHaveBeenCalled()

    act(() => { expect(dispatchNativeBack()).toBe(true) })
    expect(navigate).toHaveBeenCalledOnce()
    expect(navigate).toHaveBeenCalledWith('/')
    expect(leaveFiles).not.toHaveBeenCalled()
    unregisterLeaveFiles()
  })

  it('closes an upload transfer center before discarding a new-directory draft', async () => {
    useFileManagerMock.mockImplementation(() => {
      const [newDirName, setNewDirName] = useState('')
      return createManager({ newDirName, setNewDirName })
    })
    const pickAndUpload = vi.fn()
    const fileTransfer = createFileTransferContext(pickAndUpload)
    render(<FileManagerTransferHarness fileTransfer={fileTransfer} />)

    await userEvent.click(screen.getByRole('button', { name: 'New directory' }))
    const draft = screen.getByRole('textbox', { name: 'Directory name' }) as HTMLInputElement
    await userEvent.type(draft, 'release-assets')
    await userEvent.click(screen.getByRole('button', { name: 'Upload files' }))
    expect(pickAndUpload).toHaveBeenCalledWith('machine-1', '/')
    expect(screen.getByRole('dialog', { name: 'Data Transfer Center' })).toBeTruthy()

    act(() => { expect(dispatchNativeBack()).toBe(true) })
    expect(screen.queryByRole('dialog', { name: 'Data Transfer Center' })).toBeNull()
    expect(screen.getByRole('textbox', { name: 'Directory name' })).toHaveProperty('value', 'release-assets')

    act(() => { expect(dispatchNativeBack()).toBe(true) })
    expect(screen.queryByRole('textbox', { name: 'Directory name' })).toBeNull()
  })

  it('closes an upload transfer center before discarding an inline rename draft', async () => {
    useFileManagerMock.mockReturnValue(createManager({
      entries: [fileEntry],
      visibleEntries: [fileEntry],
      total: 1,
    }))
    const pickAndUpload = vi.fn()
    const fileTransfer = createFileTransferContext(pickAndUpload)
    render(<FileManagerTransferHarness fileTransfer={fileTransfer} />)

    await userEvent.click(screen.getByRole('button', { name: 'More actions for notes.txt' }))
    await userEvent.click(within(screen.getByRole('dialog', { name: 'notes.txt' })).getByRole('button', { name: 'Rename' }))
    const renameDraft = screen.getByRole('textbox', { name: 'Rename entry' }) as HTMLInputElement
    await userEvent.clear(renameDraft)
    await userEvent.type(renameDraft, 'release-notes.txt')
    await userEvent.click(screen.getByRole('button', { name: 'Upload files' }))
    expect(pickAndUpload).toHaveBeenCalledWith('machine-1', '/')
    expect(screen.getByRole('dialog', { name: 'Data Transfer Center' })).toBeTruthy()

    act(() => { expect(dispatchNativeBack()).toBe(true) })
    expect(screen.queryByRole('dialog', { name: 'Data Transfer Center' })).toBeNull()
    expect(screen.getByRole('textbox', { name: 'Rename entry' })).toHaveProperty('value', 'release-notes.txt')

    act(() => { expect(dispatchNativeBack()).toBe(true) })
    expect(screen.queryByRole('textbox', { name: 'Rename entry' })).toBeNull()
  })

  it('captures one transfer intent before awaiting the native download offset', async () => {
    useFileManagerMock.mockReturnValue(createManager({
      entries: [fileEntry],
      visibleEntries: [fileEntry],
      total: 1,
    }))
    const offset = deferred<number>()
    const intent = {}
    const order: string[] = []
    const fileTransfer = createFileTransferContext(vi.fn())
    fileTransfer.beginTransferIntent = vi.fn(() => {
      order.push('begin')
      return intent
    })
    fileTransfer.getDownloadResumeOffset = vi.fn(async () => {
      order.push('offset')
      return await offset.promise
    })
    fileTransfer.startDownload = vi.fn(async () => undefined)
    render(<FileManagerTransferHarness fileTransfer={fileTransfer} />)

    await userEvent.click(screen.getByRole('button', { name: 'More actions for notes.txt' }))
    await userEvent.click(within(screen.getByRole('dialog', { name: 'notes.txt' })).getByRole('button', { name: 'Download' }))

    expect(order).toEqual(['begin', 'offset'])
    expect(fileTransfer.startDownload).not.toHaveBeenCalled()
    offset.resolve(12)
    await waitFor(() => expect(fileTransfer.startDownload).toHaveBeenCalledWith(
      'machine-1',
      'notes.txt',
      42,
      '/notes.txt',
      12,
      intent,
    ))
    expect(fileTransfer.beginTransferIntent).toHaveBeenCalledOnce()
  })

  it('discards a captured transfer intent when download preflight fails', async () => {
    useFileManagerMock.mockReturnValue(createManager({
      entries: [fileEntry],
      visibleEntries: [fileEntry],
      total: 1,
    }))
    const intent = {}
    const fileTransfer = createFileTransferContext(vi.fn())
    fileTransfer.beginTransferIntent = vi.fn(() => intent)
    fileTransfer.discardTransferIntent = vi.fn()
    fileTransfer.getDownloadResumeOffset = vi.fn(async () => { throw new Error('offset unavailable') })
    fileTransfer.startDownload = vi.fn()
    render(<FileManagerTransferHarness fileTransfer={fileTransfer} />)

    await userEvent.click(screen.getByRole('button', { name: 'More actions for notes.txt' }))
    await userEvent.click(within(screen.getByRole('dialog', { name: 'notes.txt' })).getByRole('button', { name: 'Download' }))

    await waitFor(() => expect(fileTransfer.discardTransferIntent).toHaveBeenCalledWith('machine-1', intent))
    expect(fileTransfer.startDownload).not.toHaveBeenCalled()
  })

  it('closes only the latest visible inline state at the shared workspace priority', async () => {
    useFileManagerMock.mockImplementation(() => {
      const [newDirName, setNewDirName] = useState('')
      return createManager({
        entries: [fileEntry],
        visibleEntries: [fileEntry],
        total: 1,
        newDirName,
        setNewDirName,
      })
    })
    renderFileManager()

    await userEvent.click(screen.getByRole('button', { name: 'More actions for notes.txt' }))
    await userEvent.click(within(screen.getByRole('dialog', { name: 'notes.txt' })).getByRole('button', { name: 'Rename' }))
    await userEvent.clear(screen.getByRole('textbox', { name: 'Rename entry' }))
    await userEvent.type(screen.getByRole('textbox', { name: 'Rename entry' }), 'release-notes.txt')
    await userEvent.click(screen.getByRole('button', { name: 'New directory' }))
    await userEvent.type(screen.getByRole('textbox', { name: 'Directory name' }), 'release-assets')

    act(() => { expect(dispatchNativeBack()).toBe(true) })
    expect(screen.queryByRole('textbox', { name: 'Directory name' })).toBeNull()
    expect(screen.getByRole('textbox', { name: 'Rename entry' })).toHaveProperty('value', 'release-notes.txt')

    act(() => { expect(dispatchNativeBack()).toBe(true) })
    expect(screen.queryByRole('textbox', { name: 'Rename entry' })).toBeNull()
  })

  it('keeps file selection under a transfer opened from the reachable active-transfer summary', async () => {
    useFileManagerMock.mockImplementation(() => {
      const [selectionMode, setSelectionMode] = useState(false)
      return createManager({
        entries: [fileEntry],
        visibleEntries: [fileEntry],
        total: 1,
        selectionMode,
        setSelectionMode,
      })
    })
    const fileTransfer = createFileTransferContext(vi.fn())
    render(<FileManagerTransferHarness fileTransfer={fileTransfer} />)

    await userEvent.click(screen.getByRole('button', { name: 'Select files' }))
    expect(screen.getByRole('button', { name: 'Cancel' })).toBeTruthy()
    expect(screen.queryByRole('button', { name: 'Upload files' })).toBeNull()
    expect(screen.queryByRole('button', { name: 'More actions for notes.txt' })).toBeNull()
    await userEvent.click(screen.getByRole('button', { name: 'Open transfer center, 1 active' }))

    act(() => { expect(dispatchNativeBack()).toBe(true) })
    expect(screen.queryByRole('dialog', { name: 'Data Transfer Center' })).toBeNull()
    expect(screen.getByRole('button', { name: 'Cancel' })).toBeTruthy()

    act(() => { expect(dispatchNativeBack()).toBe(true) })
    expect(screen.queryByRole('button', { name: 'Cancel' })).toBeNull()
    expect(screen.getByRole('button', { name: 'Upload files' })).toBeTruthy()
  })

  it('keeps clipboard mode under a transfer opened from the reachable active-transfer summary', async () => {
    useFileManagerMock.mockImplementation(() => {
      const [clipboard, setClipboard] = useState<UseFileManagerResult['clipboard']>(null)
      return createManager({
        entries: [fileEntry],
        visibleEntries: [fileEntry],
        total: 1,
        clipboard,
        setClipboard,
        copy: (paths) => setClipboard({ mode: 'copy', paths }),
      })
    })
    const fileTransfer = createFileTransferContext(vi.fn())
    render(<FileManagerTransferHarness fileTransfer={fileTransfer} />)

    await userEvent.click(screen.getByRole('button', { name: 'More actions for notes.txt' }))
    await userEvent.click(within(screen.getByRole('dialog', { name: 'notes.txt' })).getByRole('button', { name: 'Copy' }))
    expect(screen.getByRole('button', { name: 'Paste' })).toBeTruthy()
    expect(screen.queryByRole('button', { name: 'Upload files' })).toBeNull()
    await userEvent.click(screen.getByRole('button', { name: 'Open transfer center, 1 active' }))

    act(() => { expect(dispatchNativeBack()).toBe(true) })
    expect(screen.queryByRole('dialog', { name: 'Data Transfer Center' })).toBeNull()
    expect(screen.getByRole('button', { name: 'Paste' })).toBeTruthy()

    act(() => { expect(dispatchNativeBack()).toBe(true) })
    expect(screen.queryByRole('button', { name: 'Paste' })).toBeNull()
    expect(screen.getByRole('button', { name: 'Upload files' })).toBeTruthy()
  })

  it('keeps delete confirmation associations unique across two file managers', () => {
    useFileManagerMock.mockReturnValue(createManager({
      currentPath: '/tmp',
      entries: [fileEntry],
      visibleEntries: [fileEntry],
      total: 1,
    }))
    renderTwoFileManagers()

    const triggers = screen.getAllByRole('button', { name: 'More actions for notes.txt' })
    fireEvent.click(triggers[0]!)
    fireEvent.click(triggers[1]!)
    const actionSheets = Array.from(document.querySelectorAll<HTMLElement>('[data-testid="action-sheet-backdrop"] [role="dialog"]'))
    expect(actionSheets).toHaveLength(2)
    actionSheets.forEach((sheet) => {
      const deleteButton = Array.from(sheet.querySelectorAll<HTMLButtonElement>('button'))
        .find((button) => button.textContent?.trim() === 'Delete')
      expect(deleteButton).toBeTruthy()
      fireEvent.click(deleteButton!)
    })

    const confirmations = Array.from(document.querySelectorAll<HTMLElement>('[data-testid="anytty-file-delete-confirm"] [role="dialog"]'))
    expect(confirmations).toHaveLength(2)
    expectUniqueAssociations(confirmations, ['Delete this entry?', 'Delete this entry?'], ['/tmp/notes.txt', '/tmp/notes.txt'])
  })

  it('keeps bookmark editor associations unique across two file managers', () => {
    useFileManagerMock.mockReturnValue(createManager({
      pathBookmarks: [{
        id: 'bookmark-1',
        path: '/srv/app',
        label: 'Production',
        createdAt: '2026-01-01T00:00:00Z',
        updatedAt: '2026-01-01T00:00:00Z',
        version: 1,
      }],
    }))
    renderTwoFileManagers()

    const bookmarkTriggers = screen.getAllByRole('button', { name: 'Path bookmarks' })
    fireEvent.click(bookmarkTriggers[0]!)
    fireEvent.click(bookmarkTriggers[1]!)
    const editButtons = Array.from(document.querySelectorAll<HTMLButtonElement>('button[aria-label="Edit bookmark Production"]'))
    expect(editButtons).toHaveLength(2)
    fireEvent.click(editButtons[0]!)
    fireEvent.click(editButtons[1]!)

    const editors = Array.from(document.querySelectorAll<HTMLElement>('[role="dialog"]'))
    expect(editors).toHaveLength(2)
    expectUniqueAssociations(editors, ['Edit bookmark', 'Edit bookmark'], ['/srv/app', '/srv/app'])
  })
})

function FileManagerWithTransfer() {
  const [transferOpen, setTransferOpen] = useState(true)
  return (
    <>
      <FileManager machineId="machine-1" session={{} as ProtoClientSession} />
      <FileTransferPanel
        transfers={[{
          id: 'transfer-1',
          name: 'payload.bin',
          direction: 'download',
          totalSize: 1,
          transferredSize: 0,
          status: 'pending',
          startedAt: 1,
          updatedAt: 1,
        }]}
        hasActiveTransfers
        onCancel={vi.fn()}
        onDismiss={vi.fn()}
        open={transferOpen}
        onOpenChange={setTransferOpen}
      />
    </>
  )
}

function FileManagerTransferHarness({ fileTransfer }: { fileTransfer: FileTransferContext }) {
  const [transferOpen, setTransferOpen] = useState(false)
  const snapshot = fileTransfer.getSnapshot()
  return (
    <>
      <FileManager
        machineId="machine-1"
        session={{} as ProtoClientSession}
        fileTransfer={fileTransfer}
        onOpenTransferCenter={() => setTransferOpen(true)}
      />
      <FileTransferPanel
        transfers={snapshot.transfers}
        hasActiveTransfers={snapshot.hasActiveTransfers}
        onCancel={fileTransfer.cancelTransfer}
        onDismiss={fileTransfer.dismissTransfer}
        open={transferOpen}
        onOpenChange={setTransferOpen}
      />
    </>
  )
}

function createFileTransferContext(pickAndUpload: FileTransferContext['pickAndUpload']): FileTransferContext {
  return {
    subscribe: () => () => {},
    getSnapshot: () => ({
      transfers: [{
        id: 'upload-1',
        name: 'release-assets.zip',
        direction: 'upload',
        totalSize: 10,
        transferredSize: 0,
        status: 'pending',
        startedAt: 1,
        updatedAt: 1,
      }],
      hasActiveTransfers: true,
    }),
    startDownload: vi.fn(),
    startUpload: vi.fn(),
    pickAndUpload,
    cancelTransfer: vi.fn(),
    dismissTransfer: vi.fn(),
    isNative: true,
  }
}

function expectAssociation(dialog: HTMLElement, attribute: 'aria-labelledby' | 'aria-describedby', text: string) {
  const id = dialog.getAttribute(attribute)
  expect(id).toBeTruthy()
  const target = document.getElementById(id!)
  expect(dialog.contains(target)).toBe(true)
  expect(target?.textContent).toBe(text)
  return id!
}

function expectUniqueAssociations(dialogs: HTMLElement[], titles: string[], descriptions: string[]) {
  const ids = dialogs.flatMap((dialog, index) => [
    expectAssociation(dialog, 'aria-labelledby', titles[index]!),
    expectAssociation(dialog, 'aria-describedby', descriptions[index]!),
  ])
  expect(new Set(ids).size).toBe(ids.length)
}

const fileEntry = {
  name: 'notes.txt',
  type: 'file',
  size: 42,
}

function renderFileManager() {
  return render(
    <FileManager
      machineId="machine-1"
      session={{} as ProtoClientSession}
    />,
  )
}

function renderTwoFileManagers() {
  return render(
    <>
      <FileManager machineId="machine-1" session={{} as ProtoClientSession} />
      <FileManager machineId="machine-2" session={{} as ProtoClientSession} />
    </>,
  )
}

function createManager(overrides: Partial<UseFileManagerResult> = {}): UseFileManagerResult {
  return {
    machineId: 'machine-1',
    currentPath: '/',
    entries: [],
    visibleEntries: [],
    total: 0,
    loading: false,
    error: null,
    sortState: { field: 'name', direction: 'asc' },
    showHidden: false,
    newDirName: '',
    creatingDirectory: false,
    actionMessage: null,
    preview: null,
    previewPath: null,
    previewLoading: false,
    previewError: null,
    fileApi: {} as UseFileManagerResult['fileApi'],
    selectionMode: false,
    selectedPaths: new Set(),
    clipboard: null,
    pathBookmarks: [],
    pathBookmarksLoading: false,
    pathBookmarkError: null,
    setSelectionMode: vi.fn(),
    toggleSelect: vi.fn(),
    selectAll: vi.fn(),
    deselectAll: vi.fn(),
    setClipboard: vi.fn(),
    copy: vi.fn(),
    cut: vi.fn(),
    copyFilePaths: vi.fn(async () => undefined),
    paste: vi.fn(async () => undefined),
    batchDelete: vi.fn(async () => undefined),
    setNewDirName: vi.fn(),
    setSort: vi.fn(),
    toggleShowHidden: vi.fn(),
    openEntry: vi.fn(async () => undefined),
    openPreview: vi.fn(async () => undefined),
    streamPreview: vi.fn(),
    closePreview: vi.fn(),
    createDirectory: vi.fn(async () => undefined),
    deleteEntry: vi.fn(async () => undefined),
    renameEntry: vi.fn(async () => undefined),
    navigate: vi.fn(async () => undefined),
    refresh: vi.fn(async () => undefined),
    addCurrentPathBookmark: vi.fn(async () => undefined),
    updatePathBookmark: vi.fn(async () => undefined),
    removePathBookmark: vi.fn(async () => undefined),
    refreshPathBookmarks: vi.fn(async () => undefined),
    ...overrides,
  }
}

function deferred<T>() {
  let resolve!: (value: T | PromiseLike<T>) => void
  let reject!: (reason?: unknown) => void
  const promise = new Promise<T>((accept, decline) => {
    resolve = accept
    reject = decline
  })
  return { promise, resolve, reject }
}
