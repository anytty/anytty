import { useCallback, useEffect, useRef, useState } from 'react'
import {
  connectionSnapshotFromStatus,
  connectionStatusIsSettled,
} from '../connection/connectionState'
import type { RtcConnectionPhase } from '../core/transport'

export interface MachineNetworkStatusState {
  connectionStatus: string | null
  connectionPhase: RtcConnectionPhase | null
  showMachineNetworkOverlay: boolean
  showDelayedMachineNetworkOverlay: boolean
  setMachineNetworkMachineId(machineId: string | null): void
  updateConnectionStatus(status: string, phase?: RtcConnectionPhase | undefined): void
  clearConnectionStatus(): void
  clearConnectionStatusSoon(): void
}

export function useMachineNetworkStatus(): MachineNetworkStatusState {
  const [connectionStatus, setConnectionStatus] = useState<string | null>(null)
  const [connectionPhase, setConnectionPhase] = useState<RtcConnectionPhase | null>(null)
  const [showDelayedMachineNetworkOverlay, setShowDelayedMachineNetworkOverlay] = useState(false)
  const machineIdRef = useRef<string | null>(null)
  const clearTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null)

  const clearTimer = useCallback(() => {
    if (!clearTimerRef.current) return
    clearTimeout(clearTimerRef.current)
    clearTimerRef.current = null
  }, [])

  const setMachineNetworkMachineId = useCallback((machineId: string | null) => {
    machineIdRef.current = machineId
  }, [])

  const clearConnectionStatus = useCallback(() => {
    clearTimer()
    setConnectionStatus(null)
    setConnectionPhase(null)
  }, [clearTimer])

  const clearConnectionStatusSoon = useCallback(() => {
    clearTimer()
    clearTimerRef.current = setTimeout(() => {
      clearTimerRef.current = null
      setConnectionStatus(null)
      setConnectionPhase(null)
    }, 1200)
  }, [clearTimer])

  const updateConnectionStatus = useCallback((status: string, phase?: RtcConnectionPhase | undefined) => {
    const snapshot = connectionSnapshotFromStatus({
      machineId: machineIdRef.current ?? 'machine',
      statusText: status,
      ...(phase ? { phase } : {}),
    })
    clearTimer()
    setConnectionStatus(snapshot.statusText)
    setConnectionPhase(snapshot.phase)
  }, [clearTimer])

  useEffect(() => () => {
    clearTimer()
  }, [clearTimer])

  const showMachineNetworkOverlay = Boolean(connectionStatus && !connectionStatusIsSettled(connectionPhase))

  useEffect(() => {
    if (showMachineNetworkOverlay) {
      const timer = setTimeout(() => {
        setShowDelayedMachineNetworkOverlay(true)
      }, 300)
      return () => clearTimeout(timer)
    } else {
      setShowDelayedMachineNetworkOverlay(false)
    }
  }, [showMachineNetworkOverlay])

  return {
    connectionStatus,
    connectionPhase,
    showMachineNetworkOverlay,
    showDelayedMachineNetworkOverlay,
    setMachineNetworkMachineId,
    updateConnectionStatus,
    clearConnectionStatus,
    clearConnectionStatusSoon,
  }
}
