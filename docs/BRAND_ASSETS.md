# AnyTTY Brand Asset Inventory

Updated: 2026-09-07. Local assets are prepared; no store submission or app release has been made.

## Approved Source

Use [mascot.png](assets/brand/mascot.png), the approved blue mascot from
`anytty-homepage-code.zip`, entry `anytty-homepage-code/assets/mascot.png`.
Its SHA-256 is `de0b8d8d45cd61aeb06fd8a64f162e7ec48bddf8d79f93b563d648ae94b858fb`.
This is the same artwork used on the website, not a newly drawn terminal symbol.

The original is 417 x 490 pixels. Larger exports meet output dimensions but do
not add source detail. Obtain an approved vector or higher-resolution master
before making large print artwork; do not invent a replacement mascot.

## Updated Locally

| Surface | Asset / source | Status |
| --- | --- | --- |
| Android launcher | `clients/flutter/android/app/src/main/res/mipmap-*` | Five densities, regular, round and adaptive foreground |
| Android adaptive background | `clients/flutter/android/app/src/main/res/values/ic_launcher_background.xml` | Opaque `#F6F8FA` |
| iPhone / iPad / App Store | `clients/flutter/ios/Runner/Assets.xcassets/AppIcon.appiconset` | All existing slots, including opaque 1024 x 1024 |
| README / documentation | [logo.png](assets/logo.png) | New mascot, existing README layout retained |
| Google Play listing icon | `artifacts/store-screenshots/google-play/assets/app-icon-512.png` (local output) | 32-bit PNG, opaque artwork, no pre-applied store mask |
| App Store source icon | `artifacts/store-screenshots/app-store/assets/app-icon-1024.png` (local output) | RGB PNG without alpha |
| Google Play feature graphic | `artifacts/store-screenshots/google-play/assets/feature-graphic-{en,zh}.png` (local outputs) | 1024 x 500; new logo and core / multi-client copy |
| Existing five promotional pages | `artifacts/store-screenshots/2026-09-05/android/promo/rendered` (local outputs) | New masthead logo; original product screenshots retained |
| Marketing repository | `anytty-site/docs/assets/logo.png`, `anytty-site/site/public/assets/app-icon.webp` | Documentation and compatibility asset updated in that repository |

Open `artifacts/brand-refresh/2026-09-07/index.html` locally to inspect the exports.
Generated reviews and device screenshots under `artifacts/` are not distributed
with the source repository; regenerate them or use the retained local outputs.
The [generated manifest](assets/brand/generated-icons.json) records all 35 icon outputs and hashes.

## Preserve Real Product Screens

The original Flutter UI used text and functional icons, not the old brand bitmap.
The subsequent [mobile refresh](../clients/flutter/docs/MOBILE_REFRESH.md) adds
the approved mascot to selected entry points and native loading states.
Existing iOS launch images are empty placeholders; Android's legacy launch screen
is plain. No new splash screen or terminal mockup was introduced.
Raw product screenshots without the old logo were not repainted. Historical device
home-screen captures can still show an old installed-app icon and should be
recaptured from the new binary if reused for marketing.

## Store Publication Checklist

- Google Play Console: replace the main listing icon and English / Chinese feature
  graphics. Check every additional locale and custom store listing for copied old artwork.
- Google Play Console: replace any screenshot carrying an old promotional masthead
  with its corresponding updated image. The five supplied promotional pages are
  Android artwork, not substitutes for correctly sized iPhone / iPad screenshots.
- Android: build and release a new AAB / APK to change the installed launcher icon.
- App Store Connect: upload a new app version containing the updated asset catalog.
  Apple takes the app icon from the build; changing a separate listing image is insufficient.
- App Store Connect: inspect localized screenshots, custom product pages, in-app
  event art and preview videos for embedded old branding. No backend inventory was
  available in this task, so these items are pending inspection, not marked complete.
- Check developer-profile avatars, social profiles, paid campaign art and external
  documents separately. Do not replace a company identity automatically with a product icon.
- Recapture any launch / home-screen images from the new app. Keep terminal, Web,
  TUI and mobile screenshots faithful to the actual client being advertised.

Suggested shared copy:

> One core. Every client.
>
> Sessions keep running on your own machines. Connect from TUI, CLI, Web and mobile.

The Chinese feature graphic uses the approved positioning: multi-client coexistence
and freedom of the core. Existing feature descriptions were otherwise preserved.

## Regenerate And Verify

From the public repository root, with dependencies installed:

```sh
npm run icons
npm run icons:check
node scripts/render-store-branding.mjs
node artifacts/store-screenshots/2026-09-05/android/promo/render.mjs
```

The two renderers require Google Chrome and a locally available `playwright` module.
If it is installed outside this repository, set `PLAYWRIGHT_MODULE` to its absolute
module path. It is a build-time tool, not an application dependency. The existing
promotional renderer also requires the fonts referenced by its template.

Icon checks cover output sizes, source consistency, PNG channel format, opacity,
Google Play's size limit, iOS catalog slots and Android adaptive-mask safety.
Promotional rendering checks image loading, text overflow and layout overlaps.
These checks do not replace a signed app build or physical-device verification.

Pre-change local artwork is retained in `.artifacts/brand-refresh-20260907/`.
The obsolete untracked SVG was moved there as `legacy-logo.svg`, not discarded.

## Official Requirements

- [Google Play preview assets](https://support.google.com/googleplay/android-developer/answer/9866151?hl=en)
- [Google Play icon specifications](https://developer.android.com/distribute/google-play/resources/icon-design-specifications)
- [Apple: Add an app icon](https://developer.apple.com/help/app-store-connect/manage-app-information/add-an-app-icon)
