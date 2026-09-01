const body = document.body;
const screens = [...document.querySelectorAll('[data-screen]')];
const previewButtons = [...document.querySelectorAll('[data-preview-nav]')];
const themeMeta = document.querySelector('meta[name="theme-color"]');
const backdrop = document.querySelector('[data-sheet-backdrop]');
const pairContent = document.querySelector('[data-sheet-pair]');
const terminalContent = document.querySelector('[data-sheet-terminal]');
const sheetTitle = document.querySelector('[data-sheet-title]');
const toast = document.querySelector('.toast');
const tuner = document.querySelector('[data-theme-tuner]');
const tunerBackdrop = document.querySelector('[data-tuner-backdrop]');
const radiusOutput = document.querySelector('[data-radius-output]');
const resourceBackdrop = document.querySelector('[data-resource-backdrop]');
const petalMenu = document.querySelector('[data-petal-menu]');
const petalCanvas = document.querySelector('[data-petal-canvas]');
const terminalSurface = document.querySelector('.terminal-surface');
const consoleScreen = document.querySelector('.console-screen');
let toastTimer;
let toastUndoAction;
let holdTimer;
let holdStart;

const paletteTokenNames = [
  '--accent',
  '--accent-pressed',
  '--accent-soft',
  '--accent-ink',
  '--bg',
  '--surface',
  '--surface-soft',
  '--surface-strong',
  '--line',
  '--terminal',
  '--terminal-raised',
  '--terminal-soft',
  '--terminal-strong',
  '--terminal-line',
  '--terminal-ink',
  '--terminal-muted',
  '--terminal-accent',
  '--terminal-accent-soft',
  '--terminal-accent-ink',
  '--terminal-on-accent',
];

const presets = {
  cyan: { accent: '#32d5d0', background: '#0e1012', surface: '#181b1e' },
  sky: { accent: '#32b9ef', background: '#0d1115', surface: '#171c20' },
  mint: { accent: '#42ddb0', background: '#0d1110', surface: '#171c1a' },
  iris: { accent: '#819cff', background: '#0f1118', surface: '#191c26' },
};
let paletteValues = { ...presets.cyan };
let activeRgbToken = 'accent';

const petalActionCatalog = {
  history: { label: '历史记录', icon: 'history', group: '终端' },
  search: { label: '搜索', icon: 'search', group: '终端' },
  more: { label: '更多', icon: 'ellipsis', group: '分组', acceptsChildren: true },
  selection: { label: '选择', icon: 'scan-text', group: '编辑' },
  paste: { label: '粘贴', icon: 'clipboard-paste', group: '编辑' },
  resources: { label: '资源快照', icon: 'activity', group: '终端' },
  keyboard: { label: '系统键盘', icon: 'keyboard', group: '输入' },
  'quick-keys': { label: '快捷键面板', icon: 'zap', group: '输入' },
  enter: { label: '回车', icon: 'corner-down-left', group: '按键' },
  escape: { label: '退出键', icon: 'badge-x', group: '按键' },
  'command-bar': { label: '快捷栏', icon: 'sliders-horizontal', group: '终端' },
  files: { label: '当前目录', icon: 'folder-open', group: '终端' },
  'input-tools': { label: '输入操作', icon: 'command', group: '分组', acceptsChildren: true },
  'session-tools': { label: '布局操作', icon: 'panels-top-left', group: '分组', acceptsChildren: true },
  settings: { label: '终端设置', icon: 'settings', group: '终端' },
  'copy-screen': { label: '复制屏幕', icon: 'copy', group: '编辑' },
};

const petalSlotPositions = [
  { left: 50, top: 11 },
  { left: 78, top: 22 },
  { left: 90, top: 50 },
  { left: 78, top: 78 },
  { left: 50, top: 89 },
  { left: 22, top: 78 },
  { left: 10, top: 50 },
  { left: 22, top: 22 },
];

const petalEditorDefaults = {
  root: ['history', null, 'search', 'more', null, 'selection', null, 'paste'],
  more: ['command-bar', null, 'files', 'input-tools', null, 'session-tools', null, 'settings'],
  'input-tools': Array(8).fill(null),
  'session-tools': Array(8).fill(null),
};

let petalEditorRings = clonePetalRings(petalEditorDefaults);
let petalEditorRing = 'root';
let petalEditorPress;
let petalEditorDrag;
let petalEditorPressTimer;
let suppressPetalClickUntil = 0;

function clonePetalRings(value) {
  return Object.fromEntries(Object.entries(value).map(([key, items]) => [key, [...items]]));
}

function refreshIcons() {
  if (window.lucide) {
    window.lucide.createIcons({ attrs: { 'stroke-width': 1.8 } });
  }
}

function setTheme(theme) {
  const next = theme === 'dark' ? 'dark' : 'light';
  body.dataset.theme = next;
  applyThemeTokens();
  themeMeta.setAttribute('content', next === 'dark' ? paletteValues.background : '#f2f5f3');

  document.querySelectorAll('[data-theme-toggle]').forEach((button) => {
    button.innerHTML = `<i data-lucide="${next === 'dark' ? 'sun' : 'moon'}"></i>`;
    button.setAttribute('aria-label', next === 'dark' ? '切换到浅色主题' : '切换到深色主题');
  });

  document.querySelectorAll('[data-appearance]').forEach((button) => {
    const selected = button.dataset.appearance === next;
    button.classList.toggle('is-selected', selected);
    button.setAttribute('aria-pressed', String(selected));
  });

  document.querySelectorAll('[data-tuner-theme]').forEach((button) => {
    const selected = button.dataset.tunerTheme === next;
    button.classList.toggle('is-selected', selected);
    button.setAttribute('aria-pressed', String(selected));
  });
  refreshIcons();
}

