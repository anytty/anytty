import { useEffect } from 'react'
import { App as CapApp } from '@capacitor/app'
import { Capacitor } from '@capacitor/core'
import { dispatchNativeBack } from '@anytty/ui'

let nextAndroidBackListenerGeneration = 1
let activeAndroidBackListenerGeneration = 0

export function handleAndroidBackButton(): void {
  if (dispatchNativeBack()) return
  if (!Capacitor.isNativePlatform() || Capacitor.getPlatform() !== 'android') return
  if (!Capacitor.isPluginAvailable('App')) return

  try {
    void CapApp.exitApp().catch(() => undefined)
  } catch {}
}

export function useAndroidBackButton(): void {
  useEffect(() => {
    if (!Capacitor.isNativePlatform() || Capacitor.getPlatform() !== 'android') return undefined
    if (!Capacitor.isPluginAvailable('App')) return undefined

    const generation = nextAndroidBackListenerGeneration++
    let active = true
    activeAndroidBackListenerGeneration = generation
    let subscription: ReturnType<typeof CapApp.addListener>
    try {
      subscription = CapApp.addListener('backButton', () => {
        if (!active || activeAndroidBackListenerGeneration !== generation) return
        handleAndroidBackButton()
      })
    } catch {
      active = false
      if (activeAndroidBackListenerGeneration === generation) activeAndroidBackListenerGeneration = 0
      return undefined
    }
    void subscription.catch(() => {
      active = false
      if (activeAndroidBackListenerGeneration === generation) activeAndroidBackListenerGeneration = 0
    })
    return () => {
      active = false
      if (activeAndroidBackListenerGeneration === generation) activeAndroidBackListenerGeneration = 0
      void subscription.then((handle) => handle.remove()).catch(() => undefined)
    }
  }, [])
}
