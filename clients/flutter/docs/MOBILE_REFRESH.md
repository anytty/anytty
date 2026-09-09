# Mobile Brand, Recovery And File Previews

## Implemented

- The approved mascot is bundled in `assets/brand/mascot.png` and shared by the
  device header, settings, browser new tab, connection and preview loading states.
- The browser new tab retains real search, bookmarks and history. Its device
  switch action uses the active endpoint, not a sample or marketing destination.
- Native loading motion uses a small fixed-box stride animation. It stops for
  reduced motion and inactive ticker modes. No animated GIF download is required.
- Foreground recovery rechecks OS connectivity with a bounded wait. Wi-Fi to
  mobile transitions are signaled even when both transports report connectivity.
- The file manager retains and observes its endpoint session. Backgrounding
  makes the old directory non-interactive. A replacement session reloads the
  current path. Read/recovery waits have a 20-second deadline, retry and exit.
- File mutations are not replayed after timeout. The user is told that their
  outcome is unknown and must refresh the directory before repeating an action.
- File previews keep their fetch future across rebuilds, expose retry on failure,
  and cancel the UI deadline when dismissed.
- HTML remains source-first. The browser icon opens a separate embedded WebView
  using the preview bytes fetched from the selected endpoint. Returning to source
  preserves the file manager and its session context.
- The WebView also renders Markdown, JSON, CSV/TSV, XML/YAML, diffs and plain text
  using the existing `@file-viewer` text and spreadsheet renderers. The previous
  Capacitor implementation was inspected at commit `2520e6f^`; the same package
  family is still used by `clients/ui/src/files/preview/FileViewerPreview.tsx`.

## Deliberate Limits

- HTML is a static, sandboxed preview: inline styles and embedded images work;
  file scripts, external resources, form submissions and external navigation are
  blocked. Relative sibling assets are not fetched automatically.
- No JavaScript-to-native bridge, credentials or file-system API is exposed to
  documents. The trusted bundled renderer can run JavaScript; user HTML cannot.
- Preview data stays local after retrieval through the current device session.
  No cloud document conversion or third-party viewer service is involved.
- Only complete supported text previews up to the existing 4 MB preview limit
  receive the render action. Truncated sources remain readable as source.
- PDF, Word, Excel binaries, videos and 3D models are not added by this change.
  Supporting those requires the existing bounded file-transfer path and separate
  format/resource verification, not bypassing the current preview safety checks.
- Installed applications are unchanged until a new build is installed. No store
  upload or release is included. Physical-device lock/unlock and WebView checks
  are still required before a production release.

## Build And Check

From the public repository root:

```sh
npm run preview:build
```

This builds the offline renderer into `clients/flutter/assets/file-viewer/`.
The HTML asset is approximately 4.93 MB before application-package compression.
Renderer packages come from existing workspace dependencies. Notices are bundled.
Regenerate this asset after editing `file-viewer/main.tsx` or upgrading its packages.

From `clients/flutter`:

```sh
flutter analyze
flutter test
```

Focused coverage includes `app_lifecycle_controller_test.dart`,
`file_manager_recovery_test.dart`, `anytty_brand_loader_test.dart`,
`browser_new_tab_page_test.dart`, `file_preview_sheet_test.dart` and
`file_web_preview_test.dart`. Recovery tests simulate paused/resumed lifecycle,
replacement sessions and a stalled endpoint; they do not emulate physical radio
or OS process eviction.

With Chrome and Playwright available, run `scripts/check-mobile-file-viewer.mjs`
from the repository root. Set `PLAYWRIGHT_MODULE` if the tool is installed outside
this repository. It checks HTML sandboxing, offline rendering and light/dark
rendering of HTML, Markdown, JSON and CSV.

The local visual review at `clients/flutter/artifacts/mobile-refresh/index.html` contains actual
Flutter widget captures and rendered document test fixtures, not fabricated client
screens. The document fixtures are samples rather than user files.
