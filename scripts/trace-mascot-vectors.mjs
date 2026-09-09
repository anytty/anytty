import { createCanvas, loadImage } from '@napi-rs/canvas';
import { createRequire } from 'node:module';
import { mkdir, writeFile } from 'node:fs/promises';
import { fileURLToPath } from 'node:url';
import path from 'node:path';

const root = fileURLToPath(new URL('../', import.meta.url));
const toolsDirectory = process.argv[2];
if (!toolsDirectory) throw new Error('Pass a temporary directory containing the potrace authoring dependency.');
const { Potrace } = createRequire(path.resolve(toolsDirectory, 'package.json'))('potrace');
const out = path.join(root, 'docs/assets/brand/motion/vector');
await mkdir(out, { recursive: true });
const source = await loadImage(path.join(root, 'docs/assets/brand/mascot.png'));
const { width: w, height: h } = source;
const canvas = createCanvas(w, h);
const ctx = canvas.getContext('2d');
ctx.drawImage(source, 0, 0);
const { data } = ctx.getImageData(0, 0, w, h);

function maskWhere(predicate) {
  return Uint8Array.from({ length: w * h }, (_, p) => {
    const i = p * 4;
    return data[i + 3] > 128 && predicate(data[i], data[i + 1], data[i + 2], p % w, Math.floor(p / w)) ? 1 : 0;
  });
}

function components(mask) {
  const seen = new Uint8Array(mask.length);
  const results = [];
  for (let seed = 0; seed < mask.length; seed++) {
    if (!mask[seed] || seen[seed]) continue;
    const points = [seed];
    seen[seed] = 1;
    for (let q = 0; q < points.length; q++) {
      const p = points[q], x = p % w, y = Math.floor(p / w);
      for (const next of [x > 0 ? p - 1 : -1, x < w - 1 ? p + 1 : -1, y > 0 ? p - w : -1, y < h - 1 ? p + w : -1]) {
        if (next >= 0 && mask[next] && !seen[next]) { seen[next] = 1; points.push(next); }
      }
    }
    if (points.length > 35) results.push(points);
  }
  return results.sort((a, b) => b.length - a.length);
}

function filled(points) {
  const mask = new Uint8Array(w * h);
  for (const p of points) mask[p] = 1;
  const inverse = Uint8Array.from(mask, v => 1 - v);
  const exterior = components(inverse).find(c => c.includes(0));
  if (!exterior) throw new Error('Mask must have a transparent outer margin.');
  const result = new Uint8Array(w * h).fill(1);
  for (const p of exterior) result[p] = 0;
  return result;
}

function closeGaps(mask, radius) {
  const expand = (input, dilate) => Uint8Array.from(input, (_, p) => {
    const x = p % w, y = Math.floor(p / w);
    for (let dy = -radius; dy <= radius; dy++) for (let dx = -radius; dx <= radius; dx++) {
      const value = x + dx >= 0 && x + dx < w && y + dy >= 0 && y + dy < h ? input[(y + dy) * w + x + dx] : 0;
      if (Boolean(value) === dilate) return dilate ? 1 : 0;
    }
    return dilate ? 0 : 1;
  });
  return expand(expand(mask, true), false);
}

function bodySilhouette(points) {
  const sorted = points.filter(p => Math.floor(p / w) >= 245).map(p => [p % w, Math.floor(p / w)]).sort((a, b) => a[0] - b[0] || a[1] - b[1]);
  const cross = (a, b, c) => (b[0] - a[0]) * (c[1] - a[1]) - (b[1] - a[1]) * (c[0] - a[0]);
  const chain = values => {
    const result = [];
    for (const p of values) { while (result.length > 1 && cross(result.at(-2), result.at(-1), p) <= 0) result.pop(); result.push(p); }
    return result.slice(0, -1);
  };
  const hull = [...chain(sorted), ...chain([...sorted].reverse())];
  ctx.fillStyle = '#ffffff'; ctx.fillRect(0, 0, w, h); ctx.fillStyle = '#000000'; ctx.beginPath();
  hull.forEach(([x, y], i) => i ? ctx.lineTo(x, y) : ctx.moveTo(x, y)); ctx.closePath(); ctx.fill();
  const pixels = ctx.getImageData(0, 0, w, h).data;
  return Uint8Array.from({ length: w * h }, (_, p) => pixels[p * 4] < 128 ? 1 : 0);
}

