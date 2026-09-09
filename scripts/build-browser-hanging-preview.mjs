import { createCanvas, loadImage } from '@napi-rs/canvas';
import { createRequire } from 'node:module';
import { createHash } from 'node:crypto';
import { readFile, writeFile, mkdir, copyFile } from 'node:fs/promises';
import { fileURLToPath } from 'node:url';
import path from 'node:path';

const root = fileURLToPath(new URL('../', import.meta.url));
const toolsDirectory = process.argv[2];
if (!toolsDirectory) throw new Error('Pass the authoring directory containing potrace and lucide-static.');
const { Potrace } = createRequire(path.resolve(toolsDirectory, 'package.json'))('potrace');
const sourceFile = path.join(root, 'docs/assets/brand/mascot.png');
const vectorDirectory = path.join(root, 'docs/assets/brand/motion/vector');
const out = path.join(root, 'artifacts/browser-hanging');
await mkdir(out, { recursive: true });
const source = await loadImage(sourceFile);
const { width, height } = source;
const canvas = createCanvas(width, height), context = canvas.getContext('2d');
context.drawImage(source, 0, 0);
const pixels = context.getImageData(0, 0, width, height);
const mask = Uint8Array.from({ length: width * height }, (_, i) => {
  const p = i * 4;
  const [r, g, b, a] = pixels.data.subarray(p, p + 4);
  const x = i % width, y = Math.floor(i / width);
  const originalHand = y > 280 && y < 400 && ((x > 130 && x < 230) || (x > 305 && x < 405)) && b - r > 95 && b - g > 35;
  return a > 128 && ((r < 80 && g < 95 && b < 105) || originalHand) ? 1 : 0;
});
const neighbors = p => [p % width ? p - 1 : -1, p % width < width - 1 ? p + 1 : -1, p >= width ? p - width : -1, p < width * (height - 1) ? p + width : -1];
const seen = new Uint8Array(mask.length);
let body = [];
for (let seed = 0; seed < mask.length; seed++) {
  if (!mask[seed] || seen[seed]) continue;
  const component = [seed]; seen[seed] = 1;
  for (let i = 0; i < component.length; i++) for (const p of neighbors(component[i])) {
    if (p >= 0 && mask[p] && !seen[p]) { seen[p] = 1; component.push(p); }
  }
  if (component.length > body.length) body = component;
}
if (body.length < 10000) throw new Error('The approved body contour was not found.');
// Preserve the actual outer contour, not a convex hull. Fill only enclosed phone/hand holes.
const silhouette = new Uint8Array(mask.length);
for (const p of body) silhouette[p] = 1;
const exterior = new Uint8Array(mask.length), queue = [0]; exterior[0] = 1;
for (let i = 0; i < queue.length; i++) for (const p of neighbors(queue[i])) {
  if (p >= 0 && !silhouette[p] && !exterior[p]) { exterior[p] = 1; queue.push(p); }
}
for (let p = 0; p < mask.length; p++) {
  const value = exterior[p] ? 255 : 0;
  pixels.data.set([value, value, value, 255], p * 4);
}
context.putImageData(pixels, 0, 0);
const tracer = new Potrace({ color: '#070a0b', threshold: 128, optTolerance: 0.4 });
await new Promise((resolve, reject) => tracer.loadImage(canvas.toBuffer('image/png'), error => error ? reject(error) : resolve()));
const bodyPath = tracer.getPathTag();
await writeFile(path.join(out, 'body-contour.svg'), tracer.getSVG());

