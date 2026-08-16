package com.anytty.app;

import static org.junit.Assert.assertFalse;
import static org.junit.Assert.assertTrue;

import org.junit.Test;

public class AnyTTYLocalUrlTest {
    @Test
    public void acceptsOnlyCanonicalLocalHttpUrls() {
        assertTrue(AnyTTYLocalUrl.isCanonical("http://localhost"));
        assertTrue(AnyTTYLocalUrl.isCanonical("http://localhost/scan?mode=qr#camera"));

        assertFalse(AnyTTYLocalUrl.isCanonical("https://localhost/"));
        assertFalse(AnyTTYLocalUrl.isCanonical("http://localhost:80/"));
        assertFalse(AnyTTYLocalUrl.isCanonical("http://LOCALHOST/"));
        assertFalse(AnyTTYLocalUrl.isCanonical("http://localhost./"));
        assertFalse(AnyTTYLocalUrl.isCanonical("http://localhost@evil.test/"));
        assertFalse(AnyTTYLocalUrl.isCanonical("http://localhost/a/../b"));
        assertFalse(AnyTTYLocalUrl.isCanonical("http://localhost\\evil.test/"));
        assertFalse(AnyTTYLocalUrl.isCanonical("file:///android_asset/index.html"));
        assertFalse(AnyTTYLocalUrl.isCanonical("content://localhost/item"));
    }
}
