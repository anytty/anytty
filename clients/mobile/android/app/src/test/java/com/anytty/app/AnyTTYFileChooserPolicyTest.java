package com.anytty.app;

import static org.junit.Assert.assertFalse;
import static org.junit.Assert.assertTrue;

import org.junit.Test;

public class AnyTTYFileChooserPolicyTest {
    @Test
    public void allowsOnlySingleImageSelectionFromCanonicalAppPage() {
        assertTrue(AnyTTYFileChooserPolicy.allowsSingleImage(
            "http://localhost/device", new String[] { "image/*" }, 0, 0
        ));
        assertTrue(AnyTTYFileChooserPolicy.allowsSingleImage(
            "http://localhost", new String[] { "image/png", "image/jpeg" }, 0, 0
        ));

        assertFalse(AnyTTYFileChooserPolicy.allowsSingleImage("https://localhost/", new String[] { "image/*" }, 0, 0));
        assertFalse(AnyTTYFileChooserPolicy.allowsSingleImage("http://localhost", new String[] { "*/*" }, 0, 0));
        assertFalse(AnyTTYFileChooserPolicy.allowsSingleImage("http://localhost", new String[] { "image/*", "video/*" }, 0, 0));
        assertFalse(AnyTTYFileChooserPolicy.allowsSingleImage("http://localhost", new String[] { "image/*" }, 1, 0));
    }
}
