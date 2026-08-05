package com.anytty.app;

import android.webkit.WebResourceRequest;
import android.webkit.WebView;

import com.getcapacitor.Bridge;
import com.getcapacitor.BridgeWebViewClient;

public final class AnyTTYWebViewClient extends BridgeWebViewClient {
    public AnyTTYWebViewClient(Bridge bridge) {
        super(bridge);
    }

    @Override
    public boolean shouldOverrideUrlLoading(WebView view, WebResourceRequest request) {
        return !request.isForMainFrame() || !AnyTTYLocalUrl.isCanonical(request.getUrl().toString());
    }

    @SuppressWarnings("deprecation")
    @Override
    public boolean shouldOverrideUrlLoading(WebView view, String url) {
        return !AnyTTYLocalUrl.isCanonical(url);
    }
}
