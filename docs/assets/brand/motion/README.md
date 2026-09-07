# AnyTTY Mascot Motion Samples, Revision 3

Review sample only. This does not replace the Flutter loader or the approved logo.

- `index.html`: offline **Rive player**, with nine scenario tabs, pause/replay, background and size controls, and original-image comparison.
- `anytty-mascot.riv`: vector runtime asset, with no embedded raster images or remote asset requests.
- `scene.json`: editable scene specification, including the hierarchy and animation tracks.
- `vector/`: SVG paths traced from the approved `../mascot.png`, hand-authored free arms/feet for new poses, and their imported Rive scene fragments.
- `source.json`: original source checksum and revision metadata.
- `scenarios.json`: scenario names, durations, repeat behavior, and intended use.
- [SCENE_PLAN.md](SCENE_PLAN.md): scenario-first design, playback rules, and review gates.
- `runtime/`, `preview-data.js`: pinned official Rive Canvas 2.42.0 runtime and embedded WASM for offline review; licenses included.

## Motion Contract

Artboard: `AnyTTY Mascot` (513 x 546), 60 fps. State machine: `Mascot`.
Initial state: `connecting`. Each scenario has a same-named trigger and timeline.

| Scenario | Duration | Repeats | Action |
| --- | --- | --- | --- |
| `welcome` | 2.2s | No | Open-hand wave, head greeting and tail follow-through, without phone |
| `running` | 0.9s | Yes | Alternating legs and arms with body bounce, without phone |
| `searching` | 3.6s | Yes | Crouches and leans to look around, without phone |
| `wake` | 2.6s | No | Drowsy crouch, stretch, then alert standing pose, without phone |
| `connecting` | 4s | Yes | Looks for a connection, lifts phone, signal bars sequence, tail follows |
| `processing` | 3s | Yes | Looks down and alternates taps with both hands, phone lines change |
| `history` | 1.2s | No | Pull-down hand gesture, scrolling lines, glances back |
| `success` | 1.5s | No | Anticipation, head lift and hand acknowledgement, checkmark |
| `failure` | 1.8s | No | Short head shake, lowered phone, warning mark; settles without repeating |

Every state keys the same animated properties, including resets for other screen symbols.
This prevents previous poses, limbs or icons leaking into an interrupted action. Transitions are 120ms.
Loop endpoints match; result states retain their result symbol. Symbols on the mascot phone are
animation cues, not screenshots or depictions of the actual AnyTTY terminal UI.

The white fringe in revision 1 was baked into opaque source pixels. This revision traces clean
color contours instead of carrying those pixels into the runtime. It preserves the recognizable
source silhouette, proportions and palette while reconstructing hidden shoulders/body and phone
surfaces for gestures. Original PNG, production logos, and App loaders remain unchanged.
Revision 3 adds articulated legs, feet and open hands for phone-free poses; these are deliberate
extensions of the character, not shapes present in the original logo. They require visual approval.
The rig uses independent vector parts and transforms, not a 3D or bone-skinned model.
Eye blinks use eye-group squash; pupils are clipped to the eye whites.

## Rebuild

From the public repository:

1. Install `potrace`, `rive-mcp-server`, `@rive-app/canvas@2.42.0`, `lucide-static`, and `playwright-core` in a temporary authoring directory, not the project dependencies.
2. To retrace source paths, run `node scripts/trace-mascot-vectors.mjs <authoring-directory>` and invoke the generated `vector/import-jobs.json` jobs through the authoring tool's public `riv_import_svg` API. These jobs also import the hand-authored `free-arm.svg` and `free-foot.svg` without retracing or overwriting them. Skip this when changing only motion.
3. Run `node scripts/build-mascot-motion.mjs` (delegates to `build-mascot-scenarios.mjs`).
4. Pass `create.json` to `riv_create`, with this asset directory as the working directory.
5. Place the official rive-wasm MIT `LICENSE` at `<authoring-directory>/RIVE-LICENSE`; run `node scripts/build-mascot-preview.mjs <authoring-directory>`.
6. Run `node scripts/check-mascot-motion.mjs <authoring-directory>`. This uses installed macOS Chrome in a separate headless instance and writes screenshots to `artifacts/mascot-motion/`.

The [Potrace authoring tool](https://github.com/tooolbox/node-potrace) traces the source contours.
[rive-mcp](https://github.com/ODU33104/rive-mcp) is third-party freeware, not an official Rive
authoring SDK; its license permits unrestricted generated assets. It was used only through its
public API, and its implementation is not copied into this repository. The official Rive runtime
and Lucide preview icons are distributed with their licenses.

Official Rive runtime loading, state-trigger transitions, screenshots, and canvas-pixel checks
validate this sample on desktop, small-phone portrait, and landscape, in both background modes.
Tests cover distinct scene poses, phone-free poses, nonblank/moving canvases, pause, no external network requests,
reduced motion, off-canvas clipping, white exterior pixels, and matching loop track endpoints.
No claim is made that the generated file round-trips into the Rive web editor as an editable project.
No workspace file has been saved or uploaded to the Rive account.

Before production integration, approve the motion at 48-96 logical pixels, then wire real app
states, pause/dispose, and reduced motion into Flutter. Never delay a successful connection to
finish a loop; do not replay failure indefinitely or replace retry/status text with animation.
History is an entry cue, not an additional mandatory 1.2-second wait. Keep the existing loader
until approval. The review player selects named timelines directly so paused/reduced-motion
samples show the selected pose immediately; the asset also includes all nine state-machine triggers.
Rive 2.42 still supports timeline playback and trigger inputs but marks these APIs deprecated for a
future major release; pin the runtime and plan a data-binding migration before adopting that major.

Legacy revision-1 `parts/`, `preview.apng`, and `blink.png` are retained only as old review material;
they are not loaded by the current player or included in the current `.riv`.
