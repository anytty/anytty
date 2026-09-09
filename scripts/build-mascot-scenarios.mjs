import { mkdir, readFile, writeFile } from 'node:fs/promises';
import { createHash } from 'node:crypto';
import { fileURLToPath } from 'node:url';
import path from 'node:path';

const root = fileURLToPath(new URL('../', import.meta.url));
const out = path.join(root, 'docs/assets/brand/motion');
await mkdir(out, { recursive: true });
const ink = '#070a0b', green = '#2bce99', blue = '#1bb3ff';
const groups = [
  { id: 'character', x: 48, y: 28 },
  { id: 'tail', parent: 'character', x: 149, y: 444 },
  { id: 'body', parent: 'character', x: 0, y: 0 },
  { id: 'phone', parent: 'character', x: 270, y: 365 },
  { id: 'hand-left', parent: 'character', x: 155, y: 344 },
  { id: 'hand-right', parent: 'character', x: 383, y: 342 },
  { id: 'head', parent: 'character', x: 250, y: 231 },
  { id: 'eye-left', parent: 'head', x: -70, y: -58 },
  { id: 'eye-right', parent: 'head', x: 104, y: -132 },
  { id: 'pupil-left', parent: 'eye-left', x: 0, y: 0 },
  { id: 'pupil-right', parent: 'eye-right', x: 0, y: 0 },
  { id: 'free-legs', parent: 'character', x: 0, y: 0, opacity: 0 },
  ...['left', 'right'].flatMap((side, i) => [
    { id: `leg-${side}`, parent: 'free-legs', x: 230 + i * 75, y: 439 },
    { id: `calf-${side}`, parent: `leg-${side}`, x: 0, y: 43 },
    { id: `foot-${side}`, parent: `calf-${side}`, x: 0, y: 32, scaleX: i ? 1 : -1 },
    { id: `free-arm-${side}`, parent: 'character', x: i ? 350 : 185, y: 315, scaleX: i ? -1 : 1, opacity: 0 },
  ]),
  { id: 'yawn', parent: 'head', x: 45, y: -19, opacity: 0 },
  ...['connecting', 'processing', 'history', 'success', 'failure'].map(id => ({ id: `screen-${id}`, parent: 'phone', x: 0, y: 0, opacity: 0 })),
];
const shapes = [];
async function traced(id, parent, origin, extra = {}) {
  const fragment = JSON.parse(await readFile(path.join(out, 'vector', `${id}.json`), 'utf8'));
  for (const shape of fragment.shapes) shapes.push({ ...shape, parent, x: shape.x - origin[0], y: shape.y - origin[1], ...extra });
}
const rect = (id, parent, x, y, width, height, color, extra = {}) => shapes.push({ id, parent, type: 'rect', x, y, width, height, cornerRadius: 3, fill: { color }, ...extra });
await traced('tail', 'tail', [149, 444]);
for (const side of ['right', 'left']) {
  shapes.push({ id: `thigh-${side}`, parent: `leg-${side}`, type: 'rect', x: 0, y: 22, width: 24, height: 58, cornerRadius: 12, fill: { color: blue } });
  shapes.push({ id: `shin-${side}`, parent: `calf-${side}`, type: 'rect', x: 0, y: 18, width: 22, height: 50, cornerRadius: 11, fill: { color: blue } });
  await traced('free-foot', `foot-${side}`, [26, 8], { id: `foot-${side}-path` });
}
await traced('body', 'body', [0, 0]);
// Reconstruct occluded surfaces so larger gestures cannot expose cutout holes.
shapes.push({ id: 'shoulders', parent: 'body', type: 'ellipse', x: 270, y: 270, width: 204, height: 150, fill: { color: ink } });
rect('phone-face', 'phone', 0, -2, 112, 214, green, { cornerRadius: 14 });
for (let i = 0; i < 4; i++) rect(`signal-${i}`, 'screen-connecting', -25 + i * 17, 12 - i * 6, 9, 12 + i * 12, '#076d52');
for (let i = 0; i < 3; i++) {
  rect(`work-line-${i}`, 'screen-processing', 0, -30 + i * 26, 62 - i * 10, 7, '#076d52');
  rect(`history-line-${i}`, 'screen-history', 0, -35 + i * 27, 60 - i * 6, 7, '#076d52');
}
shapes.push({ id: 'success-check', parent: 'screen-success', type: 'polygon', x: 0, y: 0, closed: false, points: [{ x: -25, y: 0 }, { x: -7, y: 19 }, { x: 28, y: -23 }], stroke: { color: '#065c45', thickness: 10, cap: 'round', join: 'round' } });
rect('failure-mark', 'screen-failure', 0, -12, 10, 36, '#79450c', { cornerRadius: 5 });
shapes.push({ id: 'failure-dot', parent: 'screen-failure', type: 'ellipse', x: 0, y: 20, width: 10, height: 10, fill: { color: '#79450c' } });
await traced('hand-left', 'hand-left', [155, 344], { stroke: { color: ink, thickness: 9, join: 'round' } });
await traced('hand-right', 'hand-right', [383, 342], { stroke: { color: ink, thickness: 9, join: 'round' } });
for (const side of ['left', 'right']) await traced('free-arm', `free-arm-${side}`, [50, 8], { id: `free-arm-${side}-path`, stroke: { color: ink, thickness: 4, join: 'round' } });
await traced('head', 'head', [250, 231]);
await traced('eye-left', 'eye-left', [180, 173], { stroke: { color: '#ffffff', thickness: 1.4 } });
await traced('pupil-left', 'pupil-left', [180, 173], { clipBy: 'eye-left-p0' });
await traced('eye-right', 'eye-right', [354, 99], { stroke: { color: '#ffffff', thickness: 1.4 } });
await traced('pupil-right', 'pupil-right', [354, 99], { clipBy: 'eye-right-p0' });
await traced('nostrils', 'head', [250, 231]);
shapes.push({ id: 'yawn-mouth', parent: 'yawn', type: 'ellipse', x: 0, y: 0, width: 18, height: 8, fill: { color: ink } });