function hexToRgb(hex) {
  const value = hex.replace('#', '');
  return {
    r: Number.parseInt(value.slice(0, 2), 16),
    g: Number.parseInt(value.slice(2, 4), 16),
    b: Number.parseInt(value.slice(4, 6), 16),
  };
}

function rgbToHex(rgb) {
  const channel = (value) => Math.max(0, Math.min(255, Math.round(value))).toString(16).padStart(2, '0');
  return `#${channel(rgb.r)}${channel(rgb.g)}${channel(rgb.b)}`;
}

function rgbToHsv(rgb) {
  const r = rgb.r / 255;
  const g = rgb.g / 255;
  const b = rgb.b / 255;
  const max = Math.max(r, g, b);
  const min = Math.min(r, g, b);
  const delta = max - min;
  let h = 0;
  if (delta !== 0) {
    if (max === r) h = 60 * (((g - b) / delta) % 6);
    else if (max === g) h = 60 * ((b - r) / delta + 2);
    else h = 60 * ((r - g) / delta + 4);
  }
  if (h < 0) h += 360;
  return { h, s: max === 0 ? 0 : delta / max, v: max };
}

function hsvToRgb(hsv) {
  const c = hsv.v * hsv.s;
  const x = c * (1 - Math.abs(((hsv.h / 60) % 2) - 1));
  const m = hsv.v - c;
  let channels;
  if (hsv.h < 60) channels = [c, x, 0];
  else if (hsv.h < 120) channels = [x, c, 0];
  else if (hsv.h < 180) channels = [0, c, x];
  else if (hsv.h < 240) channels = [0, x, c];
  else if (hsv.h < 300) channels = [x, 0, c];
  else channels = [c, 0, x];
  return {
    r: Math.round((channels[0] + m) * 255),
    g: Math.round((channels[1] + m) * 255),
    b: Math.round((channels[2] + m) * 255),
  };
}

function mixHex(first, second, amount) {
  const a = hexToRgb(first);
  const b = hexToRgb(second);
  const channel = (name) => Math.round(a[name] + (b[name] - a[name]) * amount).toString(16).padStart(2, '0');
  return `#${channel('r')}${channel('g')}${channel('b')}`;
}

function isLightColor(hex) {
  const rgb = hexToRgb(hex);
  const linear = [rgb.r, rgb.g, rgb.b].map((channel) => {
    const value = channel / 255;
    return value <= 0.04045 ? value / 12.92 : ((value + 0.055) / 1.055) ** 2.4;
  });
  return (0.2126 * linear[0] + 0.7152 * linear[1] + 0.0722 * linear[2]) > 0.42;
}

function applyThemeTokens() {
  paletteTokenNames.forEach((name) => body.style.removeProperty(name));
  const dark = body.dataset.theme === 'dark';
  const base = dark ? paletteValues.background : '#f2f5f3';
  const terminalInk = isLightColor(paletteValues.background) ? '#17211e' : '#e9eef0';
  body.style.setProperty('--accent', paletteValues.accent);
  body.style.setProperty('--accent-pressed', mixHex(paletteValues.accent, dark ? '#ffffff' : '#000000', dark ? 0.22 : 0.12));
  body.style.setProperty('--accent-soft', mixHex(base, paletteValues.accent, dark ? 0.2 : 0.13));
  body.style.setProperty('--accent-ink', mixHex(paletteValues.accent, dark ? '#ffffff' : '#000000', dark ? 0.48 : 0.24));
  body.style.setProperty('--terminal', paletteValues.background);
  body.style.setProperty('--terminal-raised', paletteValues.surface);
  body.style.setProperty('--terminal-soft', mixHex(paletteValues.surface, terminalInk, 0.06));
  body.style.setProperty('--terminal-strong', mixHex(paletteValues.surface, terminalInk, 0.13));
  body.style.setProperty('--terminal-line', mixHex(paletteValues.surface, terminalInk, 0.11));
  body.style.setProperty('--terminal-ink', terminalInk);
  body.style.setProperty('--terminal-muted', mixHex(terminalInk, paletteValues.background, 0.48));
  body.style.setProperty('--terminal-accent', paletteValues.accent);
  body.style.setProperty('--terminal-accent-soft', mixHex(paletteValues.surface, paletteValues.accent, 0.22));
  body.style.setProperty('--terminal-accent-ink', mixHex(paletteValues.accent, terminalInk, 0.2));
  body.style.setProperty('--terminal-on-accent', isLightColor(paletteValues.accent) ? '#0a1b17' : '#ffffff');
  if (!dark) return;
  body.style.setProperty('--bg', paletteValues.background);
  body.style.setProperty('--surface', paletteValues.surface);
  body.style.setProperty('--surface-soft', mixHex(paletteValues.surface, '#ffffff', 0.05));
  body.style.setProperty('--surface-strong', mixHex(paletteValues.surface, '#ffffff', 0.1));
  body.style.setProperty('--line', mixHex(paletteValues.surface, '#ffffff', 0.09));
}

function applyPalette(values, selectedPreset) {
  paletteValues = {
    accent: values.accent.toLowerCase(),
    background: values.background.toLowerCase(),
    surface: values.surface.toLowerCase(),
  };
  applyThemeTokens();

  document.querySelectorAll('[data-preset]').forEach((button) => {
    const selected = button.dataset.preset === selectedPreset;
    button.classList.toggle('is-selected', selected);
    button.setAttribute('aria-pressed', String(selected));
  });
  updateRgbEditor();
}

function currentPalette() {
  return { ...paletteValues };
}

