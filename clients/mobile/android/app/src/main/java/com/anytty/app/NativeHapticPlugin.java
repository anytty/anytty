package com.anytty.app;

import android.app.Activity;
import android.os.Build;
import android.os.VibrationEffect;
import android.os.Vibrator;
import android.os.VibratorManager;
import android.content.Context;
import android.view.HapticFeedbackConstants;
import android.view.View;

import com.getcapacitor.Plugin;
import com.getcapacitor.PluginCall;
import com.getcapacitor.PluginMethod;
import com.getcapacitor.annotation.CapacitorPlugin;

import org.json.JSONArray;
import org.json.JSONException;

@CapacitorPlugin(name = "NativeHaptic")
public class NativeHapticPlugin extends Plugin {

    @PluginMethod()
    public void impact(PluginCall call) {
        Activity activity = getActivity();
        if (activity == null) {
            call.resolve();
            return;
        }
        Object pattern = call.getData().opt("pattern");
        activity.runOnUiThread(() -> {
            View view = getBridge().getWebView();
            if (view != null) {
                performViewFeedback(view, pattern);
            } else {
                vibratePattern(pattern);
            }
            call.resolve();
        });
    }

    private void performViewFeedback(View view, Object pattern) {
        int constant = HapticFeedbackConstants.KEYBOARD_TAP;
        if (isErrorPattern(pattern) && Build.VERSION.SDK_INT >= Build.VERSION_CODES.R) {
            constant = HapticFeedbackConstants.REJECT;
        } else if (isErrorPattern(pattern)) {
            constant = HapticFeedbackConstants.LONG_PRESS;
        } else if (isLongPattern(pattern) && Build.VERSION.SDK_INT >= Build.VERSION_CODES.R) {
            constant = HapticFeedbackConstants.CONFIRM;
        } else if (isLongPattern(pattern)) {
            constant = HapticFeedbackConstants.LONG_PRESS;
        } else if (isSelectionPattern(pattern)) {
            constant = HapticFeedbackConstants.CLOCK_TICK;
        }
        boolean handled = view.performHapticFeedback(
                constant,
                HapticFeedbackConstants.FLAG_IGNORE_GLOBAL_SETTING
        );
        if (!handled) {
            vibratePattern(pattern);
        }
    }

    private boolean vibratePattern(Object pattern) {
        Vibrator vibrator = vibrator();
        if (vibrator == null || !vibrator.hasVibrator()) return false;
        try {
            if (pattern instanceof JSONArray) {
                JSONArray values = (JSONArray) pattern;
                long[] timings = new long[values.length() + 1];
                timings[0] = 0;
                for (int i = 0; i < values.length(); i += 1) {
                    timings[i + 1] = Math.max(0, values.getLong(i));
                }
                vibrate(vibrator, timings);
                return true;
            }
            if (pattern instanceof Number) {
                vibrate(vibrator, Math.max(1, ((Number) pattern).longValue()));
                return true;
            }
        } catch (JSONException ignored) {
        }
        return false;
    }

    private Vibrator vibrator() {
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.S) {
            VibratorManager manager = (VibratorManager) getContext().getSystemService(Context.VIBRATOR_MANAGER_SERVICE);
            return manager != null ? manager.getDefaultVibrator() : null;
        }
        return (Vibrator) getContext().getSystemService(Context.VIBRATOR_SERVICE);
    }

    private void vibrate(Vibrator vibrator, long durationMs) {
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.O) {
            vibrator.vibrate(VibrationEffect.createOneShot(durationMs, VibrationEffect.DEFAULT_AMPLITUDE));
        } else {
            vibrator.vibrate(durationMs);
        }
    }

    private void vibrate(Vibrator vibrator, long[] timings) {
        if (timings.length == 0) return;
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.O) {
            vibrator.vibrate(VibrationEffect.createWaveform(timings, -1));
        } else {
            vibrator.vibrate(timings, -1);
        }
    }

    private boolean isSelectionPattern(Object pattern) {
        return pattern instanceof Number && ((Number) pattern).longValue() <= 8;
    }

    private boolean isLongPattern(Object pattern) {
        return pattern instanceof Number && ((Number) pattern).longValue() >= 20;
    }

    private boolean isErrorPattern(Object pattern) {
        return pattern instanceof JSONArray;
    }
}
