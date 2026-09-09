# Brand Motion Integration

The authoritative assets are `docs/assets/brand/motion/rigged/*.riv`.
No online Rive editor, hosted asset URL or preview-only runtime is needed.

## Production Entry Points

- Flutter: `AnyttyBrandLoader` selects a named Rive timeline via
  `AnyttyMascotScene`. File reconnection uses `connecting`, page/file loading
  uses `processing`, history entry uses `history`, and file errors use `failure`.
  History still uses its existing minimum 700ms mask, not a new artificial delay.
- Flutter browser: `BrowserPerchedMascot` layers the original black torso behind
  the field and hands above it. Feet/tail attach to the actual bottom edge.
  Focus still transfers to the toolbar and hides the mascot; no horizontal travel.
- Public Web: `MachineWorkspace` connection status uses `BrandMotion`.
- Cloud: consumes `@anytty/ui/brand-motion` for loading and the existing login
  illustration. Public client source is not copied into the private repository.
- Website: `brand-motion.astro` adds one short original-pose animation beside
  the existing AnyTTY headline. Product screenshots and page layout are retained.

All nine scene names remain available in both client components: connecting,
processing, history, success, failure, welcome, running, searching and wake.
Unused scene names are not attached to unrelated actions merely to display them.

## Runtime And Fallback

Flutter uses stable `rive 0.14.11` with the Flutter renderer; Web uses pinned
`@rive-app/webgl2 2.42.0`. Web WASM is bundled locally and loaded asynchronously.
Decorative motion plays once. Loading motion loops only while its view is active.
Background/offscreen views pause; reduced motion shows a still pose. Transparent
PNG fallbacks are rendered from the same Rive artwork, not the old matte PNG.

Regenerate distributable assets after rebuilding/finalizing the approved rig:

```sh
node scripts/export-brand-motion.mjs <authoring-tools-directory> \
  ../anytty-site/site/public/assets/brand/motion
```

The tools directory provides `playwright-core` and Chrome must be installed.
The export waits for embedded mesh textures to decode before capturing PNGs.

## Verification

```sh
# From clients/flutter, once per test environment:
dart run rive_native:setup --platform macos
flutter test test/anytty_brand_loader_test.dart test/browser_perched_mascot_test.dart
```

The native test requires real Rive rendering, checks all nine scenes, body pixels,
changing frames, fixed bounds, pause/resume and reduced motion. Existing browser
tests cover phone/tablet/landscape, English/Chinese and 3x accessibility text.

Web component lifecycle tests: `clients/ui/src/brand/BrandMotion.test.tsx`.
Real browser QA: `node scripts/check-brand-motion-integration.mjs <tools-directory>`
with website on port 4323 and Cloud on 4179. Screenshots and pixel/motion results
are written to `artifacts/brand-motion-integration`.
The website's default development base path is `/anytty-site/`.

Cloud retains the 490 KiB initial and 760 KiB application JavaScript budgets.
Its separate 200 KiB Rive budget also asserts the runtime is not an initial import.

## Browser Interaction

The mobile new-tab page uses compact saved-link shortcuts (long-press for removal),
a device switcher and at most three recent pages. Full history remains available.
The mobile toolbar keeps bookmark actions in the overflow menu to leave room for
the address field. `BrowserAddressBar` owns a bounded autocomplete list with an
explicit typed-address option, touch-outside dismissal and Escape cancellation.
Submitting uses one selection/navigation path; tapping the page dismisses editing
without activating content behind the popup. Cancel restores the current URL and
invalidates queued new-tab focus requests. Browser navigation/proxy/session
ownership is unchanged.
