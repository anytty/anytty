package com.anytty.app;

import android.os.Bundle;
import android.webkit.WebSettings;
import android.webkit.WebView;

import com.getcapacitor.BridgeActivity;

public class MainActivity extends BridgeActivity {
    @Override
    protected void onCreate(Bundle savedInstanceState) {
        AnyTTYDebugLog.init(this);
        registerPlugin(NativeConnectionPlugin.class);
        registerPlugin(NativeFilePickerPlugin.class);
        registerPlugin(NativeHapticPlugin.class);
        super.onCreate(savedInstanceState);

        boolean isDebug = (getApplicationInfo().flags & android.content.pm.ApplicationInfo.FLAG_DEBUGGABLE) != 0;
        WebView.setWebContentsDebuggingEnabled(isDebug);

        WebView webView = getBridge().getWebView();
        if (webView != null) {
            webView.setWebChromeClient(new AnyTTYWebChromeClient(getBridge()));
            getBridge().setWebViewClient(new AnyTTYWebViewClient(getBridge()));
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
        }
        AnyTTYDebugLog.event(AnyTTYDebugEvent.ACTIVITY_CREATED, isDebug);
    }
}
