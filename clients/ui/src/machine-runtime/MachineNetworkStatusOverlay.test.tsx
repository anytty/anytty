import { cleanup, render, screen } from '@testing-library/react'
import { afterEach, describe, expect, it } from 'vitest'
import { anyttyI18n } from '../i18n'
import { MachineNetworkStatusOverlay } from './MachineNetworkStatusOverlay'

describe('MachineNetworkStatusOverlay', () => {
  afterEach(async () => {
    cleanup()
    await anyttyI18n.changeLanguage('en')
  })

  it('uses theme variables so light and dark terminal themes stay readable', () => {
    render(<MachineNetworkStatusOverlay phase="waiting_network" status="Waiting for network..." />)

    const overlay = screen.getByTestId('anytty-machine-network-overlay')
    expect(overlay.className).toContain('bg-[var(--anytty-overlay)]')
    expect(overlay.className).toContain('pointer-events-auto')
    expect(overlay.className).not.toContain('pointer-events-none')

    const card = overlay.firstElementChild
    if (!(card instanceof HTMLDivElement)) throw new Error('overlay card was not rendered')
    expect(card.className).toContain('border-[var(--anytty-border)]')
    expect(card.className).toContain('bg-[var(--anytty-surface)]')
    expect(card.className).toContain('text-[var(--anytty-text)]')
    expect(card.className).toContain('rounded-lg')
    expect(card.className).toContain('shadow-sm')
    expect(card.textContent).toContain('Your phone is offline.')
  })

  it('uses the stable phase instead of exposing a native implementation message', async () => {
    await anyttyI18n.changeLanguage('zh-CN')
    render(<MachineNetworkStatusOverlay phase="signaling" status="JNI runtime handle 42" />)

    const overlay = screen.getByTestId('anytty-machine-network-overlay')
    expect(overlay.textContent).toContain('正在连接设备')
    expect((overlay.textContent.match(/正在连接设备/g) ?? []).length).toBe(1)
    expect(overlay.textContent).not.toMatch(/JNI|runtime|handle/i)
  })
})
