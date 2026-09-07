# Browser Hanging Pose Review

The newer Rive interaction review is at
`docs/assets/brand/motion/rigged/index.html` (public repository root).
It adds four-bone tail skinning, blinking, pointer/caret gaze and the opaque
original black body. Horizontal travel has been removed; the character stays
at the field's right edge even during focus and typing. See that directory's
README for the runtime contract and QA.
The approved Rive files now power `BrowserPerchedMascot` in the Flutter app.
The field retains focus routing to the toolbar. The mascot does not move
horizontally; its three clipped layers reserve 70 logical pixels above and 46
below the field, including when accessibility text enlarges the field.
Native Rive drives blinking and the skinned tail for one 14-second idle cycle.
It pauses when hidden, offstage, backgrounded or reduced motion is enabled.
The notes below describe the earlier pose study, not the production widget.

The earlier static approval step followed feedback that the low cyan perch did
not match the approved logo.

The corrected pose puts the entire character behind the input field rather
than suspending it below with long arms. The approved head angle/proportions,
eyes and nostrils stay above the field. Original hands rotate to grip the upper
border; the long raised arms from the first draft are removed. The opaque field
occludes the black torso, while small feet and the tail emerge below it.
The tail uses its original contour with a rotation; feet reuse the existing
`free-foot` pose-extension contour and still need visual approval.
Source contour checksums are stored with the preview. The body is retraced from
the original outer contour without a convex hull, with old phone/hand regions
filled black; this reconstruction is now hidden by the field.

The rear character is at z-index 0, the opaque field at 1, and only the hands
at 2. Both mascot layers ignore pointers. In the 140 x 174 view box the field
runs from y=70 to y=128; the fingers overlap the top border without entering the
text line. Head space and foot/tail space are reserved. It cannot intercept typing.
No cursor-following, eye-following, blinking or idle loop is implemented here.
After pose approval, the intended interaction is eyes leading toward the text
caret or pointer/touch target, followed by body movement along the border.

From the public repository:

```sh
node scripts/build-browser-hanging-preview.mjs <temporary-authoring-directory>
node scripts/check-browser-hanging-preview.mjs <temporary-authoring-directory>
```

The authoring directory supplies `potrace`, `lucide-static` and `playwright-core`.
Open `artifacts/browser-hanging/index.html` directly. It has light/dark review,
the original PNG comparison, and a local-only test input. The original PNG may
still show its previously identified matte; the new vector artwork does not
embed that raster. QA covers desktop, phone and landscape, source checksums,
vector bounds/pixels, input behavior, static frames and offline asset loading.
