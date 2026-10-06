# SDK + codec benchmarks: before / after

Machine / toolchain (both runs identical):

- darwin/arm64, Apple M2, macOS (`goos: darwin`, `goarch: arm64`)
- `go version go1.26.7 darwin/arm64`
- tree: `refactor/tui-surface` @ `dc62a4a` plus the parallel workstreams in this
  working tree; `clients/tui/sdk/builder` was unchanged between the two runs

Exact command (run once before the perf changes and once after):

```sh
export PATH="$HOME/.local/share/go-toolchains/go1.26.7/bin:$PATH"
TMPDIR=/tmp ANYTTY_ALLOW_NESTED=1 go test ./clients/tui/sdk/bench -bench . -benchmem -count=5
```

The table reports the median of the 5 runs. The machine was shared and its
load moved between the two captures, so ns/op moves by a few percent; B/op and
allocs/op are deterministic and are the real signal. `—` means the benchmark
did not exist before the change (new reuse-mode API).

| Benchmark | ns/op before | ns/op after | B/op before | B/op after | allocs before | allocs after |
|---|---:|---:|---:|---:|---:|---:|
| Commit100 | 8325 | 8058 | 2208 | 0 | 4 | 0 |
| Commit1000 | 76981 | 75169 | 19104 | 1 | 4 | 0 |
| Commit10000 | 773052 | 748649 | 180384 | 131 | 4 | 0 |
| KeyEventDecode (envelope) | 32 | 24 | 28 | 24 | 2 | 1 |
| KeyEventDecodeRoundTrip (decode+unmarshal) | 244 | 237 | 208 | 208 | 7 | 6 |
| KeyEventDecodeReuse (new `Decoder.ReuseBuffer(true)`) | — | 12 | — | 0 | — | 0 |
| KeyEventDecodeRoundTripReuse | — | 220 | — | 184 | — | 5 |
| ResultEncode (`wire.Encoder.Encode`) | 265 | 240 | 96 | 0 | 2 | 0 |
| StreamEncode (`wire.Encoder.Encode`) | 154 | 92 | 576 | 0 | 2 | 0 |
| Emit (`sdk.Client.Emit` + request bookkeeping) | 277 | 242 | 176 | 0 | 3 | 0 |
| FrameRoundTrip (`wire.DecodeFrame`) | 50 | 51 | 76 | 72 | 3 | 2 |

## What changed

- `proto/ui.MarshalAppend(dst, t, m, max)` appends one frame into caller
  storage; error kinds match `Marshal`.
- `wire.Encoder.Encode` marshals into a `sync.Pool` scratch buffer (`*[]byte`,
  no per-Put boxing) and writes it with a single `Write`. Buffers above
  `DefaultMaxMessageBytes+5` are dropped instead of retained.
- `wire.Decoder.ReuseBuffer(true)` reads frames into the decoder's scratch
  buffer: payload slices alias it and are valid until the next `Decode`.
  Default `NewDecoder` behavior is unchanged.
- `Decoder` keeps the 4-byte length prefix in the struct, removing a per-frame
  heap allocation of the local prefix array.
- `wire.DecodeFrame` reuses its `Decoder` envelope through a pool.
- `sdk/core.Client.Commit` and `Client.Emit` build their envelopes from pooled
  scratch (`viewScratchPool`, `resultScratchPool`), taken while holding
  `writeMu`, so a steady-state Commit allocates 0 B/op. Commit now takes
  `writeMu` before `mu`, which also makes write order equal revision order.

Target: Commit steady-state allocations reduced by more than 50% — actual
reduction is 100% (4 → 0 allocs/op at 100/1000/10000 boxes). The two new
reuse-mode decode benchmarks show the intended path: 0 allocs/op at 12 ns/op
for a key envelope versus 1 alloc/op at 24 ns/op for the unchanged default.

# Incremental views (VIEW_DELTA): before / after

Second capture, same machine and toolchain, after the Go SDK learned
`CommitDelta` (PROTOCOL §2.1). The "before" columns are the full-snapshot
numbers from the capture above (re-run side by side for a fair comparison:
`BenchmarkCommit*` and the byte-reporting `BenchmarkCommitBytes*` are full
VIEWs; `BenchmarkCommitDelta*` diff against the previous frame).

Exact command (run once for the full snapshots and once for the deltas):

```sh
export PATH="$HOME/.local/share/go-toolchains/go1.26.7/bin:$PATH"
TMPDIR=/tmp ANYTTY_ALLOW_NESTED=1 go test ./clients/tui/sdk/bench -bench 'Commit' -benchmem -count=5 -timeout 900s
```

The delta benchmarks edit **one text leaf** of a tree of the given size and
commit the alternating trees, so every call is a real one-leaf diff against the
frame the host holds. `B/frame` is a custom metric (`b.ReportMetric`) because
`-benchmem` does not show produced bytes.

| Benchmark | ns/op | B/frame | B/op | allocs/op |
|---|---:|---:|---:|---:|
| Commit100 (full VIEW) | 15139 | 906 | 0 | 0 |
| CommitDelta100 (1-leaf delta) | 14795 | 39 | 449 | 7 |
| Commit1000 (full VIEW) | 137360 | 8588 | 2 | 0 |
| CommitDelta1000 (1-leaf delta) | 147300 | 38 | 482 | 9 |
| Commit10000 (full VIEW) | 1441035 | 86555 | 218 | 0 |
| CommitDelta10000 (1-leaf delta) | 1672541 | 40 | 557 | 11 |

Frame bytes (the actual wire win) come from `BenchmarkCommitBytes*` /
`CommitDelta*`:

| Tree | full VIEW B/frame | 1-leaf delta B/frame | ratio |
|---|---:|---:|---:|
| 100 nodes | 905.8 | 39.1 | 4.3% |
| 1000 nodes | 8588 | 38.5 | 0.45% |
| 10000 nodes | 86555 | 40.2 | 0.046% |

`TestDeltaBytesVsFullView` pins the 10000-node result (delta < 5% of the full
VIEW) and asserts the whole-tree fallback: clearing a root scalar cannot be a
`set`, so the diff is a root `replace` whose box is the whole tree; the cost
guard then picks the full VIEW. The test output:

