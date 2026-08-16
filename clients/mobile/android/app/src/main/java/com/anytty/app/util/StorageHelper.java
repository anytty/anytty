package com.anytty.app.util;

import android.content.Context;
import android.content.SharedPreferences;

public class StorageHelper {

    private static final String CAPACITOR_PREFS = "CapacitorStorage";
    private static final String NATIVE_PREFS = "AnyTTYNativePrefs";

    // ─── Capacitor Preferences (written by JS side) ───────────────────────────

    public static String getString(Context context, String key) {
        return context.getSharedPreferences(CAPACITOR_PREFS, Context.MODE_PRIVATE)
                .getString(key, null);
    }

    // ─── Native Preferences ───────────────────────────────────────────────────

    public static String getNativeString(Context context, String key) {
        return context.getSharedPreferences(NATIVE_PREFS, Context.MODE_PRIVATE)
                .getString(key, null);
    }

    public static void putNativeString(Context context, String key, String value) {
        context.getSharedPreferences(NATIVE_PREFS, Context.MODE_PRIVATE)
                .edit().putString(key, value).apply();
    }

    public static boolean getNativeBoolean(Context context, String key, boolean defaultValue) {
        return context.getSharedPreferences(NATIVE_PREFS, Context.MODE_PRIVATE)
                .getBoolean(key, defaultValue);
    }

    public static void putNativeBoolean(Context context, String key, boolean value) {
        context.getSharedPreferences(NATIVE_PREFS, Context.MODE_PRIVATE)
                .edit().putBoolean(key, value).apply();
    }
}