const blue = components(maskWhere((r, g, b) => b - r > 95 && b - g > 35 && b > 175));
const dark = components(maskWhere((r, g, b) => r < 80 && g < 95 && b < 105));
const green = components(maskWhere((r, g, b) => g - r > 95 && g - b > 25 && b > 100));
const whites = components(maskWhere((r, g, b, x, y) => r > 185 && g > 200 && b > 210 && ((x > 124 && x < 240 && y > 115 && y < 233) || (x > 312 && x < 399 && y > 43 && y < 158))));
const center = points => points.reduce((a, p) => [a[0] + p % w / points.length, a[1] + Math.floor(p / w) / points.length], [0, 0]);
const head = blue.find(p => center(p)[1] < 220);
const tail = blue.find(p => center(p)[0] < 110 && center(p)[1] > 220);
const handLeft = blue.find(p => center(p)[0] > 130 && center(p)[0] < 240 && center(p)[1] > 250);
const handRight = blue.find(p => center(p)[0] > 300 && center(p)[1] > 250);
const pupils = dark.filter(p => center(p)[1] < 180 && p.length > 1000).sort((a, b) => center(a)[0] - center(b)[0]);
const nostrils = dark.filter(p => p.length < 500 && center(p)[1] > 155 && center(p)[1] < 215);
const eyeWhites = whites.sort((a, b) => center(a)[0] - center(b)[0]);
const parts = [
  ['tail', tail, '#1bb3ff'], ['body', dark[0], '#070a0b'],
  ['phone', green[0], '#2bce99'], ['hand-left', handLeft, '#1bb3ff'], ['hand-right', handRight, '#1bb3ff'],
  ['head', head, '#1bb3ff'], ['eye-left', [...eyeWhites[0], ...pupils[0]], '#ffffff'], ['eye-right', [...eyeWhites[1], ...pupils[1]], '#ffffff'],
  ['pupil-left', pupils[0], '#050809'], ['pupil-right', pupils[1], '#050809'],
  ['nostrils', nostrils.flat(), '#050809'],
];

const paths = [];
const jobs = [];
for (const [id, points, color] of parts) {
  if (!points?.length) throw new Error(`Missing source component: ${id}`);
  let mask = filled(points);
  if (id.startsWith('eye-')) mask = closeGaps(mask, 3);
  if (id === 'body') mask = bodySilhouette(points);
  const image = ctx.createImageData(w, h);
  for (let i = 0; i < mask.length; i++) {
    const value = mask[i] ? 0 : 255;
    image.data.set([value, value, value, 255], i * 4);
  }
  ctx.putImageData(image, 0, 0);
  const tracer = new Potrace({ color, threshold: 128, turdSize: 5, optTolerance: 0.6, alphaMax: 1.1 });
  await new Promise((resolve, reject) => tracer.loadImage(canvas.toBuffer('image/png'), error => error ? reject(error) : resolve()));
  await writeFile(path.join(out, `${id}.svg`), tracer.getSVG());
  paths.push(`<g id="${id}">${tracer.getPathTag()}</g>`);
  jobs.push({ name: 'riv_import_svg', arguments: { svgPath: path.join(out, `${id}.svg`), outSpec: path.join(out, `${id}.json`), idPrefix: `${id}-` } });
  console.log(id, points.length, center(points).map(Math.round));
}
await writeFile(path.join(out, 'traced-master.svg'), `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 ${w} ${h}" width="${w}" height="${h}">${paths.join('')}</svg>`);
// Pose extensions are authored separately from the source-logo contours.
for (const id of ['free-arm', 'free-foot']) {
  jobs.push({ name: 'riv_import_svg', arguments: { svgPath: path.join(out, `${id}.svg`), outSpec: path.join(out, `${id}.json`), idPrefix: `${id}-` } });
}
await writeFile(path.join(out, 'import-jobs.json'), JSON.stringify(jobs, null, 2));
