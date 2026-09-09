import assert from 'node:assert/strict';
import { createRequire } from 'node:module';
import { dirname, join } from 'node:path';
import { fileURLToPath, pathToFileURL } from 'node:url';

const require = createRequire(import.meta.url);
const { chromium } = require(process.env.PLAYWRIGHT_MODULE || 'playwright');
const root = join(dirname(fileURLToPath(import.meta.url)), '..');
const browser = await chromium.launch({ channel: 'chrome', args: ['--allow-file-access-from-files'] });
try {
  const page = await browser.newPage({ viewport: { width: 1280, height: 640 }, deviceScaleFactor: 1 });
  await page.goto(pathToFileURL(join(root, 'docs/assets/brand/social-preview.html')).href);
  await page.evaluate(async () => {
    await document.fonts.ready;
    await Promise.all([...document.images].map(image => image.decode()));
  });
  const output = join(root, 'docs/assets/brand/social-preview.png');
  const png = await page.screenshot({ path: output });
  assert.equal(png.readUInt32BE(16), 1280);
  assert.equal(png.readUInt32BE(20), 640);
  assert.ok(png.length < 1024 * 1024, 'Keep the social preview below 1 MB');
  console.log(`${output} (1280 × 640, ${png.length} bytes)`);
} finally {
  await browser.close();
}
