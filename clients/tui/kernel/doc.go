// Package kernel implements the TUI v2 box model defined by
// tui2/docs/PROTOCOL.zh-CN.md §2, §9.1 and §9.5.
//
// The kernel is a pure, standard-library-only solver: it turns a tree of
// Node values into a Frame holding absolute rects, styled text lines,
// overlay frames (Pos subtrees), cursor placement and hit testing. It
// deliberately knows nothing about processes, PTYs, terminal sizes or
// business objects (slot/tab/pane), and it does not implement scrolling
// (protocol §9.5).
//
// Rules (protocol §9.1, authoritative):
//
//   - Defaults: empty id, Visible=true, Flow=col, Size={0,0,0}, Pos=null,
//     no content, no cursor, no input, not focused.
//   - Size 0 on an axis means "use the intrinsic content size"; when there
//     is no intrinsic size the axis stretches to fill the parent.
//   - Visible=false is display:none: not rendered, not hit, not occupying
//     space, so siblings close the gap.
//   - The box rect is pure geometry: the kernel has no border concept and
//     the content area is the full rect. Chrome (borders, titles, badges,
//     colors) belongs to the content or the host component, which declares
//     its own inset.
//   - Flow col/row: fixed and intrinsic sizes are assigned first, then the
//     leftover is split by Flex (equal split when no Flex is set); the
//     integer remainder goes to the last flexible child.
//   - Flow stack: children fill the parent rect by default.
//   - Pos != nil: absolute placement inside the parent rect; the whole
//     subtree composites above the regular flow and produces an entry in
//     Frame.OverlayFrames.
//   - Z order: declaration order among siblings (later is on top); Pos
//     subtrees are above the regular flow.
//   - Hit returns the smallest-area visible node with a non-empty id that
//     contains the point; ties go to the later declaration.
//
// Frames never contain ANSI escape bytes: text lines carry an opaque style
// string (explicit style or host-internal token) that the host renderer
// resolves to SGR.
package kernel
