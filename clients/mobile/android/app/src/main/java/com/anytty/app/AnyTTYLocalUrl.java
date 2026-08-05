package com.anytty.app;

import java.net.URI;
import java.net.URISyntaxException;

final class AnyTTYLocalUrl {
    private AnyTTYLocalUrl() {}

    static boolean isCanonical(String value) {
        if (value == null || value.indexOf('\\') >= 0) return false;
        try {
            URI uri = new URI(value);
            return "http".equals(uri.getScheme()) &&
                "localhost".equals(uri.getRawAuthority()) &&
                "localhost".equals(uri.getHost()) &&
                uri.getPort() == -1 &&
                uri.getUserInfo() == null &&
                uri.normalize().toASCIIString().equals(value);
        } catch (URISyntaxException error) {
            return false;
        }
    }
}