function updateRgbEditor() {
  const names = { accent: '强调色', background: '深色背景', surface: '深色表面' };
  const value = paletteValues[activeRgbToken];
  const rgb = hexToRgb(value);
  const hsv = rgbToHsv(rgb);
  document.querySelector('[data-rgb-preview]').style.backgroundColor = value;
  document.querySelector('[data-rgb-name]').textContent = names[activeRgbToken];
  document.querySelector('[data-rgb-hex]').textContent = value.toUpperCase();
  const colorPlane = document.querySelector('[data-color-plane]');
  colorPlane.style.background = `linear-gradient(to top, #000000, transparent), linear-gradient(to right, #ffffff, hsl(${hsv.h} 100% 50%))`;
  colorPlane.setAttribute('aria-valuetext', value.toUpperCase());
  const thumb = document.querySelector('[data-color-plane-thumb]');
  thumb.style.left = `${hsv.s * 100}%`;
  thumb.style.top = `${(1 - hsv.v) * 100}%`;
  document.querySelector('[data-hue-range]').value = String(Math.round(hsv.h));
  document.querySelectorAll('[data-rgb-channel]').forEach((input) => {
    input.value = String(rgb[input.dataset.rgbChannel]);
  });
  document.querySelectorAll('[data-rgb-token]').forEach((button) => {
    const selected = button.dataset.rgbToken === activeRgbToken;
    button.classList.toggle('is-selected', selected);
    button.setAttribute('aria-pressed', String(selected));
  });
}

function setRadius(value) {
  const radius = Number(value);
  body.style.setProperty('--radius-sm', `${Math.max(8, radius - 4)}px`);
  body.style.setProperty('--radius-md', `${radius}px`);
  body.style.setProperty('--radius-lg', `${radius + 4}px`);
  body.style.setProperty('--radius-sheet', `${radius + 10}px`);
  radiusOutput.value = `${radius} px`;
  radiusOutput.textContent = `${radius} px`;
}

function openTuner() {
  tuner.hidden = false;
  tunerBackdrop.hidden = false;
  window.requestAnimationFrame(() => {
    tuner.classList.add('is-open');
    tunerBackdrop.classList.add('is-open');
  });
}

function closeTuner() {
  tuner.classList.remove('is-open');
  tunerBackdrop.classList.remove('is-open');
  window.setTimeout(() => {
    tuner.hidden = true;
    tunerBackdrop.hidden = true;
  }, 190);
}

function showScreen(name) {
  const target = screens.find((screen) => screen.dataset.screen === name);
  if (!target) return;

  closePetal();

  screens.forEach((screen) => {
    if (screen.classList.contains('is-active') && screen !== target) {
      screen.classList.add('is-exiting');
      window.setTimeout(() => screen.classList.remove('is-exiting'), 220);
    }
    screen.classList.toggle('is-active', screen === target);
  });

  previewButtons.forEach((button) => {
    const previewName = name === 'petal-settings' ? 'settings' : name;
    button.classList.toggle('is-active', button.dataset.previewNav === previewName);
  });
}

function showToast(message, undoAction) {
  window.clearTimeout(toastTimer);
  toast.querySelector('span').textContent = message;
  const undoButton = toast.querySelector('[data-undo-petal]');
  toastUndoAction = undoAction;
  undoButton.hidden = !undoAction;
  toast.hidden = false;
  window.requestAnimationFrame(() => toast.classList.add('is-visible'));
  toastTimer = window.setTimeout(() => {
    toast.classList.remove('is-visible');
    toastUndoAction = undefined;
    window.setTimeout(() => {
      toast.hidden = true;
      undoButton.hidden = true;
    }, 180);
  }, 2600);
}

function openSheet(type) {
  const isPairing = type === 'pair';
  sheetTitle.textContent = isPairing ? '配对设备' : '新建终端';
  pairContent.hidden = !isPairing;
  terminalContent.hidden = isPairing;
  backdrop.hidden = false;
  window.requestAnimationFrame(() => backdrop.classList.add('is-open'));
  window.setTimeout(() => {
    const input = backdrop.querySelector(isPairing ? '#pair-code' : '#terminal-name');
    input?.focus({ preventScroll: true });
  }, 240);
}

function closeSheet() {
  backdrop.classList.remove('is-open');
  window.setTimeout(() => { backdrop.hidden = true; }, 190);
}

function openResources(trigger) {
  const name = trigger.dataset.resourceName || 'api-server';
  const cpu = trigger.dataset.resourceCpu || '12.6%';
  const memory = trigger.dataset.resourceMemory || '1.8 GB';
  resourceBackdrop.querySelector('[data-resource-title]').textContent = name;
  resourceBackdrop.querySelector('[data-current-cpu]').textContent = cpu;
  resourceBackdrop.querySelector('[data-current-memory]').textContent = memory;
  resourceBackdrop.querySelector('[data-chart-cpu]').textContent = cpu;
  resourceBackdrop.querySelector('[data-chart-memory]').textContent = memory;
  resourceBackdrop.hidden = false;
  window.requestAnimationFrame(() => resourceBackdrop.classList.add('is-open'));
}

function closeResources() {
  resourceBackdrop.classList.remove('is-open');
  window.setTimeout(() => { resourceBackdrop.hidden = true; }, 190);
}

function openPetal(clientX, clientY) {
  const rect = consoleScreen.getBoundingClientRect();
  const edge = 112;
  const x = Math.max(edge, Math.min(rect.width - edge, clientX - rect.left));
  const y = Math.max(edge, Math.min(rect.height - edge, clientY - rect.top));
  petalCanvas.style.setProperty('--petal-x', `${x}px`);
  petalCanvas.style.setProperty('--petal-y', `${y}px`);
  petalMenu.hidden = false;
  window.requestAnimationFrame(() => petalMenu.classList.add('is-open'));
  navigator.vibrate?.(12);
}