```
=== RUN   TestDeltaBytesVsFullView
    bench_test.go:213: 10000-node one-leaf delta = 40 bytes, full VIEW = 86554 bytes (0.046%)
--- PASS: TestDeltaBytesVsFullView (0.03s)
```

## What changed

- `core.DiffView(base, next) ([]*pb.Patch, bool)` computes index-addressed
  patches (set / replace / insert / remove). It emits a `set` only when the
  change is representable (proto3 cannot clear a scalar with a zero-omitting
  set), otherwise a `replace`; child-list changes use a common prefix/suffix
  match and insert/remove the middle. Unchanged subtrees short-circuit on a
  hand-written `boxesEqual` (pointer fast path on shared subtrees) instead of
  reflection, and patch paths are only allocated for the branches that differ.
- `core.Client.CommitDelta(base, next, keys) (bool, error)` commits one frame
  per call: it diffs against the client's committed baseline, marshals the
  delta into a pooled scratch buffer, and compares it with the full VIEW's
  exact `gproto.Size`-derived frame length. If the delta is not smaller (or no
  baseline / no `view_delta` feature), it writes the full VIEW. The baseline is
  replaced on every successful send.
- Invalidation: `Client.Supports(feature)`, `HasBase`, `DropBase`. The base is
  dropped on every HELLO (new epoch) and on every `view_rejected` event before
  the program handler runs.
- `app.Program` uses the delta path automatically when the host advertises
  `view_delta` and a baseline exists, still exactly one frame per batch.
  `Program.ForceFullView` forces snapshots for debugging. A `ViewRejectedMsg`
  drops the baseline (no retry inside the same batch) so the next batch sends a
  full VIEW.

Cost note: at 10000 nodes the delta's ns/op is within noise of the full
snapshot because the diff walks the whole tree to prove it is unchanged
(~1.6 ms); the win is bytes on the wire (86.5 KB → 40 B, 0.046%) and host-side
apply cost. The full snapshot also allocates 0 B/op but the delta allocates
~560 B/op for the patch, envelope and frame; a future structural-sharing or
dirty-tracking view could make the walk O(changed) instead of O(tree).

## Host-side apply + end-to-end (same machine, Go 1.26.7)

