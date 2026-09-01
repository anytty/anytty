package com.anytty.app;

/** Token fence for the evaluateJavascript renderer watchdog fallback. */
final class NativeRendererJavascriptWatchdogState {
    static final long NO_PROBE = 0L;

    private boolean foreground;
    private long nextProbeId;
    private long activeProbeId;

    void enterForeground() {
        foreground = true;
    }

    void leaveForeground() {
        foreground = false;
        activeProbeId = NO_PROBE;
    }

    long beginProbe() {
        if (!foreground || activeProbeId != NO_PROBE) return NO_PROBE;
        if (nextProbeId == Long.MAX_VALUE) {
            throw new IllegalStateException("renderer JavaScript watchdog probe ID is exhausted");
        }
        activeProbeId = ++nextProbeId;
        return activeProbeId;
    }

    boolean completeProbe(long probeId) {
        return finishProbe(probeId);
    }

    boolean timeoutProbe(long probeId) {
        return finishProbe(probeId);
    }

    private boolean finishProbe(long probeId) {
        if (!foreground || probeId == NO_PROBE || activeProbeId != probeId) return false;
        activeProbeId = NO_PROBE;
        return true;
    }
}
