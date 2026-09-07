# Rive Rig Review, Revision 4

Open `index.html` directly for the browser interaction, or `scenarios.html` for
all nine scene animations. Both run offline with the official Rive WebGL2
runtime, pinned to 2.42.0. They are review assets, not a Flutter integration or
a project saved in the online Rive editor. The previous samples remain intact.

## Rig

- `anytty-mascot.riv`: connecting, processing, history, success, failure,
  welcome, running, searching and wake; the existing `Mascot` trigger contract.
- `browser-back.riv` and `browser-front.riv`: synchronized layers behind and
  in front of the real input. Hands cover its upper edge; the field covers the
  torso region. The opaque black body uses the original outer contour, with the
  neck and lower belly visible around the field. Feet and tail emerge below y=128.
- Tail: one RootBone, three child Bones, one Skin, four Tendons and 165 weighted
  mesh vertices. Root-region weights are pinned to the fixed first bone.
- Head, pupils, eyelids and limbs use independent Rive control nodes. Rigid
  parts retain their vector contours; they are not unnecessarily mesh-warped.
- The tail texture is a transparent 3x rendering of the approved traced SVG,
  not the old PNG with a white matte. It is a skinned raster, not a vector skin.
- Existing free-body pose extensions are preserved for comparison, not newly
  approved logo artwork. The browser keeps the original head and hand contours.

Browser timelines are `idle`, `gaze-x`, `gaze-y`, `head-follow`, and `placement`.
Gaze/head timelines map one second to -1..1; placement maps one second to
0..1460 artboard units. The 1600-wide artboard is drawn at logical 1:1 scale and
clipped by the responsive canvas. Placement is sampled only for responsive
right-edge anchoring: focus, typing and blur never move the character sideways.
The old travel/stepping interaction is removed. Input coordinates only direct
the gaze. JavaScript feeds timeline positions; it does not deform artwork or
animate SVG/CSS parts. Browser interaction uses timeline composition rather
than numeric state-machine inputs. A Flutter adapter must preserve that contract.

The decorative idle stops after 14 seconds. Interaction can wake it; pause,
reduced motion, offscreen visibility and page visibility are respected. The
input remains usable throughout. The review form never navigates or sends data.

## Rebuild

From the public repository, using the existing temporary authoring directory:

```sh
node scripts/build-mascot-rigged.mjs <authoring-directory>
node <authoring-directory>/call.mjs batch docs/assets/brand/motion/rigged/create-jobs.json
node scripts/finalize-mascot-rigged.mjs <authoring-directory>
node scripts/build-mascot-rigged-preview.mjs <authoring-directory>
node scripts/check-mascot-rigged.mjs <authoring-directory>
```

The authoring directory supplies the public MCP bridge, `rive-mcp-server`,
`@modelcontextprotocol/sdk`, `@rive-app/webgl2@2.42.0`, `lucide-static`,
`playwright-core`, and the Rive license. The third-party authoring implementation
is neither copied nor modified; only its public API and generated assets are used.
`body-contour.json` is the checked-in vector import of the original body outline
from the earlier pose study (`artifacts/browser-hanging/body-contour.svg`), not
the convex-hull body used by some earlier full-body pose extensions.

The finalizer must run once on freshly created files. It normalizes the generated
mesh triangle field to the varuint encoding read by the official runtime, then
pins root weights through the public editing API. It validates the exact target
field and index range before rewriting. See the official runtime's
[mesh decoder](https://github.com/rive-app/rive-runtime/blob/main/src/shapes/mesh.cpp).
Skipping this step produces missing triangles in the production renderer.

`rig-validation.json` records bone/skin counts and lint results. Browser QA
screenshots and `report.json` are under `artifacts/mascot-rigged/`: desktop,
phone, landscape, 36 tail-contact/opaque-body samples, visible deformation, no
tail matte, fixed placement during focus/typing/blur, pause, reduced motion,
offline playback and nine scene views.
Visual acceptance and on-device Flutter performance testing remain separate.