function closePetal() {
  window.clearTimeout(holdTimer);
  holdStart = undefined;
  terminalSurface.classList.remove('is-holding');
  if (petalMenu.hidden) return;
  petalMenu.classList.remove('is-open');
  window.setTimeout(() => { petalMenu.hidden = true; }, 150);
}

async function copyThemeConfig() {
  const config = {
    schema: 'anytty-ui-theme/v1',
    mode: body.dataset.theme,
    colors: {
      accent: paletteValues.accent.toUpperCase(),
      darkBackground: paletteValues.background.toUpperCase(),
      darkSurface: paletteValues.surface.toUpperCase(),
    },
    radius: Number(document.querySelector('[data-radius-range]').value),
  };
  const text = JSON.stringify(config, null, 2);
  const output = document.querySelector('[data-config-output]');
  output.textContent = text;
  output.hidden = false;
  try {
    await navigator.clipboard.writeText(text);
  } catch {
    const fallback = document.createElement('textarea');
    fallback.value = text;
    fallback.style.position = 'fixed';
    fallback.style.opacity = '0';
    document.body.append(fallback);
    fallback.select();
    document.execCommand('copy');
    fallback.remove();
  }
  showToast('主题配置已复制');
}

function petalEditorSnapshot() {
  return { rings: clonePetalRings(petalEditorRings), ring: petalEditorRing };
}

function restorePetalEditor(snapshot) {
  petalEditorRings = clonePetalRings(snapshot.rings);
  petalEditorRing = snapshot.ring;
  renderPetalEditor();
}

function commitPetalEditor(message, snapshot) {
  renderPetalEditor();
  showToast(message, () => {
    restorePetalEditor(snapshot);
    showToast('已撤销花瓣布局修改');
  });
}

function assignedPetalIds() {
  return new Set(Object.values(petalEditorRings).flat().filter(Boolean));
}

function findPetalLocation(actionId) {
  for (const [ring, items] of Object.entries(petalEditorRings)) {
    const index = items.indexOf(actionId);
    if (index >= 0) return { ring, index };
  }
  return null;
}

function findPetalParentRing(actionId) {
  return findPetalLocation(actionId)?.ring || 'root';
}

function petalRingLevel(ring) {
  let level = 1;
  let cursor = ring;
  const seen = new Set();
  while (cursor !== 'root' && !seen.has(cursor)) {
    seen.add(cursor);
    level += 1;
    cursor = findPetalParentRing(cursor);
  }
  return level;
}

function ensurePetalRing(actionId) {
  if (!petalEditorRings[actionId]) petalEditorRings[actionId] = Array(8).fill(null);
  return petalEditorRings[actionId];
}

function removePetalAction(actionId) {
  const location = findPetalLocation(actionId);
  if (location) petalEditorRings[location.ring][location.index] = null;
  return location;
}

function unassignPetalSubtree(actionId) {
  removePetalAction(actionId);
  const children = petalEditorRings[actionId];
  if (!children) return;
  children.filter(Boolean).forEach(unassignPetalSubtree);
  petalEditorRings[actionId] = Array(8).fill(null);
}

function firstEmptyPetalSlot(ring) {
  return ring.findIndex((item) => !item);
}

function placePetalWithShift(ring, actionId, targetIndex) {
  if (!ring[targetIndex]) {
    ring[targetIndex] = actionId;
    return true;
  }
  const emptyIndexes = ring.flatMap((item, index) => item ? [] : [index]);
  if (!emptyIndexes.length) return false;
  const empty = emptyIndexes.reduce((best, index) => (
    Math.abs(index - targetIndex) < Math.abs(best - targetIndex) ? index : best
  ));
  if (empty > targetIndex) {
    for (let index = empty; index > targetIndex; index -= 1) ring[index] = ring[index - 1];
  } else {
    for (let index = empty; index < targetIndex; index += 1) ring[index] = ring[index + 1];
  }
  ring[targetIndex] = actionId;
  return true;
}

function petalHaptic(duration = 10) {
  const toggle = document.querySelector('[aria-label="花瓣菜单触觉反馈"]');
  if (toggle?.getAttribute('aria-checked') === 'true') navigator.vibrate?.(duration);
}

