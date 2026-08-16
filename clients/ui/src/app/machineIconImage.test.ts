import { afterEach, describe, expect, it, vi } from 'vitest'
import { compressMachineIconImage, machineIconImageErrorCode } from './machineIconImage'

afterEach(() => {
  vi.restoreAllMocks()
  vi.unstubAllGlobals()
})

describe('machine icon image compression', () => {
  it('rejects non-image input before decoding it', async () => {
    const file = new File(['plain text'], 'notes.txt', { type: 'text/plain' })

    await expect(compressMachineIconImage(file)).rejects.toMatchObject({ message: 'invalid_image' })
  })

  it('center-crops and stores a small WebP data URL', async () => {
    const close = vi.fn()
    const drawImage = vi.fn()
    vi.stubGlobal('createImageBitmap', vi.fn(async () => ({ width: 200, height: 100, close })))
    vi.spyOn(HTMLCanvasElement.prototype, 'getContext').mockReturnValue({ drawImage } as unknown as CanvasRenderingContext2D)
    vi.spyOn(HTMLCanvasElement.prototype, 'toBlob').mockImplementation((callback) => {
      callback(new Blob(['compressed'], { type: 'image/webp' }))
    })

    const result = await compressMachineIconImage(new File(['source'], 'device.png', { type: 'image/png' }))

    expect(result).toMatch(/^data:image\/webp;base64,/)
    expect(drawImage).toHaveBeenCalledWith(expect.anything(), 50, 0, 100, 100, 0, 0, 128, 128)
    expect(close).toHaveBeenCalledOnce()
  })

  it('maps unexpected failures to a stable UI error code', () => {
    expect(machineIconImageErrorCode(new Error('decoder crashed'))).toBe('image_processing_failed')
  })
})