const defaults = new Map();
function register(target, property, initial) { defaults.set(`${target}:${property}`, { target, property, initial }); }
for (const g of groups) for (const property of ['x', 'y', 'rotation', 'scaleX', 'scaleY', 'opacity']) {
  register(g.id, property, g[property] ?? (property.startsWith('scale') || property === 'opacity' ? 1 : 0));
}
for (let i = 0; i < 4; i++) register(`signal-${i}`, 'opacity', 1);
for (let i = 0; i < 3; i++) register(`work-line-${i}`, 'width', 62 - i * 10);

const scenarios = [
  { name: 'connecting', label: 'Connecting', seconds: 4, loop: true, use: 'Session connection and reconnection' },
  { name: 'processing', label: 'Processing', seconds: 3, loop: true, use: 'File loading, preview preparation, background work' },
  { name: 'history', label: 'History', seconds: 1.2, loop: false, use: 'Entering terminal history after a pull-down gesture' },
  { name: 'success', label: 'Success', seconds: 1.5, loop: false, use: 'A confirmed successful operation; never mere connection progress' },
  { name: 'failure', label: 'Connection lost', seconds: 1.8, loop: false, use: 'Connection lost or a failed attempt; pair with a real retry action' },
  { name: 'welcome', label: 'Welcome', seconds: 2.2, loop: false, use: 'An intentional greeting or first session, not a claim of network success', phone: false },
  { name: 'running', label: 'Running', seconds: 0.9, loop: true, use: 'An active pending task or transfer with unknown duration', phone: false },
  { name: 'searching', label: 'Searching', seconds: 3.6, loop: true, use: 'File search or device discovery while results are pending', phone: false },
  { name: 'wake', label: 'Wake', seconds: 2.6, loop: false, use: 'An optional wake cue after background inactivity; not network recovery confirmation', phone: false },
];
function animation(scenario, build) {
  const duration = Math.round(scenario.seconds * 60);
  const tracks = new Map();
  const set = (target, property, points) => {
    if (!defaults.has(`${target}:${property}`)) throw new Error(`Missing reset default: ${target}:${property}`);
    const complete = points.map(([frame, value]) => [frame, target === 'tail' && property === 'rotation' ? Math.max(-12, Math.min(7, value)) : value]);
    if (complete[0][0] !== 0) complete.unshift([0, defaults.get(`${target}:${property}`).initial]);
    if (complete.at(-1)[0] !== duration) complete.push([duration, complete.at(-1)[1]]);
    tracks.set(`${target}:${property}`, { target, property, keyframes: complete.map(([frame, value], i) => ({ frame, value, ...(i < complete.length - 1 ? { easing: 'ease-in-out' } : {}) })) });
  };
  if (scenario.phone !== false) set(`screen-${scenario.name}`, 'opacity', scenario.loop ? [[0, 1], [duration, 1]] : [[0, 0], [12, 1], [duration, 1]]);
  build(set, duration);
  return { name: scenario.name, fps: 60, duration, loop: scenario.loop ? 'loop' : 'oneshot', tracks: [...tracks.values()] };
}
const blink = (set, frame, duration) => {
  set('eye-left', 'scaleY', [[0, 1], [frame, 1], [frame + 5, 0.035], [frame + 8, 0.035], [frame + 15, 1], [duration, 1]]);
  set('eye-right', 'scaleY', [[0, 1], [frame + 1, 1], [frame + 6, 0.035], [frame + 9, 0.035], [frame + 16, 1], [duration, 1]]);
};
const gaze = (set, points, yPoints) => {
  for (const side of ['left', 'right']) { set(`pupil-${side}`, 'x', points); if (yPoints) set(`pupil-${side}`, 'y', yPoints); }
};
function freePose(set, duration) {
  const constant = (target, property, value) => set(target, property, [[0, value], [duration, value]]);
  constant('phone', 'opacity', 0);
  for (const side of ['left', 'right']) {
    constant(`hand-${side}`, 'opacity', 0);
    constant(`free-arm-${side}`, 'opacity', 1);
    constant(`free-arm-${side}`, 'rotation', side === 'left' ? 12 : -12);
  }
  constant('free-legs', 'opacity', 1);
  constant('character', 'x', 54); constant('character', 'y', 60);
  constant('character', 'scaleX', 0.82); constant('character', 'scaleY', 0.82);
  constant('body', 'x', 47); constant('body', 'y', 84);
  constant('body', 'scaleX', 0.82); constant('body', 'scaleY', 0.78);
  constant('tail', 'x', 190); constant('tail', 'y', 425);
  constant('tail', 'scaleX', 0.78); constant('tail', 'scaleY', 0.78);
}
const animations = [
  animation(scenarios[0], (set, d) => {
    set('head', 'rotation', [[0, 0], [28, -5], [72, -5], [110, 5], [162, 5], [205, -2], [240, 0]]);
    gaze(set, [[0, 0], [18, -15], [78, -15], [98, -2], [170, -2], [206, -7], [240, 0]]);
    set('phone', 'y', [[0, 365], [32, 357], [74, 357], [110, 365], [180, 365], [240, 365]]);
    for (const side of ['left', 'right']) set(`hand-${side}`, 'y', [[0, side === 'left' ? 344 : 342], [32, side === 'left' ? 336 : 334], [74, side === 'left' ? 336 : 334], [110, side === 'left' ? 344 : 342], [240, side === 'left' ? 344 : 342]]);
    set('tail', 'rotation', [[0, 0], [42, 12], [93, -8], [150, 10], [203, -5], [240, 0]]);
    blink(set, 177, d);
    for (let i = 0; i < 4; i++) set(`signal-${i}`, 'opacity', [[0, 0.25], [16 + i * 12, 0.25], [28 + i * 12, 1], [85 + i * 12, 1], [103 + i * 12, 0.25], [155 + i * 12, 1], [200 + i * 7, 0.25], [240, 0.25]]);
  }),
  animation(scenarios[1], (set, d) => {
    set('head', 'rotation', [[0, 0], [26, 7], [85, 7], [112, 3], [145, 6], [180, 0]]);
    set('head', 'y', [[0, 231], [26, 236], [95, 236], [125, 232], [150, 235], [180, 231]]);
    gaze(set, [[0, 0], [22, -7], [145, -7], [180, 0]], [[0, 0], [22, 10], [145, 10], [180, 0]]);
    set('hand-left', 'rotation', [[0, 0], [30, -4], [40, -13], [49, -4], [62, -12], [73, -3], [106, -3], [118, -10], [132, -3], [180, 0]]);
    set('hand-right', 'rotation', [[0, 0], [43, 0], [53, 10], [63, 0], [76, 9], [87, 0], [125, 0], [138, 8], [150, 0], [180, 0]]);
    set('tail', 'rotation', [[0, 0], [40, 11], [85, -7], [135, 8], [180, 0]]);
    blink(set, 95, d);
    for (let i = 0; i < 3; i++) set(`work-line-${i}`, 'width', [[0, 62 - i * 10], [25 + i * 8, 20], [48 + i * 8, 62], [104 + i * 8, 24], [138 + i * 8, 62 - i * 10], [180, 62 - i * 10]]);
  }),
  animation(scenarios[2], (set, d) => {
    set('head', 'rotation', [[0, 0], [12, 4], [36, -7], [52, -3], [72, 0]]);
    gaze(set, [[0, 0], [12, -5], [31, -18], [50, -10], [72, 0]], [[0, 0], [12, 8], [36, -5], [72, 0]]);
    set('hand-left', 'y', [[0, 344], [13, 320], [33, 365], [49, 340], [72, 344]]);
    set('hand-left', 'rotation', [[0, 0], [13, -14], [33, 8], [49, -3], [72, 0]]);
    set('screen-history', 'y', [[0, -14], [13, -14], [33, 15], [49, 0], [72, 0]]);
    set('tail', 'rotation', [[0, 0], [17, -10], [39, 12], [56, -3], [72, 0]]);
    blink(set, 44, d);
  }),
  animation(scenarios[3], (set, d) => {
    set('head', 'rotation', [[0, 0], [13, 5], [29, -8], [43, -3], [62, -5], [90, 0]]);
    set('head', 'y', [[0, 231], [13, 237], [29, 221], [43, 228], [65, 225], [90, 231]]);
    set('hand-left', 'rotation', [[0, 0], [13, 5], [28, -36], [41, -23], [53, -34], [66, -20], [90, 0]]);
    set('hand-left', 'y', [[0, 344], [13, 346], [28, 320], [65, 320], [90, 344]]);
    set('tail', 'rotation', [[0, 0], [13, -5], [31, 17], [46, -11], [64, 10], [90, 0]]);
    set('screen-success', 'scaleX', [[0, 0.6], [18, 0.6], [32, 1.14], [46, 1], [90, 1]]);
    set('screen-success', 'scaleY', [[0, 0.6], [18, 0.6], [32, 1.14], [46, 1], [90, 1]]);
    blink(set, 39, d);
  }),
  animation(scenarios[4], (set, d) => {
    set('head', 'rotation', [[0, 0], [18, 5], [34, -8], [49, 7], [64, -5], [80, 2], [108, 0]]);
    set('head', 'y', [[0, 231], [18, 234], [80, 234], [108, 231]]);
    gaze(set, [[0, 0], [18, -12], [70, -12], [108, 0]], [[0, 0], [18, 7], [70, 7], [108, 0]]);
    set('eye-left', 'scaleY', [[0, 1], [18, 0.72], [75, 0.72], [108, 1]]);
    set('eye-right', 'scaleY', [[0, 1], [20, 0.72], [77, 0.72], [108, 1]]);
    set('tail', 'rotation', [[0, 0], [26, -10], [67, -10], [108, 0]]);
    set('phone', 'y', [[0, 365], [26, 371], [76, 371], [108, 365]]);
    set('hand-left', 'y', [[0, 344], [26, 350], [76, 350], [108, 344]]);
    set('hand-right', 'y', [[0, 342], [26, 348], [76, 348], [108, 342]]);
  }),
  animation(scenarios[5], (set, d) => {
    freePose(set, d);
    set('head', 'rotation', [[0, 4], [18, 4], [34, -5], [98, -5], [132, 0]]);
    gaze(set, [[0, 0], [20, -10], [110, -10], [132, -5]]);
    set('free-arm-left', 'rotation', [[0, 12], [15, 8], [35, 97], [48, 114], [60, 91], [74, 112], [87, 93], [103, 80], [132, 12]]);
    set('free-arm-right', 'rotation', [[0, -12], [38, -22], [95, -22], [132, -12]]);
    set('tail', 'rotation', [[0, 0], [30, -7], [56, 6], [80, -6], [110, 4], [132, 0]]);
    blink(set, 89, d);
  }),
  animation(scenarios[6], (set, d) => {
    freePose(set, d);
    set('character', 'rotation', [[0, 5], [d, 5]]);
    set('character', 'x', [[0, 76], [d, 76]]);
    set('character', 'y', [[0, 55], [7, 59], [14, 45], [22, 48], [27, 55], [34, 59], [41, 45], [49, 48], [54, 55]]);
    set('leg-left', 'rotation', [[0, -25], [14, 0], [27, 25], [41, 0], [54, -25]]);
    set('leg-right', 'rotation', [[0, 25], [14, 0], [27, -25], [41, 0], [54, 25]]);
    set('calf-left', 'rotation', [[0, 25], [14, 10], [27, 65], [41, 70], [54, 25]]);
    set('calf-right', 'rotation', [[0, 65], [14, 70], [27, 25], [41, 10], [54, 65]]);
    set('foot-left', 'scaleX', [[0, 1], [54, 1]]);
    set('foot-left', 'rotation', [[0, 0], [14, -12], [27, -45], [41, -30], [54, 0]]);
    set('foot-right', 'rotation', [[0, -45], [14, -30], [27, 0], [41, -12], [54, -45]]);
    set('free-arm-left', 'rotation', [[0, 37], [14, 0], [27, -27], [41, 0], [54, 37]]);
    set('free-arm-right', 'rotation', [[0, -27], [14, 0], [27, 37], [41, 0], [54, -27]]);
    set('head', 'rotation', [[0, -3], [14, -6], [27, -3], [41, -6], [54, -3]]);
    set('head', 'y', [[0, 233], [14, 228], [27, 233], [41, 228], [54, 233]]);
    set('tail', 'rotation', [[0, 0], [14, -9], [27, 5], [41, -9], [54, 0]]);
    gaze(set, [[0, -3], [54, -3]]);
  }),
  animation(scenarios[7], (set, d) => {
    freePose(set, d);
    set('character', 'y', [[0, 82], [d, 82]]);
    set('character', 'scaleY', [[0, 0.78], [d, 0.78]]);
    set('body', 'scaleY', [[0, 0.7], [d, 0.7]]);
    set('body', 'y', [[0, 107], [d, 107]]);
    set('head', 'y', [[0, 254], [42, 264], [82, 264], [135, 248], [175, 248], [216, 254]]);
    set('head', 'x', [[0, 250], [42, 219], [82, 219], [135, 275], [175, 275], [216, 250]]);
    set('head', 'rotation', [[0, 2], [42, -9], [82, -9], [135, 8], [175, 8], [216, 2]]);
    gaze(set, [[0, -7], [24, -22], [86, -22], [113, 0], [176, 0], [216, -7]], [[0, 5], [40, 10], [86, 10], [126, 0], [176, 0], [216, 5]]);
    set('free-arm-left', 'rotation', [[0, 20], [42, 35], [84, 35], [140, 10], [176, 10], [216, 20]]);
    set('free-arm-right', 'rotation', [[0, -20], [42, -10], [84, -10], [140, -36], [176, -36], [216, -20]]);
    set('tail', 'rotation', [[0, 0], [55, -10], [90, -6], [148, 5], [183, 3], [216, 0]]);
    blink(set, 90, d);
  }),
  animation(scenarios[8], (set, d) => {
    freePose(set, d);
    set('character', 'y', [[0, 126], [26, 126], [64, 31], [85, 31], [115, 66], [156, 60]]);
    set('character', 'scaleY', [[0, 0.70], [26, 0.70], [64, 0.87], [85, 0.87], [115, 0.81], [156, 0.82]]);
    set('character', 'scaleX', [[0, 0.88], [26, 0.88], [64, 0.79], [85, 0.79], [115, 0.83], [156, 0.82]]);
    set('character', 'x', [[0, 39], [26, 39], [64, 62], [85, 62], [115, 51], [156, 54]]);
    set('head', 'rotation', [[0, 9], [26, 9], [64, -7], [85, -7], [115, 2], [156, 0]]);
    set('head', 'y', [[0, 246], [26, 246], [64, 213], [85, 213], [115, 235], [156, 231]]);
    set('free-arm-left', 'rotation', [[0, -14], [26, -14], [64, 104], [85, 104], [115, 18], [156, 12]]);
    set('free-arm-right', 'rotation', [[0, 14], [26, 14], [64, -104], [85, -104], [115, -18], [156, -12]]);
    set('eye-left', 'scaleY', [[0, 0.04], [26, 0.04], [40, 0.48], [56, 0.08], [85, 0.08], [110, 1], [156, 1]]);
    set('eye-right', 'scaleY', [[0, 0.04], [28, 0.04], [42, 0.48], [58, 0.08], [87, 0.08], [112, 1], [156, 1]]);
    set('yawn', 'opacity', [[0, 0], [30, 0], [48, 1], [84, 1], [105, 0], [156, 0]]);
    set('yawn', 'scaleY', [[0, 1], [48, 2.5], [84, 2.5], [105, 1], [156, 1]]);
    set('tail', 'rotation', [[0, -10], [26, -10], [70, 7], [90, 5], [120, -3], [156, 0]]);
  }),
];

