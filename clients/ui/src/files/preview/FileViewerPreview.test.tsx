import { act, cleanup, fireEvent, render, screen, waitFor } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { FileViewerPreview } from './FileViewerPreview'

const fileViewerMock = vi.hoisted(() => vi.fn((props: { filename?: string; file?: File; onStateChange?(state: unknown): void }) => (
  <div data-testid="file-viewer" data-filename={props.filename} data-file-name={props.file?.name} data-size={props.file?.size} />
)))
const fileViewerHandleMock = vi.hoisted(() => ({
  zoomIn: vi.fn(async () => null),
  zoomOut: vi.fn(async () => null),
  resetZoom: vi.fn(async () => null),
}))

vi.mock('@file-viewer/react', async () => {
  const React = await import('react')
  return {
    default: React.forwardRef((props: { filename?: string; file?: File; onStateChange?(state: unknown): void }, ref) => {
      React.useImperativeHandle(ref, () => fileViewerHandleMock)
      return fileViewerMock(props)
    }),
  }
})

describe('FileViewerPreview', () => {
  afterEach(() => {
    cleanup()
    fileViewerMock.mockClear()
    fileViewerHandleMock.zoomIn.mockClear()
    fileViewerHandleMock.zoomOut.mockClear()
    fileViewerHandleMock.resetZoom.mockClear()
  })

  it('passes Markdown content directly to File Viewer as a local Blob', async () => {
    const markdown = '| Name | State |\n| --- | --- |\n| TUI | Ready |'
    render(
      <FileViewerPreview
        preview={{
          path: '/docs/status.md',
          name: 'status.md',
          size: markdown.length,
          mimeType: 'text/markdown',
          category: 'text',
          isText: true,
          content: markdown,
        }}
        streamPreview={vi.fn()}
      />,
    )

    await waitFor(() => expect(screen.getByTestId('file-viewer')).toBeTruthy())
    expect(screen.getByTestId('file-viewer').dataset.filename).toBe('status.md')
    expect(screen.getByTestId('file-viewer').dataset.fileName).toBe('status.md')
    expect(fileViewerMock).toHaveBeenCalled()
    const props = fileViewerMock.mock.calls.at(-1)?.[0]
    expect(props.file).toBeInstanceOf(File)
    expect(props.options.rendererMode).toBe('extend')
    expect(props.options.toolbar).toMatchObject({
      download: false,
      exportHtml: false,
      print: false,
      search: false,
      theme: false,
      zoom: false,
    })
    expect(props.options.text).toEqual({ toolbar: false })
  })

  it('downloads a binary preview into a named in-memory File', async () => {
    const streamPreview = vi.fn(async () => ({ blob: new Blob(['image'], { type: 'image/png' }), size: 5 }))
    render(
      <FileViewerPreview
        preview={{
          path: '/images/logo.png',
          name: 'logo.png',
          size: 5,
          mimeType: 'image/png',
          category: 'image',
          isText: false,
        }}
        streamPreview={streamPreview}
      />,
    )

    await waitFor(() => expect(screen.getByTestId('file-viewer').dataset.fileName).toBe('logo.png'))
    expect(streamPreview).toHaveBeenCalledWith('/images/logo.png', 'image/png', expect.objectContaining({ signal: expect.any(AbortSignal) }))
    const props = fileViewerMock.mock.calls.at(-1)?.[0]
    expect(props.file).toBeInstanceOf(File)
    expect(props.file?.type).toBe('image/png')
  })

  it('maps two-finger pinch gestures to the viewer zoom controller', async () => {
    render(
      <FileViewerPreview
        preview={{
          path: '/images/logo.png',
          name: 'logo.png',
          size: 5,
          mimeType: 'image/png',
          category: 'image',
          isText: false,
        }}
        streamPreview={vi.fn(async () => ({ blob: new Blob(['image'], { type: 'image/png' }), size: 5 }))}
      />,
    )

    await waitFor(() => expect(screen.getByTestId('file-viewer')).toBeTruthy())
    const surface = screen.getByTestId('file-viewer').parentElement as HTMLElement
    fireEvent.touchStart(surface, { touches: [{ clientX: 0, clientY: 0 }, { clientX: 100, clientY: 0 }] })
    fireEvent.touchMove(surface, { touches: [{ clientX: 0, clientY: 0 }, { clientX: 105, clientY: 0 }] })
    expect(fileViewerHandleMock.zoomIn).not.toHaveBeenCalled()
    fireEvent.touchMove(surface, { touches: [{ clientX: 0, clientY: 0 }, { clientX: 120, clientY: 0 }] })
    expect(fileViewerHandleMock.zoomIn).toHaveBeenCalledOnce()
    fireEvent.touchMove(surface, { touches: [{ clientX: 0, clientY: 0 }, { clientX: 100, clientY: 0 }] })
    expect(fileViewerHandleMock.zoomOut).toHaveBeenCalledOnce()
  })

  it('places search and zoom controls in one toolbar and uses an unambiguous clear action', async () => {
    render(
      <FileViewerPreview
        preview={{
          path: '/docs/status.txt',
          name: 'status.txt',
          size: 5,
          mimeType: 'text/plain',
          category: 'text',
          isText: true,
          content: 'ready',
        }}
        streamPreview={vi.fn()}
      />,
    )

    await waitFor(() => expect(screen.getByTestId('file-viewer')).toBeTruthy())
    const props = fileViewerMock.mock.calls.at(-1)?.[0]
    act(() => props.onStateChange?.({
      loading: false,
      ready: true,
      error: null,
      lastEvent: null,
      lifecycle: null,
      availability: null,
      search: null,
      zoom: { scale: 1.25, label: '125%', canZoomIn: true, canZoomOut: true, canReset: true },
      location: null,
    }))

    const toolbar = screen.getByRole('search')
    expect(toolbar.contains(screen.getByRole('button', { name: 'Clear search' }))).toBe(true)
    expect(toolbar.contains(screen.getByRole('button', { name: 'Reset zoom' }))).toBe(true)
    fireEvent.click(screen.getByRole('button', { name: 'Reset zoom' }))
    expect(fileViewerHandleMock.resetZoom).toHaveBeenCalledOnce()
  })
})