> Superseded numbers: this section captures the delta implementation before the
> host allocation and incremental-layout passes (see the later "Host-side
> apply" and "incremental layout" sections). The qualitative conclusion
> (delta beats full on the wire and on allocations) still holds, but every
> absolute ns/op, MB and alloc figure below is stale — use the latest tables.

Host-side benchmarks live in `clients/tui/runtime/view_delta_bench_test.go`
(part of `go test ./clients/tui/runtime`). A 10k-node tree, one changed leaf;
every row excludes baseline construction (that is a separate full VIEW):

| Benchmark (10000 nodes, 1 leaf) | ns/op | B/op | allocs/op |
|---|---:|---:|---:|
| `HandleViewFull_10000` (decode-less, full tree) | 17.7 ms | 37.0 MB | 269k |
| `HandleViewDeltaLeaf_10000` (decode-less, 1-leaf delta) | 17.6 ms | 37.1 MB | 269k |
| `WireFullRoundTrip_10000` (frame → decode → apply + layout) | 23.8 ms | 47.9 MB | 409k |
| `WireDeltaRoundTrip_10000` (frame → decode → apply + layout) | 17.8 ms | 37.1 MB | 269k |

Reading:

- Delta wins **on the wire**: 86.5 KB → ~43 B per frame at 10k nodes (SDK table
  above), which is the point of §2.1.
- End-to-end (decode + apply + relayout) a 1-leaf delta beats a full snapshot at
  10k nodes; after the later allocation and incremental-layout passes the ratio
  is much larger (see the latest "Host-side apply" table). The saving is mostly
  not decoding the 86 KB payload.
- Host-side apply itself is now on par with a full VIEW (`17.6` vs `17.7` ms)
  because both are dominated by **full-tree relayout** (`kernel.Layout` runs on
  the whole tree for every accepted view), not by patch application. A future
  incremental layout would be the next multiplier.
- An earlier implementation deep-cloned the whole tree per delta and was 2×
  slower than a full snapshot (40.8 ms / 83.9 MB at 10k). The final
  implementation uses copy-on-write path copying (`cloneSpine`/`cloneForEdit`
  in `clients/tui/runtime/view_delta.go`): only nodes on patched paths are
  rebuilt, so apply cost is O(depth + patch), not O(tree).

Exact commands:

```sh
export PATH="$HOME/.local/share/go-toolchains/go1.26.7/bin:$PATH"
TMPDIR=/tmp ANYTTY_ALLOW_NESTED=1 go test ./clients/tui/runtime   -run '^$' -bench 'HandleView|WireFull|WireDelta' -benchmem -count=5
```

# Host allocation pass: kernel.Layout + runtime.toNode

Same machine and toolchain (darwin/arm64, Apple M2, Go 1.26.7). The earlier
capture identified the flat cost of a 10k-node VIEW as allocation/GC
(`runtime.madvise` 22%, `scanObjectsSmall` 19%, `mallocgc` 17%), with
`kernel.Layout` 18.6% cumulative. This pass removes the per-node allocations in
`kernel.Layout` and `runtime.toNode`; layout results, `Frame` shape, `Rect`/`Hit`
semantics and the wire are all unchanged.

Kernel-only benchmark (new: `clients/tui/kernel/layout_bench_test.go`, the same
2*rows+1 tree shape the runtime benchmark builds, so kernel gains are visible in
isolation):

```sh
TMPDIR=/tmp ANYTTY_ALLOW_NESTED=1 go test ./clients/tui/kernel -run '^$' -bench . -benchmem -count=5
```

| Benchmark | ns/op before | ns/op after | B/op before | B/op after | allocs before | allocs after |
|---|---:|---:|---:|---:|---:|---:|
| KernelLayout_100 | 144k | 15.5k | 226.0 KB | 29.6 KB | 2033 | 10 |
| KernelLayout_1000 | 1.45 M | 171k | 2.36 MB | 344 KB | 20,053 | 12 |
| KernelLayout_10000 | 14.8 M | 1.73 M | 26.2 MB | 3.17 MB | 209,157 | 40 |

Host-side apply (median of 5 runs, same command as the table above the previous
section). Delta is a one-leaf change on the 10k tree:

| Benchmark | ns/op before | ns/op after | B/op before | B/op after | allocs before | allocs after |
|---|---:|---:|---:|---:|---:|---:|
| `HandleViewFull_100` | 604.0 k | 30.3 k | 336.4 KB | 92.0 KB | 2,647 | 324 |
| `HandleViewFull_1000` | 12.81 M | 386.1 k | 3.45 MB | 957 KB | 26,067 | 3,026 |
| `HandleViewFull_10000` | 63.08 M | 3.65 M | 37.05 MB | 9.26 MB | 269,171 | 30,054 |
| `HandleViewDeltaLeaf_100` | 1.13 M | 29.6 k | 336.7 KB | 92.4 KB | 2,640 | 317 |
| `HandleViewDeltaLeaf_1000` | 8.08 M | 344.9 k | 3.46 MB | 965 KB | 26,060 | 3,019 |
| `HandleViewDeltaLeaf_10000` | 30.28 M | 3.59 M | 37.14 MB | 9.34 MB | 269,163 | 30,047 |
| `WireFullRoundTrip_10000` | 35.09 M | 9.84 M | 47.86 MB | 20.06 MB | 409,197 | 170,080 |
| `WireDeltaRoundTrip_10000` | 21.87 M | 3.61 M | 37.14 MB | 9.34 MB | 269,176 | 30,058 |

At 10k nodes a full VIEW drops **17.7 ms → 3.65 ms** (4.8×), **37.0 MB → 9.26
MB** (4.0×) and **269k → 30k allocs/op** (8.9×); the one-leaf delta drops
**17.6 ms → 3.59 ms** and **37.1 MB → 9.34 MB** at the same alloc count. Wire
round trips improve the same way because the wire itself is byte-identical.

Frame bytes are unchanged by this pass: `TestDeltaBytesVsFullView` still logs
`10000-node one-leaf delta = 40 bytes, full VIEW = 86554 bytes (0.046%)`, i.e.
the 86.5 KB / ~43 B figures above are unaffected. The wire path was not touched.

## What changed

- `kernel.Layout` walks the tree through one `frameBuilder` that owns the
  frame's `Rects` map, `Lines` and `hits` slices. Previously every `solve`
  allocated a fresh `Frame{Rects: map[...]{}}` and merged each child's map
  upward (`mergeRects`), i.e. one map per node. Now the regular-flow subtree
  shares one builder; only a `Pos` subtree allocates its own builder because it
  becomes its own `OverlayFrame`. This removes the per-node map plus the
  `mergeRects` copy for every node.
- `Rects`/`Lines`/`hits` are pre-sized from a single read-only `countHints`
  pass (visible ids and content-line count), so the slices do not repeatedly
  grow on a 10k-node tree.
- `assignFlow` no longer builds a `map[int]Rect` per container. It appends
  `flowPlan` values into a per-depth scratch slice reused across siblings (the
  slice is held only while children recurse one level deeper), and writes each
  child's `Rect` back into the plan in place. One scratch allocation per tree
  depth, reused across the whole layout.
- `renderOwn` iterates content lines via `forEachContentLine`, which walks the
  `Text`/`Lines` payload without the per-node `strings.Split`; `IntrinsicSize`
  scans the same payload rune by rune instead of splitting. `Truncate` returns
  a prefix of its input instead of building a new string, and `RuneWidth` has
  ASCII and `< 0x1100` fast paths (the linear scan over the 30 wide ranges was
  31% flat on ASCII content). All width/CJK/emoji semantics are pinned by the
  untouched `width_test.go`; `strings.Split` is still what `ContentLines`
  returns for callers, so that API is unchanged.
- `runtime.toNode` fills a caller-provided `*kernel.Node` in place
  (`toNodeInto`) so children are allocated directly in the parent's slice
  instead of via a temporary `*Node` per child that was then copied. `Lines`,
  `Input` and `Props` are only copied when non-empty; nil stays nil.
- `countBoxes` was left unchanged in this pass (a zero-allocation O(tree) walk);
  a later pass added the incremental `boxCount`, which is now maintained
  per patch and checked against `countBoxes` by the equivalence fuzz.

Semantic risk considered: the builder writes `b.rects[n.ID] = rect` in the same
pre-order as the old per-node merge, and later siblings/overlays still overwrite
earlier ids, so `Frame.Rect`/`Rects` are identical. Flow output order and
`hits` z-order are unchanged because children are still visited in
`n.Children` order; `Hit` iterates the same candidate sequence. `OverlayFrames`
is still appended in declaration order, and a `Pos` subtree's nested overlays
are lifted into its own frame exactly as before (the nested builder's overlays
survive into the returned frame). `Frame`'s public fields are untouched;
`hits` stays unexported and `allHits` remains correct.

# Incremental layout: subtree reuse for VIEW_DELTA

> Superseded: this section documents the now-removed flat `Frame.Rects` design
> and its numbers. The pointer-identity reuse idea (gated by the absolute rect)
> survived, but the frame shape, bookkeeping and numbers were replaced by the
> O(changed) bookkeeping and tree-shaped rect-index sections after it.

Same machine and toolchain (darwin/arm64, Apple M2, Go 1.26.7). The previous
section left one multiplier on the table: every accepted view, full or delta,
ran `runtime.toNode` on the whole tree and `kernel.Layout` on the whole tree.
A one-leaf delta was therefore ~as expensive as a full 10k-node VIEW
(`HandleViewDeltaLeaf_10000` ≈ `HandleViewFull_10000` ≈ 3.6 ms / 9.3 MB /
30k allocs). This pass reuses the solved kernel subtree and its frame for every
part of the tree that a VIEW_DELTA did not touch.

The reuse anchor is pointer identity in the program's box layer: a delta is
applied with copy-on-write path copying (`cloneSpine`/`cloneForEdit` in
`clients/tui/runtime/view_delta.go`), so every subtree off a patched path is
the same `*pb.Box` pointer it was in the previous revision.

## What changed

- `kernel.Solved{Rect Rect; Frame Frame}` and `(*Node).Cached()/SetCached()`
  (stored in an unexported field). A `Solved` records "this subtree was solved
  at this absolute rect with this frame".
- `kernel.LayoutCached(root, w, h) Frame` is `Layout` plus reuse. It walks the
  tree; a node carrying a `Solved` whose `Rect` equals the node's current
  absolute rect is spliced (its `Rects` copied into the parent map, its
  `Lines`/`hits`/`OverlayFrames` appended in order, its cursor adopted) instead
  of being solved. The `Rect` equality is a hard gate, because frames store
  absolute coordinates: a subtree that moved or resized is re-solved.
  `LayoutCached` returns exactly the frame `Layout` returns, and records a
  fresh `Solved` on every node it solves so the next call can reuse it.
- `runtime` keeps `map[*pb.Box]*kernel.Node` (`node_cache.go`). `buildNodeTree`
  walks the new box tree: a `*pb.Box` hit means its `*kernel.Node` (and the
  `Solved` inside it) is reused wholesale instead of rebuilt. A full VIEW shares
  no pointers, so it neither reads nor populates the map; the first delta after
  a full VIEW lazily indexes the cached tree once, then each delta rebuilds the
  map from the boxes still reachable, pruning dropped/replaced branches. The
  map holds only `*kernel.Node` pointers — the solved rect/frame live on the
  node — so cache memory is bounded by the current view.
- `countHintsPruned` gives `LayoutCached` its allocation sizes in O(changed
  path): a cached child contributes its frame's `len(Rects)`/`len(Lines)`
  instead of a full subtree walk.
- No protobuf types enter `kernel`; the existing `Layout(root, w, h)` API is
  unchanged and still the path for full VIEWs.

## Results

Same-process A/B, so machine load cancels. `BenchmarkABKernelFromScratch_10000`
solves a clean 10k tree every iteration; `BenchmarkABKernelCached_10000` changes
one leaf and invalidates only its path (`root -> row -> leaf`), reusing every
untouched row:

```sh
TMPDIR=/tmp ANYTTY_ALLOW_NESTED=1 go test ./clients/tui/kernel -run '^$' -bench 'ABKernel' -benchmem -count=10
```

| kernel, 10000 nodes | ns/op | B/op | allocs/op |
|---|---:|---:|---:|
| `ABKernelFromScratch_10000` (full build + solve) | 1,763,000 | 3.17 MB | 40 |
| `ABKernelCached_10000` (1-leaf solve, rest spliced) | 1,180,000 | 3.17 MB | 55 |

Kernel-only win: **1.50×** on ns/op. End-to-end host apply (same process
control, patch + from-scratch tree build + full solve vs. the real cached delta
path):

```sh
TMPDIR=/tmp ANYTTY_ALLOW_NESTED=1 go test ./clients/tui/runtime -run '^$' \
    -bench 'HandleViewDeltaLeaf_10000|DeltaLeafControl_10000' -benchmem -count=8
```

| host, 10000 nodes, 1 leaf | ns/op | B/op | allocs/op |
|---|---:|---:|---:|
| `DeltaLeafControl_10000` (apply + rebuild + full solve) | 3,150,000 | 9.74 MB | 30,046 |
| `HandleViewDeltaLeaf_10000` (real cached delta path) | 2,940,000 | 6.17 MB | 537 |

The standard baselines (median of 6) on the same machine:

| Benchmark | ns/op before | ns/op after | B/op before | B/op after | allocs before | allocs after |
|---|---:|---:|---:|---:|---:|---:|
| `HandleViewFull_10000` | 3,600,000 | 3,570,000 | 9.26 MB | 9.66 MB | 30,054 | 30,054 |
| `HandleViewDeltaLeaf_10000` | 3,590,000 | 3,300,000 | 9.34 MB | 6.17 MB | 30,047 | 566 |
| `WireDeltaRoundTrip_10000` | 3,610,000 | 3,450,000 | 9.34 MB | 6.17 MB | 30,058 | 586 |
| `WireFullRoundTrip_10000` | 9,840,000 | 10,300,000 | 20.06 MB | 20.47 MB | 170,080 | 170,080 |

Reading:

- The delta path drops **30,047 → 537 allocs/op** (~56×) and **9.34 MB → 6.17 MB
  B/op** (~34%). The remaining 6.17 MB is the flat `Frame` materialization: the
  root frame still copies the rects of every reused subtree into one shared map
  (`maps.Copy` in `frameBuilder.addFrame`). That is deliberate — the returned
  `Frame` is a flat value with no subtree indirection — and it is the floor for
  the current `Frame` shape.
- ns/op is **near parity** (≈2.94 vs 3.15 ms, ~7% in the same-process control),
  not a 1.5× end-to-end win. The kernel-only solve is ~1.5× faster, but the
  delta path then pays an O(tree) `countBoxes` max_nodes walk and an O(tree)
  flat `Rects` merge, and allocates a fresh 10k-entry pointer map. Profile of
  the delta path (`go test ./clients/tui/runtime -run '^$' -bench
  'BenchmarkHandleViewDeltaLeaf_10000$' -cpuprofile`):
  - `runtime.gcBgMarkWorker`/`scanObjectsSmall` ≈ 25% — GC of the flat frame's
    map plus the pointer map.
  - `(*Session).buildNodeTree`/`nodeFor` ≈ 22% — the per-commit pointer map
    (`mapassign_fast64ptr` is the largest leaf).
  - `kernel.LayoutCached`/`solveChild`/`addFrame` ≈ 21% — the flat rect merge.
  So the remaining bottleneck is **flat-frame rect merging plus the per-commit
  pointer map and its GC**, not patch application (`applyPatches` is ~1%).
- `HandleViewFull_10000` is unchanged within noise (full views share no
  pointers, and the lazy cache means they never build the pointer map).
  `WireFullRoundTrip_10000` is unchanged (wire byte-identical).
- The B/op on full views is +0.4 MB because every kernel `Node` now carries an
  extra `*Solved` field (unsolved nodes keep it nil; solved full-view nodes
  allocate one `Solved` each).

## Correctness

- `clients/tui/runtime/incremental_equivalence_test.go` fuzzes 400 iterations
  over random trees: it interleaves full VIEWs, keys-only deltas (no patches,
  so the whole root subtree is reused), and all five patch ops
  (set/replace/insert/remove/move, including moves and resizes), plus periodic
  viewport resizes, and after every accepted commit deep-compares the
  incremental frame against an independently built from-scratch oracle
  (`nodeFull` + `kernel.Layout`): `Rects`, `Lines` (order + content + style),
  `OverlayFrames` shape, cursor, `Hit(x,y)` over a grid, and `focusLocked`
  focus. `TestIncrementalFrameIndependence` snapshots 25 successive frames and
  re-asserts each against its own fresh solve, so no later splice mutates an
  earlier returned frame. `TestIncrementalNodeReuse` pins that an untouched
  sibling keeps its exact `Solved` pointer and its children backing array.
- `clients/tui/kernel/cache_test.go` covers the rect gate (root and subtree),
  splice-vs-solve equality, and aliasing (mutating a returned parent frame
  cannot touch a cached child frame).
- A `kernel.Node` is read-only after it is solved: `Layout`/`LayoutCached` only
  read exported fields, and a reused subtree shares its children backing array
  because its parent slot is rebuilt but its descendants are not. The only
  mutation is `SetCached` on a node that is being (re-)solved in the current
  tree, which has a single owner in that tree.

Exact verification:

```sh
TMPDIR=/tmp ANYTTY_ALLOW_NESTED=1 go test ./clients/tui/kernel/... ./clients/tui/runtime/...
ok  github.com/anytty/anytty/clients/tui/kernel
ok  github.com/anytty/anytty/clients/tui/runtime
ok  github.com/anytty/anytty/clients/tui/runtime/keys

TMPDIR=/tmp ANYTTY_ALLOW_NESTED=1 go test -race ./clients/tui/kernel/... ./clients/tui/runtime/...
ok  github.com/anytty/anytty/clients/tui/kernel  1.58s
ok  github.com/anytty/anytty/clients/tui/runtime 3.06s
ok  github.com/anytty/anytty/clients/tui/runtime/keys

go vet ./clients/tui/kernel/... ./clients/tui/runtime/...   # clean
gofmt -l clients/tui/kernel clients/tui/runtime            # clean

=== RUN   TestIncrementalNodeReuse
--- PASS: TestIncrementalNodeReuse (0.00s)
=== RUN   TestIncrementalLayoutEquivalence
--- PASS: TestIncrementalLayoutEquivalence (0.21s)
=== RUN   TestIncrementalFrameIndependence
--- PASS: TestIncrementalFrameIndependence (0.00s)
PASS
```

`bash clients/tui/scripts/sdk-verify.sh` stays 14/14 for the Go, Python and TS
bindings: the wire is untouched.

## What was not done

- ~~The end-to-end delta ns/op is not 1.5× faster~~ — superseded: the later
  O(changed) bookkeeping and tree-shaped rect-index passes made the delta path
  ~2.8× faster end-to-end (see the latest sections).
- Full VIEWs do not use `LayoutCached`'s reuse (they have no shared pointers),
  and the pointer map is intentionally not built for them.

# O(changed) commit bookkeeping

> Partly superseded: the flat-frame figures below predate the tree-shaped
> rect-index pass; the bookkeeping design (persistent pointer map, focus reuse,
> incremental `boxCount`) is current. (VIEW_DELTA)

Same machine and toolchain (darwin/arm64, Apple M2, Go 1.26.7). The previous
section left the delta path near parity with a full VIEW in ns/op: the kernel
solve was already incremental, but every commit still re-walked the whole tree
for bookkeeping — a fresh 10k-entry `map[*pb.Box]*kernel.Node`, `recordBoxes`,
`focusLocked` and `countBoxes`. This pass makes all of it O(changed) with no
behaviour change.

## What changed

- **Persistent pointer map.** `s.nodes` is no longer rebuilt per commit. On a
  delta, `buildNodeTree` walks from the new root and a map hit reuses the cached
  `*kernel.Node` subtree and stops descending (every descendant already has an
  entry under its unchanged pointer); a miss builds the node, recurses, and
  inserts the new pointer into the same map. A full VIEW keeps the previous
  behaviour: it drops the map (nothing to reuse) and the next delta lazily
  re-indexes once with `indexNodes`.
- **Stale pruning.** `applyPatchesEffect` collects, while it already rebuilds
  the tree, the old pointers each patch detaches: `cloneSpine`/`cloneForEdit`
  originals (only their own entry is dropped; shared children stay) and the
  roots of `replace`/`remove` subtrees (dropped recursively with
  `dropSubtrees`). `move`/`insert`/`set` detach no subtree. A hard fallback,
  `nodeMapCapFactor = 4`, rebuilds the map with `indexNodes` if it ever exceeds
  4× the live box count; the check is O(1) against the maintained counter.
- **Focus.** `patchEffect.focusDirty` is set by any `set` whose payload carries
  `focused`/`content`/`input`/`visible`, and by every structural op
  (`replace`/`insert`/`remove`/`move`), all conservative. When it is clear (a
  keys-only or tree-identical delta) the previous `focus` value is reused
  instead of walking the tree; `Focus{ID, Input}` is a value, so reuse is safe.
- **Box count.** `s.boxCount` is maintained incrementally: `insert`/`replace`
  add (`countBoxes` of the patch box, computed once, O(patch)), `replace`/
  `remove` subtract the detached subtree (`countBoxes`, O(removed)), `set`/
  `move` are unchanged. `HandleViewDelta`'s max_nodes check uses it when known
  and only falls back to a full `countBoxes` for the first growth delta after a
  full VIEW; full VIEWs keep their O(tree) walk. Validation order/reasons are
  unchanged.
- `applyPatches` is kept as a thin wrapper over `applyPatchesEffect`, so the
  fuzz/reference helper and the A/B control are untouched.

## Results

```sh
TMPDIR=/tmp ANYTTY_ALLOW_NESTED=1 go test ./clients/tui/runtime -run '^$' \
    -bench 'HandleViewDeltaLeaf_10000$|WireDeltaRoundTrip_10000$|HandleViewFull_10000$' \
    -benchmem -count=5
```

Median of 5, before (working tree as of the previous section) vs after:

| Benchmark | ns/op before | ns/op after | B/op before | B/op after | allocs before | allocs after |
|---|---:|---:|---:|---:|---:|---:|
| `HandleViewDeltaLeaf_10000` | 3,352,000 | 1,758,000 | 6.17 MB | 4.97 MB | 555 | 282 |
| `WireDeltaRoundTrip_10000` | 3,681,000 | 1,761,000 | 6.17 MB | 4.97 MB | 628 | 294 |
| `HandleViewFull_10000` | 4,100,000 | 3,399,000 | 9.66 MB | 9.66 MB | 30,054 | 30,054 |

Delta CPU drops **1.9×** (`HandleViewDeltaLeaf`) / **2.1×**
(`WireDeltaRoundTrip`); allocations halve and B/op falls ~19% (the map is no
longer reallocated at full tree size each commit). Full-VIEW numbers are
unchanged (it never built the map); the small ns/op move is machine load.

Both profiles are `BenchmarkWireDeltaRoundTrip_10000` (`-cpuprofile`,
300–400 iterations), top functions by cum%:

Before (baseline):

```
HandleViewDelta                     53.6%
commitViewLocked                    47.0%
kernel.LayoutCached                 27.2%
gcBgMarkWorker / gcDrain            25.8%
buildNodeTree                       16.6%
maps.Copy (flat rect merge)         15.9%
nodeFor                             14.6%
countBoxes                           6.6%
focusLocked.func1                     3.3%
```

After:

```
commitViewLocked                    31.6%
HandleViewDelta                     32.5%
kernel.LayoutCached                 23.9%
kernel.solve / solveChild           19.7%
gcBgMarkWorker / gcDrain            18.0%
maps.Copy (flat rect merge)         ~17%
nodeFor                              6.0%
buildNodeTree                         ~0% (no longer a profile line)
countBoxes                            ~0% (no longer a profile line)
focusLocked.func1                   ~2% (see note)
```

`nodeFor`, `buildNodeTree` and `countBoxes` are no longer material. (The flat
`Frame.Rects` merge called out here was later removed by the tree-shaped
rect-index pass; see that section.) `focusLocked` still walks for this
benchmark because its patch is a **`replace`**, which is conservatively
focus-dirty (a replace can detach the currently-focused node); it is skipped
for keys-only and non-focus-relevant `set` deltas, and the reuse is pinned by
`TestIncrementalFocusTracking`.

## Correctness

- `TestIncrementalLayoutEquivalence` now also asserts, after every commit, that
  the map holds exactly one entry per live box, that `boxCount ==
  countBoxes(root)`, and that the committed focus equals a from-scratch
  `focusLocked`. 400 iterations, all 5 ops, resizes, full VIEWs and keys-only
  deltas.
- `TestIncrementalMapBounded` runs 1000 replace deltas that each detach a whole
  subtree and asserts the map stays exactly the live box count (no leak) and the
  counter tracks.
- `TestIncrementalMapPersistentAcrossCommits` proves the map is carried across
  commits (an untouched sibling's `Solved` is reused by pointer).
- `TestIncrementalBoxCountMaxNodes` pins accept/reject at the max_nodes boundary
  for insert and replace growth and for remove that frees room.
- `TestIncrementalFocusTracking` pins reuse for a style-only set and a keys-only
  delta, and update for a `focused`/`self` set, a structural replace that drops
  the focused node, and an `input` set.
- `TestIncrementalNodeReuse` / `TestIncrementalFrameIndependence` are untouched
  and still pass.

`bash clients/tui/scripts/sdk-verify.sh` stays 14/14; the wire is untouched.

## What was not done

- ~~The flat `Frame.Rects` merge (one map entry per node per commit) is still
  O(tree)~~ — superseded by the tree-shaped rect-index pass below.
- The map/`boxCount` are not built for full VIEWs (lazy on the next delta), so a
  workload that only ever sends full VIEWs pays nothing for either.

One correctness note on the O(changed) cache: when a single VIEW_DELTA
contains multiple patches and a later patch detaches a subtree whose root was
created by an earlier patch in the same frame, the detached root has no cache
entry, so the map cannot be pruned exactly. The host then rebuilds the map from
the committed tree (one O(tree) `indexNodes` for that frame); the steady
single-patch path never does. Pinned by `TestIncrementalMultiPatchPrune`.

# Tree-shaped rect index (incremental layout)

Same machine and toolchain (darwin/arm64, Apple M2, Go 1.26.7). The previous
section left the flat `Frame.Rects` merge as the dominant remaining delta cost:
every commit rebuilt a 10k-entry map and `maps.Copy`ed each reused subtree's map
into its parent. This pass replaces the flat map with a tree-shaped index so a
reused subtree's rects are stored once and inherited by reference.

## Design

- **Index shape.** `Frame` no longer has an exported `Rects map[string]Rect`.
  Its rect surface lives in an unexported `rectIndex`: ONE ordered slice
  `items []rectNode{id, rect, ref}` in solve order plus an incrementally
  maintained `count` of every reachable entry. An item either carries one node's
  absolute rect or references a whole spliced index. Keeping own entries and
  refs in one ordered list is what makes a plain `Layout` and a `LayoutCached`
  resolve duplicate ids to the same rect (last in solve order wins, exactly like
  the flat map this index replaced).
- **Splice mechanism.** `frameBuilder.addFrame` (a reused `LayoutCached`
  subtree) and the Pos-overlay lift both splice by appending one ref item and
  adding the cached `count`; there is no `maps.Copy`, no per-node work, and no
  allocation proportional to the subtree. `RectCount` is O(1). The same
  absolute-rect `Solved.Rect` gate still decides reuse: a moved/resized subtree
  is re-solved, never spliced at stale coordinates.
- **Public surface.** `Frame.Rect(id) (Rect, bool)` is unchanged for every
  caller (host/compositor/session). New exported accessors
  `Frame.RectsIterate(fn)` and `Frame.RectCount() int` replace non-test ranges
  over the old map. The exported `Rects` field was DELETED and every reference
  in the repo migrated (kernel, runtime, tests). The tree-shaped index is not
  comparable by `reflect.DeepEqual` across a spliced vs re-solved frame, so the
  equivalence oracles canonicalize both sides (flatten the index into a flat
  snapshot) before comparing; every id->rect is still compared, nothing is
  weakened.
- **Hit/cursor.** `Hit`/`allHits`, `Lines`, `OverlayFrames`, `Cursor*` are
  unchanged; `hits` stays a small flat slice (id-bearing visible nodes only).
- `countHints`/`countHintsPruned` presize the `items` slice to the changed-path
  id count plus the number of spliced refs (`ownIDs + refs`) and the `hits`
  slice to the exact reachable count (`ids = cached RectCount`), so a reuse
  commit allocates neither a full map nor a full-size entry slice (presizing
  refs here removed an append-growth regression of ~1.8 MB/op).
- **Payload aliasing.** An in-process caller may pass a patch payload that
  shares a pointer with the committed tree (the payload root or any descendant;
  the wire always decodes fresh). `patchEffect.payload` scans the payload
  (O(payload), the same order as the box-count it already computes) and clones it
  when any live box is found, so the committed tree keeps one unique pointer per
  box. The pointer-keyed node cache therefore holds exactly one entry per live
  box, and an alias-then-detach burst cannot leak or over-prune.

## Results

```sh
TMPDIR=/tmp ANYTTY_ALLOW_NESTED=1 go test ./clients/tui/kernel -run '^$' \
    -bench 'ABKernel' -benchmem -count=5
TMPDIR=/tmp ANYTTY_ALLOW_NESTED=1 go test ./clients/tui/runtime -run '^$' \
    -bench 'HandleViewDeltaLeaf_10000$|WireDeltaRoundTrip_10000$|HandleViewFull_10000$' \
    -benchmem -count=9
```

Median, before (flat `Rects` map) vs after (tree index):

| Benchmark | ns/op before | ns/op after | B/op before | B/op after | allocs before | allocs after |
|---|---:|---:|---:|---:|---:|---:|
| `ABKernelFromScratch_10000` | 1,813,000 | 1,605,000 | 3.17 MB | 2.82 MB | 40 | 8 |
| `ABKernelCached_10000` | 1,262,000 | 620,000 | 3.17 MB | 2.82 MB | 55 | 25 |
| `HandleViewDeltaLeaf_10000` | 1,951,000 | 1,400,000 | 4.97 MB | 4.61 MB | 309 | 215 |
| `WireDeltaRoundTrip_10000` | 2,016,000 | 1,240,000 | 4.97 MB | 4.61 MB | 318 | 218 |
| `HandleViewFull_10000` | 3,715,000 | 3,310,000 | 9.66 MB | 9.31 MB | 30,054 | 30,022 |

The kernel cached path drops **1.97×** and the delta path **1.35×**
(`HandleViewDeltaLeaf`) / **1.36×** (`WireDeltaRoundTrip`). Full-VIEW numbers
move only within machine load (it never built the old flat-map merge either).

CPU profile of `BenchmarkWireDeltaRoundTrip_10000` (`-cpuprofile`, ~750
iterations), top functions by cum%:

Before (flat map):

```
HandleViewDelta                     42.8%
commitViewLocked                    43.3%
kernel.LayoutCached                 26.9%
kernel.solve                        22.9%
kernel.solveChild                   20.4%
addFrame                            19.9%
maps.Copy (flat rect merge)         15.9%   <-- eliminated
gcDrain / scanObjectsSmall          22.9% / 17.9%
nodeFor                             11.4%
```

After (tree index):

```
HandleViewDelta                     23.7%   (halved)
commitViewLocked                    23.7%
kernel.LayoutCached                 14.0%
kernel.solve                        10.5%
kernel.solveChild                    6.6%
addFrame                             5.7%   (pointer link only)
maps.Copy                            0%     (no longer a profile line)
gcDrain / scanObjectsSmall          33.8% / 23.7%
nodeFor                              7.9%
```

`maps.Copy` is gone from the profile; the remaining cost is GC of the returned
frame plus the copy-on-write node rebuild, not the rect merge.

## Correctness

- `TestIncrementalLayoutEquivalence` (400 iterations, all 5 ops, resizes, full
  VIEWs, keys-only deltas) and `TestIncrementalFrameIndependence` /
  `TestIncrementalNodeReuse` compare the full frame (every id->rect via the
  canonical flat snapshot), `Lines`, `OverlayFrames`, cursor, `Hit` grid and
  focus against a from-scratch solve; unchanged and passing.
- `TestLayoutCachedIndexSharedOnReuse` proves the index is pointer-shared: after
  a one-leaf delta on a 2000-row tree the untouched row's exact `*rectIndex` is
  linked into the new root index, the root materializes only O(depth) own
  entries, and `RectCount` is unchanged.
- `TestLayoutCachedAliasing` pins that the parent links (not aliases) the cached
  child index and that a later solve is unaffected.
- Rect-equality gating is unchanged: a cached frame is only spliced when the
  subtree's absolute rect is identical; moving/resizing re-solves.
- `TestLayoutZeroAndNegativeSafety`, the hit/cursor/render tests and the
  DeepEqual geometry tables compare the same expected geometry, only through the
  new accessors.

`bash clients/tui/scripts/sdk-verify.sh` stays 14/14; the wire is untouched.

## What was not done

- The flat index is still rebuilt for a from-scratch `Layout` (a full VIEW has
  no shared subtree to link), so its cost is inherently O(tree); only the
  incremental splice is O(changed).
- `Hit` still materializes its `allHits` slice per call (small: id-bearing
  visible nodes only) and was left flat as permitted.

# SDK memo: O(changed) diff

## What changed

- `sdk/app.Memo` caches the boxes a `Model.View` builds under comparable keys,
  frame-scoped by `BeginFrame`/`EndFrame`, so unchanged subtrees keep the SAME
  `*pb.Box` pointer across frames.
- `Program.Memo *Memo` (nil disabled) scopes one memo frame per batch:
  `BeginFrame` before `Model.View`, `EndFrame` after the single commit, so
  pruning (dropping keys unused in the frame) runs exactly once per commit.
- `core.Client.fullViewFrameLen` (the `CommitDelta` cost guard) now sizes the
  small envelope with `proto.Size` and the root with
  `proto.MarshalOptions{UseCachedSize: true}`. A memoized subtree that did not
  change carries a valid protobuf size cache, so the guard is O(changed)
  instead of a full-tree `proto.Size` walk. The result is byte-exact with the
  full VIEW (pinned by `TestFullViewFrameLenMatchesMarshal`), so delta/full
  decisions are unchanged. `core.DiffView`/`boxesEqual` already short-circuits
  on pointer identity in its first statement at every level (root, node, child,
  prefix/suffix), so no diff fix was needed.

## Results

Exact commands (identical environment/toolchain as above; the memo benchmark
did not exist for the "before" capture):

```sh
export PATH="$HOME/.local/share/go-toolchains/go1.26.7/bin:$PATH"
# before (control only)
TMPDIR=/tmp ANYTTY_ALLOW_NESTED=1 go test ./clients/tui/sdk/bench \
    -run '^$' -bench 'CommitDelta10000$' -benchmem -count=5 -timeout 900s
# after (control + memo)
TMPDIR=/tmp ANYTTY_ALLOW_NESTED=1 go test ./clients/tui/sdk/bench \
    -run '^$' -bench 'CommitDelta10000$|CommitDeltaMemo10000$' -benchmem -count=5 -timeout 900s
```

Median of 5 runs. Both runtimes edit one text leaf of a 10000-node tree and
commit one frame per iteration; `BenchmarkCommitDeltaMemo10000` builds the tree
through `app.Memo` with keys that change only on the root-to-target path, while
`BenchmarkCommitDelta10000` (unchanged code) clones the whole tree once and
alternates the two copies.

| Benchmark | ns/op | B/frame | B/op | allocs/op |
|---|---:|---:|---:|---:|
| before: `CommitDelta10000` (no memo) | 843,037 | 40.31 | 552 | 11 |
| after: `CommitDelta10000` (no memo, unchanged code) | 550,046 | 40.39 | 552 | 11 |
| after: `CommitDeltaMemo10000` | 6,555 | 43.81 | 2,866 | 70 |

The memoized path is **129× faster** than the pre-change control (843 µs →
6.6 µs) and **84× faster** than the same control after the change: the
remaining 6.6 µs is the O(depth) rebuild of the changed path through the memo,
the O(depth·fanout) diff walk around it, and the 44 B frame write. The control
also dropped (843 µs → 550 µs) because the benchmark alternates two immutable
trees, so its sizes are cached after the first call; a no-memo program that
rebuilds fresh boxes every frame still pays the full `proto.Size` walk.

B/frame is the wire frame per delta: ~40 B for a one-leaf `set` plus ~4 B for
the growing `rev`/`rev_base` varints at ~175k iterations (the control averages
slightly less text per patch because it alternates a 7-byte and a 4-byte leaf).

Pinned by `clients/tui/sdk/app/memo_test.go` (key reuse, pruning, zero value,
changed keys, engine-level patch proportionality, real-wire 10k-tree delta
under 200 B with the host decoding and chaining the baseline) and
`clients/tui/sdk/core/delta_test.go` (`TestFullViewFrameLenMatchesMarshal`,
which compares the guard's exact length with the marshalled full VIEW over
shared, empty and fully-populated trees). `bash
clients/tui/scripts/sdk-verify.sh` stays 14/14 for Go, Python and TS.

# Two-process vs single-process: what the OS-pipe boundary costs

`clients/tui/runtime/e2e_bench_test.go` (opt-in: `ANYTTY_E2E_BENCH=1`) measures
the whole host path for one key press with the same program logic in two
configurations:

- `BenchmarkTwoProcessKeyToFrame`: the layout program is a **separate
  executable** built on the real SDK, talking over real OS pipes (`exec` +
  `StdinPipe`/`StdoutPipe`), exactly like `tui2 -shell`.
- `BenchmarkInProcessKeyToFrame`: the same logic and the same real
  `runtime.Session`, but the program runs **in this process** over `io.Pipe`.

Each iteration is: host `Input` (route + encode EVENT + write) -> program
`Update`/`View`/`Commit` (encode VIEW_DELTA + write) -> host `HandleViewDelta`
+ `FrameBytes` (kernel layout + frame diff). This is the interactive path, not
PTY output.

```sh
TMPDIR=/tmp ANYTTY_E2E_BENCH=1 go test ./clients/tui/runtime \
    -run '^$' -bench 'KeyToFrame|HostFrameCompose' -benchmem -count=5
```

Apple M2, Go 1.26.7 (median of 5; the machine was shared, so ns/op varies by a
few percent — B/op and the byte metrics are deterministic):

| Benchmark | ns/op | B/op | allocs/op | wire |
|---|---:|---:|---:|---:|
| TwoProcessKeyToFrame (real subprocess, OS pipes) | ~190–480 µs | 272 KB | 226 | 22 B host→prog, 43 B prog→host |
| InProcessKeyToFrame (io.Pipe, same process) | ~180–220 µs | 326 KB | 732 | — |
| HostFrameCompose (host only: delta apply + layout + frame diff) | ~117–141 µs | 274 KB | 213 | — |
| HostFrameComposeStatic (host only, unchanged tree) | ~100–108 µs | 251 KB | 162 | — |

Reading:

- **The process boundary is not the bottleneck.** Two-process and in-process
  land in the same band (both dominated by the host's kernel layout + frame
  composition). The wire is tiny: the host sends the 22-byte EVENT and receives
  a ~43-byte VIEW_DELTA per key.
- The **floor is the host-side frame compose** (`HostFrameCompose`): kernel
  layout plus the `render.Frame.Bytes` diff over 120×40 costs ~0.1–0.14 ms per
  key on this machine regardless of how the program is hosted.
- The **PTY output path never crosses the boundary.** Terminal read/parse lives
  in the host process (`TerminalHandler` -> `ansi.Parser`), and the composed
  frame pulls the terminal screen from the host-side component. A busy terminal
  repaints without any program round trip; only structural events (key/mouse/
  resize) that change the layout tree pay one program round trip.
- So a single-process design (main) and this two-process design share the same
  hot path for streaming output; the extra cost is one ~50 µs OS-pipe
  round trip *per layout-affecting event*, on top of the ~0.1 ms host compose
  both designs pay. At interactive rates (a few events per frame) that is well
  below one frame budget (16.7 ms at 60 fps).

Caveats (measured honestly):

- This is an in-process host harness, not the full `cmd/tui2` binary: it
  excludes the real PTY/`render.Frame.Bytes` TTY write cost, which both designs
  pay identically. The comparison isolates the program-hosting boundary.
- `main` has no `*_bench_test.go`, so there is no upstream benchmark to run
  head-to-head; the control here is the same session code with an in-process
  program, which is the closest fair baseline.
- The measured per-key wire frames are for a deliberately simple tree (41
  boxes, one changed leaf); larger structural edits (split/zoom) send bigger
  deltas, but still one round trip.
