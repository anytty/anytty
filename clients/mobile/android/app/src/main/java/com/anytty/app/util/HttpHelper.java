package com.anytty.app.util;

import java.io.ByteArrayOutputStream;
import java.io.IOException;
import java.io.InputStream;
import java.io.OutputStream;
import java.net.HttpURLConnection;
import java.net.URL;
import java.nio.charset.StandardCharsets;
import java.util.Map;

public class HttpHelper {

    private static final int DEFAULT_TIMEOUT = 10000;

    public static class Response {
        public final int statusCode;
        public final byte[] body;

        public Response(int statusCode, byte[] body) {
            this.statusCode = statusCode;
            this.body = body;
        }

        public String bodyString() {
            return new String(body, StandardCharsets.UTF_8);
        }

        public boolean isOk() {
            return statusCode >= 200 && statusCode < 300;
        }
    }

    public static Response get(String url, Map<String, String> headers) throws IOException {
        return request("GET", url, headers, null, DEFAULT_TIMEOUT);
    }

    public static Response get(String url, Map<String, String> headers, int timeoutMs) throws IOException {
        return request("GET", url, headers, null, timeoutMs);
    }

    public static Response post(String url, Map<String, String> headers, String body) throws IOException {
        byte[] bodyBytes = body != null ? body.getBytes(StandardCharsets.UTF_8) : null;
        return request("POST", url, headers, bodyBytes, DEFAULT_TIMEOUT);
    }

    public static Response post(String url, Map<String, String> headers, String body, int timeoutMs) throws IOException {
        byte[] bodyBytes = body != null ? body.getBytes(StandardCharsets.UTF_8) : null;
        return request("POST", url, headers, bodyBytes, timeoutMs);
    }

    public static Response request(String method, String url, Map<String, String> headers, byte[] body, int timeoutMs) throws IOException {
        HttpURLConnection conn = null;
        try {
            conn = (HttpURLConnection) new URL(url).openConnection();
            conn.setRequestMethod(method);
            conn.setConnectTimeout(timeoutMs);
            conn.setReadTimeout(timeoutMs);
            conn.setInstanceFollowRedirects(true);

            if (headers != null) {
                for (Map.Entry<String, String> entry : headers.entrySet()) {
                    conn.setRequestProperty(entry.getKey(), entry.getValue());
                }
            }

            if (body != null && ("POST".equals(method) || "PUT".equals(method) || "PATCH".equals(method))) {
                conn.setDoOutput(true);
                try (OutputStream os = conn.getOutputStream()) {
                    os.write(body);
                    os.flush();
                }
            }

            int statusCode = conn.getResponseCode();
            InputStream is = statusCode >= 400 ? conn.getErrorStream() : conn.getInputStream();
            byte[] responseBody = is != null ? readAll(is) : new byte[0];
            return new Response(statusCode, responseBody);
        } finally {
            if (conn != null) conn.disconnect();
        }
    }

    /** Quick reachability probe using HEAD request. Returns -1 on network error. */
    public static int probe(String url, int timeoutMs) {
        HttpURLConnection conn = null;
        try {
            conn = (HttpURLConnection) new URL(url).openConnection();
            conn.setRequestMethod("HEAD");
            conn.setConnectTimeout(timeoutMs);
            conn.setReadTimeout(timeoutMs);
            conn.setInstanceFollowRedirects(false);
            return conn.getResponseCode();
        } catch (Exception e) {
            return -1;
        } finally {
            if (conn != null) conn.disconnect();
        }
    }

    private static byte[] readAll(InputStream is) throws IOException {
        ByteArrayOutputStream baos = new ByteArrayOutputStream();
        byte[] buf = new byte[4096];
        int n;
        while ((n = is.read(buf)) != -1) {
            baos.write(buf, 0, n);
        }
        return baos.toByteArray();
    }
}
