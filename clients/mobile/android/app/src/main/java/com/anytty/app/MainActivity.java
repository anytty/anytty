package com.anytty.app;

import android.content.pm.PackageInfo;
import android.os.Bundle;
import android.os.Build;
import android.os.Handler;
import android.os.Looper;
import android.os.SystemClock;
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
    private static final long RENDERER_TERMINATION_FALLBACK_MS = 2_000L;
    private static final long RENDERER_JAVASCRIPT_PROBE_INTERVAL_MS = 2_000L;
    private static final long RENDERER_JAVASCRIPT_CALLBACK_TIMEOUT_MS = 10_000L;

    private final Handler mainHandler = new Handler(Looper.getMainLooper());
    private final NativeRendererJavascriptWatchdogState rendererJavascriptWatchdog =
        new NativeRendererJavascriptWatchdogState();
    private boolean resumed;
    private boolean rendererJavascriptWatchdogEnabled;
    private Runnable pendingRendererJavascriptTimeout;
    private boolean rendererRecoveryScheduled;
    private boolean rendererRecoveryPending;
    private boolean rendererUnresponsive;
    private boolean rendererTerminationPending;
    private long rendererUnresponsiveSince = -1L;
    private WebView unresponsiveWebView;
    private WebViewRenderProcess unresponsiveRenderer;
    private WebView pendingRecoveryWebView;
    private final Runnable runRendererJavascriptProbe = this::runRendererJavascriptProbe;
    private final Runnable recoverTerminatedRendererFallback = () -> {
        if (rendererRecoveryScheduled || !rendererTerminationPending) return;
        requestRendererRecovery(unresponsiveWebView != null ? unresponsiveWebView : getBridge().getWebView());
    };
    private final Runnable recoverUnresponsiveRenderer = () -> {
        if (!resumed || rendererRecoveryScheduled || !rendererUnresponsive) return;
        AnyTTYDebugLog.event(AnyTTYDebugEvent.WEBVIEW_RENDERER_UNRESPONSIVE);
        if (terminateRenderer(unresponsiveRenderer)) {
            rendererTerminationPending = true;
            mainHandler.postDelayed(recoverTerminatedRendererFallback, RENDERER_TERMINATION_FALLBACK_MS);
            return;
        }
        requestRendererRecovery(unresponsiveWebView != null ? unresponsiveWebView : getBridge().getWebView());
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
    public void onStart() {
        super.onStart();
        refreshNativeForeground();
    }

    @Override
    public void onResume() {
        super.onResume();
        resumed = true;
        refreshNativeForeground();
        AnyTTYDebugLog.event(AnyTTYDebugEvent.ACTIVITY_RESUMED);
        if (rendererRecoveryPending) {
            scheduleRendererRecovery(pendingRecoveryWebView);
        } else if (rendererUnresponsive) {
            scheduleUnresponsiveRendererRecovery();
        }
        startRendererJavascriptWatchdog();
    }

    @Override
    public void onPause() {
        resumed = false;
        AnyTTYDebugLog.event(AnyTTYDebugEvent.ACTIVITY_PAUSED);
        stopRendererJavascriptWatchdog();
        mainHandler.removeCallbacks(recoverUnresponsiveRenderer);
        super.onPause();
    }

    @Override
    public void onStop() {
        NativeConnectionRuntimeOwner.handleActivityBackground();
        super.onStop();
    }

    @Override
    public void onDestroy() {
        resumed = false;
        AnyTTYDebugLog.event(AnyTTYDebugEvent.ACTIVITY_DESTROYED);
        stopRendererJavascriptWatchdog();
        clearUnresponsiveTracking();
        rendererRecoveryPending = false;
        pendingRecoveryWebView = null;
        super.onDestroy();
    }

    private void refreshNativeForeground() {
        try {
            NativeConnectionRuntimeOwner.handleActivityForeground(getApplicationContext());
        } catch (Exception failure) {
            AnyTTYDebugLog.connection(
                "activity_foreground refresh_failed type=" + failure.getClass().getSimpleName()
            );
        }
    }

    private void installRendererResponsivenessClient(WebView webView) {
        // Some WebView builds advertise the renderer callback but do not report a pure JS
        // main-thread stall. Keep the active probe on every supported WebView as a backstop.
        rendererJavascriptWatchdogEnabled = true;
        if (WebViewFeature.isFeatureSupported(WebViewFeature.WEB_VIEW_RENDERER_CLIENT_BASIC_USAGE)) {
            WebViewCompat.setWebViewRenderProcessClient(webView, new WebViewRenderProcessClient() {
                @Override
                public void onRenderProcessUnresponsive(WebView view, WebViewRenderProcess renderer) {
                    if (rendererUnresponsive && unresponsiveWebView == view) {
                        if (unresponsiveRenderer == null) unresponsiveRenderer = renderer;
                        return;
                    }
                    rendererUnresponsive = true;
                    rendererUnresponsiveSince = SystemClock.elapsedRealtime();
                    unresponsiveWebView = view;
                    unresponsiveRenderer = renderer;
                    scheduleUnresponsiveRendererRecovery();
                }

                @Override
                public void onRenderProcessResponsive(WebView view, WebViewRenderProcess renderer) {
                    if (!rendererUnresponsive || unresponsiveWebView != view) return;
                    // A vendor WebView can report responsive after accepting terminate() but
                    // omit onRenderProcessGone. Keep the fallback rebuild armed in that case.
                    if (rendererTerminationPending) return;
                    clearUnresponsiveTracking();
                }
            });
        }
    }

    private void startRendererJavascriptWatchdog() {
        if (!rendererJavascriptWatchdogEnabled || !resumed || rendererRecoveryScheduled) return;
        rendererJavascriptWatchdog.enterForeground();
        mainHandler.removeCallbacks(runRendererJavascriptProbe);
        mainHandler.postDelayed(runRendererJavascriptProbe, RENDERER_JAVASCRIPT_PROBE_INTERVAL_MS);
    }

    private void stopRendererJavascriptWatchdog() {
        rendererJavascriptWatchdog.leaveForeground();
        mainHandler.removeCallbacks(runRendererJavascriptProbe);
        if (pendingRendererJavascriptTimeout != null) {
            mainHandler.removeCallbacks(pendingRendererJavascriptTimeout);
            pendingRendererJavascriptTimeout = null;
        }
    }

    private void runRendererJavascriptProbe() {
        if (!rendererJavascriptWatchdogEnabled || !resumed || rendererRecoveryScheduled) return;
        WebView webView = getBridge().getWebView();
        if (webView == null) return;
        long probeId = rendererJavascriptWatchdog.beginProbe();
        if (probeId == NativeRendererJavascriptWatchdogState.NO_PROBE) return;

        Runnable timeout = () -> handleRendererJavascriptTimeout(probeId, webView);
        pendingRendererJavascriptTimeout = timeout;
        mainHandler.postDelayed(timeout, RENDERER_JAVASCRIPT_CALLBACK_TIMEOUT_MS);
        try {
            webView.evaluateJavascript("void 0", ignored -> handleRendererJavascriptCallback(probeId));
        } catch (RuntimeException failure) {
            mainHandler.removeCallbacks(timeout);
            if (pendingRendererJavascriptTimeout == timeout) pendingRendererJavascriptTimeout = null;
            if (rendererJavascriptWatchdog.timeoutProbe(probeId)) recoverJavascriptRenderer(webView);
        }
    }

    private void handleRendererJavascriptCallback(long probeId) {
        if (!rendererJavascriptWatchdog.completeProbe(probeId)) return;
        if (pendingRendererJavascriptTimeout != null) {
            mainHandler.removeCallbacks(pendingRendererJavascriptTimeout);
            pendingRendererJavascriptTimeout = null;
        }
        startRendererJavascriptWatchdog();
    }

    private void handleRendererJavascriptTimeout(long probeId, WebView webView) {
        if (!rendererJavascriptWatchdog.timeoutProbe(probeId)) return;
        pendingRendererJavascriptTimeout = null;
        recoverJavascriptRenderer(webView);
    }

    private void recoverJavascriptRenderer(WebView webView) {
        AnyTTYDebugLog.event(AnyTTYDebugEvent.WEBVIEW_JAVASCRIPT_UNRESPONSIVE);
        WebViewRenderProcess renderer = null;
        try {
            if (WebViewFeature.isFeatureSupported(WebViewFeature.GET_WEB_VIEW_RENDERER)) {
                renderer = WebViewCompat.getWebViewRenderProcess(webView);
            }
        } catch (RuntimeException ignored) {
            // Fall through to rebuilding the WebView when a vendor implementation disagrees
            // with its advertised feature set.
        }
        rendererUnresponsive = true;
        rendererUnresponsiveSince = SystemClock.elapsedRealtime();
        unresponsiveWebView = webView;
        unresponsiveRenderer = renderer;
        if (terminateRenderer(renderer)) {
            rendererTerminationPending = true;
            mainHandler.postDelayed(recoverTerminatedRendererFallback, RENDERER_TERMINATION_FALLBACK_MS);
            return;
        }
        requestRendererRecovery(webView);
    }

    private boolean terminateRenderer(WebViewRenderProcess renderer) {
        if (
            renderer == null ||
            Build.VERSION.SDK_INT < Build.VERSION_CODES.O ||
            !WebViewFeature.isFeatureSupported(WebViewFeature.WEB_VIEW_RENDERER_TERMINATE)
        ) return false;
        try {
            return renderer.terminate();
        } catch (RuntimeException failure) {
            AnyTTYDebugLog.connection(
                "webview_renderer_terminate failed type=" + failure.getClass().getSimpleName()
            );
            return false;
        }
    }

    private void scheduleUnresponsiveRendererRecovery() {
        mainHandler.removeCallbacks(recoverUnresponsiveRenderer);
        if (!resumed) return;
        long elapsed = rendererUnresponsiveSince < 0L
            ? 0L
            : Math.max(0L, SystemClock.elapsedRealtime() - rendererUnresponsiveSince);
        mainHandler.postDelayed(
            recoverUnresponsiveRenderer,
            Math.max(0L, RENDERER_UNRESPONSIVE_GRACE_MS - elapsed)
        );
    }

    private boolean handleRendererGone(WebView webView, android.webkit.RenderProcessGoneDetail detail) {
        boolean didCrash = Build.VERSION.SDK_INT >= Build.VERSION_CODES.O && detail.didCrash();
        AnyTTYDebugLog.event(AnyTTYDebugEvent.WEBVIEW_RENDERER_GONE, didCrash);
        requestRendererRecovery(webView);
        return true;
    }

    private void requestRendererRecovery(WebView webView) {
        if (!resumed) {
            rendererRecoveryPending = true;
            pendingRecoveryWebView = webView;
            clearUnresponsiveTracking();
            return;
        }
        scheduleRendererRecovery(webView);
    }

    private void scheduleRendererRecovery(WebView webView) {
        if (rendererRecoveryScheduled || isFinishing() || isDestroyed()) return;
        rendererRecoveryScheduled = true;
        stopRendererJavascriptWatchdog();
        rendererRecoveryPending = false;
        pendingRecoveryWebView = null;
        clearUnresponsiveTracking();
        if (webView != null) {
            if (webView.getParent() instanceof ViewGroup) {
                ((ViewGroup) webView.getParent()).removeView(webView);
            }
            webView.destroy();
        }
        mainHandler.post(this::recreate);
    }

    private void clearUnresponsiveTracking() {
        mainHandler.removeCallbacks(recoverUnresponsiveRenderer);
        mainHandler.removeCallbacks(recoverTerminatedRendererFallback);
        rendererUnresponsive = false;
        rendererTerminationPending = false;
        rendererUnresponsiveSince = -1L;
        unresponsiveWebView = null;
        unresponsiveRenderer = null;
    }
}
