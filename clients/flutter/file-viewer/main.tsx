import React, { useRef, useState } from 'react'
import { createRoot } from 'react-dom/client'
import FileViewer, { type FileViewerHandle, type ViewerOptions, type ViewerState } from '@file-viewer/react'
import textRenderer from '@file-viewer/renderer-text'
import spreadsheetRenderer from '@file-viewer/renderer-spreadsheet'
import { Search, Minus, Plus, RotateCcw, ChevronUp, ChevronDown, X } from 'lucide-react'

type Preview = { name: string; mime: string; content: string; dark: boolean; chinese: boolean; html: boolean }
const payload = document.getElementById('preview-data')!.textContent!.trim()
const data: Preview = JSON.parse(new TextDecoder().decode(Uint8Array.from(atob(payload), char => char.charCodeAt(0))))
const bytes = Uint8Array.from(atob(data.content), char => char.charCodeAt(0))
const root = document.getElementById('root')!
document.documentElement.style.colorScheme = data.dark ? 'dark' : 'light'

if (data.html) {
  const frame = document.createElement('iframe')
  frame.title = data.name
  // Remote file content receives no scripts, forms, navigation, or app bridge.
  frame.setAttribute('sandbox', '')
  frame.srcdoc = new TextDecoder().decode(bytes)
  root.append(frame)
} else {
  const file = new File([bytes], data.name, { type: data.mime })
  const options: ViewerOptions = {
      autoRenderers: false,
      rendererMode: 'extend',
      renderers: [textRenderer, spreadsheetRenderer] as unknown as NonNullable<ViewerOptions['renderers']>,
      theme: data.dark ? 'dark' : 'light',
      locale: data.chinese ? 'zh-CN' : 'en-US',
      toolbar: { download: false, exportHtml: false, print: false, theme: false, search: false, zoom: false },
      text: { toolbar: false },
      fit: { mode: 'width', resize: 'until-interaction', padding: 8 },
  }
  createRoot(root).render(<DocumentPreview file={file} options={options} />)
}

function DocumentPreview({ file, options }: { file: File; options: ViewerOptions }) {
  const viewer = useRef<FileViewerHandle | null>(null)
  const [state, setState] = useState<ViewerState | null>(null)
  const [searching, setSearching] = useState(false)
  const [query, setQuery] = useState('')
  const text = (en: string, zh: string) => data.chinese ? zh : en
  const iconStyle = { width: 20, height: 20 }
  return <div className="document" data-theme={data.dark ? 'dark' : 'light'}>
    <div className="tools" role="toolbar" aria-label={text('Document tools', '文档工具')}>
      <button aria-label={text('Search document', '搜索文档')} title={text('Search document', '搜索文档')} aria-expanded={searching} onClick={() => setSearching(!searching)}><Search style={iconStyle} /></button>
      <span className="tool-space" />
      <button aria-label={text('Zoom out', '缩小')} title={text('Zoom out', '缩小')} disabled={!state?.zoom?.canZoomOut} onClick={() => void viewer.current?.zoomOut()}><Minus style={iconStyle} /></button>
      <output className="zoom">{state?.zoom?.label ?? '100%'}</output>
      <button aria-label={text('Zoom in', '放大')} title={text('Zoom in', '放大')} disabled={!state?.zoom?.canZoomIn} onClick={() => void viewer.current?.zoomIn()}><Plus style={iconStyle} /></button>
      <button aria-label={text('Reset zoom', '重置缩放')} title={text('Reset zoom', '重置缩放')} disabled={!state?.zoom?.canReset} onClick={() => void viewer.current?.resetZoom()}><RotateCcw style={iconStyle} /></button>
    </div>
    {searching && <form className="search" onSubmit={event => { event.preventDefault(); void viewer.current?.searchDocument(query) }}>
      <input aria-label={text('Search document', '搜索文档')} placeholder={text('Search', '搜索')} value={query} onChange={event => setQuery(event.target.value)} />
      <button type="submit" aria-label={text('Find', '查找')}><Search style={iconStyle} /></button>
      <output>{state?.search?.total ? `${state.search.currentIndex + 1}/${state.search.total}` : '0/0'}</output>
      <button type="button" aria-label={text('Previous match', '上一个匹配')} disabled={!state?.search?.total} onClick={() => void viewer.current?.previousSearchResult()}><ChevronUp style={iconStyle} /></button>
      <button type="button" aria-label={text('Next match', '下一个匹配')} disabled={!state?.search?.total} onClick={() => void viewer.current?.nextSearchResult()}><ChevronDown style={iconStyle} /></button>
      <button type="button" aria-label={text('Close search', '关闭搜索')} onClick={() => { setSearching(false); setQuery(''); void viewer.current?.clearDocumentSearch() }}><X style={iconStyle} /></button>
    </form>}
    <FileViewer ref={viewer} file={file} filename={data.name} options={options} onStateChange={setState} style={{ flex: 1, minHeight: 0, width: '100%' }} />
  </div>
}
