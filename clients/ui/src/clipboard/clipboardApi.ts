import type { ProtoClientSession } from '../core/protoClientSession'
import { createRemoteStorageApi, storageText } from '../storage/remoteStorageApi'

export interface RemoteClipboardEntry {
  id: string
  text: string
  preview: string
  paneId?: string | undefined
  sourceApp?: string | undefined
  createdAt: string
  version: number
}

export interface RemoteClipboardApi {
  list(): Promise<RemoteClipboardEntry[]>
  putText(text: string): Promise<RemoteClipboardEntry>
  updateText(id: string, text: string): Promise<RemoteClipboardEntry>
  delete(id: string): Promise<void>
}

const clipboardStorageAppId = 'anytty.clipboard'
const clipboardHistoryPrefix = 'history/'
const clipboardRecordVersion = 1

export function createRemoteClipboardApi(session: ProtoClientSession): RemoteClipboardApi {
  const storage = createRemoteStorageApi(session)

  return {
    async list() {
      const entries = await storage.list({
        appId: clipboardStorageAppId,
        scope: 'public',
        prefix: clipboardHistoryPrefix,
      })
      return entries
        .map((entry) => decodeClipboardEntry(entry.key, storageText(entry), entry.version, entry.updatedAt))
        .filter((entry): entry is RemoteClipboardEntry => entry !== null)
        .sort((a, b) => Date.parse(b.createdAt) - Date.parse(a.createdAt))
    },
    async putText(text) {
      return await writeClipboardText(storage, text)
    },
    async updateText(id, text) {
      const trimmed = id.trim()
      if (!trimmed) throw new Error('clipboard id is required')
      const existing = await storage.get({
        appId: clipboardStorageAppId,
        scope: 'public',
        key: clipboardHistoryPrefix + trimmed,
      })
      const createdAt = decodeClipboardEntry(existing.key, storageText(existing), existing.version, existing.updatedAt)?.createdAt
      return await writeClipboardText(storage, text, trimmed, createdAt)
    },
    async delete(id) {
      const trimmed = id.trim()
      if (!trimmed) return
      await storage.delete({
        appId: clipboardStorageAppId,
        scope: 'public',
        key: clipboardHistoryPrefix + trimmed,
      })
    },
  }
}

async function writeClipboardText(
  storage: ReturnType<typeof createRemoteStorageApi>,
  text: string,
  id = `clip-${Date.now().toString(36)}-${Math.random().toString(36).slice(2, 10)}`,
  createdAt?: string,
): Promise<RemoteClipboardEntry> {
  const now = new Date().toISOString()
  const record = {
    schema_version: clipboardRecordVersion,
    id,
    text,
    preview: clipboardPreview(text),
    source_app: 'remote-ui',
    created_at: createdAt || now,
  }
  const entry = await storage.put({
    appId: clipboardStorageAppId,
    scope: 'public',
    key: clipboardHistoryPrefix + id,
    value: JSON.stringify(record),
  })
  return decodeClipboardEntry(entry.key, storageText(entry), entry.version, entry.updatedAt) ?? {
    id,
    text,
    preview: record.preview,
    sourceApp: 'remote-ui',
    createdAt: record.created_at,
    version: entry.version,
  }
}

function decodeClipboardEntry(key: string, raw: string, version: number, updatedAt?: string): RemoteClipboardEntry | null {
  try {
    const record = JSON.parse(raw) as Record<string, unknown>
    const text = typeof record.text === 'string' ? record.text : ''
    if (!text) return null
    const id = stringValue(record.id) || (key.startsWith(clipboardHistoryPrefix) ? key.slice(clipboardHistoryPrefix.length) : key)
    if (!id) return null
    return {
      id,
      text,
      preview: stringValue(record.preview) || clipboardPreview(text),
      paneId: stringValue(record.pane_id) || undefined,
      sourceApp: stringValue(record.source_app) || undefined,
      createdAt: stringValue(record.created_at) || updatedAt || new Date(0).toISOString(),
      version,
    }
  } catch {
    return null
  }
}

function clipboardPreview(text: string): string {
  const trimmed = text.replace(/\n/g, ' ').trim()
  if (!trimmed) return '(empty)'
  const runes = Array.from(trimmed)
  return runes.length > 72 ? `${runes.slice(0, 72).join('')}...` : trimmed
}

function stringValue(value: unknown): string {
  return typeof value === 'string' ? value.trim() : ''
}
