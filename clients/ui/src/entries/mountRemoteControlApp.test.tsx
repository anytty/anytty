import { cleanup, screen } from '@testing-library/react'
import { afterEach, describe, expect, it } from 'vitest'
import { mountRemoteControlApp } from './mountRemoteControlApp'

function contrastRatio(foreground: string, background: string): number {
  const luminance = (value: string) => {
    const channels = value.slice(1).match(/.{2}/g)!.map((channel) => Number.parseInt(channel, 16) / 255)
    const linear = channels.map((channel) => channel <= 0.04045 ? channel / 12.92 : ((channel + 0.055) / 1.055) ** 2.4)
    return 0.2126 * linear[0]! + 0.7152 * linear[1]! + 0.0722 * linear[2]!
  }
  const values = [luminance(foreground), luminance(background)].sort((left, right) => right - left)
  return (values[0]! + 0.05) / (values[1]! + 0.05)
}

describe('mountRemoteControlApp', () => {
  afterEach(() => {
    cleanup()
    document.body.innerHTML = ''
  })

  it.each(['light', 'dark'] as const)('uses readable %s app theme semantics', async (appTheme) => {
    document.body.innerHTML = '<div id="root"></div>'
    const root = mountRemoteControlApp({ appTheme })

    const unavailable = await screen.findByTestId('anytty-cloud-unavailable')
    const background = unavailable.style.getPropertyValue('--anytty-app-surface')
    const text = unavailable.style.getPropertyValue('--anytty-app-text')
    expect(contrastRatio(text, background)).toBeGreaterThanOrEqual(4.5)
    expect(screen.getByRole('alert', { name: 'AnyTTY Cloud 暂不可用' })).toBe(unavailable)
    expect(unavailable.getAttribute('aria-describedby')).toBe('anytty-cloud-unavailable-description')
    expect(screen.getByRole('heading', { name: 'AnyTTY Cloud 暂不可用' }).className).toContain('text-[var(--anytty-app-text)]')
    expect(screen.getByText('云端服务正在重构。Direct 和 SSH 客户端不受影响。').className).toContain('text-[var(--anytty-app-text)]')
    expect(screen.queryByRole('button')).toBeNull()
    root.unmount()
  })
})
