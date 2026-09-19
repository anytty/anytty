# Shared Browser Proxy

`OpenSession(session)` adapts the existing Go application session and resource
protocol into an opener. Pass that opener to `Start(ctx, opener)` to create the
IPv4-loopback HTTP/CONNECT proxy.
The package is available to native Go consumers as well as the Flutter binding;
it does not import platform UI or cloud implementation packages.

Flutter starts/stops the listener through `EngineCommand.browser_proxy_listen`
and configures WebView with its returned port. HTTP parsing, upload/download
bytes, credit ACKs and resource cleanup stay in Go. Dart no longer marshals a
binding protobuf and queues a blocking FFI call for each browser data frame.
The previous Dart implementation lives under Flutter test support only, to
retain its existing regression cases as a reference.

## Ownership And Bounds

- The binding session owns one listener. Renderer retirement, engine shutdown,
  session closure and explicit stop close it and its accepted local sockets.
- Listen only on `127.0.0.1:0`; target hostnames and loopback addresses are sent
  unchanged to the daemon. Never dial or resolve the target on the client.
- Each proxy accepts at most 64 sockets and opens at most 16 remote resources.
- Headers are limited to 64 KiB with a 10-second deadline. Go's HTTP parser
  handles request framing and chunked uploads; proxy credentials are removed.
- Request 512 KiB send/receive windows and reject negotiated windows over 1 MiB.
  Legacy peers without credit use a 4 MiB bounded download queue. All queues
  also have a 4096-frame limit. Uploads use at most 32 KiB per frame.
- A receiver processes upload credit separately from the download writer.
  Remote closure drains already-received response bytes, including when an ACK
  fails after confirmed closure. Malformed credit is not tolerated.
- Connections progress in independent goroutines. This removes the single Dart
  writer queue but does not bypass shared transport congestion or relay limits.
- `Close` immediately revokes local access; `Wait` joins workers. Native sends
  already inside transport code may take that transport's bounded timeout to
  unwind. The proxy does not close a shared endpoint session itself.

Plain HTTP uses one request per resource and `Connection: close`. CONNECT and
WebSocket upgrades preserve the duplex byte stream. TLS remains end-to-end;
there is no certificate interception or verification bypass.

## Diagnostics And Verification

`anytty browser proxy` logs resource queue/open times, byte counts, upload credit
and transport send times, socket-write and ACK times. It excludes URLs, headers,
credentials and payloads. `anytty transport stage=datachannel_send` separates
DataChannel buffering, send-lock and channel-send time for slow calls. Both
prefixes are admitted by the Android diagnostic logger.

Run `go test -race ./client/browserproxy ./client/binding ./proto/bindingpb` and
Flutter's `native_browser_http_proxy_test.dart` for the native data path and UI
lifecycle. The legacy Dart tests are not evidence that the Go implementation
works; Go transfer tests cover HTTP, chunked upload, CONNECT, WebSocket, credit,
response drain and independent progress with a deliberately blocked connection.
Real network throughput and WebView speed-test correctness require separate
device measurements; passing these tests does not establish a throughput claim.
