import { cleanup, fireEvent, render, screen, waitFor } from '@testing-library/react'
import { afterAll, afterEach, beforeAll, describe, expect, it, vi } from 'vitest'
import { FileViewerPreview } from './FileViewerPreview'

describe('FileViewerPreview renderer integration', () => {
  const originalArrayBuffer = File.prototype.arrayBuffer
  const originalText = File.prototype.text
  const originalResizeObserver = globalThis.ResizeObserver

  beforeAll(() => {
    File.prototype.arrayBuffer = function arrayBuffer() {
      return readFile(this, 'arrayBuffer')
    }
    File.prototype.text = function text() {
      return readFile(this, 'text')
    }
    globalThis.ResizeObserver = class ResizeObserver {
      observe() {}
      unobserve() {}
      disconnect() {}
    }
  })

  afterAll(() => {
    File.prototype.arrayBuffer = originalArrayBuffer
    File.prototype.text = originalText
    globalThis.ResizeObserver = originalResizeObserver
  })

  afterEach(cleanup)

  it('renders a Markdown table with the real File Viewer renderer', async () => {
    const markdown = '| Name | State |\n| --- | --- |\n| TUI | Ready |'
    const rendered = render(
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

    await waitFor(() => {
      const host = viewerHost(rendered.container)
      const content = host?.shadowRoot?.textContent ?? host?.textContent ?? ''
      expect(content).toContain('TUI')
      expect(content).toContain('Ready')
      expect(content).not.toContain('不支持')
      expect(content).not.toContain('Unsupported')
      expect(host?.shadowRoot?.querySelector('.file-viewer-web-theme-button')).toBeNull()
      expect(Array.from(host?.shadowRoot?.querySelectorAll('button') ?? []).some((button) => /download/i.test(button.ariaLabel))).toBe(false)
      expect(Array.from(host?.shadowRoot?.querySelectorAll('button') ?? []).some((button) => /print/i.test(button.ariaLabel))).toBe(false)
      expect(Array.from(host?.shadowRoot?.querySelectorAll('button') ?? []).some((button) => /html/i.test(button.ariaLabel))).toBe(false)
      expect(host?.shadowRoot?.querySelector('style[data-anytty-file-viewer]')?.textContent).toContain('.fv-video-shell')
      expect(host?.shadowRoot?.querySelector('style[data-anytty-file-viewer]')?.textContent).toContain('.code-area')
    }, { timeout: 8_000 })

    const searchInput = screen.getByRole('searchbox', { name: 'Search document' })
    searchInput.focus()
    fireEvent.change(searchInput, { target: { value: 'Ready' } })
    fireEvent.submit(screen.getByRole('search'))
    await waitFor(() => expect(screen.getByText('1/1')).toBeTruthy())
    expect(document.activeElement).toBe(searchInput)

    fireEvent.click(screen.getByRole('button', { name: 'Next match' }))
    await waitFor(() => expect(document.activeElement).toBe(searchInput))

  }, 10_000)

  it('removes the duplicate text renderer heading and line summary', async () => {
    const rendered = render(
      <FileViewerPreview
        preview={{
          path: '/logs/app.txt',
          name: 'app.txt',
          size: 11,
          mimeType: 'text/plain',
          category: 'text',
          isText: true,
          content: 'hello\nworld',
        }}
        streamPreview={vi.fn()}
      />,
    )

    await waitFor(() => {
      const shadowRoot = viewerHost(rendered.container)?.shadowRoot
      expect(shadowRoot?.querySelector('.code-viewer')).toBeTruthy()
      expect(shadowRoot?.querySelector('.code-toolbar')).toBeNull()
    }, { timeout: 8_000 })
  }, 10_000)

  it('downloads and renders an image through the real core renderer', async () => {
    const png = new Uint8Array([
      137, 80, 78, 71, 13, 10, 26, 10, 0, 0, 0, 13, 73, 72, 68, 82,
      0, 0, 0, 1, 0, 0, 0, 1, 8, 4, 0, 0, 0, 181, 28, 12, 2,
      0, 0, 0, 11, 73, 68, 65, 84, 120, 218, 99, 252, 255, 31, 0, 3,
      3, 2, 0, 239, 191, 105, 127, 0, 0, 0, 0, 73, 69, 78, 68, 174,
      66, 96, 130,
    ])
    const streamPreview = vi.fn(async () => ({ blob: new Blob([png], { type: 'image/png' }), size: png.byteLength }))
    const rendered = render(
      <FileViewerPreview
        preview={{
          path: '/images/pixel.png',
          name: 'pixel.png',
          size: png.byteLength,
          mimeType: 'image/png',
          category: 'image',
          isText: false,
        }}
        streamPreview={streamPreview}
      />,
    )

    await waitFor(() => {
      const host = viewerHost(rendered.container)
      expect(host?.shadowRoot?.querySelector('.image-viewer img')).toBeTruthy()
      expect(host?.shadowRoot?.textContent).not.toContain('cannot be previewed online')
    }, { timeout: 8_000 })
    expect(screen.queryByRole('search')).toBeNull()
    expect(streamPreview).toHaveBeenCalledOnce()
  }, 10_000)

  it('hides package installation guidance for a missing ZIP renderer', async () => {
    const streamPreview = vi.fn(async () => ({ blob: new Blob(['zip'], { type: 'application/zip' }), size: 3 }))
    const rendered = render(
      <FileViewerPreview
        preview={{
          path: '/archives/example.zip',
          name: 'example.zip',
          size: 3,
          mimeType: 'application/zip',
          category: 'unsupported',
          isText: false,
        }}
        streamPreview={streamPreview}
      />,
    )

    await waitFor(() => {
      const content = viewerHost(rendered.container)?.shadowRoot?.querySelector('.file-viewer-missing-renderer')?.textContent ?? ''
      expect(content).toContain('This file type is not available for in-app preview.')
      expect(content).not.toMatch(/install|@file-viewer|renderer|preset/i)
    }, { timeout: 8_000 })
    expect(screen.queryByRole('search')).toBeNull()
  }, 10_000)
})

function viewerHost(container: HTMLElement): HTMLElement | null {
  return container.firstElementChild?.lastElementChild as HTMLElement | null
}

function readFile(file: Blob, mode: 'arrayBuffer'): Promise<ArrayBuffer>
function readFile(file: Blob, mode: 'text'): Promise<string>
function readFile(file: Blob, mode: 'arrayBuffer' | 'text'): Promise<ArrayBuffer | string> {
  return new Promise((resolve, reject) => {
    const reader = new FileReader()
    reader.onerror = () => reject(reader.error)
    reader.onload = () => resolve(reader.result as ArrayBuffer | string)
    if (mode === 'arrayBuffer') reader.readAsArrayBuffer(file)
    else reader.readAsText(file)
  })
}
