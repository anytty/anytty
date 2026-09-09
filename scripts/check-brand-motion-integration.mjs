import { mkdir, writeFile } from 'node:fs/promises';
import { createRequire } from 'node:module';
import path from 'node:path';
import assert from 'node:assert/strict';
const require = createRequire(path.join(process.argv[2], 'package.json'));
const { chromium } = require('playwright-core');
const { createCanvas, loadImage } = await import('@napi-rs/canvas');
const out = path.resolve('artifacts/brand-motion-integration');
await mkdir(out, { recursive: true });
const browser = await chromium.launch({ channel: 'chrome', headless: true });
const report = [];
try {
  for (const [name, url, selector, widths] of [
    ['site', 'http://127.0.0.1:4323/anytty-site/zh-CN/', 'anytty-brand-motion', [375, 1280]],
    ['cloud', 'http://127.0.0.1:4179/login', '[data-brand-motion]', [1280]],
  ]) {
    for (const { width, colorScheme } of widths.flatMap(width => ['light', 'dark'].map(colorScheme => ({ width, colorScheme })))) {
      const page = await browser.newPage({ viewport: { width, height: 900 }, deviceScaleFactor: 2, colorScheme });
      const errors = [];
      page.on('pageerror', error => errors.push(error.message));
      const response = await page.goto(url);
      assert(response.ok(), `${url}: ${response.status()}`);
      const mascot = page.locator(selector).last();
      await mascot.locator('canvas').waitFor({ state: 'visible' });
      await page.waitForTimeout(400);
      const frames = [];
      for (let frame = 0; frame < 2; frame++) {
        const png = await mascot.screenshot();
        const img = await loadImage(png), canvas = createCanvas(img.width, img.height), ctx = canvas.getContext('2d');
        ctx.drawImage(img, 0, 0);
        const pixels = ctx.getImageData(0, 0, img.width, img.height).data;
        let cyan = 0;
        for (let i = 0; i < pixels.length; i += 4) if (pixels[i] < 80 && pixels[i + 1] > 100 && pixels[i + 2] > 180) cyan++;
        assert(cyan > 100, `${name}: actual mascot pixels must be visible`);
        frames.push(png.toString('base64'));
        await page.waitForTimeout(600);
      }
      assert.notEqual(frames[0], frames[1], `${name}: Rive must move`);
      assert.equal(await page.evaluate(() => document.documentElement.scrollWidth > innerWidth), false);
      await page.screenshot({ path: path.join(out, `${name}-${width}-${colorScheme}.png`) });
      await page.emulateMedia({ reducedMotion: 'reduce' });
      await mascot.locator('img').waitFor({ state: 'visible' });
      assert(await mascot.locator('canvas').isHidden());
      assert.deepEqual(errors, []);
      report.push({ name, width, colorScheme, nonblank: true, moving: true, reducedMotion: true });
      await page.close();
    }
  }
  await writeFile(path.join(out, 'report.json'), JSON.stringify(report, null, 2));
  console.log(report);
} finally { await browser.close(); }
