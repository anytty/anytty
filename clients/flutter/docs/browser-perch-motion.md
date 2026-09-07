# Browser Perch Motion

The new-tab companion rests on the search field, with a low rounded cyan body,
front paws over the border, and the approved head/eye contours. The body, paws,
and curled tail are new pose drawings, not changes to the static production logo.
This replaces the static mascot only on the browser new-tab page. Other loaders
and the nine Rive review samples are unchanged.

## Behavior

- Visible only when search and toolbar address fields are empty/unfocused and the keyboard is closed. `about:blank` is treated as empty.
- A 12-second sequence includes two blinks, a small head/gaze shift, and a short tail movement, then rests. It does not imply loading or network status.
- Focus or entered text hides it in 140ms. The fixed field position and pointer-transparent mascot allow typing immediately.
- Reduced motion shows a static pose. Background/inactive App states and disabled TickerMode stop the controller; disposal releases it.
- The App uses Flutter vector paths and an AnimationController, with no new dependency. This is not a Rive runtime integration or a GIF.

## Rebuild And Review

From the public repository, regenerate approved contours with:

```sh
node scripts/build-browser-mascot-paths.mjs
```

From `clients/flutter`, run:

```sh
flutter test test/browser_perched_mascot_test.dart test/browser_new_tab_page_test.dart test/browser_session_test.dart test/browser_load_progress_test.dart
```

Set `ANYTTY_BRAND_PREVIEW_DIR` to the repository's absolute `artifacts/browser-perch`
directory and `ANYTTY_BRAND_PREVIEW_FONT` to an available Unicode font when running
the tests to export actual Flutter screenshots and a native-rendered sprite sheet.
Then, from the repository root:

```sh
node scripts/build-browser-perch-preview.mjs <temporary-authoring-directory>
node scripts/check-browser-perch-preview.mjs <temporary-authoring-directory>
```

The temporary directory supplies `lucide-static` and `playwright-core`; these are
not App dependencies. Open `artifacts/browser-perch/index.html` directly without
a server. The review HTML plays exported native frames and simulates focus/text
behavior; submission is local only and does not navigate. Actual App screenshots
are linked separately. The App itself renders vectors at device resolution.

Checks cover phone/tablet/landscape layouts, Chinese/English, 3x text scaling,
light/dark, pointer passthrough, both input controllers, motion lifecycle,
nonblank/distinct frames, canvas bounds, and white exterior pixels.
No device build or installation is performed by these scripts.
