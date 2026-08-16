import type { FileEntry } from './fileApi'
import { extension as modelExtension } from './modelFileTypes'
import { anyttyI18n, anyttyIntlLocale } from '../i18n'

export function basename(path: string): string {
  const normalized = path.replace(/\\/g, '/').replace(/\/+$/, '')
  const index = normalized.lastIndexOf('/')
  return index >= 0 ? normalized.slice(index + 1) : normalized
}

export function extension(name: string): string {
  return modelExtension(name)
}

export function joinPath(base: string, name: string): string {
  const normalizedBase = normalizeFilePath(base)
  const child = name.replace(/\\/g, '/').replace(/^\/+/, '')
  if (normalizedBase === '/') return `/${child}`
  return `${normalizedBase.replace(/\/+$/, '')}/${child}`
}

export function fileEntryPath(base: string, entry: Pick<FileEntry, 'path' | 'name'>): string {
  return entry.path ? normalizeFilePath(entry.path) : joinPath(base, entry.name)
}

export function normalizeFilePath(path: string): string {
  const normalized = path.trim().replace(/\\/g, '/')
  if (!normalized || normalized === '/') return '/'
  if (/^[A-Za-z]:\/+$/i.test(normalized)) return `${normalized.slice(0, 2)}/`
  return normalized.replace(/\/+$/, '') || '/'
}

export function parentPath(path: string): string {
  const normalized = normalizeFilePath(path)
  if (normalized === '/') return '/'
  if (/^[A-Za-z]:\/$/i.test(normalized)) return normalized
  const uncRoot = normalized.match(/^(\/\/[^/]+\/[^/]+)(?:\/|$)/)?.[1]
  if (uncRoot === normalized) return normalized
  const index = normalized.lastIndexOf('/')
  if (index === 2 && /^[A-Za-z]:/i.test(normalized)) return `${normalized.slice(0, 2)}/`
  if (index <= 0) return '/'
  return normalized.slice(0, index)
}

export interface PathBreadcrumb {
  label: string
  path: string
}

export function pathBreadcrumbs(path: string): PathBreadcrumb[] {
  const normalized = normalizeFilePath(path)
  const drive = normalized.match(/^([A-Za-z]:)(?:\/(.*))?$/)
  if (drive) {
    const root = `${drive[1]}/`
    return [
      { label: '/', path: '/' },
      ...breadcrumbsFromSegments(root, drive[1]!, drive[2]),
    ]
  }

  const unc = normalized.match(/^(\/\/[^/]+\/[^/]+)(?:\/(.*))?$/)
  if (unc) {
    return breadcrumbsFromSegments(unc[1]!, unc[1]!, unc[2])
  }

  return breadcrumbsFromSegments('/', '/', normalized.replace(/^\/+/, ''))
}

function breadcrumbsFromSegments(rootPath: string, rootLabel: string, remainder: string | undefined): PathBreadcrumb[] {
  const breadcrumbs: PathBreadcrumb[] = [{ label: rootLabel, path: rootPath }]
  let current = rootPath.replace(/\/+$/, '')
  for (const segment of remainder?.split('/').filter(Boolean) ?? []) {
    current = `${current}/${segment}`
    breadcrumbs.push({ label: segment, path: current })
  }
  return breadcrumbs
}

export function clamp(value: number, min: number, max: number): number {
  return Math.min(Math.max(value, min), max)
}

export function formatBytes(bytes: number, decimals = 1) {
  if (!+bytes) return '0 B'
  const k = 1024
  const dm = decimals < 0 ? 0 : decimals
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return `${parseFloat((bytes / Math.pow(k, i)).toFixed(dm))} ${sizes[i]}`
}

export function isMarkdownFile(name: string, mimeType: string): boolean {
  const ext = extension(name)
  return mimeType === 'text/markdown' || ext === 'md' || ext === 'markdown' || ext === 'mdx'
}

export function fileEntryMeta(entry: Pick<FileEntry, 'type' | 'size' | 'modTime' | 'childCount' | 'linkTarget' | 'hardLink' | 'linkCount' | 'inode'>): string {
  const parts: string[] = []
  const isDirectory = entry.type === 'dir' || entry.type === 'symlink-dir'
  if (isDirectory) {
    if (typeof entry.childCount === 'number') {
      parts.push(anyttyI18n.t('files.meta.items', { count: entry.childCount }))
    } else {
      parts.push(anyttyI18n.t('files.meta.folder'))
    }
  } else {
    parts.push(formatBytes(entry.size))
  }
  const modified = formatModifiedTime(entry.modTime)
  if (modified) parts.push(modified)
  if (entry.linkTarget) parts.push(`-> ${entry.linkTarget}`)
  else if (entry.hardLink) {
    const hardLink = entry.linkCount && entry.linkCount > 1
      ? anyttyI18n.t('files.meta.hardLinkCount', { count: entry.linkCount })
      : anyttyI18n.t('files.meta.hardLink')
    parts.push(entry.inode ? `${hardLink} · inode ${entry.inode}` : hardLink)
  }
  return parts.join(' · ')
}

export function fileEntryMenuSubtitle(entry: Pick<FileEntry, 'type' | 'size' | 'childCount' | 'linkTarget' | 'hardLink' | 'linkCount' | 'inode'>): string {
  const isDirectory = entry.type === 'dir' || entry.type === 'symlink-dir'
  if (isDirectory) {
    const count = typeof entry.childCount === 'number'
      ? anyttyI18n.t('files.meta.items', { count: entry.childCount })
      : anyttyI18n.t('files.meta.folder')
    if (entry.linkTarget) return `${count} · -> ${entry.linkTarget}`
    return count
  }
  if (entry.linkTarget) return `${formatBytes(entry.size)} · -> ${entry.linkTarget}`
  if (entry.hardLink) {
    const hardLink = entry.linkCount && entry.linkCount > 1
      ? anyttyI18n.t('files.meta.hardLinkCount', { count: entry.linkCount })
      : anyttyI18n.t('files.meta.hardLink')
    return entry.inode ? `${formatBytes(entry.size)} · ${hardLink} · inode ${entry.inode}` : `${formatBytes(entry.size)} · ${hardLink}`
  }
  return `${formatBytes(entry.size)} · ${anyttyI18n.t('files.meta.file')}`
}

function formatModifiedTime(value: string | undefined): string {
  if (!value) return ''
  const timestamp = Date.parse(value)
  if (!Number.isFinite(timestamp)) return value
  return new Date(timestamp).toLocaleString(anyttyIntlLocale(), {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
  })
}
