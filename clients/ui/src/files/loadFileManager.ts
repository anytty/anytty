import type { ComponentType } from 'react'
import type { FileManagerProps } from './FileManager'

export type FileManagerComponent = ComponentType<FileManagerProps>

let fileManagerPromise: Promise<FileManagerComponent> | null = null

export function loadFileManager(): Promise<FileManagerComponent> {
  if (!fileManagerPromise) {
    fileManagerPromise = import('./FileManager').then((module) => module.FileManager)
  }
  return fileManagerPromise
}

export function reloadAfterFileManagerLoadFailure(): void {
  window.location.reload()
}
