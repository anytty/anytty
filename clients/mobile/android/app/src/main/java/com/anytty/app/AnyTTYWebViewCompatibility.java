package com.anytty.app;

import android.content.pm.PackageInfo;

final class AnyTTYWebViewCompatibility {
    static final int MINIMUM_MAJOR_VERSION = 101;

    private AnyTTYWebViewCompatibility() {}

    static boolean isSupported(PackageInfo webViewPackage) {
        return majorVersion(webViewPackage) >= MINIMUM_MAJOR_VERSION;
    }

    static int majorVersion(PackageInfo webViewPackage) {
        return webViewPackage == null ? -1 : majorVersion(webViewPackage.versionName);
    }

    static int majorVersion(String versionName) {
        if (versionName == null || versionName.isEmpty()) return -1;
        int separator = versionName.indexOf('.');
        String major = separator < 0 ? versionName : versionName.substring(0, separator);
        try {
            return Integer.parseInt(major);
        } catch (NumberFormatException ignored) {
            return -1;
        }
    }
}
