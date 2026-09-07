import assert from 'node:assert/strict';
import { createCanvas, loadImage } from '@napi-rs/canvas';
import { readFile } from 'node:fs/promises';
import { dirname, join } from 'node:path';
import { fileURLToPath } from 'node:url';

const root = join(dirname(fileURLToPath(import.meta.url)), '..');
const manifest = JSON.parse(await readFile(join(root, 'docs/assets/brand/generated-icons.json'), 'utf8'));
for (const output of manifest.outputs) {
  const buffer = await readFile(join(root, output.path));
  const image = await loadImage(buffer);
  assert.equal(image.width, output.width, output.path);
  assert.equal(image.height, output.height, output.path);
  assert.equal(buffer[24], 8, 'Icons must have 8-bit channels');
  if (output.shape === 'square') assert.equal(buffer[25], 2, 'App Store icons must be opaque RGB PNGs');
  if (output.shape === 'play') {
    assert.equal(buffer[25], 6, 'Google Play icons must be 32-bit PNGs');
    assert.ok(buffer.length <= 1024 * 1024, 'Google Play icon exceeds 1 MB');
  }
  const canvas = createCanvas(image.width, image.height);
  const context = canvas.getContext('2d');
  context.drawImage(image, 0, 0);
  const pixels = context.getImageData(0, 0, image.width, image.height).data;
  let painted = 0;
  let radius = 0;
  for (let y = 0; y < image.height; y++) {
    for (let x = 0; x < image.width; x++) {
      const offset = (y * image.width + x) * 4;
      if (output.shape === 'square' || output.shape === 'play') assert.equal(pixels[offset + 3], 255);
      if (pixels[offset + 3]) {
        painted++;
        radius = Math.max(radius, Math.hypot(x + 0.5 - image.width / 2, y + 0.5 - image.height / 2));
      }
    }
  }
  assert.ok(painted > image.width * image.height * 0.08, `Blank or undersized icon: ${output.path}`);
  if (output.shape === 'foreground') assert.ok(radius <= image.width * 33 / 108, `Android adaptive mask clips the mascot: ${output.path}`);
}

const directory = 'clients/flutter/ios/Runner/Assets.xcassets/AppIcon.appiconset';
const ios = JSON.parse(await readFile(join(root, directory, 'Contents.json'), 'utf8'));
for (const slot of ios.images) {
  const expected = Math.round(Number.parseFloat(slot.size) * Number.parseFloat(slot.scale));
  const output = manifest.outputs.find(item => item.path === `${directory}/${slot.filename}`);
  assert.equal(output?.width, expected, slot.filename);
}
console.log(`Validated ${manifest.outputs.length} icon dimensions, alpha channels, store limits, and Android mask safety.`);
