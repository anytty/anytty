package com.anytty.app;

import android.net.Uri;
import android.webkit.ConsoleMessage;
import android.webkit.GeolocationPermissions;
import android.webkit.PermissionRequest;
import android.webkit.ValueCallback;
import android.webkit.WebChromeClient;
import android.webkit.WebView;

import com.getcapacitor.Bridge;
import com.getcapacitor.BridgeWebChromeClient;

public class AnyTTYWebChromeClient extends BridgeWebChromeClient {
    private static final String DIAGNOSTIC_PREFIX = "[anytty:diagnostic] ";

    public AnyTTYWebChromeClient(Bridge bridge) {
        super(bridge);
    }

    @Override
    public void onPermissionRequest(PermissionRequest request) {
        String[] resources = request.getResources();
        if (
            AnyTTYLocalUrl.isCanonical(request.getOrigin().toString()) &&
            resources.length == 1 &&
            PermissionRequest.RESOURCE_VIDEO_CAPTURE.equals(resources[0])
        ) {
            super.onPermissionRequest(request);
            return;
        }
        request.deny();
    }

    @Override
    public void onGeolocationPermissionsShowPrompt(String origin, GeolocationPermissions.Callback callback) {
        callback.invoke(origin, false, false);
    }

    @Override
    public boolean onConsoleMessage(ConsoleMessage message) {
        String value = message == null ? "" : message.message();
        if (value.startsWith(DIAGNOSTIC_PREFIX)) {
            AnyTTYDebugLog.connection("web " + value.substring(DIAGNOSTIC_PREFIX.length()));
        }
        return super.onConsoleMessage(message);
    }

    @Override
    public boolean onShowFileChooser(
        WebView webView,
        ValueCallback<Uri[]> filePathCallback,
        WebChromeClient.FileChooserParams fileChooserParams
    ) {
        if (
            webView != null &&
            fileChooserParams != null &&
            AnyTTYFileChooserPolicy.allowsSingleImage(
                webView.getUrl(),
                fileChooserParams.getAcceptTypes(),
                fileChooserParams.getMode(),
                WebChromeClient.FileChooserParams.MODE_OPEN
            )
        ) {
            return super.onShowFileChooser(webView, filePathCallback, fileChooserParams);
        }
        filePathCallback.onReceiveValue(null);
        return true;
    }
}
