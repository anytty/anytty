import { strict as assert } from 'node:assert';
import { createRequire } from 'node:module';
import { mkdir, readFile, writeFile } from 'node:fs/promises';
import { createHash } from 'node:crypto';
import { fileURLToPath, pathToFileURL } from 'node:url';
import path from 'node:path';

const root = fileURLToPath(new URL('../', import.meta.url));
const directory = path.join(root, 'docs/assets/brand/motion');
const artifacts = path.join(root, 'artifacts/mascot-motion');
await mkdir(artifacts, { recursive: true });
const requireTools = createRequire(path.resolve(process.argv[2], 'package.json'));
const { chromium } = requireTools('playwright-core');
const scene = JSON.parse(await readFile(path.join(directory, 'scene.json')));
const source = JSON.parse(await readFile(path.join(directory, 'source.json')));
assert.equal(source.sha256, createHash('sha256').update(await readFile(path.join(directory, '../mascot.png'))).digest('hex'));
assert.equal(scene.images?.length ?? 0, 0, 'Rive scene must not embed the white-matted source PNG.');
assert.equal(scene.animations.length, 9);
const freeScenarios = new Set(source.scenarios.filter(s => s.phone === false).map(s => s.name));
assert.deepEqual([...freeScenarios].sort(), ['running', 'searching', 'wake', 'welcome']);
const common = scene.animations[0].tracks.map(t => `${t.target}:${t.property}`).sort();
for (const animation of scene.animations) {
  assert.deepEqual(animation.tracks.map(t => `${t.target}:${t.property}`).sort(), common);
  for (const track of animation.tracks) {
    assert.equal(track.keyframes[0].frame, 0);
    assert.equal(track.keyframes.at(-1).frame, animation.duration);
    for (let i = 1; i < track.keyframes.length; i++) assert.ok(track.keyframes[i].frame >= track.keyframes[i - 1].frame);
    if (animation.loop === 'loop') assert.equal(track.keyframes[0].value, track.keyframes.at(-1).value, `${animation.name}/${track.target} loop seam`);
  }
  if (freeScenarios.has(animation.name)) {
    assert.ok(animation.tracks.find(t => t.target === 'phone' && t.property === 'opacity').keyframes.every(k => k.value === 0));
    assert.ok(animation.tracks.find(t => t.target === 'free-legs' && t.property === 'opacity').keyframes.every(k => k.value === 1));
  }
}
const browser = await chromium.launch({ headless: true, executablePath: '/Applications/Google Chrome.app/Contents/MacOS/Google Chrome' });
const reports = [];
try {
  for (const viewport of [{ width: 1440, height: 900 }, { width: 375, height: 812 }, { width: 844, height: 390 }]) {
    const page = await browser.newPage({ viewport, deviceScaleFactor: 2 });
    const errors = [], remoteRequests = [];
    page.on('pageerror', error => errors.push(error.message));
    page.on('request', request => { if (/^https?:/.test(request.url())) remoteRequests.push(request.url()); });
    await page.goto(pathToFileURL(path.join(directory, 'index.html')).href);
    await page.waitForSelector('body[data-ready=true]');
    assert.equal(await page.evaluate(() => document.documentElement.scrollWidth > innerWidth), false);
    const canvas = page.locator('#mascot');
    const first = await canvas.screenshot();
    await page.waitForTimeout(500);
    const second = await canvas.screenshot();
    assert.notDeepEqual(first, second, 'Rive canvas must move.');
    const poses = [];
    for (const scenario of scene.animations) {
      await page.locator(`[data-scenario=${scenario.name}]`).click();
      await page.getByRole('button', { name: '重播', exact: true }).click();
      await page.waitForTimeout(530);
      await page.getByRole('button', { name: '暂停', exact: true }).click();
      const pixels = await canvas.evaluate(el => {
        const ctx = el.getContext('2d');
        const { data } = ctx.getImageData(0, 0, el.width, el.height);
        let opaque = 0, brightExterior = 0, clipped = 0, greenPixels = 0;
        for (let y = 1; y < el.height - 1; y++) for (let x = 1; x < el.width - 1; x++) {
          const p = (y * el.width + x) * 4;
          if (data[p + 3] < 32) continue;
          opaque++;
          if(data[p+1] > data[p+2]+25 && data[p+1] > data[p]+80)greenPixels++;
          if (x < 2 || y < 2 || x >= el.width - 2 || y >= el.height - 2) clipped++;
          const edge = [p - 4, p + 4, p - el.width * 4, p + el.width * 4].some(n => data[n + 3] < 32);
          if (edge && data[p] > 100 && data[p + 1] > 100 && data[p + 2] > 100) brightExterior++;
        }
        return { opaque, brightExterior, clipped, greenPixels, width: el.width, height: el.height };
      });
      assert.ok(pixels.opaque > 1000, `${scenario.name} must be visible.`);
      assert.equal(pixels.brightExterior, 0, `${scenario.name} must not have a white matte.`);
      assert.equal(pixels.clipped, 0);
      if(freeScenarios.has(scenario.name))assert.equal(pixels.greenPixels,0,'New poses must not contain the phone.');
      const darkPath = path.join(artifacts, `${viewport.width}-${scenario.name}-dark.png`);
      await page.screenshot({ path: darkPath, fullPage: true });
      const pose = await canvas.screenshot();
      poses.push(createHash('sha256').update(pose).digest('hex'));
      await page.waitForTimeout(150);
      assert.deepEqual(await canvas.screenshot(), pose, 'Pause must freeze the real player.');
      await page.getByLabel('浅色', { exact: true }).check();
      await page.screenshot({ path: path.join(artifacts, `${viewport.width}-${scenario.name}-light.png`), fullPage: true });
      await page.getByLabel('深色', { exact: true }).check();
      reports.push({ viewport, scenario: scenario.name, ...pixels });
    }
    assert.equal(new Set(poses).size, 9, 'Scenarios must have distinct poses.');
    await page.getByLabel('对比原图', { exact: true }).check();
    assert.equal(await page.locator('.original').isVisible(), true);
    assert.equal(await page.evaluate(() => document.documentElement.scrollWidth > innerWidth), false);
    await page.emulateMedia({ reducedMotion: 'reduce' });
    await page.reload();
    await page.waitForSelector('body[data-ready=true]');
    assert.equal(await page.getByRole('button', { name: '播放', exact: true }).isVisible(), true);
    const reducedFrame = await canvas.screenshot();
    await page.waitForTimeout(400);
    assert.deepEqual(await canvas.screenshot(), reducedFrame);
    const reducedPixels=await canvas.evaluate(el=>Array.from(el.getContext('2d').getImageData(0,0,el.width,el.height).data).filter((v,i)=>i%4===3&&v>32).length);
    assert.ok(reducedPixels>1000,'Reduced motion must show a real static pose, not a blank canvas.');
    assert.deepEqual(errors, []);
    assert.deepEqual(remoteRequests, [], 'Offline preview must not request runtime or assets from a CDN.');
    await page.close();
    console.log(`PASS ${viewport.width}x${viewport.height}: 9 scenarios, 4 phone-free poses, both backgrounds, no white matte, offline, reduced motion.`);
  }
} finally { await browser.close(); }
await writeFile(path.join(artifacts, 'report.json'), JSON.stringify(reports, null, 2) + '\n');
