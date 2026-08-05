// @vitest-environment jsdom

import { App as CapApp } from '@capacitor/app'
import { Capacitor } from '@capacitor/core'
import { addNativeBackHandler } from '@anytty/ui'
import { cleanup, render } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { createElement, StrictMode } from 'react'
import { handleAndroidBackButton, useAndroidBackButton } from './androidBack'

const appPluginMock = vi.hoisted(() => ({
  addListener: vi.fn(),
  exitApp: vi.fn(),
}))

vi.mock('@capacitor/app', () => ({ App: appPluginMock }))

describe('Android native back bridge', () => {
  beforeEach(() => {
    appPluginMock.addListener.mockReset().mockResolvedValue({ remove: vi.fn(async () => undefined) })
    appPluginMock.exitApp.mockReset().mockResolvedValue(undefined)
    vi.spyOn(Capacitor, 'isNativePlatform').mockReturnValue(true)
    vi.spyOn(Capacitor, 'getPlatform').mockReturnValue('android')
    vi.spyOn(Capacitor, 'isPluginAvailable').mockReturnValue(true)
  })

  afterEach(() => {
    cleanup()
    vi.restoreAllMocks()
  })

  it('does not exit when one registered surface consumes Back', () => {
    const handler = vi.fn()
    const unregister = addNativeBackHandler(handler, 1)

    handleAndroidBackButton()

    expect(handler).toHaveBeenCalledOnce()
    expect(CapApp.exitApp).not.toHaveBeenCalled()
    unregister()
  })

  it('exits from the Android root with the standard Capacitor App plugin', () => {
    handleAndroidBackButton()

    expect(CapApp.exitApp).toHaveBeenCalledOnce()
  })

  it('does not exit on Web and fails closed when the App plugin is unavailable', () => {
    vi.mocked(Capacitor.isNativePlatform).mockReturnValue(false)
    handleAndroidBackButton()
    expect(CapApp.exitApp).not.toHaveBeenCalled()

    vi.mocked(Capacitor.isNativePlatform).mockReturnValue(true)
    vi.mocked(Capacitor.isPluginAvailable).mockReturnValue(false)
    expect(() => handleAndroidBackButton()).not.toThrow()
    expect(CapApp.exitApp).not.toHaveBeenCalled()
  })

  it('contains synchronous and asynchronous exit plugin failures', async () => {
    appPluginMock.exitApp.mockRejectedValueOnce(new Error('plugin rejected'))
    expect(() => handleAndroidBackButton()).not.toThrow()
    await Promise.resolve()

    appPluginMock.exitApp.mockImplementationOnce(() => { throw new Error('plugin missing') })
    expect(() => handleAndroidBackButton()).not.toThrow()
  })

  it('invalidates StrictMode listeners before deferred remove handles arrive', async () => {
    const firstSubscription = deferred<{ remove: () => Promise<void> }>()
    const secondSubscription = deferred<{ remove: () => Promise<void> }>()
    const firstRemove = vi.fn(async () => undefined)
    const secondRemove = vi.fn(async () => undefined)
    const callbacks: Array<() => void> = []
    appPluginMock.addListener
      .mockImplementationOnce((_event: string, callback: () => void) => {
        callbacks.push(callback)
        return firstSubscription.promise
      })
      .mockImplementationOnce((_event: string, callback: () => void) => {
        callbacks.push(callback)
        return secondSubscription.promise
      })
    const handler = vi.fn()
    const unregister = addNativeBackHandler(handler, 1)

    const view = render(createElement(StrictMode, null, createElement(AndroidBackHarness)))
    expect(callbacks).toHaveLength(2)

    callbacks[0]?.()
    expect(handler).not.toHaveBeenCalled()
    callbacks[1]?.()
    expect(handler).toHaveBeenCalledOnce()

    firstSubscription.resolve({ remove: firstRemove })
    await firstSubscription.promise
    await Promise.resolve()
    expect(firstRemove).toHaveBeenCalledOnce()

    view.unmount()
    callbacks[1]?.()
    expect(handler).toHaveBeenCalledOnce()
    secondSubscription.resolve({ remove: secondRemove })
    await secondSubscription.promise
    await Promise.resolve()
    expect(secondRemove).toHaveBeenCalledOnce()
    unregister()
  })
})

function AndroidBackHarness() {
  useAndroidBackButton()
  return null
}

function deferred<T>() {
  let resolve!: (value: T | PromiseLike<T>) => void
  let reject!: (reason?: unknown) => void
  const promise = new Promise<T>((res, rej) => {
    resolve = res
    reject = rej
  })
  return { promise, resolve, reject }
}
