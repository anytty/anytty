import type { ProtoClientSession } from '../core/protoClientSession'
import type { RemoteRuntimeStorage } from '../core/transport'
import { createRemoteStorageApi, storageText } from '../storage/remoteStorageApi'

export interface PathBookmark {
  id: string
  path: string
  label: string
  createdAt: string
  updatedAt: string
  version: number
}

export interface PathBookmarkApi {
  list(): Promise<PathBookmark[]>
  add(path: string, label?: string): Promise<PathBookmark>
  update(id: string, input: { label?: string | undefined; path?: string | undefined }): Promise<PathBookmark>
  remove(id: string): Promise<void>
}

const pathBookmarkStorageAppId = 'anytty.paths'
const pathBookmarkPrefix = 'bookmarks/'
const pathBookmarkRecordVersion = 1
const localPathBookmarkStoragePrefix = 'anytty.path-bookmarks.v1:'

/**
 * App-local bookmarks are authoritative. On first use, older daemon-side bookmarks are
 * imported once so upgrading does not discard paths the user already saved.
 */
export function createPersistentPathBookmarkApi(
  machineId: string,
  session?: ProtoClientSession,
  storage: RemoteRuntimeStorage | undefined = browserStorage(),
): PathBookmarkApi {
  if (!machineId || !storage) return session ? createPathBookmarkApi(session) : emptyPathBookmarkApi()
  const key = localPathBookmarkStoragePrefix + machineId
  const remote = session ? createPathBookmarkApi(session) : null

  const read = (): PathBookmark[] | null => decodeLocalPathBookmarks(storage.getItem(key))
  const write = (bookmarks: PathBookmark[]): PathBookmark[] => {
    const normalized = sortPathBookmarks(bookmarks)
    storage.setItem(key, JSON.stringify(normalized))
    return normalized
  }
  const ensure = async (): Promise<PathBookmark[]> => {
    const local = read()
    if (local) return local
    if (remote) {
      try {
        return write(await remote.list())
      } catch {
        // The local store must remain usable while the device is offline.
      }
    }
    return write([])
  }

  return {
    list: ensure,
    async add(path, label) {
      const bookmarks = await ensure()
      const normalizedPath = normalizeBookmarkedPath(path)
      const existing = bookmarks.find((bookmark) => bookmark.path === normalizedPath)
      const now = new Date().toISOString()
      if (existing) {
        const updated = { ...existing, label: label?.trim() || existing.label, updatedAt: now, version: existing.version + 1 }
        write(bookmarks.map((bookmark) => bookmark.id === existing.id ? updated : bookmark))
        return updated
      }
      const bookmark: PathBookmark = {
        id: createBookmarkId(normalizedPath),
        path: normalizedPath,
        label: label?.trim() || bookmarkLabel(normalizedPath),
        createdAt: now,
        updatedAt: now,
        version: 1,
      }
      write([...bookmarks, bookmark])
      return bookmark
    },
    async update(id, input) {
      const bookmarks = await ensure()
      const current = bookmarks.find((bookmark) => bookmark.id === id.trim())
      if (!current) throw new Error('Bookmark not found')
      const path = input.path !== undefined ? normalizeBookmarkedPath(input.path) : current.path
      const updated: PathBookmark = {
        ...current,
        path,
        label: input.label?.trim() || (input.path !== undefined ? bookmarkLabel(path) : current.label),
        updatedAt: new Date().toISOString(),
        version: current.version + 1,
      }
      write(bookmarks.map((bookmark) => bookmark.id === current.id ? updated : bookmark))
      return updated
    },
    async remove(id) {
      const bookmarks = await ensure()
      write(bookmarks.filter((bookmark) => bookmark.id !== id.trim()))
    },
  }
}