function renderPetalEditor() {
  const slotsContainer = document.querySelector('[data-petal-editor-slots]');
  if (!slotsContainer) return;
  const nodesContainer = document.querySelector('[data-petal-editor-nodes]');
  const guide = document.querySelector('[data-petal-editor-guides]');
  const currentRing = ensurePetalRing(petalEditorRing);
  const currentMeta = petalEditorRing === 'root'
    ? { label: '主菜单', icon: 'flower-2' }
    : petalActionCatalog[petalEditorRing];

  guide.innerHTML = `
    <ellipse cx="50" cy="50" rx="40" ry="39"></ellipse>
    ${petalSlotPositions.map((position) => `<line x1="50" y1="50" x2="${position.left}" y2="${position.top}"></line>`).join('')}
  `;
  slotsContainer.innerHTML = petalSlotPositions.map((position, index) => `
    <span class="petal-editor-slot" data-petal-slot="${index}" style="--slot-left:${position.left}%;--slot-top:${position.top}%"></span>
  `).join('');
  nodesContainer.innerHTML = currentRing.map((actionId, index) => {
    if (!actionId) return '';
    const action = petalActionCatalog[actionId];
    const position = petalSlotPositions[index];
    const childCount = ensurePetalRing(actionId).filter(Boolean).length;
    return `
      <button class="editor-petal-node" type="button"
        data-petal-drag-source="${actionId}" data-petal-node="${actionId}"
        ${action.acceptsChildren ? 'data-petal-group-target="true"' : ''}
        style="--slot-left:${position.left}%;--slot-top:${position.top}%"
        aria-label="${action.label}${action.acceptsChildren ? `，包含 ${childCount} 个内部花瓣` : ''}">
        <i data-lucide="${action.icon}"></i><span>${action.label}</span>
        ${action.acceptsChildren ? `<b>${childCount}</b>` : ''}
      </button>
    `;
  }).join('');

  document.querySelector('[data-petal-ring-title]').textContent = currentMeta.label;
  document.querySelector('[data-petal-ring-level]').textContent = `第 ${petalRingLevel(petalEditorRing)} 层`;
  document.querySelector('[data-petal-ring-count]').textContent = `${currentRing.filter(Boolean).length} / 8`;
  const mark = document.querySelector('.petal-ring-mark');
  mark.innerHTML = `<i data-lucide="${currentMeta.icon}"></i>`;

  const hub = document.querySelector('[data-petal-editor-hub]');
  const isRoot = petalEditorRing === 'root';
  hub.classList.toggle('is-back', !isRoot);
  hub.innerHTML = `<i data-lucide="${isRoot ? 'flower-2' : 'undo-2'}"></i><span>${isRoot ? '主菜单' : currentMeta.label}</span>`;
  hub.setAttribute('aria-label', isRoot ? '主菜单' : '返回上层花瓣');

  const assigned = assignedPetalIds();
  const available = Object.entries(petalActionCatalog).filter(([id]) => !assigned.has(id));
  const library = document.querySelector('[data-petal-action-library]');
  library.innerHTML = available.length ? available.map(([id, action]) => `
    <button class="petal-action-item" type="button" data-petal-drag-source="${id}" data-petal-library-action="${id}" aria-label="添加${action.label}到当前花瓣">
      <span class="petal-action-icon"><i data-lucide="${action.icon}"></i></span>
      <span class="petal-action-copy"><strong>${action.label}</strong><small>${action.group}</small></span>
      <i data-lucide="grip-vertical"></i>
    </button>
  `).join('') : '<div class="petal-library-empty">所有动作均已加入菜单</div>';
  document.querySelector('[data-petal-library-count]').textContent = String(available.length);
  document.querySelectorAll('[data-petal-settings-status]').forEach((status) => {
    status.textContent = `已启用 ${assigned.size} 个操作`;
  });
  refreshIcons();
}

function openPetalEditorRing(actionId) {
  if (!petalActionCatalog[actionId]?.acceptsChildren) return;
  ensurePetalRing(actionId);
  petalEditorRing = actionId;
  renderPetalEditor();
}

function backPetalEditorRing() {
  if (petalEditorRing === 'root') return;
  petalEditorRing = findPetalParentRing(petalEditorRing);
  renderPetalEditor();
}

function addPetalToCurrentRing(actionId) {
  const destination = ensurePetalRing(petalEditorRing);
  const slot = firstEmptyPetalSlot(destination);
  if (slot < 0) {
    showToast('当前花瓣已满');
    return;
  }
  const snapshot = petalEditorSnapshot();
  removePetalAction(actionId);
  destination[slot] = actionId;
  petalHaptic();
  commitPetalEditor(`${petalActionCatalog[actionId].label}已加入${petalEditorRing === 'root' ? '主菜单' : petalActionCatalog[petalEditorRing].label}`, snapshot);
}

function clearPetalDropHighlights() {
  document.querySelectorAll('.is-drop-target, .is-group-target').forEach((element) => {
    element.classList.remove('is-drop-target', 'is-group-target');
  });
}

function setPetalDragTarget(target) {
  clearPetalDropHighlights();
  if (!target) return;
  if (target.type === 'group') target.element.classList.add('is-group-target');
  else target.element.classList.add('is-drop-target');
}

function petalDropTargetAt(clientX, clientY, sourceId) {
  for (const node of document.querySelectorAll('[data-petal-group-target]')) {
    if (node.dataset.petalNode === sourceId) continue;
    const rect = node.getBoundingClientRect();
    if (Math.hypot(clientX - (rect.left + rect.width / 2), clientY - (rect.top + rect.height / 2)) <= 39) {
      return { type: 'group', actionId: node.dataset.petalNode, element: node };
    }
  }

  if (petalEditorRing !== 'root') {
    const hub = document.querySelector('[data-petal-editor-hub]');
    const rect = hub.getBoundingClientRect();
    if (Math.hypot(clientX - (rect.left + rect.width / 2), clientY - (rect.top + rect.height / 2)) <= 44) {
      return { type: 'hub', element: hub };
    }
  }

  let nearest;
  for (const slot of document.querySelectorAll('[data-petal-slot]')) {
    const rect = slot.getBoundingClientRect();
    const distance = Math.hypot(clientX - (rect.left + rect.width / 2), clientY - (rect.top + rect.height / 2));
    if (distance <= 47 && (!nearest || distance < nearest.distance)) {
      nearest = { type: 'slot', index: Number(slot.dataset.petalSlot), element: slot, distance };
    }
  }
  if (nearest) return nearest;

  const library = document.querySelector('[data-petal-library-drop]');
  const rect = library.getBoundingClientRect();
  if (clientX >= rect.left && clientX <= rect.right && clientY >= rect.top && clientY <= rect.bottom) {
    return { type: 'library', element: library };
  }
  return null;
}

