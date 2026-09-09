import { createCanvas, loadImage } from '@napi-rs/canvas';
import { mkdir, readFile, writeFile } from 'node:fs/promises';
import { dirname, join } from 'node:path';
import { fileURLToPath } from 'node:url';
import { deflateSync } from 'node:zlib';
import { createHash } from 'node:crypto';
import assert from 'node:assert/strict';

const root = join(dirname(fileURLToPath(import.meta.url)), '..');
const logoPath = join(root, 'docs/assets/brand/mascot.png');
const background = '#F6F8FA';
const source = await readFile(logoPath);
const sourceHash = createHash('sha256').update(source).digest('hex');
assert.equal(sourceHash, 'de0b8d8d45cd61aeb06fd8a64f162e7ec48bddf8d79f93b563d648ae94b858fb', 'Use the approved website mascot, not a replacement drawing.');
const logo = await loadImage(source);
const check = process.argv.includes('--check');
const outputs = [];
const mobileMascot = join(root, 'clients/flutter/assets/brand/mascot.png');
if (check) {
  assert.deepEqual(await readFile(mobileMascot), source, 'Mobile mascot differs from the approved source');
} else {
  await mkdir(dirname(mobileMascot), { recursive: true });
  await writeFile(mobileMascot, source);
}

const sourceCanvas = createCanvas(logo.width, logo.height);
sourceCanvas.getContext('2d').drawImage(logo, 0, 0);
const sourcePixels = sourceCanvas.getContext('2d').getImageData(0, 0, logo.width, logo.height).data;
let sourceRadius = 0;
for (let y = 0; y < logo.height; y++) {
  for (let x = 0; x < logo.width; x++) {
    if (sourcePixels[(y * logo.width + x) * 4 + 3] > 0) {
      sourceRadius = Math.max(sourceRadius, Math.hypot(x + 0.5 - logo.width / 2, y + 0.5 - logo.height / 2));
    }
  }
}

const crcTable = Array.from({ length: 256 }, (_, index) => {
  let value = index;
  for (let bit = 0; bit < 8; bit++) {
    value = value & 1 ? 0xEDB88320 ^ (value >>> 1) : value >>> 1;
  }
  return value >>> 0;
});

function pngChunk(type, data) {
  const name = Buffer.from(type, 'ascii');
  const length = Buffer.alloc(4);
  length.writeUInt32BE(data.length);
  let crc = 0xFFFFFFFF;
  for (const byte of Buffer.concat([name, data])) {
    crc = crcTable[(crc ^ byte) & 0xFF] ^ (crc >>> 8);
  }
  const checksum = Buffer.alloc(4);
  checksum.writeUInt32BE((crc ^ 0xFFFFFFFF) >>> 0);
  return Buffer.concat([length, name, data, checksum]);
}

function encodeRgbPng(canvas, size) {
  const rgba = canvas.getContext('2d').getImageData(0, 0, size, size).data;
  const scanline = size * 3 + 1;
  const pixels = Buffer.alloc(scanline * size);
  for (let y = 0; y < size; y++) {
    const row = y * scanline;
    pixels[row] = 0;
    for (let x = 0; x < size; x++) {
      const source = (y * size + x) * 4;
      const target = row + 1 + x * 3;
      pixels[target] = rgba[source];
      pixels[target + 1] = rgba[source + 1];
      pixels[target + 2] = rgba[source + 2];
    }
  }
  const header = Buffer.alloc(13);
  header.writeUInt32BE(size, 0);
  header.writeUInt32BE(size, 4);
  header[8] = 8;
  header[9] = 2;
  return Buffer.concat([
    Buffer.from([137, 80, 78, 71, 13, 10, 26, 10]),
    pngChunk('IHDR', header),
    pngChunk('IDAT', deflateSync(pixels, { level: 9 })),
    pngChunk('IEND', Buffer.alloc(0)),
  ]);
}

