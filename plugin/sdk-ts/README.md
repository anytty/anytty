# AnyTTY TypeScript plugin SDK

Node.js SDK for the same generated Protobuf protocol as `plugin/sdk` (Go).
Every call goes through the host's stdio bridge to the selected authenticated
daemon, including messages addressed to the current TUI. The SDK does not render
UI, dispatch operations locally, or establish an independent network connection.

From the repository root:

```sh
npm ci --prefix plugin/sdk-ts --workspaces=false
npm --prefix plugin/sdk-ts test
node scripts/generate-plugin-sdk.mjs
```

Generation requires `protoc` and the repository's `protoc-gen-es` installation.
`proto/apipb/plugin.proto` is the only wire schema. The generated SDK exports all
its types and schemas. This package is currently independently managed with its
own lockfile; it is not automatically published.

```ts
import { create } from '@bufbuild/protobuf'
import { randomUUID } from 'node:crypto'
import {
  Client, PluginRegisterRequestSchema, PluginMessageSchema,
} from '@anytty/plugin-sdk'

const root = Client.stdio()
const endpoint = root.forEndpoint('studio') // Client-local connection alias.
const registration = await endpoint.register(create(PluginRegisterRequestSchema, {
  address: {
    pluginId: 'org.example.status',
    pluginInstanceId: randomUUID(),
    tuiInstanceId: process.env.ANYTTY_TUI_INSTANCE_ID,
  },
}))

// This self-addressed message STILL traverses the daemon.
const waiting = endpoint.receive(25_000)
await endpoint.send(create(PluginMessageSchema, {
  destination: registration.address,
  body: { case: 'payload', value: {
    schema: 'org.example.status.example', version: 1,
    data: new Uint8Array([8, 1]), // Your own generated Protobuf bytes.
  } },
}))
const batch = await waiting
// Consume typed batch.messages; inspect resyncRequired before applying state.
await endpoint.unregister()
root.close()
```

Each `forEndpoint()` client owns a separate registration while sharing one
multiplexed transport. A pending `receive()` does not block `send()` or other
endpoints. The host must validate the installed plugin's identity and endpoint
scope before forwarding; passing an endpoint alias is not an authorization grant.

`send()` resolves on **daemon delivery acknowledgement**, not completion of a
panel operation. A caller requiring completion must set a request ID and consume
the matching typed `PluginReply` from its receive loop. Do not run competing
receive loops on one registration; the daemon permits one active receive.

`state()` supports generated `get`, `put`, and `watch` requests. Only the daemon
service can write its namespace. Watch starts with an atomic versioned snapshot;
subsequent full snapshots arrive as `stateChanged`. If `resyncRequired` is true,
re-establish affected watches and ignore older snapshot revisions. A delivery
sequence is diagnostic, not a resumable event-log cursor.

Calls accept `{ signal, timeoutMs }`. Cancellation sends a control frame to cancel the matching host executor context,
rejects the local pending call, and discards late responses. Control frames bypass
worker admission, even when all 32 host workers are busy. Deadlines are validated
and enforced at the host. Cancellation does **not** undo a remote side effect;
the remote receive slot can take a brief scheduling interval to release. Long-poll receive is bounded by 25 seconds; default call
timeout is 30 seconds. The SDK never automatically retries writes. `unregister()`
requests daemon cleanup; child `close()` cancels only that client's calls, while
root `close()` rejects all calls on the shared transport. EOF and framing errors
also reject pending calls. stdout is reserved for Protobuf; use stderr for logs.

Frames use unsigned-varint lengths and a 4 MiB maximum, matching Go. There are at
most 128 pending RPCs and 8 MiB of buffered outbound frames. Generated `uint64`
fields are `bigint`, preserving values above JavaScript's safe integer limit.

Tests include malformed/fragmented frames, concurrent send/receive, out-of-order
correlation, cancellation, delayed replies, endpoint isolation, disconnects, host cancellation, deadline enforcement, and
a real TypeScript → Go SDK bridge → two independently owned Protobuf daemons
round trip. The test helper never connects to the user's running daemon.
