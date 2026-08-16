import { describe, expect, it } from 'vitest'
import { createTerminalOptions } from './Terminal'
import { DEFAULT_TERMINAL_SETTINGS } from './terminalSettings'

describe('terminal accessibility', () => {
  it('enables xterm screen-reader output without changing terminal input settings', () => {
    const options = createTerminalOptions(DEFAULT_TERMINAL_SETTINGS)

    expect(options.screenReaderMode).toBe(true)
    expect(options.convertEol).toBe(false)
    expect(options.scrollback).toBe(DEFAULT_TERMINAL_SETTINGS.scrollback)
  })
})
