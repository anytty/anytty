export type RemoteResumeType = 'quick' | 'normal' | 'cold'

export interface RemoteNetworkState {
  phoneOnline: boolean
  connectionType: string
  appActive: boolean
  jsFrozenRecovery: boolean
  networkReady: boolean
  resumeType: RemoteResumeType | null
  resumeDuration: number
}

export interface NativeNetworkStatusPlugin {
  getStatus?: (() => Promise<NativeNetworkStatus>) | undefined
  addListener?: ((
    eventName: 'networkStatusChange',
    handler: (status: NativeNetworkStatus) => void,
  ) => NativePluginListenerHandle | Promise<NativePluginListenerHandle>) | undefined
}

type NetworkStateListener = (state: RemoteNetworkState, prevState: RemoteNetworkState) => void

const HEARTBEAT_INTERVAL_MS = 3000
const FREEZE_THRESHOLD_MS = 5000
const QUICK_RESUME_THRESHOLD_MS = 30000
const NORMAL_RESUME_THRESHOLD_MS = 300000
const NOTIFY_DEBOUNCE_MS = 100

export class RemoteNetworkStateManager {
  private readonly listeners = new Set<NetworkStateListener>()
  private readonly snapshotListeners = new Set<() => void>()
  private cleanup: Array<() => void> = []
  private initialized = false
  private heartbeatTimer: ReturnType<typeof setInterval> | null = null
  private notifyTimer: ReturnType<typeof setTimeout> | null = null
  private lastHeartbeat = Date.now()
  private backgroundTimestamp: number | null = null
  private lastResumeAt = 0

  private phoneOnline = browserNavigator()?.onLine ?? true
  private connectionType = networkConnectionType()
  private appActive = browserDocument()?.hidden === false || browserDocument()?.hidden === undefined
  private jsFrozenRecovery = false
  private resumeType: RemoteResumeType | null = null
  private resumeDuration = 0
  private lastSnapshot: RemoteNetworkState = this.createSnapshot()

  constructor(private readonly nativeNetworkPlugin?: NativeNetworkStatusPlugin) {}

  get state(): RemoteNetworkState {
    return this.lastSnapshot
  }

  init(): void {
    if (this.initialized) return
    this.initialized = true
    this.initPhoneNetwork()
    void this.initNativeNetwork()
    this.initAppLifecycle()
    void this.initNativeAppLifecycle()
    this.initHeartbeat()
  }

  subscribe(listener: NetworkStateListener): () => void {
    this.listeners.add(listener)
    return () => this.listeners.delete(listener)
  }

  subscribeSnapshot(listener: () => void): () => void {
    this.snapshotListeners.add(listener)
    return () => this.snapshotListeners.delete(listener)
  }

  destroy(): void {
    for (const fn of this.cleanup) fn()
    this.cleanup = []
    if (this.heartbeatTimer) {
      clearInterval(this.heartbeatTimer)
      this.heartbeatTimer = null
    }
    if (this.notifyTimer) {
      clearTimeout(this.notifyTimer)
      this.notifyTimer = null
    }
    this.listeners.clear()
    this.snapshotListeners.clear()
    this.initialized = false
  }

  private initPhoneNetwork(): void {
    const win = browserWindow()
    if (!win) return
    const onOnline = () => {
      this.phoneOnline = true
      this.scheduleNotify()
    }
    const onOffline = () => {
      this.phoneOnline = false
      this.scheduleNotify()
    }
    win.addEventListener('online', onOnline)
    win.addEventListener('offline', onOffline)
    this.cleanup.push(() => {
      win.removeEventListener('online', onOnline)
      win.removeEventListener('offline', onOffline)
    })

    const connection = browserNetworkConnection()
    if (!connection?.addEventListener || !connection.removeEventListener) return
    const onChange = () => {
      this.connectionType = networkConnectionType()
      this.phoneOnline = browserNavigator()?.onLine ?? this.phoneOnline
      this.scheduleNotify()
    }
    connection.addEventListener('change', onChange)
    this.cleanup.push(() => connection.removeEventListener?.('change', onChange))
  }

  private initAppLifecycle(): void {
    const doc = browserDocument()
    if (!doc) return
    const onVisibilityChange = () => {
      if (doc.visibilityState === 'hidden') {
        this.onBackground()
        return
      }
      if (doc.visibilityState === 'visible') {
        this.onResume()
      }
    }
    doc.addEventListener('visibilitychange', onVisibilityChange)
    this.cleanup.push(() => doc.removeEventListener('visibilitychange', onVisibilityChange))
  }

  private async initNativeNetwork(): Promise<void> {
    const network = this.nativeNetworkPlugin ?? capacitorNetworkPlugin()
    if (!network) return
    try {
      const status = await network.getStatus?.()
      if (status) {
        this.phoneOnline = status.connected
        this.connectionType = status.connectionType ?? this.connectionType
        this.scheduleNotify()
      }
      const handle = await network.addListener?.('networkStatusChange', (status) => {
        this.phoneOnline = status.connected
        this.connectionType = status.connectionType ?? this.connectionType
        this.scheduleNotify()
      })
      this.addNativeCleanup(handle)
    } catch (err) {
      reportListenerError(err)
    }
  }

