import { act, renderHook, waitFor } from '@testing-library/react'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import type { ProtoClientSession } from '../core/protoClientSession'
import { useFileManager } from './useFileManager'

const mocks = vi.hoisted(() => ({
  listDir: vi.fn(),
  stat: vi.fn(),
  preview: vi.fn(),
  stream: vi.fn(),
}))

vi.mock('./fileApi', async (importOriginal) => {
  const actual = await importOriginal<typeof import('./fileApi')>()
  return {
    ...actual,
    createFileApi: () => ({
      listDir: mocks.listDir,
      stat: mocks.stat,
      preview: mocks.preview,
      mkdir: vi.fn(),
      delete: vi.fn(),
      rename: vi.fn(),
      copy: vi.fn(),
      move: vi.fn(),
      batchDelete: vi.fn(),
      downloadOpen: vi.fn(),
    }),
    createFilePreviewSource: () => ({ preview: mocks.preview, stream: mocks.stream }),
  }
})

vi.mock('./pathBookmarks', async (importOriginal) => {
  const actual = await importOriginal<typeof import('./pathBookmarks')>()
  return {
    ...actual,
    createPathBookmarkApi: () => ({
      list: vi.fn(async () => []),
      add: vi.fn(),
      update: vi.fn(),
      remove: vi.fn(),
    }),
  }
})

describe('useFileManager preview support gate', () => {
  beforeEach(() => {
    mocks.listDir.mockReset().mockResolvedValue({ path: '/', parent: '/', entries: [], total: 0 })
    mocks.stat.mockReset()
    mocks.preview.mockReset()
    mocks.stream.mockReset()
  })

  it('rejects a known unsupported extension before requesting preview metadata or file bytes', async () => {
    const session = { stamp: { endpointId: 'machine-1', generation: 1 } } as ProtoClientSession
    const { result } = renderHook(() => useFileManager({ machineId: 'machine-1', session }))
    await waitFor(() => expect(result.current.loading).toBe(false))

    await act(async () => result.current.openPreview('/downloads/archive.zip'))

    expect(result.current.previewPath).toBe('/downloads/archive.zip')
    expect(result.current.previewLoading).toBe(false)
    expect(result.current.previewError?.surface).toBe('modal')
    expect(mocks.preview).not.toHaveBeenCalled()
    expect(mocks.stream).not.toHaveBeenCalled()
  })

  it('opens a symbolic link to its target directory', async () => {
    mocks.stat.mockResolvedValue({ path: '/shared/docs', name: 'docs', type: 'dir', size: 0 })
    mocks.listDir.mockResolvedValueOnce({ path: '/workspace', parent: '/', entries: [], total: 0 })
      .mockResolvedValueOnce({ path: '/shared/docs', parent: '/shared', entries: [], total: 0 })
    const session = { stamp: { endpointId: 'machine-1', generation: 1 } } as ProtoClientSession
    const { result } = renderHook(() => useFileManager({ machineId: 'machine-1', session, initialPath: '/workspace' }))
    await waitFor(() => expect(result.current.loading).toBe(false))

    await act(async () => result.current.openEntry('/workspace/docs', {
      name: 'docs', type: 'symlink', size: 0, linkTarget: '/shared/docs',
    }))

    expect(mocks.stat).toHaveBeenCalledWith('/shared/docs')
    expect(result.current.currentPath).toBe('/shared/docs')
  })

  it('follows chained relative links before previewing the target file', async () => {
    mocks.stat
      .mockResolvedValueOnce({
        path: '/workspace/releases/current', name: 'current', type: 'symlink', size: 0, linkTarget: '../report.txt',
      })
      .mockResolvedValueOnce({
        path: '/workspace/report.txt', name: 'report.txt', type: 'file', size: 6,
      })
    mocks.preview.mockResolvedValue({
      path: '/workspace/report.txt', name: 'report.txt', size: 6, mimeType: 'text/plain', category: 'text', isText: true, content: 'report',
    })
    const session = { stamp: { endpointId: 'machine-1', generation: 1 } } as ProtoClientSession
    const { result } = renderHook(() => useFileManager({ machineId: 'machine-1', session, initialPath: '/workspace' }))
    await waitFor(() => expect(result.current.loading).toBe(false))

    await act(async () => result.current.openEntry('/workspace/latest', {
      name: 'latest', type: 'symlink', size: 0, linkTarget: 'releases/current',
    }))

    expect(mocks.stat).toHaveBeenNthCalledWith(1, '/workspace/releases/current')
    expect(mocks.stat).toHaveBeenNthCalledWith(2, '/workspace/releases/../report.txt')
    expect(mocks.preview).toHaveBeenCalledWith('/workspace/report.txt')
    expect(result.current.previewPath).toBe('/workspace/report.txt')
  })

  it('opens a hard link through its own path without trying to resolve a target', async () => {
    mocks.preview.mockResolvedValue({
      path: '/workspace/hard.txt', name: 'hard.txt', size: 6, mimeType: 'text/plain', category: 'text', isText: true, content: 'shared',
    })
    const session = { stamp: { endpointId: 'machine-1', generation: 1 } } as ProtoClientSession
    const { result } = renderHook(() => useFileManager({ machineId: 'machine-1', session, initialPath: '/workspace' }))
    await waitFor(() => expect(result.current.loading).toBe(false))

    await act(async () => result.current.openEntry('/workspace/hard.txt', {
      name: 'hard.txt', type: 'file', size: 6, hardLink: true, linkCount: 2,
    }))

    expect(mocks.stat).not.toHaveBeenCalled()
    expect(mocks.preview).toHaveBeenCalledWith('/workspace/hard.txt')
  })

  it('stops a cyclic symbolic link without opening a preview', async () => {
    mocks.stat
      .mockResolvedValueOnce({ path: '/workspace/loop-b', name: 'loop-b', type: 'symlink', size: 0, linkTarget: 'loop-a' })
      .mockResolvedValueOnce({ path: '/workspace/loop-a', name: 'loop-a', type: 'symlink', size: 0, linkTarget: 'loop-b' })
    const session = { stamp: { endpointId: 'machine-1', generation: 1 } } as ProtoClientSession
    const { result } = renderHook(() => useFileManager({ machineId: 'machine-1', session, initialPath: '/workspace' }))
    await waitFor(() => expect(result.current.loading).toBe(false))

    await act(async () => result.current.openEntry('/workspace/loop-a', {
      name: 'loop-a', type: 'symlink', size: 0, linkTarget: 'loop-b',
    }))

    expect(mocks.stat).toHaveBeenCalledTimes(2)
    expect(mocks.preview).not.toHaveBeenCalled()
    expect(result.current.error?.message).toBe('This shortcut points back to itself or forms a loop.')
  })
})
