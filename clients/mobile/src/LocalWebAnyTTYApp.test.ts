// @vitest-environment jsdom

import { render, screen } from '@testing-library/react'
import type { MachineWorkspaceProps } from '@anytty/ui'
import { createElement } from 'react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

const workspaceState = vi.hoisted(() => ({ props: null as MachineWorkspaceProps | null }))

vi.mock('@anytty/ui/machine-workspace', () => ({
  MachineWorkspace: (props: MachineWorkspaceProps) => {
    workspaceState.props = props
    return createElement('div', {
      'data-testid': 'local-machine-workspace',
      'data-machine-id': props.initialMachine?.machineId,
      'data-has-machine-back': String(Boolean(props.onBack)),
      'data-has-reauthorization': String(Boolean(props.onNeedsReauthorization)),
      'data-web-layout': String(Boolean(props.webLayout)),
    }, 'Terminal list')
  },
}))

import { LocalWebAnyTTYApp, parseLocalWebBootstrap } from './LocalWebAnyTTYApp'

const bootstrap = {
  bridge: {
    port: 43123,
    token: 'abcdefghijklmnopqrstuvwxyz0123456789_-ABCDE',
  },
  machine: { id: 'local' as const, name: 'Studio', platform: 'darwin' },
}

describe('LocalWebAnyTTYApp', () => {
  beforeEach(() => {
    const values = new Map<string, string>()
    vi.stubGlobal('localStorage', {
      getItem: (key: string) => values.get(key) ?? null,
      setItem: (key: string, value: string) => values.set(key, value),
      removeItem: (key: string) => values.delete(key),
    })
  })

  afterEach(() => vi.unstubAllGlobals())

  it('mounts the fixed local terminal workspace without machine navigation or pairing', async () => {
    render(createElement(LocalWebAnyTTYApp, { bootstrap, initialAppThemeStyle: {} }))

    const workspace = await screen.findByTestId('local-machine-workspace')
    expect(workspace.getAttribute('data-machine-id')).toBe('local')
    expect(workspace.getAttribute('data-has-machine-back')).toBe('false')
    expect(workspace.getAttribute('data-has-reauthorization')).toBe('false')
    expect(workspace.getAttribute('data-web-layout')).toBe('true')
    expect(screen.queryByText('Add device')).toBeNull()
    expect(screen.queryByText('添加设备')).toBeNull()
    await expect(workspaceState.props?.connector.connect({ machineId: 'remote' })).rejects.toThrow('local web cannot connect to another machine')
  })
})

describe('parseLocalWebBootstrap', () => {
  it('accepts a complete daemon bootstrap payload', () => {
    expect(
      parseLocalWebBootstrap(bootstrap),
    ).toEqual({
      bridge: {
        port: 43123,
        token: 'abcdefghijklmnopqrstuvwxyz0123456789_-ABCDE',
      },
      machine: { id: 'local', name: 'Studio', platform: 'darwin' },
    })
  })

  it.each([
    null,
    {},
    {
      bridge: { port: 0, token: 'abcdefghijklmnopqrstuvwxyz0123456789_-ABCDE' },
      machine: { id: 'local', name: 'Studio', platform: 'darwin' },
    },
    {
      bridge: { port: 43123, token: 'short' },
      machine: { id: 'local', name: 'Studio', platform: 'darwin' },
    },
    {
      bridge: {
        port: 43123,
        token: 'abcdefghijklmnopqrstuvwxyz0123456789_-ABCDE',
      },
      machine: { id: 'remote', name: 'Studio', platform: 'darwin' },
    },
  ])('rejects invalid payload %#', (payload) => {
    expect(parseLocalWebBootstrap(payload)).toBeNull()
  })
})