  private async initNativeAppLifecycle(): Promise<void> {
    const app = capacitorAppPlugin()
    if (!app) return
    try {
      const handle = await app.addListener?.('appStateChange', ({ isActive }) => {
        if (isActive) this.onResume()
        else this.onBackground()
      })
      this.addNativeCleanup(handle)
    } catch (err) {
      reportListenerError(err)
    }
  }

  private initHeartbeat(): void {
    this.lastHeartbeat = Date.now()
    this.heartbeatTimer = setInterval(() => {
      const now = Date.now()
      const drift = now - this.lastHeartbeat - HEARTBEAT_INTERVAL_MS
      this.lastHeartbeat = now
      if (drift <= FREEZE_THRESHOLD_MS) return
      this.jsFrozenRecovery = true
      this.scheduleNotify()
    }, HEARTBEAT_INTERVAL_MS)
  }

  private onBackground(): void {
    this.backgroundTimestamp = Date.now()
    this.appActive = false
    this.scheduleNotify()
  }

  private onResume(): void {
    const now = Date.now()
    if (now - this.lastResumeAt < 300) return
    this.lastResumeAt = now
    const duration = this.backgroundTimestamp ? now - this.backgroundTimestamp : 0
    this.backgroundTimestamp = null
    this.appActive = true
    this.resumeType = classifyResume(duration)
    this.resumeDuration = duration
    this.scheduleNotify()
  }

  private scheduleNotify(): void {
    if (this.notifyTimer) return
    this.notifyTimer = setTimeout(() => {
      this.notifyTimer = null
      this.notify()
    }, NOTIFY_DEBOUNCE_MS)
  }

  private notify(): void {
    const prev = this.lastSnapshot
    const curr = this.createSnapshot()
    this.lastSnapshot = curr
    for (const listener of this.listeners) {
      try {
        listener(curr, prev)
      } catch (err) {
        reportListenerError(err)
      }
    }
    for (const listener of this.snapshotListeners) {
      try {
        listener()
      } catch (err) {
        reportListenerError(err)
      }
    }

    let shouldRenotify = false
    if (this.jsFrozenRecovery) {
      this.jsFrozenRecovery = false
      shouldRenotify = true
    }
    if (this.resumeType) {
      this.resumeType = null
      this.resumeDuration = 0
    }
    if (shouldRenotify) this.scheduleNotify()
  }

  private createSnapshot(): RemoteNetworkState {
    const nativeLifecycleOwnsRecovery = this.nativeNetworkPlugin !== undefined
    return {
      phoneOnline: this.phoneOnline,
      connectionType: this.connectionType,
      appActive: this.appActive,
      jsFrozenRecovery: this.jsFrozenRecovery,
      // The native transport remains alive while the WebView is backgrounded. appActive is
      // informational and must not tear down or block an otherwise healthy terminal session.
      networkReady: this.phoneOnline && (nativeLifecycleOwnsRecovery || !this.jsFrozenRecovery),
      resumeType: this.resumeType,
      resumeDuration: this.resumeDuration,
    }
  }

  private addNativeCleanup(handle: NativePluginListenerHandle | null | undefined): void {
    if (!handle?.remove) return
    const cleanup = () => {
      void Promise.resolve(handle.remove()).catch(reportListenerError)
    }
    if (!this.initialized) {
      cleanup()
      return
    }
    this.cleanup.push(cleanup)
  }
}

function classifyResume(duration: number): RemoteResumeType {
  if (duration < QUICK_RESUME_THRESHOLD_MS) return 'quick'
  if (duration < NORMAL_RESUME_THRESHOLD_MS) return 'normal'
  return 'cold'
}

function networkConnectionType(): string {
  const connection = browserNetworkConnection()
  return connection?.effectiveType ?? connection?.type ?? ''
}

function browserNetworkConnection(): (EventTarget & { effectiveType?: string; type?: string }) | undefined {
  return (browserNavigator() as (Navigator & { connection?: EventTarget & { effectiveType?: string; type?: string } }) | undefined)?.connection
}

function browserWindow(): Window | undefined {
  return typeof window === 'undefined' ? undefined : window
}

function browserDocument(): Document | undefined {
  return typeof document === 'undefined' ? undefined : document
}

function browserNavigator(): Navigator | undefined {
  return typeof navigator === 'undefined' ? undefined : navigator
}

export interface NativePluginListenerHandle {
  remove(): void | Promise<void>
}

export interface NativeNetworkStatus {
  connected: boolean
  connectionType?: string | undefined
}

interface NativeAppPlugin {
  addListener?: ((
    eventName: 'appStateChange',
    handler: (state: { isActive: boolean }) => void,
  ) => NativePluginListenerHandle | Promise<NativePluginListenerHandle>) | undefined
}

function capacitorNetworkPlugin(): NativeNetworkStatusPlugin | undefined {
  return capacitorPlugin('Network') as NativeNetworkStatusPlugin | undefined
}

function capacitorAppPlugin(): NativeAppPlugin | undefined {
  return capacitorPlugin('App') as NativeAppPlugin | undefined
}

function capacitorPlugin(name: string): unknown {
  const cap = (globalThis as { Capacitor?: { Plugins?: Record<string, unknown> } }).Capacitor
  return cap?.Plugins?.[name]
}

function reportListenerError(error: unknown): void {
  if (typeof console === 'undefined' || typeof console.error !== 'function') return
  console.error(error)
}
