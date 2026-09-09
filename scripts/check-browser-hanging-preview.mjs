import { strict as assert } from 'node:assert';
import { createRequire } from 'node:module';
import { readFile } from 'node:fs/promises';
import { createHash } from 'node:crypto';
import { fileURLToPath, pathToFileURL } from 'node:url';
import path from 'node:path';

const root = fileURLToPath(new URL('../', import.meta.url));
const require = createRequire(path.resolve(process.argv[2], 'package.json'));
const { chromium } = require('playwright-core');
const directory = path.join(root, 'artifacts/browser-hanging');
const source = JSON.parse(await readFile(path.join(directory, 'source.json')));
assert.equal(source.sha256, createHash('sha256').update(await readFile(path.join(root, source.source))).digest('hex'));
for (const [name, hash] of Object.entries(source.contours)) {
  assert.equal(hash, createHash('sha256').update(await readFile(path.join(root, `docs/assets/brand/motion/vector/${name}.json`))).digest('hex'));
}
const browser = await chromium.launch({ headless: true, executablePath: '/Applications/Google Chrome.app/Contents/MacOS/Google Chrome' });
try {
  for (const viewport of [{ width: 1280, height: 800 }, { width: 375, height: 812 }, { width: 844, height: 390 }]) {
    const page = await browser.newPage({ viewport, deviceScaleFactor: 2 });
    const errors = [], remote = [];
    page.on('pageerror', e => errors.push(e.message));
    page.on('request', r => { if (/^https?:/.test(r.url())) remote.push(r.url()); });
    await page.goto(pathToFileURL(path.join(directory, 'index.html')).href);
    assert.equal(await page.locator('.reference img').evaluate(i => i.complete && i.naturalWidth > 0), true);
    assert.equal(await page.evaluate(() => document.documentElement.scrollWidth > innerWidth), false);
    const bounds = await page.locator('.mascot-back svg').evaluate(svg => {
      const box = svg.getBBox();
      return { x: box.x, y: box.y, right: box.x + box.width, bottom: box.y + box.height };
    });
    assert.ok(bounds.x >= 0 && bounds.y >= 0 && bounds.right <= 140 && bounds.bottom <= 174, JSON.stringify(bounds));
    const pixels = await page.locator('.mascot-back svg').evaluate(async svg => {
      const image = new Image();
      image.src = 'data:image/svg+xml;charset=utf-8,' + encodeURIComponent(new XMLSerializer().serializeToString(svg));
      await image.decode();
      const canvas = document.createElement('canvas'); canvas.width = 280; canvas.height = 348;
      const ctx = canvas.getContext('2d'); ctx.drawImage(image, 0, 0, 280, 348);
      const { data } = ctx.getImageData(0, 0, 280, 348);
      let opaque = 0, whiteExterior = 0;
      for (let y = 1; y < 347; y++) for (let x = 1; x < 279; x++) {
        const p = (y * 280 + x) * 4;
        if (data[p + 3] < 32) continue;
        opaque++;
        const edge = [p - 4, p + 4, p - 1120, p + 1120].some(n => data[n + 3] < 32);
        if (edge && data[p] > 200 && data[p + 1] > 200 && data[p + 2] > 200) whiteExterior++;
      }
      return { opaque, whiteExterior };
    });
    assert.ok(pixels.opaque > 10000);
    assert.equal(pixels.whiteExterior, 0);
    const tailContact = await page.locator('.mascot-back svg').evaluate(async svg => {
      const isolated = svg.cloneNode(true);
      for (const child of [...isolated.children]) if (child.id !== 'lower-tail') child.remove();
      const image = new Image();
      image.src = 'data:image/svg+xml;charset=utf-8,' + encodeURIComponent(new XMLSerializer().serializeToString(isolated));
      await image.decode();
      const canvas = document.createElement('canvas'); canvas.width = 280; canvas.height = 348;
      const ctx = canvas.getContext('2d'); ctx.drawImage(image, 0, 0, 280, 348);
      const { data } = ctx.getImageData(0, 0, 280, 348);
      // Require a solid root across the field edge, including an occluded margin.
      let continuousColumns = 0;
      for (let x = 0; x < 140; x++) {
        let solid = true;
        for (let y = 124 * 2; y <= 132 * 2; y++) {
          if (data[(y * 280 + x) * 4 + 3] < 240) { solid = false; break; }
        }
        if (solid) continuousColumns++;
      }
      return continuousColumns / 2;
    });
    assert.ok(tailContact >= 6, `Tail root must overlap the field without a gap: ${tailContact}px`);
    const mascot = await page.locator('.mascot-back').boundingBox();
    const field = await page.locator('.field').boundingBox();
    assert.equal(field.y, mascot.y + 70);
    assert.equal(await page.locator('.mascot-back').evaluate(el => getComputedStyle(el).pointerEvents), 'none');
    assert.equal(await page.locator('.mascot-front').evaluate(el => getComputedStyle(el).pointerEvents), 'none');
    assert.equal(await page.locator('.mascot-back').evaluate(el => getComputedStyle(el).zIndex), '0');
    assert.equal(await page.locator('.field').evaluate(el => getComputedStyle(el).zIndex), '1');
    assert.equal(await page.locator('.mascot-front').evaluate(el => getComputedStyle(el).zIndex), '2');
    assert.equal(await page.locator('#raised-arms').count(), 0);
    const handsBox = await page.locator('.mascot-front svg').evaluate(svg => { const b = svg.getBBox(); return {top:b.y,bottom:b.y+b.height}; });
    assert.ok(handsBox.top < 70 && handsBox.bottom > 70 && handsBox.bottom < 82);
    for (const id of ['foot-left', 'foot-right', 'lower-tail']) {
      const box = await page.locator(`#${id}`).boundingBox();
      assert.ok(box.y + box.height > field.y + field.height, `${id} must show below the field`);
    }
    for (const dark of [false, true]) {
      await page.locator('#dark').setChecked(dark);
      await page.screenshot({ path: path.join(directory, `${viewport.width}-${dark ? 'dark' : 'light'}.png`), fullPage: true });
      await page.screenshot({ path: path.join(directory, `${viewport.width}-${dark ? 'dark' : 'light'}-tail.png`), clip: { x: mascot.x, y: field.y + field.height - 8, width: 65, height: 42 } });
    }
    await page.locator('#query').fill('localhost:3000');
    await page.locator('#query').press('Enter');
    assert.match(await page.locator('#result').textContent(), /localhost:3000/);
    assert.deepEqual(await page.locator('.field').boundingBox(), field);
    await page.locator('#compare').uncheck();
    assert.equal(await page.locator('.reference').isVisible(), false);
    await page.emulateMedia({ reducedMotion: 'reduce' });
    const before = await page.locator('.mascot-back').screenshot();
    await page.waitForTimeout(250);
    assert.deepEqual(await page.locator('.mascot-back').screenshot(), before);
    assert.deepEqual(errors, []); assert.deepEqual(remote, []);
    await page.close();
    console.log(`PASS ${viewport.width}x${viewport.height}: source contours, bounds, light/dark, input, static, offline.`);
  }
} finally { await browser.close(); }
