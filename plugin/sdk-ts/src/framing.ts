import { fromBinary, toBinary } from '@bufbuild/protobuf'
import { PluginBridgeFrameSchema, type PluginBridgeFrame } from './generated/apipb/plugin_pb.js'

export const MAX_FRAME_BYTES = 4 * 1024 * 1024

export function encodeFrame(frame: PluginBridgeFrame): Uint8Array {
  const data = toBinary(PluginBridgeFrameSchema, frame)
  if (data.length === 0 || data.length > MAX_FRAME_BYTES) throw new Error('Invalid plugin frame size')
  const prefix: number[] = []
  let size = data.length
  while (size >= 128) { prefix.push((size & 127) | 128); size = Math.floor(size / 128) }
  prefix.push(size)
  const output = new Uint8Array(prefix.length + data.length)
  output.set(prefix); output.set(data, prefix.length)
  return output
}

/** Incremental bounded decoder; no length-derived allocation before validation. */
export class FrameDecoder {
  private length = 0
  private shift = 0
  private prefixBytes = 0
  private body: Uint8Array | undefined
  private offset = 0
  push(chunk: Uint8Array, onFrame: (frame: PluginBridgeFrame) => void): void {
    let index = 0
    while (index < chunk.length) {
      if (!this.body) {
        const byte = chunk[index++]
        this.prefixBytes++
        if (this.prefixBytes > 5) throw new Error('Invalid plugin varint length')
        this.length += (byte & 127) * 2 ** this.shift
        if (this.length > MAX_FRAME_BYTES) throw new Error('Plugin frame exceeds size limit')
        if ((byte & 128) !== 0) { this.shift += 7; continue }
        if (this.length === 0) throw new Error('Empty plugin frame')
        this.body = new Uint8Array(this.length)
      }
      const count = Math.min(this.length - this.offset, chunk.length - index)
      this.body.set(chunk.subarray(index, index + count), this.offset)
      index += count; this.offset += count
      if (this.offset === this.length) {
        const frame = fromBinary(PluginBridgeFrameSchema, this.body)
        this.body = undefined; this.length = 0; this.offset = 0; this.shift = 0; this.prefixBytes = 0
        onFrame(frame)
      }
    }
  }
  finish(): void { if (this.prefixBytes !== 0 || this.body) throw new Error('Truncated plugin frame') }
}