function beginPetalEditorDrag() {
  if (!petalEditorPress) return;
  const action = petalActionCatalog[petalEditorPress.actionId];
  const ghost = document.createElement('div');
  ghost.className = 'petal-drag-ghost';
  ghost.innerHTML = `<i data-lucide="${action.icon}"></i><span>${action.label}</span>`;
  document.body.append(ghost);
  petalEditorDrag = {
    actionId: petalEditorPress.actionId,
    source: petalEditorPress.source,
    ghost,
    target: null,
    snapshot: petalEditorSnapshot(),
  };
  petalEditorPress.source.classList.add('is-dragging');
  document.querySelector('[data-petal-editor-stage]').classList.add('is-drag-active');
  petalHaptic(12);
  refreshIcons();
  updatePetalEditorDrag(petalEditorPress.clientX, petalEditorPress.clientY);
}

function updatePetalEditorDrag(clientX, clientY) {
  if (!petalEditorDrag) return;
  petalEditorDrag.ghost.style.left = `${clientX}px`;
  petalEditorDrag.ghost.style.top = `${clientY}px`;
  const target = petalDropTargetAt(clientX, clientY, petalEditorDrag.actionId);
  const changed = target?.type !== petalEditorDrag.target?.type ||
    target?.index !== petalEditorDrag.target?.index ||
    target?.actionId !== petalEditorDrag.target?.actionId;
  petalEditorDrag.target = target;
  setPetalDragTarget(target);
  if (changed && target) petalHaptic(6);
}

function cleanupPetalEditorDrag() {
  window.clearTimeout(petalEditorPressTimer);
  petalEditorDrag?.source.classList.remove('is-dragging');
  petalEditorDrag?.ghost.remove();
  document.querySelector('[data-petal-editor-stage]')?.classList.remove('is-drag-active');
  clearPetalDropHighlights();
  petalEditorPress = undefined;
  petalEditorDrag = undefined;
}

function applyPetalDrop(drag) {
  const { actionId, target, snapshot } = drag;
  if (!target) return;
  const action = petalActionCatalog[actionId];
  const sourceLocation = findPetalLocation(actionId);

  if (target.type === 'library') {
    if (!sourceLocation) return;
    unassignPetalSubtree(actionId);
    petalHaptic(14);
    commitPetalEditor(`${action.label}已移出菜单`, snapshot);
    return;
  }

  if (target.type === 'group') {
    const destination = ensurePetalRing(target.actionId);
    const empty = firstEmptyPetalSlot(destination);
    if (empty < 0) {
      showToast(`${petalActionCatalog[target.actionId].label}已满`);
      return;
    }
    removePetalAction(actionId);
    destination[empty] = actionId;
    petalEditorRing = target.actionId;
    petalHaptic(14);
    commitPetalEditor(`${action.label}已放入${petalActionCatalog[target.actionId].label}`, snapshot);
    return;
  }

  if (target.type === 'hub') {
    const root = ensurePetalRing('root');
    const empty = firstEmptyPetalSlot(root);
    if (empty < 0) {
      showToast('主菜单已满');
      return;
    }
    removePetalAction(actionId);
    root[empty] = actionId;
    petalEditorRing = 'root';
    petalHaptic(14);
    commitPetalEditor(`${action.label}已移到主菜单`, snapshot);
    return;
  }

  const destination = ensurePetalRing(petalEditorRing);
  if (sourceLocation?.ring === petalEditorRing) {
    if (sourceLocation.index === target.index) return;
    const displaced = destination[target.index];
    destination[sourceLocation.index] = displaced;
    destination[target.index] = actionId;
  } else {
    removePetalAction(actionId);
    if (!placePetalWithShift(destination, actionId, target.index)) {
      restorePetalEditor(snapshot);
      showToast('当前花瓣已满');
      return;
    }
  }
  petalHaptic(14);
  commitPetalEditor(`${action.label}位置已更新`, snapshot);
}

function finishPetalEditorDrag() {
  if (!petalEditorDrag) {
    window.clearTimeout(petalEditorPressTimer);
    petalEditorPress = undefined;
    return;
  }
  const drag = petalEditorDrag;
  suppressPetalClickUntil = Date.now() + 450;
  cleanupPetalEditorDrag();
  applyPetalDrop(drag);
}