function render(size, shape) {
  const canvas = createCanvas(size, size);
  const opaque = shape === 'square' || shape === 'play';
  const context = canvas.getContext('2d', { alpha: !opaque });
  context.imageSmoothingEnabled = true;
  context.imageSmoothingQuality = 'high';

  if (opaque) {
    context.fillStyle = background;
    context.fillRect(0, 0, size, size);
  } else if (shape === 'round') {
    context.beginPath();
    context.arc(size / 2, size / 2, size / 2, 0, Math.PI * 2);
    context.clip();
    context.fillStyle = background;
    context.fillRect(0, 0, size, size);
  }

  // Fit every painted pixel inside Android's central 66dp safe circle on a 108dp layer.
  const scale = shape === 'foreground'
    ? (size * 31 / 108) / sourceRadius
    : size * 0.8 / Math.max(logo.width, logo.height);
  const width = logo.width * scale;
  const height = logo.height * scale;
  context.drawImage(logo, (size - width) / 2, (size - height) / 2, width, height);
  return shape === 'square' ? encodeRgbPng(canvas, size) : canvas.toBuffer('image/png');
}

async function writeIcon(relativePath, size, shape = 'square') {
  const output = join(root, relativePath);
  const data = render(size, shape);
  if (check) {
    assert.deepEqual(await readFile(output), data, `Stale branding asset: ${relativePath}`);
  } else {
    await mkdir(dirname(output), { recursive: true });
    await writeFile(output, data);
  }
  outputs.push({ path: relativePath, width: size, height: size, shape, sha256: createHash('sha256').update(data).digest('hex') });
}

await writeIcon('docs/assets/logo.png', 512, 'transparent');
await writeIcon('docs/assets/brand/app-icon-512.png', 512, 'play');
await writeIcon('docs/assets/brand/app-icon-1024.png', 1024);
await writeIcon('artifacts/store-screenshots/google-play/assets/app-icon-512.png', 512, 'play');
await writeIcon('artifacts/store-screenshots/app-store/assets/app-icon-1024.png', 1024);

const android = {
  mdpi: [48, 108],
  hdpi: [72, 162],
  xhdpi: [96, 216],
  xxhdpi: [144, 324],
  xxxhdpi: [192, 432],
};

for (const [density, [legacySize, foregroundSize]] of Object.entries(android)) {
  const directory = `clients/flutter/android/app/src/main/res/mipmap-${density}`;
  await writeIcon(`${directory}/ic_launcher.png`, legacySize);
  await writeIcon(`${directory}/ic_launcher_round.png`, legacySize, 'round');
  await writeIcon(`${directory}/ic_launcher_foreground.png`, foregroundSize, 'foreground');
}

const iosDirectory = 'clients/flutter/ios/Runner/Assets.xcassets/AppIcon.appiconset';
const iosManifest = JSON.parse(await readFile(join(root, iosDirectory, 'Contents.json'), 'utf8'));
const iosIcons = new Map();
for (const image of iosManifest.images) {
  if (!image.filename) continue;
  const points = Number.parseFloat(image.size.split('x')[0]);
  const scale = Number.parseFloat(image.scale);
  iosIcons.set(image.filename, Math.round(points * scale));
}
for (const [filename, size] of iosIcons) {
  await writeIcon(`${iosDirectory}/${filename}`, size);
}

const backgroundPath = join(root, 'clients/flutter/android/app/src/main/res/values/ic_launcher_background.xml');
const backgroundXML = `<?xml version="1.0" encoding="utf-8"?>\n<resources>\n    <color name="ic_launcher_background">${background}</color>\n</resources>\n`;
if (check) assert.equal(await readFile(backgroundPath, 'utf8'), backgroundXML);
else await writeFile(backgroundPath, backgroundXML);

const manifest = JSON.stringify({ source: 'docs/assets/brand/mascot.png', sourceSha256: sourceHash, sourceSize: [logo.width, logo.height], background, outputs }, null, 2) + '\n';
const manifestPath = join(root, 'docs/assets/brand/generated-icons.json');
if (check) assert.equal(await readFile(manifestPath, 'utf8'), manifest);
else await writeFile(manifestPath, manifest);
console.log(`${check ? 'Verified' : 'Generated'} ${outputs.length} AnyTTY icon assets from the approved mascot.`);