const n = value => Number(value.toFixed(4));
const hashes = {};
async function contours(name) {
  const bytes = await readFile(path.join(vectorDirectory, `${name}.json`));
  hashes[name] = createHash('sha256').update(bytes).digest('hex');
  const spec = JSON.parse(bytes);
  return spec.shapes.map(shape => {
    if (shape.type !== 'polygon') throw new Error(`Unsupported vector contour: ${name}`);
    const xy = p => `${n(p.x + shape.x)} ${n(p.y + shape.y)}`;
    const handle = (p, incoming) => {
      const angle = (incoming ? p.cubic.inRotation : p.cubic.rotation) * Math.PI / 180;
      const distance = incoming ? p.cubic.inDistance : p.cubic.outDistance;
      return `${n(p.x + shape.x + Math.cos(angle) * distance)} ${n(p.y + shape.y + Math.sin(angle) * distance)}`;
    };
    const d = shape.subpaths.map(({ points }) => `M${xy(points[0])} ` + points.slice(1).map((p, i) =>
      p.cubic && points[i].cubic ? `C${handle(points[i], false)} ${handle(p, true)} ${xy(p)}` : `L${xy(p)}`).join(' ') + ' Z').join(' ');
    return `<path d="${d}" fill="${shape.fill.color}"/>`;
  }).join('');
}
const tail = await contours('tail');
const head = await contours('head');
let eyes = '';
for (const side of ['left', 'right']) {
  const white = await contours(`eye-${side}`), pupil = await contours(`pupil-${side}`);
  eyes += `<defs><clipPath id="eye-${side}-clip">${white}</clipPath></defs>
    <g id="eyelid-${side}">${white}<g clip-path="url(#eye-${side}-clip)"><g id="pupil-${side}">${pupil}</g></g></g>`;
}
eyes += await contours('nostrils');
const left = await contours('hand-left'), right = await contours('hand-right');
const foot = await contours('free-foot');
const svg = content => `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 140 174" fill="none" aria-hidden="true">${content}</svg>`;
const rear = `<defs><clipPath id="body-mask"><rect width="140" height="128"/></clipPath></defs>
  <g id="lower-tail" transform="translate(44 118) rotate(-105) scale(.20) translate(-149 -444)">${tail}</g>
  <g id="body" clip-path="url(#body-mask)"><g transform="translate(7 0) scale(.28)">${bodyPath}</g></g>
  <g id="foot-left" transform="translate(70 126) rotate(-40) scale(-.23 .23) translate(-26 -8)">${foot}</g>
  <g id="foot-right" transform="translate(96 126) rotate(40) scale(.23) translate(-26 -8)">${foot}</g>
  <g id="original-head" transform="translate(7 0) scale(.28)">${head}${eyes}</g>`;
const hands = `<g id="grip-left" transform="translate(45 69) rotate(90) scale(.17) translate(-181 -341)">${left}</g>
  <g id="grip-right" transform="translate(108 69) rotate(-90) scale(.17) translate(-355 -341)">${right}</g>`;
await writeFile(path.join(out, 'hanging.svg'), svg(`${rear}<rect y="70" width="140" height="58" fill="#fbfdfd" stroke="#dce4e3"/>${hands}`));
await writeFile(path.join(out, 'source.json'), JSON.stringify({
  source: 'docs/assets/brand/mascot.png', sha256: createHash('sha256').update(await readFile(sourceFile)).digest('hex'),
  contours: hashes, body: 'Source outer contour, enclosed phone/hand holes filled; no convex hull.',
  changes: ['Phone removed', 'Body behind the opaque field at y=70..128', 'Original hands grip the upper edge', 'Original tail rotated below the field', 'Small feet reuse the existing pose-extension contours', 'Long raised arms removed'],
  scope: 'Browser review only: idle tail sway, clipped gaze and blinks, pointer/touch gaze. No body/caret following or App changes.',
}, null, 2));
await copyFile(sourceFile, path.join(out, 'original.png'));
let html = await readFile(path.join(root, 'scripts/templates/browser-hanging.html'), 'utf8');
html = html.replace('<!-- MASCOT_REAR -->', svg(rear)).replace('<!-- MASCOT_HANDS -->', svg(hands));
for (const [token, name] of Object.entries({ SEARCH: 'search', ARROW: 'arrow-right', PLAY: 'play', PAUSE: 'pause', REPLAY: 'rotate-ccw' })) {
  html = html.replace(`<!-- ICON_${token} -->`, await readFile(path.join(toolsDirectory, 'node_modules/lucide-static/icons', `${name}.svg`), 'utf8'));
}
await copyFile(path.join(toolsDirectory, 'node_modules/lucide-static/LICENSE'), path.join(out, 'LUCIDE-LICENSE'));
await copyFile(path.join(root, 'scripts/templates/browser-hanging-motion.js'), path.join(out, 'motion.js'));
await writeFile(path.join(out, 'index.html'), html);
console.log(path.join(out, 'index.html'));
