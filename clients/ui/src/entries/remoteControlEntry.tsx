import { mountRemoteControlApp } from './mountRemoteControlApp'
import './appStyles.css'

if (typeof document !== 'undefined' && document.getElementById('root')) {
  mountRemoteControlApp()
}
