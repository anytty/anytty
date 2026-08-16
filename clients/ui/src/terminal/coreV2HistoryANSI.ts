import type {
  CoreV2HistoryCell,
  CoreV2HistoryCellStyle,
  CoreV2HistoryRow,
} from './coreV2TerminalProtocol'

const historyGraphemeSegmenter = new Intl.Segmenter(undefined, { granularity: 'grapheme' })

export function coreV2ReflowHistoryRows(rows: CoreV2HistoryRow[], cols: number): CoreV2HistoryRow[] {
  const width = cols > 0 ? Math.trunc(cols) : 80
  const visualRows = rows.flatMap((row) => reflowHistoryRow(row, width))
  return visualRows.map((row, index) => ({ ...row, index }))
}

function reflowHistoryRow(row: CoreV2HistoryRow, cols: number): CoreV2HistoryRow[] {
  if (row.fixedGrid) {
    const cells: CoreV2HistoryCell[] = []
    let width = 0
    for (const cell of row.cells.flatMap(splitHistoryCell)) {
      const cellWidth = Math.max(1, cell.width)
      if (width + cellWidth > cols) break
      cells.push(cell)
      width += cellWidth
    }
    return [{ ...row, cells, logicalStartCol: 0, wrapped: false }]
  }
  const cells = row.cells.flatMap(splitHistoryCell)
  if (cells.length === 0) return [{ ...row, rowInLine: 0, logicalStartCol: 0, cells: [] }]

  const visualRows: CoreV2HistoryRow[] = []
  let current: CoreV2HistoryCell[] = []
  let currentWidth = 0
  let logicalStartCol = 0
  const flush = () => {
    visualRows.push({
      ...row,
      cells: current,
      tailFillStyle: undefined,
      rowInLine: visualRows.length,
      logicalStartCol,
      wrapped: true,
    })
    logicalStartCol += currentWidth
    current = []
    currentWidth = 0
  }
  for (const cell of cells) {
    const cellWidth = Math.max(1, cell.width)
    if (currentWidth > 0 && currentWidth + cellWidth > cols) flush()
    current.push(cell)
    currentWidth += cellWidth
    if (currentWidth >= cols) flush()
  }
  if (current.length > 0 || visualRows.length === 0) flush()
  const last = visualRows.at(-1)!
  last.tailFillStyle = row.tailFillStyle
  last.wrapped = row.wrapped
  return visualRows
}

function splitHistoryCell(cell: CoreV2HistoryCell): CoreV2HistoryCell[] {
  const width = Math.max(1, cell.width)
  if (!cell.text) {
    return Array.from({ length: width }, () => ({ ...cell, text: ' ', width: 1, linkUrl: undefined, linkParams: undefined }))
  }
  const graphemes = Array.from(historyGraphemeSegmenter.segment(cell.text), (part) => part.segment)
  if (graphemes.length <= 1) return [{ ...cell, width }]
  // The API only coalesces adjacent history graphemes with equal terminal width.
  const graphemeWidth = width / graphemes.length
  return graphemes.map((text) => ({ ...cell, text, width: graphemeWidth }))
}

export function coreV2HistoryRowsANSI(rows: CoreV2HistoryRow[], cols: number): string {
  return rows.map((row, index) => {
    let currentStyle = ''
    let width = 0
    let output = ''
    for (const cell of row.cells) {
      const style = styleANSI(cell.style)
      if (style !== currentStyle) {
        output += `\u001b[0m${style}`
        currentStyle = style
      }
      const cellWidth = Math.max(1, cell.width)
      output += cell.text || ' '.repeat(cellWidth)
      width += cellWidth
    }
    if (cols > width) {
      output += `\u001b[0m${styleANSI(row.tailFillStyle)}${' '.repeat(cols - width)}`
    }
    const separator = index === rows.length - 1 || row.wrapped ? '' : '\r\n'
    return `${output}\u001b[0m${separator}`
  }).join('')
}

function styleANSI(style: CoreV2HistoryCellStyle | undefined): string {
  if (!style) return ''
  const codes: string[] = []
  if (style.bold) codes.push('1')
  if (style.italic) codes.push('3')
  if (style.underline) codes.push('4')
  if (style.blink) codes.push('5')
  if (style.reverse) codes.push('7')
  if (style.strikethrough) codes.push('9')
  const foreground = colorANSI(style.fg, true)
  const background = colorANSI(style.bg, false)
  if (foreground) codes.push(foreground)
  if (background) codes.push(background)
  return codes.length > 0 ? `\u001b[${codes.join(';')}m` : ''
}

function colorANSI(value: string | undefined, foreground: boolean): string {
  const token = value?.trim() ?? ''
  if (!token) return ''
  const indexed = /^(?:(?:ansi|idx):)?(\d{1,3})$/i.exec(token)
  if (indexed) {
    const index = Math.min(255, Number(indexed[1]))
    return `${foreground ? 38 : 48};5;${index}`
  }
  const rgb = /^#([0-9a-f]{2})([0-9a-f]{2})([0-9a-f]{2})$/i.exec(token)
  if (!rgb) return ''
  return `${foreground ? 38 : 48};2;${parseInt(rgb[1]!, 16)};${parseInt(rgb[2]!, 16)};${parseInt(rgb[3]!, 16)}`
}
