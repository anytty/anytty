package com.anytty.app;

import android.content.pm.PackageInfo;
import android.os.Bundle;
import android.os.Build;
import android.os.Handler;
import android.os.Looper;
import android.view.ViewGroup;
import android.widget.Button;
import android.widget.TextView;
import android.webkit.WebSettings;
import android.webkit.WebView;

import androidx.webkit.WebViewCompat;
import androidx.webkit.WebViewFeature;
import androidx.webkit.WebViewRenderProcess;
import androidx.webkit.WebViewRenderProcessClient;

import com.getcapacitor.BridgeActivity;

public class MainActivity extends BridgeActivity {
    private static final long RENDERER_UNRESPONSIVE_GRACE_MS = 10_000L;

    private final Handler mainHandler = new Handler(Looper.getMainLooper());
    private boolean resumed;
    private boolean rendererRecoveryScheduled;
    private WebViewRenderProcess unresponsiveRenderer;
    private final Runnable recoverUnresponsiveRenderer = () -> {
        if (!resumed || rendererRecoveryScheduled || unresponsiveRenderer == null) return;
        AnyTTYDebugLog.event(AnyTTYDebugEvent.WEBVIEW_RENDERER_UNRESPONSIVE);
        if (
            Build.VERSION.SDK_INT >= Build.VERSION_CODES.O &&
            WebViewFeature.isFeatureSupported(WebViewFeature.WEB_VIEW_RENDERER_TERMINATE) &&
            unresponsiveRenderer.terminate()
        ) {
            return;
        }
        scheduleRendererRecovery(getBridge().getWebView());
    };

    @Override
    protected void onCreate(Bundle savedInstanceState) {
        AnyTTYDebugLog.init(this);
        PackageInfo webViewPackage = WebViewCompat.getCurrentWebViewPackage(this);
        if (!AnyTTYWebViewCompatibility.isSupported(webViewPackage)) {
            super.onCreate(savedInstanceState);
            showUnsupportedWebView(webViewPackage);
            return;
        }

        registerPlugin(NativeConnectionPlugin.class);
        registerPlugin(NativeFilePickerPlugin.class);
        registerPlugin(NativeHapticPlugin.class);
        super.onCreate(savedInstanceState);

        boolean isDebug = (getApplicationInfo().flags & android.content.pm.ApplicationInfo.FLAG_DEBUGGABLE) != 0;
        WebView.setWebContentsDebuggingEnabled(isDebug);

        WebView webView = getBridge().getWebView();
        if (webView != null) {
            webView.setWebChromeClient(new AnyTTYWebChromeClient(getBridge()));
            getBridge().setWebViewClient(new AnyTTYWebViewClient(getBridge(), this::handleRendererGone));
            webView.setOverScrollMode(WebView.OVER_SCROLL_NEVER);
            webView.setVerticalScrollBarEnabled(false);
            webView.setHorizontalScrollBarEnabled(false);
            WebSettings settings = webView.getSettings();
            settings.setDomStorageEnabled(true);
            settings.setCacheMode(WebSettings.LOAD_DEFAULT);
            settings.setMixedContentMode(WebSettings.MIXED_CONTENT_NEVER_ALLOW);
            settings.setAllowFileAccess(false);
            settings.setAllowContentAccess(false);
            settings.setAllowFileAccessFromFileURLs(false);
            settings.setAllowUniversalAccessFromFileURLs(false);
            settings.setGeolocationEnabled(false);
            installRendererResponsivenessClient(webView);
        }
        AnyTTYDebugLog.event(AnyTTYDebugEvent.ACTIVITY_CREATED, isDebug);
    }

    private void showUnsupportedWebView(PackageInfo webViewPackage) {
        int majorVersion = AnyTTYWebViewCompatibility.majorVersion(webViewPackage);
        AnyTTYDebugLog.event(AnyTTYDebugEvent.WEBVIEW_UNSUPPORTED, majorVersion);

        WebView webView = getBridge().getWebView();
        if (webView != null) {
            webView.stopLoading();
            webView.setVisibility(android.view.View.GONE);
        }
        setContentView(R.layout.activity_unsupported_webview);

        String installedVersion = webViewPackage == null || webViewPackage.versionName == null
            ? getString(R.string.webview_unknown_version)
            : webViewPackage.versionName;
        TextView message = findViewById(R.id.webview_unsupported_message);
        message.setText(getString(
            R.string.webview_unsupported_message,
            installedVersion,
            AnyTTYWebViewCompatibility.MINIMUM_MAJOR_VERSION
        ));
        Button retry = findViewById(R.id.webview_retry);
        retry.setOnClickListener(view -> recreate());
    }

    @Override
    public void onResume() {
        super.onResume();
        resumed = true;
        AnyTTYDebugLog.event(AnyTTYDebugEvent.ACTIVITY_RESUMED);
        if (unresponsiveRenderer != null) scheduleUnresponsiveRendererRecovery();
    }

    @Override
    public void onPause() {
        resumed = false;
        AnyTTYDebugLog.event(AnyTTYDebugEvent.ACTIVITY_PAUSED);
        mainHandler.removeCallbacks(recoverUnresponsiveRenderer);
        super.onPause();
    }

    @Override
    public void onDestroy() {
        resumed = false;
        AnyTTYDebugLog.event(AnyTTYDebugEvent.ACTIVITY_DESTROYED);
        mainHandler.removeCallbacks(recoverUnresponsiveRenderer);
        unresponsiveRenderer = null;
        super.onDestroy();
    }

    private void installRendererResponsivenessClient(WebView webView) {
        if (!WebViewFeature.isFeatureSupported(WebViewFeature.WEB_VIEW_RENDERER_CLIENT_BASIC_USAGE)) return;
        WebViewCompat.setWebViewRenderProcessClient(webView, new WebViewRenderProcessClient() {
            @Override
            public void onRenderProcessUnresponsive(WebView view, WebViewRenderProcess renderer) {
                unresponsiveRenderer = renderer;
                scheduleUnresponsiveRendererRecovery();
            }

            @Override
            public void onRenderProcessResponsive(WebView view, WebViewRenderProcess renderer) {
                if (unresponsiveRenderer != renderer) return;
                unresponsiveRenderer = null;
                mainHandler.removeCallbacks(recoverUnresponsiveRenderer);
            }
        });
    }

    private void scheduleUnresponsiveRendererRecovery() {
        mainHandler.removeCallbacks(recoverUnresponsiveRenderer);
        if (resumed) mainHandler.postDelayed(recoverUnresponsiveRenderer, RENDERER_UNRESPONSIVE_GRACE_MS);
    }

    private boolean handleRendererGone(WebView webView, android.webkit.RenderProcessGoneDetail detail) {
        AnyTTYDebugLog.event(AnyTTYDebugEvent.WEBVIEW_RENDERER_GONE, detail.didCrash());
        scheduleRendererRecovery(webView);
        return true;
    }

    private void scheduleRendererRecovery(WebView webView) {
        if (rendererRecoveryScheduled || isFinishing() || isDestroyed()) return;
        rendererRecoveryScheduled = true;
        mainHandler.removeCallbacks(recoverUnresponsiveRenderer);
        unresponsiveRenderer = null;
        if (webView != null) {
            if (webView.getParent() instanceof ViewGroup) {
                ((ViewGroup) webView.getParent()).removeView(webView);
            }
            webView.destroy();
        }
        mainHandler.post(this::recreate);
    }
}