export function createPathBookmarkApi(session: ProtoClientSession): PathBookmarkApi {
  const storage = createRemoteStorageApi(session)

  return {
    async list() {
      const entries = await storage.list({
        appId: pathBookmarkStorageAppId,
        scope: 'public',
        prefix: pathBookmarkPrefix,
      })
      return entries
        .map((entry) => decodePathBookmark(entry.key, storageText(entry), entry.version, entry.updatedAt))
        .filter((entry): entry is PathBookmark => entry !== null)
        .sort((a, b) => a.label.localeCompare(b.label, undefined, { numeric: true, sensitivity: 'base' }))
    },
    async add(path, label) {
      const normalizedPath = normalizeBookmarkedPath(path)
      const id = createBookmarkId(normalizedPath)
      const now = new Date().toISOString()
      const record = {
        schema_version: pathBookmarkRecordVersion,
        id,
        path: normalizedPath,
        label: label?.trim() || bookmarkLabel(normalizedPath),
        created_at: now,
        updated_at: now,
      }
      const entry = await storage.put({
        appId: pathBookmarkStorageAppId,
        scope: 'public',
        key: pathBookmarkPrefix + id,
        value: JSON.stringify(record),
      })
      return decodePathBookmark(entry.key, storageText(entry), entry.version, entry.updatedAt) ?? {
        id,
        path: normalizedPath,
        label: record.label,
        createdAt: now,
        updatedAt: now,
        version: entry.version,
      }
    },
    async update(id, input) {
      const trimmed = id.trim()
      if (!trimmed) throw new Error('Bookmark id is required')
      const existing = await storage.get({
        appId: pathBookmarkStorageAppId,
        scope: 'public',
        key: pathBookmarkPrefix + trimmed,
      })
      const current = decodePathBookmark(existing.key, storageText(existing), existing.version, existing.updatedAt)
      if (!current) throw new Error('Bookmark not found')
      const path = input.path !== undefined ? normalizeBookmarkedPath(input.path) : current.path
      const label = input.label?.trim() || bookmarkLabel(path)
      const now = new Date().toISOString()
      const record = {
        schema_version: pathBookmarkRecordVersion,
        id: current.id,
        path,
        label,
        created_at: current.createdAt,
        updated_at: now,
      }
      const entry = await storage.put({
        appId: pathBookmarkStorageAppId,
        scope: 'public',
        key: pathBookmarkPrefix + current.id,
        value: JSON.stringify(record),
      })
      return decodePathBookmark(entry.key, storageText(entry), entry.version, entry.updatedAt) ?? {
        id: current.id,
        path,
        label,
        createdAt: current.createdAt,
        updatedAt: now,
        version: entry.version,
      }
    },
    async remove(id) {
      const trimmed = id.trim()
      if (!trimmed) return
      await storage.delete({
        appId: pathBookmarkStorageAppId,
        scope: 'public',
        key: pathBookmarkPrefix + trimmed,
      })
    },
  }
}

export function bookmarkLabel(path: string): string {
  const normalized = normalizeBookmarkedPath(path)
  if (normalized === '/') return '/'
  const parts = normalized.split('/').filter(Boolean)
  return parts[parts.length - 1] || normalized
}

function decodePathBookmark(key: string, raw: string, version: number, updatedAt?: string): PathBookmark | null {
  try {
    const record = JSON.parse(raw) as Record<string, unknown>
    const path = normalizeBookmarkedPath(stringValue(record.path))
    if (!path) return null
    const id = stringValue(record.id) || (key.startsWith(pathBookmarkPrefix) ? key.slice(pathBookmarkPrefix.length) : key) || pathIdPrefix(path)
    return {
      id,
      path,
      label: stringValue(record.label) || bookmarkLabel(path),
      createdAt: stringValue(record.created_at) || updatedAt || new Date(0).toISOString(),
      updatedAt: stringValue(record.updated_at) || updatedAt || new Date(0).toISOString(),
      version,
    }
  } catch {
    return null
  }
}

function createBookmarkId(path: string): string {
  return `${pathIdPrefix(path)}~${randomIdPart()}`
}

function pathIdPrefix(path: string): string {
  const encoded = encodeURIComponent(normalizeBookmarkedPath(path))
  return encoded.replace(/%/g, '~')
}

function randomIdPart(): string {
  const cryptoApi = globalThis.crypto
  if (cryptoApi && 'randomUUID' in cryptoApi && typeof cryptoApi.randomUUID === 'function') {
    return cryptoApi.randomUUID().replace(/-/g, '').slice(0, 12)
  }
  return `${Date.now().toString(36)}${Math.random().toString(36).slice(2, 8)}`
}

function normalizeBookmarkedPath(path: string): string {
  const trimmed = path.trim()
  if (!trimmed || trimmed === '/') return '/'
  return trimmed.replace(/\/+$/, '') || '/'
}

function stringValue(value: unknown): string {
  return typeof value === 'string' ? value.trim() : ''
}

function browserStorage(): RemoteRuntimeStorage | undefined {
  if (typeof window === 'undefined') return undefined
  try {
    return window.localStorage
  } catch {
    return undefined
  }
}

function decodeLocalPathBookmarks(raw: string | null): PathBookmark[] | null {
  if (raw === null) return null
  try {
    const parsed = JSON.parse(raw) as unknown
    if (!Array.isArray(parsed)) return []
    return sortPathBookmarks(parsed.flatMap((value): PathBookmark[] => {
      if (!value || typeof value !== 'object') return []
      const record = value as Record<string, unknown>
      const path = normalizeBookmarkedPath(stringValue(record.path))
      const id = stringValue(record.id)
      if (!id || !path) return []
      return [{
        id,
        path,
        label: stringValue(record.label) || bookmarkLabel(path),
        createdAt: stringValue(record.createdAt) || new Date(0).toISOString(),
        updatedAt: stringValue(record.updatedAt) || new Date(0).toISOString(),
        version: typeof record.version === 'number' && Number.isFinite(record.version) ? record.version : 1,
      }]
    }))
  } catch {
    return []
  }
}

function sortPathBookmarks(bookmarks: PathBookmark[]): PathBookmark[] {
  return [...bookmarks].sort((a, b) => a.label.localeCompare(b.label, undefined, { numeric: true, sensitivity: 'base' }))
}

function emptyPathBookmarkApi(): PathBookmarkApi {
  return {
    async list() { return [] },
    async add() { throw new Error('Local storage is unavailable') },
    async update() { throw new Error('Local storage is unavailable') },
    async remove() {},
  }
}
