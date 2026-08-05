import type { ProtoClientSession } from '../core/protoClientSession'
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
