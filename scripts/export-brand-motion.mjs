import { copyFile, mkdir, writeFile } from 'node:fs/promises';
import { createRequire } from 'node:module';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const root = fileURLToPath(new URL('../', import.meta.url));
const source = path.join(root, 'docs/assets/brand/motion/rigged');
const require = createRequire(path.join(process.argv[2], 'package.json'));
const { chromium } = require('playwright-core');
const targets = [
  path.join(root, 'clients/flutter/assets/brand/motion'),
  path.join(root, 'clients/ui/src/brand/assets'),
  ...(process.argv[3] ? [path.resolve(process.argv[3])] : []),
];
const browser = await chromium.launch({ channel: 'chrome', headless: true });
try {
  const page = await browser.newPage();
  await page.setContent('<canvas></canvas>');
  await page.addScriptTag({ path: path.join(source, 'runtime/rive.js') });
  await page.addScriptTag({ path: path.join(source, 'preview-data.js') });
  for (const [name, key, width, height] of [
    ['anytty-mascot', 'riv', 513, 546], ['browser-back', 'back', 140, 174], ['browser-front', 'front', 140, 174],
  ]) {
    const png = await page.evaluate(async ({ key, width, height }) => {
      const bytes = value => Uint8Array.from(atob(value), c => c.charCodeAt(0));
      const data = window.ANYTTY_MOTION;
      rive.RuntimeLoader.setWasmBinary(bytes(data.wasm).buffer);
      rive.RuntimeLoader.setWasmFallbackUrl(null);
      const runtime = await rive.RuntimeLoader.awaitInstance();
      const file = await runtime.load(bytes(data[key]), undefined, false);
      // Embedded mesh textures finish decoding asynchronously in the Web runtime.
      await new Promise(resolve => setTimeout(resolve, 250));
      const artboard = file.defaultArtboard();
      const canvas = document.querySelector('canvas');
      canvas.width = width * 3; canvas.height = height * 3;
      const renderer = runtime.makeRenderer(canvas, true);
      for (const [name, time] of key === 'riv' ? [['connecting', 0]] : [['idle', 0], ['gaze-x', .5], ['gaze-y', .5], ['head-follow', .5]]) {
        const animation = new runtime.LinearAnimationInstance(artboard.animationByName(name), artboard);
        animation.time = time; animation.advance(0); animation.apply(1); animation.delete();
      }
      artboard.advance(0); renderer.clear(); renderer.save();
      renderer.align(runtime.Fit.fill, runtime.Alignment.topLeft,
        { minX: 0, minY: 0, maxX: canvas.width, maxY: canvas.height },
        { minX: 0, minY: 0, maxX: width, maxY: height });
      artboard.draw(renderer); renderer.restore(); renderer.flush();
      const result = canvas.toDataURL('image/png').split(',')[1];
      artboard.delete(); file.delete(); renderer.delete();
      return result;
    }, { key, width, height });
    for (const target of targets) {
      await mkdir(target, { recursive: true });
      await copyFile(path.join(source, `${name}.riv`), path.join(target, `${name}.riv`));
      await writeFile(path.join(target, `${name}.png`), Buffer.from(png, 'base64'));
    }
  }
  console.log('Exported approved Rive files and transparent static fallbacks:', targets);
} finally { await browser.close(); }