document.addEventListener('click', (event) => {
  if (event.target.closest('[data-undo-petal]')) {
    const undo = toastUndoAction;
    toastUndoAction = undefined;
    if (undo) undo();
    return;
  }

  if (event.target.closest('[data-reset-petal-layout]')) {
    const snapshot = petalEditorSnapshot();
    petalEditorRings = clonePetalRings(petalEditorDefaults);
    petalEditorRing = 'root';
    commitPetalEditor('花瓣菜单已恢复默认布局', snapshot);
    return;
  }

  if (event.target.closest('[data-petal-editor-hub]')) {
    backPetalEditorRing();
    return;
  }

  const libraryAction = event.target.closest('[data-petal-library-action]');
  if (libraryAction) {
    event.preventDefault();
    if (Date.now() < suppressPetalClickUntil) return;
    addPetalToCurrentRing(libraryAction.dataset.petalLibraryAction);
    return;
  }

  const editorNode = event.target.closest('[data-petal-node]');
  if (editorNode) {
    event.preventDefault();
    if (Date.now() < suppressPetalClickUntil) return;
    openPetalEditorRing(editorNode.dataset.petalNode);
    return;
  }

  const resourceTrigger = event.target.closest('[data-open-resources]');
  if (resourceTrigger) {
    event.preventDefault();
    event.stopPropagation();
    openResources(resourceTrigger);
    return;
  }

  if (event.target.closest('[data-close-resources]') || event.target === resourceBackdrop) {
    closeResources();
    return;
  }

  const petalAction = event.target.closest('[data-petal-action]');
  if (petalAction) {
    const action = petalAction.dataset.petalAction;
    closePetal();
    window.setTimeout(() => showToast(`${action} 操作已选择`), 140);
    return;
  }

  if (event.target.closest('[data-close-petal]') || event.target === petalMenu) {
    closePetal();
    return;
  }

  if (event.target.closest('[data-export-config]')) {
    copyThemeConfig();
    return;
  }

  const nav = event.target.closest('[data-nav]');
  if (nav) {
    event.preventDefault();
    showScreen(nav.dataset.nav);
    return;
  }

  const previewNav = event.target.closest('[data-preview-nav]');
  if (previewNav) {
    showScreen(previewNav.dataset.previewNav);
    return;
  }

  const themeToggle = event.target.closest('[data-theme-toggle]');
  if (themeToggle) {
    setTheme(body.dataset.theme === 'dark' ? 'light' : 'dark');
    return;
  }

  if (event.target.closest('[data-open-tuner]')) {
    openTuner();
    return;
  }

  if (event.target.closest('[data-close-tuner]') || event.target === tunerBackdrop) {
    closeTuner();
    return;
  }

  const tunerTheme = event.target.closest('[data-tuner-theme]');
  if (tunerTheme) {
    setTheme(tunerTheme.dataset.tunerTheme);
    return;
  }

  const preset = event.target.closest('[data-preset]');
  if (preset) {
    applyPalette(
      { ...paletteValues, accent: presets[preset.dataset.preset].accent },
      preset.dataset.preset,
    );
    return;
  }

  const rgbToken = event.target.closest('[data-rgb-token]');
  if (rgbToken) {
    activeRgbToken = rgbToken.dataset.rgbToken;
    updateRgbEditor();
    return;
  }

  if (event.target.closest('[data-tuner-reset]')) {
    applyPalette(presets.cyan, 'cyan');
    document.querySelector('[data-radius-range]').value = '14';
    setRadius(14);
    return;
  }

  const appearance = event.target.closest('[data-appearance]');
  if (appearance) {
    const preference = appearance.dataset.appearance;
    const resolved = preference === 'system'
      ? (window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light')
      : preference;
    setTheme(resolved);
    if (preference === 'system') {
      appearance.classList.add('is-selected');
      appearance.setAttribute('aria-pressed', 'true');
    }
    return;
  }

  const refresh = event.target.closest('[data-refresh]');
  if (refresh) {
    refresh.classList.add('is-spinning');
    window.setTimeout(() => refresh.classList.remove('is-spinning'), 720);
    window.setTimeout(() => showToast('状态已同步'), 480);
    return;
  }

  const open = event.target.closest('[data-open-sheet]');
  if (open) {
    openSheet(open.dataset.openSheet);
    return;
  }

  if (event.target.closest('[data-close-sheet]')) {
    closeSheet();
    return;
  }

  if (event.target === backdrop) {
    closeSheet();
    return;
  }

  const action = event.target.closest('[data-sheet-action]');
  if (action) {
    const message = terminalContent.hidden ? '设备配对请求已发送' : '终端已创建';
    closeSheet();
    window.setTimeout(() => showToast(message), 220);
    return;
  }

  const filter = event.target.closest('[data-filter]');
  if (filter) {
    document.querySelectorAll('[data-filter]').forEach((button) => {
      const selected = button === filter;
      button.classList.toggle('is-selected', selected);
      button.setAttribute('aria-pressed', String(selected));
    });
    document.querySelectorAll('[data-terminal-state]').forEach((card) => {
      card.hidden = filter.dataset.filter !== 'all' && card.dataset.terminalState !== filter.dataset.filter;
    });
    return;
  }

  const toggle = event.target.closest('.toggle');
  if (toggle) {
    const enabled = !toggle.classList.contains('is-on');
    toggle.classList.toggle('is-on', enabled);
    toggle.setAttribute('aria-checked', String(enabled));
    showToast(`${toggle.getAttribute('aria-label')}已${enabled ? '开启' : '关闭'}`);
    return;
  }

  const more = event.target.closest('[data-more]');
  if (more) {
    event.stopPropagation();
    showToast(`${more.dataset.more}：样机中保留原有功能入口`);
  }
});

document.addEventListener('keydown', (event) => {
  if (event.key === 'Escape' && !petalMenu.hidden) closePetal();
  else if (event.key === 'Escape' && !resourceBackdrop.hidden) closeResources();
  else if (event.key === 'Escape' && !backdrop.hidden) closeSheet();
  else if (event.key === 'Escape' && !tuner.hidden) closeTuner();
  if ((event.key === 'Enter' || event.key === ' ') && event.target.matches('[role="button"][data-nav]')) {
    event.preventDefault();
    showScreen(event.target.dataset.nav);
  }
});

function cancelHold() {
  window.clearTimeout(holdTimer);
  holdTimer = undefined;
  holdStart = undefined;
  terminalSurface.classList.remove('is-holding');
}

terminalSurface.addEventListener('pointerdown', (event) => {
  if (event.button !== 0 || !petalMenu.hidden) return;
  const rect = terminalSurface.getBoundingClientRect();
  holdStart = { x: event.clientX, y: event.clientY };
  terminalSurface.style.setProperty('--hold-x', `${event.clientX - rect.left}px`);
  terminalSurface.style.setProperty('--hold-y', `${event.clientY - rect.top}px`);
  terminalSurface.classList.add('is-holding');
  holdTimer = window.setTimeout(() => {
    const point = holdStart;
    cancelHold();
    if (point) openPetal(point.x, point.y);
  }, 480);
});

terminalSurface.addEventListener('pointermove', (event) => {
  if (!holdStart) return;
  if (Math.hypot(event.clientX - holdStart.x, event.clientY - holdStart.y) > 10) cancelHold();
});

terminalSurface.addEventListener('pointerup', cancelHold);
terminalSurface.addEventListener('pointercancel', cancelHold);
terminalSurface.addEventListener('pointerleave', (event) => {
  if (event.pointerType === 'mouse') cancelHold();
});
terminalSurface.addEventListener('contextmenu', (event) => {
  event.preventDefault();
  cancelHold();
  openPetal(event.clientX, event.clientY);
});

document.addEventListener('pointerdown', (event) => {
  const source = event.target.closest('[data-petal-drag-source]');
  if (!source || event.button !== 0) return;
  window.clearTimeout(petalEditorPressTimer);
  petalEditorPress = {
    actionId: source.dataset.petalDragSource,
    source,
    pointerId: event.pointerId,
    startX: event.clientX,
    startY: event.clientY,
    clientX: event.clientX,
    clientY: event.clientY,
  };
  source.setPointerCapture?.(event.pointerId);
  petalEditorPressTimer = window.setTimeout(beginPetalEditorDrag, 280);
});

document.addEventListener('pointermove', (event) => {
  if (!petalEditorPress || event.pointerId !== petalEditorPress.pointerId) return;
  petalEditorPress.clientX = event.clientX;
  petalEditorPress.clientY = event.clientY;
  if (!petalEditorDrag) {
    if (Math.hypot(event.clientX - petalEditorPress.startX, event.clientY - petalEditorPress.startY) > 8) {
      window.clearTimeout(petalEditorPressTimer);
      petalEditorPress = undefined;
    }
    return;
  }
  event.preventDefault();
  updatePetalEditorDrag(event.clientX, event.clientY);
});

document.addEventListener('pointerup', (event) => {
  if (!petalEditorPress || event.pointerId !== petalEditorPress.pointerId) return;
  finishPetalEditorDrag();
});

document.addEventListener('pointercancel', (event) => {
  if (!petalEditorPress || event.pointerId !== petalEditorPress.pointerId) return;
  cleanupPetalEditorDrag();
});

document.querySelectorAll('[data-rgb-channel]').forEach((input) => {
  input.addEventListener('input', () => {
    const channel = input.dataset.rgbChannel;
    const rgb = hexToRgb(paletteValues[activeRgbToken]);
    rgb[channel] = Math.max(0, Math.min(255, Number(input.value) || 0));
    paletteValues[activeRgbToken] = rgbToHex(rgb);
    applyPalette(currentPalette(), null);
  });
});

const colorPlane = document.querySelector('[data-color-plane]');

function updateColorPlane(event) {
  const rect = colorPlane.getBoundingClientRect();
  const saturation = Math.max(0, Math.min(1, (event.clientX - rect.left) / rect.width));
  const value = 1 - Math.max(0, Math.min(1, (event.clientY - rect.top) / rect.height));
  const hue = Number(document.querySelector('[data-hue-range]').value);
  paletteValues[activeRgbToken] = rgbToHex(hsvToRgb({ h: hue, s: saturation, v: value }));
  applyPalette(currentPalette(), null);
}

colorPlane.addEventListener('pointerdown', (event) => {
  colorPlane.setPointerCapture(event.pointerId);
  updateColorPlane(event);
});

colorPlane.addEventListener('pointermove', (event) => {
  if (colorPlane.hasPointerCapture(event.pointerId)) updateColorPlane(event);
});

colorPlane.addEventListener('keydown', (event) => {
  const hsv = rgbToHsv(hexToRgb(paletteValues[activeRgbToken]));
  if (event.key === 'ArrowLeft') hsv.s -= 0.02;
  else if (event.key === 'ArrowRight') hsv.s += 0.02;
  else if (event.key === 'ArrowUp') hsv.v += 0.02;
  else if (event.key === 'ArrowDown') hsv.v -= 0.02;
  else return;
  event.preventDefault();
  hsv.s = Math.max(0, Math.min(1, hsv.s));
  hsv.v = Math.max(0, Math.min(1, hsv.v));
  paletteValues[activeRgbToken] = rgbToHex(hsvToRgb(hsv));
  applyPalette(currentPalette(), null);
});

document.querySelector('[data-hue-range]').addEventListener('input', (event) => {
  const hsv = rgbToHsv(hexToRgb(paletteValues[activeRgbToken]));
  hsv.h = Number(event.target.value);
  paletteValues[activeRgbToken] = rgbToHex(hsvToRgb(hsv));
  applyPalette(currentPalette(), null);
});

document.querySelector('[data-radius-range]').addEventListener('input', (event) => {
  setRadius(event.target.value);
});

document.querySelector('[data-command-form]').addEventListener('submit', (event) => {
  event.preventDefault();
  const input = event.currentTarget.elements.command;
  const command = input.value.trim();
  if (!command) return;
  const output = document.querySelector('[data-terminal-output]');
  const cursorLine = output.lastElementChild;
  cursorLine.remove();
  const commandLine = document.createElement('p');
  commandLine.textContent = `lozzow@studio ~/work/anytty $ ${command}`;
  const resultLine = document.createElement('p');
  resultLine.className = 'dim';
  resultLine.textContent = command === 'clear' ? '' : `command sent · ${command}`;
  const nextPrompt = document.createElement('p');
  nextPrompt.innerHTML = '<span class="prompt">lozzow@studio</span> <span class="path">~/work/anytty</span> $ <span class="cursor-block" aria-hidden="true"></span>';
  if (command === 'clear') output.replaceChildren(nextPrompt);
  else output.append(commandLine, resultLine, nextPrompt);
  input.value = '';
  output.scrollTop = output.scrollHeight;
});

refreshIcons();
setTheme('light');
setRadius(14);
updateRgbEditor();
renderPetalEditor();
