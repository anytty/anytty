// Package sdk is the layout-program convenience binding of the TUI v2 wire
// protocol. It is layered into three subpackages and this package re-exports
// the first two for backward compatibility:
//
//   - sdk/core: the protocol client (frame loop, typed events, Emit/Commit);
//   - sdk/builder: the chainable view-tree builder and text metrics;
//   - sdk/widgets: optional program-side chrome widgets.
//
// A layout program only needs the wire protocol, so the whole SDK builds
// against the standard library plus proto/ui and proto/ui/protobuf (see
// deps_test.go): no kernel, runtime, render or component packages.
package sdk