// Key shared properties in every state so interrupting an action cannot leak its pose.
const animatedProperties = new Set(animations.flatMap(a => a.tracks.map(t => `${t.target}:${t.property}`)));
for (const a of animations) {
  const used = new Set(a.tracks.map(t => `${t.target}:${t.property}`));
  for (const key of animatedProperties) if (!used.has(key)) {
    const { target, property, initial } = defaults.get(key);
    a.tracks.push({ target, property, keyframes: [{ frame: 0, value: initial }, { frame: a.duration, value: initial }] });
  }
}
const scene = {
  artboard: { name: 'AnyTTY Mascot', width: 513, height: 546 }, groups, shapes, animations,
  stateMachine: {
    name: 'Mascot', inputs: scenarios.map(s => ({ name: s.name, type: 'trigger' })),
    states: scenarios.map(s => ({ name: s.name, animation: s.name })),
    transitions: [{ from: 'entry', to: 'connecting' }, ...scenarios.map(s => ({ from: 'any', to: s.name, condition: { input: s.name }, durationMs: 120 }))],
  },
};
await writeFile(path.join(out, 'scene.json'), JSON.stringify(scene, null, 2) + '\n');
await writeFile(path.join(out, 'create.json'), JSON.stringify({ outPath: 'anytty-mascot.riv', scene, previewTime: 0 }, null, 2) + '\n');
await writeFile(path.join(out, 'scenarios.json'), JSON.stringify(scenarios, null, 2) + '\n');
await writeFile(path.join(out, 'source.json'), JSON.stringify({ source: '../mascot.png', sha256: createHash('sha256').update(await readFile(path.join(out, '../mascot.png'))).digest('hex'), sourceSize: [417, 490], artboardSize: [513, 546], format: 'traced-vector-with-pose-extensions', revision: 3, status: 'review-sample', stateMachine: 'Mascot', scenarios }, null, 2) + '\n');
console.log(`Created vector scene with ${shapes.length} shapes and ${animations.length} scenarios.`);
