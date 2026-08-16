import { create, fromBinary, toBinary } from '@bufbuild/protobuf'
import {
  FileTransferAckSchema,
  FileTransferDataSchema,
  FileTransferFinishSchema,
  FileTransferResultSchema,
  ErrorEnvelopeSchema,
} from '../generated/wirepb/terminal_pb'

export function decodeFileTransferDataPayload(payload: Uint8Array): { offset: number; data: Uint8Array } {
  const value = fromBinary(FileTransferDataSchema, payload)
  return { offset: Number(value.offset), data: value.data.slice() }
}

export function encodeFileTransferDataPayload(value: { offset: number; data: Uint8Array }): Uint8Array {
  return toBinary(FileTransferDataSchema, create(FileTransferDataSchema, { offset: BigInt(value.offset), data: value.data }))
}

export function decodeFileStreamErrorPayload(payload: Uint8Array): string {
  const value = fromBinary(ErrorEnvelopeSchema, payload)
  return value.error?.message || 'file stream failed'
}

export function encodeFileTransferAckPayload(value: { offset: number; windowBytes: number }): Uint8Array {
  return toBinary(FileTransferAckSchema, create(FileTransferAckSchema, {
    offset: BigInt(value.offset),
    windowBytes: BigInt(value.windowBytes),
  }))
}

export function decodeFileTransferFinishPayload(payload: Uint8Array): { size: number; sha256: Uint8Array } {
  const value = fromBinary(FileTransferFinishSchema, payload)
  return { size: Number(value.size), sha256: value.sha256.slice() }
}

export function encodeFileTransferFinishPayload(value: { size: number; sha256: Uint8Array }): Uint8Array {
  return toBinary(FileTransferFinishSchema, create(FileTransferFinishSchema, {
    size: BigInt(value.size), sha256: value.sha256,
  }))
}

export function decodeFileTransferAckPayload(payload: Uint8Array): { offset: number; windowBytes: number } {
  const value = fromBinary(FileTransferAckSchema, payload)
  return { offset: Number(value.offset), windowBytes: Number(value.windowBytes) }
}

export function decodeFileTransferResultPayload(payload: Uint8Array): { path: string; size: number; sha256: Uint8Array } {
  const value = fromBinary(FileTransferResultSchema, payload)
  return { path: value.path, size: Number(value.size), sha256: value.sha256.slice() }
}
