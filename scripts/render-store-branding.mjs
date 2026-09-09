import assert from 'node:assert/strict';
import { createRequire } from 'node:module';
import { mkdir, readFile, writeFile } from 'node:fs/promises';
import { dirname, join } from 'node:path';
import { fileURLToPath, pathToFileURL } from 'node:url';
import { createHash } from 'node:crypto';

const require = createRequire(import.meta.url);
const { chromium } = require(process.env.PLAYWRIGHT_MODULE || 'playwright');
const root = join(dirname(fileURLToPath(import.meta.url)), '..');
const assets = join(root, 'artifacts/store-screenshots/google-play/assets');
const browser = await chromium.launch({ channel: 'chrome', args: ['--allow-file-access-from-files'] });
const report = [];
try {
  const page = await browser.newPage({ viewport: { width: 1024, height: 500 }, deviceScaleFactor: 1 });
  for (const language of ['en', 'zh']) {
    await page.goto(`${pathToFileURL(join(root, 'docs/assets/brand/store/feature-graphic.html'))}?lang=${language}`);
    await page.evaluate(async () => {
      await document.fonts.ready;
      await Promise.all([...document.images].map(image => image.decode()));
    });
    const errors = await page.evaluate(() => [...document.querySelectorAll('.brand-name, .brand-mark, h1, p, .details')].filter(element => {
      const rectangle = element.getBoundingClientRect();
      return element.scrollWidth > element.clientWidth + 1 || rectangle.right > 1024 || rectangle.bottom > 500 || rectangle.left < 0;
    }).map(element => element.id || element.className));
    assert.deepEqual(errors, [], `Feature graphic overflow: ${language}`);
    const output = join(assets, `feature-graphic-${language}.png`);
    const png = await page.screenshot({ path: output });
    assert.equal(png.readUInt32BE(16), 1024);
    assert.equal(png.readUInt32BE(20), 500);
    assert.equal(png[25], 2, 'Feature graphic must have no alpha channel');
    report.push({ path: output.slice(root.length + 1), width: 1024, height: 500, sha256: createHash('sha256').update(png).digest('hex') });
  }
  const source = await readFile(join(root, 'docs/assets/brand/mascot.png'));
  await mkdir(join(root, '.artifacts/brand-refresh-20260907'), { recursive: true });
  await writeFile(join(root, '.artifacts/brand-refresh-20260907/store-render-checks.json'), `${JSON.stringify({ mascotSha256: createHash('sha256').update(source).digest('hex'), outputs: report }, null, 2)}\n`);
  console.log(JSON.stringify(report, null, 2));
} finally {
  await browser.close();
}
