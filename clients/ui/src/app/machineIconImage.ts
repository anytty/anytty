const maximumSourceImageBytes = 8 * 1024 * 1024
const maximumStoredImageLength = 96 * 1024
const iconSizes = [128, 96, 72] as const
const iconQualities = [0.82, 0.7, 0.58] as const

export type MachineIconImageErrorCode = 'invalid_image' | 'image_too_large' | 'image_processing_failed'

export async function compressMachineIconImage(file: File): Promise<string> {
  if (!file.type.startsWith('image/')) throw machineIconImageError('invalid_image')
  if (file.size > maximumSourceImageBytes) throw machineIconImageError('image_too_large')

  const source = await decodeImage(file)
  try {
    for (let index = 0; index < iconSizes.length; index += 1) {
      const dataUrl = await renderSquareImage(source.image, source.width, source.height, iconSizes[index]!, iconQualities[index]!)
      if (dataUrl.length <= maximumStoredImageLength) return dataUrl
    }
  } finally {
    source.close()
  }
  throw machineIconImageError('image_too_large')
}

export function machineIconImageErrorCode(error: unknown): MachineIconImageErrorCode {
  if (error instanceof Error && isMachineIconImageErrorCode(error.message)) return error.message
  return 'image_processing_failed'
}

async function decodeImage(file: File): Promise<{ image: CanvasImageSource; width: number; height: number; close: () => void }> {
  if (typeof globalThis.createImageBitmap === 'function') {
    try {
      const bitmap = await globalThis.createImageBitmap(file)
      return { image: bitmap, width: bitmap.width, height: bitmap.height, close: () => bitmap.close() }
    } catch {
      // Some WebViews expose createImageBitmap without supporting every camera image format.
    }
  }

  const objectUrl = URL.createObjectURL(file)
  try {
    const image = await new Promise<HTMLImageElement>((resolve, reject) => {
      const element = new Image()
      element.onload = () => resolve(element)
      element.onerror = () => reject(machineIconImageError('invalid_image'))
      element.src = objectUrl
    })
    return { image, width: image.naturalWidth, height: image.naturalHeight, close: () => URL.revokeObjectURL(objectUrl) }
  } catch (error) {
    URL.revokeObjectURL(objectUrl)
    throw error
  }
}

async function renderSquareImage(
  image: CanvasImageSource,
  sourceWidth: number,
  sourceHeight: number,
  size: number,
  quality: number,
): Promise<string> {
  if (sourceWidth <= 0 || sourceHeight <= 0) throw machineIconImageError('invalid_image')
  const canvas = document.createElement('canvas')
  canvas.width = size
  canvas.height = size
  const context = canvas.getContext('2d')
  if (!context) throw machineIconImageError('image_processing_failed')

  const sourceSize = Math.min(sourceWidth, sourceHeight)
  const sourceX = Math.floor((sourceWidth - sourceSize) / 2)
  const sourceY = Math.floor((sourceHeight - sourceSize) / 2)
  context.drawImage(image, sourceX, sourceY, sourceSize, sourceSize, 0, 0, size, size)
  const blob = await new Promise<Blob | null>((resolve) => canvas.toBlob(resolve, 'image/webp', quality))
  if (!blob || blob.type !== 'image/webp') throw machineIconImageError('image_processing_failed')
  return blobToDataUrl(blob)
}

function blobToDataUrl(blob: Blob): Promise<string> {
  return new Promise((resolve, reject) => {
    const reader = new FileReader()
    reader.onload = () => typeof reader.result === 'string'
      ? resolve(reader.result)
      : reject(machineIconImageError('image_processing_failed'))
    reader.onerror = () => reject(machineIconImageError('image_processing_failed'))
    reader.readAsDataURL(blob)
  })
}

function machineIconImageError(code: MachineIconImageErrorCode): Error {
  return new Error(code)
}

function isMachineIconImageErrorCode(value: string): value is MachineIconImageErrorCode {
  return value === 'invalid_image' || value === 'image_too_large' || value === 'image_processing_failed'
}
