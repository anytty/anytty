import { registerPlugin } from '@capacitor/core'

export interface PickedFile {
  uri: string
  name: string
  size: number
  mimeType: string
}

interface NativeFilePickerPlugin {
  pickFiles(options?: { multiple?: boolean }): Promise<{ files: PickedFile[] }>
  saveFile(options: { name: string; mimeType?: string; dataBase64: string }): Promise<{
    uri: string
    path: string
    bytes: number
    sha256: string
  }>
  getDownloadResumeOffset(options: DownloadIdentity): Promise<{ offset: number }>
  appendDownloadPartial(options: DownloadIdentity & { offset: number; dataBase64: string }): Promise<{ offset: number }>
  commitDownloadPartial(options: DownloadIdentity & { name: string; mimeType?: string; sha256Base64: string }): Promise<{
    uri: string
    path: string
    bytes: number
    sha256: string
  }>
  discardDownloadPartial(options: DownloadIdentity): Promise<{ discarded: boolean }>
  openUploadSource(options: { contentUri: string; offset: number; totalSize: number }): Promise<{ handle: string; offset: number }>
  readUploadSource(options: { handle: string; length: number }): Promise<{ dataBase64: string; offset: number; eof: boolean }>
  finishUploadSource(options: { handle: string }): Promise<{ sha256Base64: string }>
  closeUploadSource(options: { handle: string }): Promise<void>
}

export interface DownloadIdentity {
  machineId: string
  remotePath: string
  totalSize: number
}

const NativeFilePicker = registerPlugin<NativeFilePickerPlugin>('NativeFilePicker')

export default NativeFilePicker
