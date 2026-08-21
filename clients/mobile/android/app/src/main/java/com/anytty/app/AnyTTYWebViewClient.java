package com.anytty.app;

import android.webkit.WebResourceRequest;
import android.webkit.RenderProcessGoneDetail;
import android.webkit.WebView;

import com.getcapacitor.Bridge;
import com.getcapacitor.BridgeWebViewClient;

public final class AnyTTYWebViewClient extends BridgeWebViewClient {
    public interface RendererGoneHandler {
        boolean onRendererGone(WebView view, RenderProcessGoneDetail detail);
    }

    private final RendererGoneHandler rendererGoneHandler;

    public AnyTTYWebViewClient(Bridge bridge, RendererGoneHandler rendererGoneHandler) {
        super(bridge);
        this.rendererGoneHandler = rendererGoneHandler;
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

    @Override
    public boolean onRenderProcessGone(WebView view, RenderProcessGoneDetail detail) {
        return rendererGoneHandler.onRendererGone(view, detail);
    }
}
