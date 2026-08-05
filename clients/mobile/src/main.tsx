import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import { Capacitor } from '@capacitor/core'
import { setHapticImpactHandler } from '@anytty/ui'
import '@anytty/ui/styles.css'
import './index.css'
import { AnyTTYApp } from './AnyTTYApp'
import NativeHaptic from './plugins/nativeHaptic'

if (Capacitor.isNativePlatform()) {
  setHapticImpactHandler((pattern) => NativeHaptic.impact({ pattern }))
}

const root = document.getElementById('root')
if (!root) throw new Error('root element not found')

createRoot(root).render(
  <StrictMode>
    <AnyTTYApp />
  </StrictMode>,
)
