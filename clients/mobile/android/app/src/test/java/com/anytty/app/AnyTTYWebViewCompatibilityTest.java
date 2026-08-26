package com.anytty.app;

import static org.junit.Assert.assertEquals;
import static org.junit.Assert.assertFalse;
import static org.junit.Assert.assertTrue;

import android.content.pm.PackageInfo;
import org.junit.Test;

public class AnyTTYWebViewCompatibilityTest {
    @Test
    public void acceptsTheSupportedBaselineAndNewerVersions() {
        assertTrue(AnyTTYWebViewCompatibility.isSupported(packageInfo("101.0.4951.61")));
        assertTrue(AnyTTYWebViewCompatibility.isSupported(packageInfo("120.0.0.0")));
    }

    @Test
    public void rejectsOlderMissingAndMalformedVersions() {
        assertFalse(AnyTTYWebViewCompatibility.isSupported(packageInfo("100.0.4896.60")));
        assertFalse(AnyTTYWebViewCompatibility.isSupported(null));
        assertEquals(-1, AnyTTYWebViewCompatibility.majorVersion("not-a-version"));
        assertEquals(-1, AnyTTYWebViewCompatibility.majorVersion(""));
    }

    private static PackageInfo packageInfo(String versionName) {
        PackageInfo info = new PackageInfo();
        info.versionName = versionName;
        return info;
    }
}
