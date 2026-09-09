import { strict as assert } from 'node:assert';
import { createRequire } from 'node:module';
import { fileURLToPath, pathToFileURL } from 'node:url';
import path from 'node:path';

const root = fileURLToPath(new URL('../', import.meta.url));
const require = createRequire(path.resolve(process.argv[2], 'package.json'));
const { chromium } = require('playwright-core');
const directory = path.join(root, 'artifacts/browser-perch');
const browser = await chromium.launch({ headless: true, executablePath: '/Applications/Google Chrome.app/Contents/MacOS/Google Chrome' });
try {
  for (const viewport of [{ width: 1280, height: 800 }, { width: 375, height: 812 }, { width: 844, height: 390 }]) {
    const page = await browser.newPage({ viewport, deviceScaleFactor: 2 });
    const errors = [], requests = [];
    page.on('pageerror', e => errors.push(e.message));
    page.on('request', r => { if (/^https?:/.test(r.url())) requests.push(r.url()); });
    await page.goto(pathToFileURL(path.join(directory, 'index.html')).href);
    await page.waitForFunction(() => loaded);
    await page.evaluate(() => { elapsed = 0; stop(); draw(); });
    const first = await page.locator('canvas').screenshot();
    await page.evaluate(() => { elapsed = 1540; draw(); });
    const blink = await page.locator('canvas').screenshot();
    assert.notDeepEqual(first, blink);
    await page.evaluate(() => { elapsed = 4900; draw(); });
    assert.notDeepEqual(first, await page.locator('canvas').screenshot());
    assert.equal(await page.evaluate(() => document.documentElement.scrollWidth > innerWidth), false);
    await page.screenshot({ path: path.join(directory, `preview-${viewport.width}-light.png`), fullPage: true });
    await page.locator('#dark').check();
    await page.screenshot({ path: path.join(directory, `preview-${viewport.width}-dark.png`), fullPage: true });
    const position = await page.locator('.field').boundingBox();
    await page.locator('#query').fill('localhost:3000');
    await page.waitForTimeout(170);
    assert.equal(await page.locator('canvas').evaluate(c => getComputedStyle(c).opacity), '0');
    assert.deepEqual(await page.locator('.field').boundingBox(), position);
    await page.locator('#query').press('Enter');
    assert.match(await page.locator('#result').textContent(), /localhost:3000/);
    await page.locator('#query').fill('');
    await page.locator('h2').click();
    await page.waitForTimeout(170);
    assert.equal(await page.locator('canvas').evaluate(c => getComputedStyle(c).opacity), '1');
    await page.emulateMedia({ reducedMotion: 'reduce' });
    await page.waitForTimeout(50);
    const still = await page.locator('canvas').screenshot();
    await page.waitForTimeout(300);
    assert.deepEqual(await page.locator('canvas').screenshot(), still);
    assert.deepEqual(errors, []);
    assert.deepEqual(requests, []);
    await page.close();
    console.log(`PASS ${viewport.width}x${viewport.height}: native frames, focus/input, light/dark, reduced motion, offline.`);
  }
} finally { await browser.close(); }
