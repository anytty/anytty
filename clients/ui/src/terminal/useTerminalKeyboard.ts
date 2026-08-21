import { useCallback, useEffect, useRef, useState } from 'react'
import { addNativeKeyboardListener } from '../platform/nativeKeyboard'
import type { TerminalHandle } from './Terminal'

const keyboardVisibleThresholdPx = 100
const keyboardClosedThresholdPx = 1
const keyboardVisibilityDebounceMs = 120
const focusFallbackDelayMs = 300

export interface UseTerminalKeyboardOptions {
  containerRef: React.RefObject<HTMLDivElement | null>
  mainRef: React.RefObject<HTMLDivElement | null>
  termWrapperRef: React.RefObject<HTMLDivElement | null>
  getTermRef: () => TerminalHandle | null
  getLayoutMode: () => TerminalKeyboardLayoutMode
  onKeyboardHide?: () => void
}

export type TerminalKeyboardLayoutMode = 'resize' | 'shift'

export interface UseTerminalKeyboardReturn {
  keyboardVisible: boolean
  fullMainHeightRef: React.MutableRefObject<number>
  markKeyboardVisible: () => void
  markKeyboardHidden: () => void
  reapplyKeyboardLayout: () => void
  resetKeyboardLayout: () => void
  handleBufferChange: (isAlternate: boolean) => void
  handleCursorMove: () => void
}

interface KeyboardMeasurement {
  keyboardHeight: number
  visibleHeight: number
  hasNativeHeight: boolean
}

