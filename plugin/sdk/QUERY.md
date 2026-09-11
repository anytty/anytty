# Reading the TUI with the SDK

Declare `ui.read` in the TUI component's manifest capabilities. A plugin can
request a typed `PluginUiSnapshot` from a registered host peer in the current
daemon route. Sending a query and returning its result both pass through that
daemon; there is no in-process shortcut for querying the local TUI.

```go
// host is the exact published peer address, including its registration epoch.
requestID, err := client.RequestUISnapshot(ctx, host)
if err != nil { return err }

// Integrate this into the plugin's one existing Receive loop. Do not create a
// competing receiver for each request, which could consume unrelated events.
for {
    batch, err := client.Receive(ctx, 20*time.Second)
    if err != nil { return err }
    for _, message := range batch.Messages {
        snapshot, err, matched := sdk.UISnapshotResult(message, requestID)
        if !matched { /* dispatch other plugin messages */; continue }
        if err != nil { return err }
        // snapshot.Owners includes workspace/tab/panel/floating references.
        // snapshot.Panels includes current committed bindings and view IDs.
        // snapshot.ActiveContext distinguishes content/floating/plugin focus.
        use(snapshot)
    }
}
```

The snapshot contains:

- The runtime TUI instance ID and reducer generation at the time of the read.
- Workspace, tab, panel and floating owner references, including inactive owners.
- Panel and floating view IDs, current committed terminal references, and focus.
- Active workspace/tab/panel/floating and focused plugin mount identifiers.

Terminal references use authenticated stable daemon identities, never another
client's endpoint aliases. If a binding's daemon identity is not known, its view
is still listed but the global terminal reference is absent. Pending attach
candidates are not presented as already committed bindings.

`active_context` is descriptive and has no `context_id`. Reading the UI does not
mint permission to change it: panel binding still requires the separately minted,
versioned context of a real user interaction. The query returns no terminal
screen/history/selection text, process environment, credentials, private route
leases, or internal reducer objects. Those content reads require separately
designed and explicitly granted APIs.

The host process bridge rejects a query without `ui.read`, even when the plugin
has general messaging, notifications or mount capabilities. The daemon separately
validates source registration, target address, identity scope and correlated reply.
As with other plugin capabilities, this API gate is not an OS process sandbox.
