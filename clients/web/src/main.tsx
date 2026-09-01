import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import '@anytty/ui/styles.css'
import './index.css'
import { AnyTTYApp } from './AnyTTYApp'

const root = document.getElementById('root')
if (!root) throw new Error('root element not found')

createRoot(root).render(
  <StrictMode>
    <AnyTTYApp />
  </StrictMode>,
)