export function useTerminalKeyboard(opts: UseTerminalKeyboardOptions): UseTerminalKeyboardReturn {
  const { containerRef, mainRef, termWrapperRef, getTermRef, getLayoutMode, onKeyboardHide } = opts
  const [keyboardVisible, setKeyboardVisible] = useState(false)
  const fullMainHeightRef = useRef(0)
  const fullWindowHeightRef = useRef(typeof window === 'undefined' ? 0 : window.innerHeight)
  const nativeKeyboardHeightRef = useRef(0)
  const keyboardRequestedRef = useRef(false)
  const shiftRafRef = useRef(0)
  const onKeyboardHideRef = useRef(onKeyboardHide)
  const getLayoutModeRef = useRef(getLayoutMode)

  onKeyboardHideRef.current = onKeyboardHide
  getLayoutModeRef.current = getLayoutMode

  const measureKeyboard = useCallback((): KeyboardMeasurement => {
    const vv = window.visualViewport
    if (window.innerHeight > fullWindowHeightRef.current) {
      fullWindowHeightRef.current = window.innerHeight
    }

    const visualViewportKeyboardHeight = vv ? Math.max(0, window.innerHeight - vv.height) : 0
    const windowShrinkKeyboardHeight = Math.max(0, fullWindowHeightRef.current - window.innerHeight)
    const nativeKeyboardHeight = nativeKeyboardHeightRef.current
    const keyboardHeight = Math.max(visualViewportKeyboardHeight, windowShrinkKeyboardHeight, nativeKeyboardHeight)
    const hasNativeHeight = nativeKeyboardHeight > keyboardClosedThresholdPx

    let visibleHeight = vv?.height ?? window.innerHeight
    if (hasNativeHeight && windowShrinkKeyboardHeight <= keyboardClosedThresholdPx) {
      visibleHeight = Math.max(0, window.innerHeight - nativeKeyboardHeight)
    }

    return { keyboardHeight, visibleHeight, hasNativeHeight }
  }, [])

  const rememberFullMainHeight = useCallback(() => {
    const main = mainRef.current
    if (!main) return
    fullMainHeightRef.current = main.clientHeight
  }, [mainRef])

  const clearKeyboardLayout = useCallback(() => {
    if (shiftRafRef.current) {
      cancelAnimationFrame(shiftRafRef.current)
      shiftRafRef.current = 0
    }
    if (containerRef.current) containerRef.current.style.height = ''
    if (termWrapperRef.current) {
      termWrapperRef.current.style.height = ''
      termWrapperRef.current.style.transform = ''
    }
    getTermRef()?.adjustInputPosition(0)
  }, [containerRef, getTermRef, termWrapperRef])

  const resetKeyboardLayout = useCallback(() => {
    nativeKeyboardHeightRef.current = 0
    keyboardRequestedRef.current = false
    clearKeyboardLayout()
    setKeyboardVisible(false)
    onKeyboardHideRef.current?.()
  }, [clearKeyboardLayout])

  const markKeyboardVisible = useCallback(() => {
    keyboardRequestedRef.current = true
    setKeyboardVisible(true)
  }, [])

  const markKeyboardHidden = useCallback(() => {
    resetKeyboardLayout()
  }, [resetKeyboardLayout])

  const adjustShift = useCallback(() => {
    if (getLayoutModeRef.current() === 'resize') return
    const wrapper = termWrapperRef.current
    if (!wrapper || fullMainHeightRef.current <= 0) return

    const { keyboardHeight } = measureKeyboard()
    if (keyboardHeight <= keyboardClosedThresholdPx) return

    const term = getTermRef()
    const cursorInfo = term?.getCursorInfo()
    let shift = 0
    if (cursorInfo) {
      const cursorPxFromTop = cursorInfo.cursorY * cursorInfo.lineHeight
      const visibleMainHeight = fullMainHeightRef.current - keyboardHeight
      if (cursorPxFromTop >= visibleMainHeight) {
        shift = Math.min(
          keyboardHeight,
          cursorPxFromTop - visibleMainHeight + cursorInfo.lineHeight * 2,
        )
      }
    }

    wrapper.style.transform = `translateY(${-shift}px)`
    term?.adjustInputPosition(keyboardHeight - shift)
  }, [getTermRef, measureKeyboard, termWrapperRef])

  const applyKeyboardLayout = useCallback(() => {
    const wrapper = termWrapperRef.current
    const container = containerRef.current
    if (!wrapper || !container) return

    if (fullMainHeightRef.current <= 0) rememberFullMainHeight()
    const { keyboardHeight, visibleHeight } = measureKeyboard()
    const keyboardIsOpen = keyboardHeight > keyboardClosedThresholdPx

    if (!keyboardIsOpen) {
      if (!keyboardRequestedRef.current) resetKeyboardLayout()
      else clearKeyboardLayout()
      return
    }

    if (keyboardHeight > keyboardVisibleThresholdPx) {
      container.style.height = `${visibleHeight}px`
    } else {
      container.style.height = ''
    }

    if (getLayoutModeRef.current() === 'resize') {
      wrapper.style.height = ''
      wrapper.style.transform = ''
      getTermRef()?.adjustInputPosition(0)
    } else {
      wrapper.style.height = `${fullMainHeightRef.current}px`
      adjustShift()
    }
  }, [
    adjustShift,
    clearKeyboardLayout,
    containerRef,
    getTermRef,
    measureKeyboard,
    rememberFullMainHeight,
    resetKeyboardLayout,
    termWrapperRef,
  ])

  const reapplyKeyboardLayout = useCallback(() => {
    applyKeyboardLayout()
  }, [applyKeyboardLayout])

  const handleBufferChange = useCallback((_isAlternate: boolean) => {
    applyKeyboardLayout()
  }, [applyKeyboardLayout])

  const handleCursorMove = useCallback(() => {
    if (shiftRafRef.current) return
    shiftRafRef.current = requestAnimationFrame(() => {
      shiftRafRef.current = 0
      adjustShift()
    })
  }, [adjustShift])

  useEffect(() => {
    const vv = window.visualViewport
    fullWindowHeightRef.current = window.innerHeight
    let visibilityDebounce: ReturnType<typeof setTimeout> | undefined
    let fallbackTimer: ReturnType<typeof setTimeout> | undefined
    let scrollRafId = 0

    const publishVisibility = (isVisible: boolean) => {
      clearTimeout(visibilityDebounce)
      if (isVisible) {
        setKeyboardVisible(true)
        return
      }
      visibilityDebounce = setTimeout(() => {
        if (!keyboardRequestedRef.current) setKeyboardVisible(false)
      }, keyboardVisibilityDebounceMs)
    }

    const update = () => {
      const { keyboardHeight } = measureKeyboard()
      if (keyboardHeight <= keyboardClosedThresholdPx || fullMainHeightRef.current <= 0) {
        rememberFullMainHeight()
      }
      applyKeyboardLayout()
      publishVisibility(keyboardHeight > keyboardVisibleThresholdPx || keyboardRequestedRef.current)
    }

    const preventPageScroll = () => {
      if (scrollRafId) return
      scrollRafId = requestAnimationFrame(() => {
        scrollRafId = 0
        window.scrollTo(0, 0)
        document.documentElement.scrollTop = 0
        document.body.scrollTop = 0
        document.documentElement.scrollLeft = 0
        document.body.scrollLeft = 0
      })
    }

    const updateSoonAfterFocusChange = () => {
      clearTimeout(fallbackTimer)
      fallbackTimer = setTimeout(update, focusFallbackDelayMs)
    }

    const onTerminalFocusIn = (event: FocusEvent) => {
      if (!termWrapperRef.current?.contains(event.target as Node)) return
      keyboardRequestedRef.current = true
      publishVisibility(true)
      updateSoonAfterFocusChange()
    }

    const onTerminalFocusOut = () => {
      setTimeout(() => {
        if (termWrapperRef.current?.contains(document.activeElement)) return
        keyboardRequestedRef.current = false
        const { keyboardHeight } = measureKeyboard()
        if (keyboardHeight <= keyboardClosedThresholdPx) resetKeyboardLayout()
      }, 0)
    }

    const handleWindowResize = () => {
      if (window.innerHeight > fullWindowHeightRef.current) {
        fullWindowHeightRef.current = window.innerHeight
      }
      update()
    }

    const removeNativeKeyboardListener = addNativeKeyboardListener((event) => {
      nativeKeyboardHeightRef.current = event.visible ? event.keyboardHeight ?? measureKeyboard().keyboardHeight : 0
      keyboardRequestedRef.current = event.visible
      if (event.visible) {
        publishVisibility(true)
        update()
        return
      }
      resetKeyboardLayout()
    })

    const onResume = () => {
      resetKeyboardLayout()
      window.setTimeout(update, keyboardVisibilityDebounceMs)
    }

    update()
    vv?.addEventListener('resize', update)
    vv?.addEventListener('scroll', update)
    window.addEventListener('resize', handleWindowResize)
    window.addEventListener('scroll', preventPageScroll, { passive: false })
    document.addEventListener('focusin', updateSoonAfterFocusChange)
    document.addEventListener('focusout', updateSoonAfterFocusChange)
    document.addEventListener('focusin', onTerminalFocusIn)
    document.addEventListener('focusout', onTerminalFocusOut)
    document.addEventListener('visibilitychange', onResume)
    document.addEventListener('anytty:resume', onResume)

    return () => {
      removeNativeKeyboardListener()
      vv?.removeEventListener('resize', update)
      vv?.removeEventListener('scroll', update)
      window.removeEventListener('resize', handleWindowResize)
      window.removeEventListener('scroll', preventPageScroll)
      document.removeEventListener('focusin', updateSoonAfterFocusChange)
      document.removeEventListener('focusout', updateSoonAfterFocusChange)
      document.removeEventListener('focusin', onTerminalFocusIn)
      document.removeEventListener('focusout', onTerminalFocusOut)
      document.removeEventListener('visibilitychange', onResume)
      document.removeEventListener('anytty:resume', onResume)
      clearTimeout(visibilityDebounce)
      clearTimeout(fallbackTimer)
      if (scrollRafId) cancelAnimationFrame(scrollRafId)
      if (shiftRafRef.current) cancelAnimationFrame(shiftRafRef.current)
    }
  }, [applyKeyboardLayout, measureKeyboard, rememberFullMainHeight, resetKeyboardLayout, termWrapperRef])

  return {
    keyboardVisible,
    fullMainHeightRef,
    markKeyboardVisible,
    markKeyboardHidden,
    reapplyKeyboardLayout,
    resetKeyboardLayout,
    handleBufferChange,
    handleCursorMove,
  }
}
