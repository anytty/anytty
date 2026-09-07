# Connection diagnostics

## Correlation

Search for `anytty connect` and follow `trace_id` from `host_open` through
authorization, Controller resolution, Edge exchange, DataChannel authentication,
and protocol readiness. Outgoing gRPC metadata propagates the trace ID.
`session_id` links Edge and daemon signaling; `request_id` links native platform
requests to Flutter queue, storage, public-key derivation, and signing timings.
Transport reuse logs both the caller trace and the transport creator trace.

`stage_ms` measures time since the previous stage in that component; `total_ms`
measures elapsed time in that component. Concurrent stages overlap: do not sum
them to infer user-visible latency. For long-lived Edge streams, use
`client_answer_sent` for setup, not the eventual stream completion timestamp.
Network attempt logs separate queue, DNS/TCP, and TLS time and identify the path
and outcome. Canceled losing attempts are not connection failures.

Logs must not contain private keys, access grants, challenge proofs, or payloads.
Diagnostic exports may contain network addresses and device identifiers and
should be treated as private operational data.

## Optimization boundaries

- Cached Edge routing gets a 750 ms head start before concurrent Controller
  refresh. Authenticated updated locators are written back asynchronously.
- TCP path attempts are bounded to three per dial and twelve process-wide;
  successful path ordering lasts 30 seconds and is topology/target scoped.
- Engine-owned gRPC pools retain at most sixteen cached transports, with a
  30-second idle timeout. Keys include destination, TLS trust, and topology.
  Independent streams hold leases; releasing a losing attempt cannot close a
  sibling stream. No session authorization results are cached.
- Flutter allows four independent platform reads/signatures in flight, with
  exclusive FIFO barriers for mutations. This does not make synchronous Dart
  cryptographic work run on multiple CPU threads.
- Credential reads still access secure storage. Only the derived public key is
  cached, bounded to sixteen entries and checked against a seed digest. Every
  challenge is signed afresh; key rotation and deletion remain observable.
- Local discovery overlaps credential/profile preparation. Its existing empty
  cache wait remains and can still dominate route preparation.
- Only the extra initial application health probe runs after publication of an
  already authenticated, protocol-ready session. TLS, identity, authorization,
  and protocol Hello remain mandatory before readiness. Failed health probes
  invalidate the session through the existing supervisor lifecycle.

## Android development observations

Development-device samples on 2026-09-07 are not a performance guarantee:

- One successful open took about 10.8 seconds, compared with an earlier
  13.4-second observation.
- A later run still hit an approximately 8-second cached Edge exchange timeout;
  the successful retry took about 13.5 seconds itself. The full user wait was
  longer. Connection latency is not yet consistently improved.
- Route preparation still took about 1.4 seconds. Individual signatures in the
  Android debug build took about 390-410 ms. Synchronous signing can also delay
  completion callbacks for otherwise fast platform reads.
- A cached gRPC `READY` state is not proof that an application-level exchange
  will finish promptly. Edge response waits require correlated server-side
  stages before blaming transport reuse, relay allocation, or daemon signaling.

The server instrumentation must be deployed separately to observe its stages.
Do not restart an active AnyTTY process or daemon to collect these diagnostics.
