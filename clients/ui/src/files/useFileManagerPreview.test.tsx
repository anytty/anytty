import { act, renderHook, waitFor } from '@testing-library/react'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import type { ProtoClientSession } from '../core/protoClientSession'
import { useFileManager } from './useFileManager'

const mocks = vi.hoisted(() => ({
  listDir: vi.fn(),
  preview: vi.fn(),
  stream: vi.fn(),
}))

vi.mock('./fileApi', async (importOriginal) => {
  const actual = await importOriginal<typeof import('./fileApi')>()
  return {
    ...actual,
    createFileApi: () => ({
      listDir: mocks.listDir,
      stat: vi.fn(),
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
})
