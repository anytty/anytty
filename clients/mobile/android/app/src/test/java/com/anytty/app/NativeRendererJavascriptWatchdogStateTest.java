package com.anytty.app;

import static org.junit.Assert.assertEquals;
import static org.junit.Assert.assertFalse;
import static org.junit.Assert.assertNotEquals;
import static org.junit.Assert.assertTrue;

import org.junit.Test;

public class NativeRendererJavascriptWatchdogStateTest {
    @Test
    public void timelyCallbackCompletesTheSingleFlightProbe() {
        NativeRendererJavascriptWatchdogState state = new NativeRendererJavascriptWatchdogState();
        state.enterForeground();

        long probe = state.beginProbe();

        assertNotEquals(NativeRendererJavascriptWatchdogState.NO_PROBE, probe);
        assertEquals(NativeRendererJavascriptWatchdogState.NO_PROBE, state.beginProbe());
        assertTrue(state.completeProbe(probe));
        assertNotEquals(NativeRendererJavascriptWatchdogState.NO_PROBE, state.beginProbe());
    }

    @Test
    public void timeoutIsAcceptedExactlyOnce() {
        NativeRendererJavascriptWatchdogState state = new NativeRendererJavascriptWatchdogState();
        state.enterForeground();
        long probe = state.beginProbe();

        assertTrue(state.timeoutProbe(probe));
        assertFalse(state.timeoutProbe(probe));
        assertFalse(state.completeProbe(probe));
    }

    @Test
    public void backgroundCancelsTheOutstandingProbe() {
        NativeRendererJavascriptWatchdogState state = new NativeRendererJavascriptWatchdogState();
        state.enterForeground();
        long probe = state.beginProbe();

        state.leaveForeground();

        assertFalse(state.timeoutProbe(probe));
        assertFalse(state.completeProbe(probe));
        assertEquals(NativeRendererJavascriptWatchdogState.NO_PROBE, state.beginProbe());
    }

    @Test
    public void oldCallbackCannotCompleteANewerForegroundRound() {
        NativeRendererJavascriptWatchdogState state = new NativeRendererJavascriptWatchdogState();
        state.enterForeground();
        long oldProbe = state.beginProbe();
        state.leaveForeground();
        state.enterForeground();
        long currentProbe = state.beginProbe();

        assertNotEquals(oldProbe, currentProbe);
        assertFalse(state.completeProbe(oldProbe));
        assertEquals(NativeRendererJavascriptWatchdogState.NO_PROBE, state.beginProbe());
        assertTrue(state.completeProbe(currentProbe));
    }
}
