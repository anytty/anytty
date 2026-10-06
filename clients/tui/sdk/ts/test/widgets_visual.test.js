'use strict';

const { test } = require('node:test');
const assert = require('node:assert/strict');

const builder = require('../src/builder');
const tokens = require('../src/widgets/tokens');
const border = require('../src/widgets/border');
const layout = require('../src/widgets/layout');
const basics = require('../src/widgets/basics');
const progress = require('../src/widgets/progress');
const theme = require('../src/widgets/theme');

function tree(widget) {
  return widget.build().build();
}

function boxText(box) {
  if (!box || !box.content) return '';
  return box.content.text || '';
}

function rowText(box) {
  if (!box) return '';
  let out = boxText(box);
  for (const child of box.children || []) out += rowText(child);
  return out;
}

function findBox(root, id) {
  if (!root) return null;
  if (root.id === id) return root;
  for (const child of root.children || []) {
    const found = findBox(child, id);
    if (found) return found;
  }
  return null;
}

function sizeOf(box) {
  return box.size || [0, 0, 0];
}

test('token constants match host names', () => {
  const cases = {
    STYLE_DEFAULT: 'default',
    STYLE_BACKGROUND: 'bg',
    STYLE_FG: 'fg',
    STYLE_FOREGROUND: 'foreground',
    STYLE_STRONG_FOREGROUND: 'strong-foreground',
    STYLE_MUTED: 'muted',
    STYLE_ACCENT: 'accent',
    STYLE_ACCENT_DIM: 'accent_dim',
    STYLE_SUCCESS: 'success',
    STYLE_OK: 'ok',
    STYLE_WARNING: 'warning',
    STYLE_DANGER: 'danger',
    STYLE_INFO: 'info',
    STYLE_CHROME: 'chrome',
    STYLE_CHROME_FOCUS: 'chrome_focus',
    STYLE_HEADER: 'header',
    STYLE_TAB_ACTIVE: 'tab_active',
    STYLE_TAB_INACTIVE: 'tab_inactive',
    STYLE_FOOTER: 'footer',
    STYLE_FOOTER_ACCENT: 'footer-accent',
    STYLE_STATUS: 'status',
    STYLE_OVERLAY: 'overlay',
    STYLE_BORDER: 'border',
    STYLE_BORDER_FOCUS: 'border_focus',
    STYLE_BORDER_DEAD: 'border_dead',
    STYLE_ACTIVE_BORDER: 'active-border',
    STYLE_INACTIVE_BORDER: 'inactive-border',
    STYLE_SELECTION: 'selection',
  };
  for (const [name, want] of Object.entries(cases)) {
    assert.equal(tokens[name], want, name);
  }
});

test('style attribute helpers are idempotent', () => {
  assert.equal(tokens.withBold(''), 'bold');
  assert.equal(tokens.withBold(tokens.withBold('accent')), 'accent;bold');
  assert.equal(tokens.withUnderline(''), 'underline');
  assert.equal(tokens.withReverse(tokens.withReverse('fg:#101010')), 'fg:#101010;reverse');
  assert.equal(tokens.withItalic(tokens.withItalic('')), 'italic');
  assert.equal(tokens.withBold('fg:#111111;bold'), 'fg:#111111;bold');
});

test('style color helpers replace or prepend segments', () => {
  assert.equal(tokens.withFg('', '#aabbcc'), 'fg:#aabbcc');
  assert.equal(tokens.withFg('bold', '#aabbcc'), 'fg:#aabbcc;bold');
  assert.equal(tokens.withFg('fg:#111111;bold', '#aabbcc'), 'fg:#aabbcc;bold');
  assert.equal(tokens.withBg('fg:#111111;bg:#000000', '#ffffff'), 'fg:#111111;bg:#ffffff');
  assert.equal(tokens.withFg('bold', ''), 'bold');
  const once = tokens.withFg('bold', '#aabbcc');
  assert.equal(tokens.withFg(once, '#aabbcc'), once);
});

test('border presets and defaults', () => {
  const normal = border.borderNormal();
  assert.equal(normal.topLeft, '┌');
  assert.equal(normal.topRight, '┐');
  assert.equal(normal.bottomLeft, '└');
  assert.equal(normal.bottomRight, '┘');
  assert.equal(normal.top, '─');
  assert.equal(normal.bottom, '─');
  assert.equal(normal.left, '│');
  assert.equal(normal.right, '│');

  const rounded = border.borderRounded();
  assert.equal(rounded.topLeft, '╭');
  assert.equal(rounded.topRight, '╮');
  assert.equal(rounded.bottomLeft, '╰');
  assert.equal(rounded.bottomRight, '╯');

  const thick = border.borderThick();
  assert.equal(thick.topLeft, '┏');
  assert.equal(thick.top, '━');
  assert.equal(thick.left, '┃');

  const double = border.borderDouble();
  assert.equal(double.topLeft, '╔');
  assert.equal(double.top, '═');
  assert.equal(double.left, '║');

  assert.equal(border.borderSetOrDefault(border.borderRounded()).topLeft, '╭');
  assert.equal(border.borderSetOrDefault(new border.BorderSet()).topLeft, '┌');
});

test('border box draws frame and title', () => {
  const root = tree(new border.BorderBox({ id: 'panel', title: 'hi', width: 10, height: 4 }));
  assert.equal(root.id, 'panel');
  assert.deepEqual(root.size, [10, 4, 0]);
  const children = root.children;
  assert.equal(children.length, 4);
  const top = rowText(children[0]);
  assert.ok(top.startsWith('┌'));
  assert.ok(top.endsWith('┐'));
  assert.ok(top.includes(' hi '));
  assert.equal(builder.displayWidth(top), 10);
  assert.equal(rowText(children[3]), '└' + '─'.repeat(8) + '┘');
  const body = children[1].children;
  assert.equal(body.length, 3);
  assert.equal(boxText(body[0]), '│');
  assert.equal(boxText(body[2]), '│');
});

test('border box truncates a too-long title', () => {
  const root = tree(new border.BorderBox({ title: 'a very long title indeed', width: 8, height: 3 }));
  const top = rowText(root.children[0]);
  assert.equal(builder.displayWidth(top), 8);
  assert.ok(top.includes(' a ver'));
  assert.ok(top.endsWith('┐'));
});

test('border box title is CJK safe', () => {
  const root = tree(new border.BorderBox({ title: '终端列表', width: 9, height: 3 }));
  const top = rowText(root.children[0]);
  assert.equal(builder.displayWidth(top), 9);
  assert.ok(top.endsWith('┐'));
});

test('border box degrades when zero sized', () => {
  const child = builder.text('body');
  const root = tree(new border.BorderBox({ child }));
  assert.equal(sizeOf(root)[0], 6);
  assert.equal(sizeOf(root)[1], 3);

  const tiny = tree(new border.BorderBox({ child, width: 1, height: 1 }));
  assert.equal(tiny.children.length, 1);
  assert.equal(rowText(tiny), 'body');

  const empty = tree(new border.BorderBox({ width: 0, height: 0 }));
  assert.equal(sizeOf(empty)[0], 2);
  assert.equal(sizeOf(empty)[1], 2);
});

test('border box styles default and embed the child', () => {
  const child = builder.text('x').height(1).flex(1);
  const root = tree(new border.BorderBox({
    child,
    title: 'T',
    width: 6,
    height: 3,
    style: 'fg:#ff0000',
    titleStyle: 'accent',
  }));
  const top = root.children[0].children;
  assert.equal(top[0].style, 'fg:#ff0000');
  assert.ok(top.length >= 2);
  assert.equal(top[1].style, 'accent');

  const def = tree(new border.BorderBox({ width: 4, height: 2 }));
  assert.equal(def.children[0].children[0].style, tokens.STYLE_BORDER);
});

test('distribute splits cells over ratios', () => {
  const cases = [
    [10, [1, 1], [5, 5]],
    [10, [1, 2], [3, 7]],
    [5, [1, 1, 1], [1, 1, 3]],
    [2, [1, 1, 1], [1, 1, 1]],
    [0, [1, 1], [1, 1]],
    [7, [], []],
    [7, [1], [7]],
  ];
  for (const [avail, ratios, want] of cases) {
    assert.deepEqual(layout.distribute(avail, ratios), want, `distribute(${avail}, ${ratios})`);
  }
});

test('split layout solves pane and divider rects', () => {
  const rowLayout = new layout.SplitLayout({ orient: 'row', weights: [1, 2], gap: 1 });
  const [panes, dividers] = rowLayout.rects(10, 3);
  assert.equal(panes.length, 2);
  assert.equal(dividers.length, 1);
  assert.deepEqual({ ...panes[0] }, { x: 0, y: 0, w: 3, h: 3 });
  assert.deepEqual({ ...dividers[0] }, { x: 3, y: 0, w: 1, h: 3 });
  assert.equal(panes[1].x, 4);
  assert.equal(panes[1].w, 6);

  const colLayout = new layout.SplitLayout({ orient: 'col', weights: [1, 1], gap: 1 });
  const [colPanes, colDividers] = colLayout.rects(4, 5);
  assert.equal(colPanes[0].h, 2);
  assert.equal(colDividers[0].y, 2);
  assert.equal(colPanes[1].y, 3);
});

test('chrome widgets build', () => {
  const title = new layout.TitleBar({
    id: 'title',
    width: 20,
    left: [new basics.Segment({ text: ' pane ' })],
    buttons: [new basics.Button({ id: 'btn:close', text: 'x', input: ['mouse'] })],
  });
  const titleBox = tree(title);
  assert.equal(titleBox.id, 'title');
  assert.ok(titleBox.children.length > 0);
  assert.equal(title.line(), ' pane ' + ' '.repeat(13) + 'x');

  const footer = new layout.Footer({
    id: 'footer',
    width: 30,
    hasBadge: true,
    badge: new basics.Segment({ text: ' CTRL ' }),
    groups: [new basics.Segment({ text: ' P PANE' }), new basics.Segment({ text: ' T TAB' })],
    right: [new basics.Segment({ text: ' ws:main ' })],
  });
  assert.equal(builder.displayWidth(footer.line()), 30);
  assert.equal(tree(footer).id, 'footer');

  const picker = new layout.Picker({
    id: 'picker',
    title: 'Terminals',
    width: 24,
    rows: [
      new layout.PickerRow({ text: 'term-1', id: 'picker:0', selectable: true }),
      new layout.PickerRow({ text: 'term-2', id: 'picker:1', selected: true, selectable: true }),
    ],
  });
  const pickerBox = tree(picker);
  assert.equal(pickerBox.id, 'picker');
  assert.ok(pickerBox.children.length >= 3);
  const selected = findBox(pickerBox, 'picker:1');
  assert.ok(boxText(selected).startsWith('▸ term-2'));
  assert.deepEqual(selected.input, ['mouse']);
  const plain = findBox(pickerBox, 'picker:0');
  assert.ok(boxText(plain).startsWith('  term-1'));

  const floating = new layout.FloatingLayer({
    id: 'float-1',
    title: 'float',
    x: 3,
    y: 2,
    width: 20,
    height: 6,
    rows: [new basics.FrameRow({ text: 'body' })],
  });
  const floatBox = tree(floating);
  assert.deepEqual(floatBox.pos, [3, 2]);
  const collapsed = tree(new layout.FloatingLayer({
    id: 'float-1',
    title: 'float',
    width: 20,
    height: 6,
    collapsed: true,
  }));
  assert.equal(sizeOf(collapsed)[1], 1);

  const toast = tree(new layout.Toast({ id: 'toast', text: 'saved', style: 'fg:#fff' }));
  assert.equal(toast.id, 'toast');
  assert.equal(toast.style, 'fg:#fff');
});

test('tab bar structure', () => {
  const bar = new basics.TabBar({
    left: new basics.Segment({ text: ' local ', style: 'chrome', id: 'workspace' }),
    items: [
      new basics.TabItem({ id: 'tab:0', title: '1:1', active: true }),
      new basics.TabItem({ id: 'tab:1', title: '2:2' }),
    ],
    plus: true,
    plusId: 'tab:new',
  });
  const root = bar.build().id('header').height(1).build();

  const workspace = findBox(root, 'workspace');
  assert.ok(workspace);
  assert.equal(workspace.style, 'chrome');
  const active = findBox(root, 'tab:0');
  assert.equal(active.style, basics.DEFAULT_ACTIVE_STYLE);
  assert.equal(boxText(active), '[1:1]');
  assert.deepEqual(active.input, ['mouse']);
  const inactive = findBox(root, 'tab:1');
  assert.equal(inactive.style, basics.DEFAULT_INACTIVE_STYLE);
  assert.equal(boxText(inactive), ' 2:2 ');
  const plus = findBox(root, 'tab:new');
  assert.equal(boxText(plus), ' + ');
  assert.equal(plus.style, basics.DEFAULT_CHROME_STYLE);
});

test('status bar structure and alignment', () => {
  const bar = new basics.StatusBar({
    left: [new basics.Segment({ text: 'NORMAL', style: 'status' })],
    right: [
      new basics.Segment({ text: 'tab 1/2', style: 'status' }),
      new basics.Segment({ text: '12:00', style: 'muted' }),
    ],
    width: 30,
  });
  const root = tree(bar);
  const children = root.children;
  assert.equal(children.length, 6);
  assert.equal(boxText(children[0]), 'NORMAL');
  assert.equal(boxText(children[1]), ' ' + basics.DEFAULT_SEPARATOR + ' ');
  const spacer = boxText(children[2]);
  assert.ok(builder.displayWidth(spacer) > 0);
  const right = boxText(children[3]) + boxText(children[4]) + boxText(children[5]);
  assert.equal(
    builder.displayWidth('NORMAL' + ' ' + basics.DEFAULT_SEPARATOR + ' ' + spacer + right),
    30,
  );
  assert.equal(
    bar.text(),
    'NORMAL ' + basics.DEFAULT_SEPARATOR + ' tab 1/2 ' + basics.DEFAULT_SEPARATOR + ' 12:00',
  );
});

test('status bar trims the left group first', () => {
  const bar = new basics.StatusBar({
    left: [
      new basics.Segment({ text: 'LONGHINTS', style: 'muted' }),
      new basics.Segment({ text: 'extra', style: 'muted' }),
    ],
    right: [new basics.Segment({ text: '12:00', style: 'muted' })],
    width: 12,
  });
  const root = tree(bar);
  let text = '';
  for (const child of root.children) text += boxText(child);
  assert.ok(builder.displayWidth(text) <= 12);
  const last = root.children[root.children.length - 1];
  assert.ok(boxText(last).includes('12:00'));
});

test('card centers lines and draws borders', () => {
  const card = new basics.Card({
    id: 'slot-1',
    title: '空槽',
    lines: ['Ctrl-F 选择终端', 'Ctrl-P 面板命令'],
    width: 20,
    height: 8,
    style: basics.DEFAULT_BORDER_STYLE,
    center: true,
  });
  const root = tree(card);
  assert.equal(sizeOf(root)[0], 20);
  assert.equal(sizeOf(root)[1], 8);
  const children = root.children;
  assert.equal(children.length, 8);
  const top = boxText(children[0]);
  assert.ok(top.includes('空槽'));
  assert.ok(top.startsWith('┌─'));
  assert.ok(top.endsWith('┐'));
  assert.equal(children[0].style, basics.DEFAULT_BORDER_STYLE);
  assert.equal(boxText(children[7]), '└' + '─'.repeat(18) + '┘');
  const row = children[3].children;
  assert.equal(row.length, 3);
  assert.equal(boxText(row[0]), '│');
  assert.equal(boxText(row[2]), '│');
  const hint = boxText(row[1]);
  assert.ok(hint.includes('Ctrl-F 选择终端'));
  assert.ok(hint.startsWith(' '));
  assert.equal(builder.displayWidth(hint), 18);
});

test('divider builds a gutter run', () => {
  const vertical = tree(new basics.Divider({
    id: 'divider:0',
    vertical: true,
    length: 4,
    style: 'fg:#aabbcc',
    input: ['mouse'],
  }));
  assert.equal(vertical.id, 'divider:0');
  assert.equal(boxText(vertical), '││││');
  assert.equal(sizeOf(vertical)[0], 1);
  assert.equal(sizeOf(vertical)[1], 4);
  assert.equal(vertical.style, 'fg:#aabbcc');
  assert.deepEqual(vertical.input, ['mouse']);

  const horizontal = tree(new basics.Divider({ length: 3 }));
  assert.equal(boxText(horizontal), '───');
  assert.equal(sizeOf(horizontal)[1], 1);
  assert.equal(horizontal.style, basics.DEFAULT_SEP_STYLE);
});

test('frame draws chrome and preserves row ids', () => {
  const frame = new basics.Frame({
    id: 'overlay:x',
    title: 'terminals',
    width: 20,
    height: 5,
    style: 'fg:#ffffff',
    rows: [new basics.FrameRow({ text: 'select', id: 'pick:0', input: ['mouse'] })],
  });
  const root = tree(frame);
  const children = root.children;
  assert.equal(children.length, 5);
  assert.ok(boxText(children[0]).includes('terminals'));
  assert.equal(children[0].style, 'fg:#ffffff');
  assert.equal(boxText(children[4]), '└' + '─'.repeat(18) + '┘');
  const row = findBox(root, 'pick:0');
  assert.ok(row);
  assert.equal(sizeOf(row)[0], 18);
  assert.equal(sizeOf(row)[1], 1);
  assert.deepEqual(row.input, ['mouse']);
});

test('button hotkey splits runs and stays clickable', () => {
  const button = new basics.Button({ id: 'btn:new', text: 'New terminal', hot: 'N' });
  const box = tree(button);
  assert.equal(box.id, 'btn:new');
  const children = box.children;
  assert.equal(children.length, 2);
  assert.equal(boxText(children[0]), 'N');
  assert.equal(children[0].style, basics.DEFAULT_BUTTON_STYLE);
  assert.equal(boxText(children[1]), 'ew terminal');

  const container = builder.row(button.build(), builder.text('   ')).width(20).build();
  assert.equal(container.children[0].id, 'btn:new');
  assert.deepEqual(container.children[0].input, ['mouse']);
});

test('key hint renders mode, separator and keys', () => {
  const hint = new basics.KeyHint({
    mode: 'SCROLL',
    keys: ['PgUp/PgDn scroll', 'y copy'],
    id: 'keyhint',
  });
  const root = tree(hint);
  const children = root.children;
  assert.equal(children.length, 3);
  assert.equal(boxText(children[0]), 'SCROLL');
  assert.equal(children[0].style, basics.DEFAULT_STATUS_STYLE);
  assert.equal(boxText(children[2]), 'PgUp/PgDn scroll · y copy');
  assert.equal(
    hint.text(),
    'SCROLL ' + basics.DEFAULT_SEPARATOR + ' PgUp/PgDn scroll · y copy',
  );
});

test('progress bar fraction, bar text and segments', () => {
  const bar = new progress.ProgressBar({
    value: 50,
    max: 200,
    width: 10,
    label: 'load ',
    style: 'fill',
    trackStyle: 'track',
    percentStyle: 'pct',
  });
  assert.equal(bar.fraction(), 0.25);
  assert.equal(bar.percent(), 25);
  const text = bar.barText();
  assert.equal([...text].length, 10);
  assert.equal((text.match(/█/g) || []).length, 3);
  const children = tree(bar).children;
  assert.equal(children.length, 4);
  assert.equal(boxText(children[0]), 'load ');
  assert.ok(!children[0].style);
  assert.equal(boxText(children[1]), '███');
  assert.equal(children[1].style, 'fill');
  assert.equal(boxText(children[2]), '░░░░░░░');
  assert.equal(children[2].style, 'track');
  assert.equal(boxText(children[3]), '25%');
  assert.equal(children[3].style, 'pct');
  assert.equal([...new progress.ProgressBar().barText()].length, progress.DEFAULT_PROGRESS_WIDTH);
});

test('progress bar clamps and rounds', () => {
  const cases = [
    [-5, 10, 0],
    [30, 10, 100],
    [1, 3, 33],
    [2, 3, 67],
    [0, 0, 0],
    [5, -1, 0],
  ];
  for (const [value, max, percent] of cases) {
    assert.equal(new progress.ProgressBar({ value, max }).percent(), percent, `${value}/${max}`);
  }
  const children = tree(new progress.ProgressBar({ value: 3, max: 4, hidePercent: true })).children;
  assert.equal(children.length, 2);
});

test('spinner frame and label', () => {
  const spinner = new progress.Spinner({
    frame: 11,
    frames: ['a', 'b', 'c'],
    style: 'spin',
    label: 'working',
    labelStyle: 'muted',
  });
  assert.equal(spinner.frameText(), 'c');
  assert.equal(new progress.Spinner({ frame: -1, frames: ['a', 'b', 'c'] }).frameText(), 'c');
  assert.notEqual(new progress.Spinner().frameText(), '');
  const children = tree(spinner).children;
  assert.equal(children.length, 2);
  assert.equal(boxText(children[0]), 'c');
  assert.equal(children[0].style, 'spin');
  assert.equal(boxText(children[1]), 'working');
  assert.equal(children[1].style, 'muted');
});

test('badge and tags', () => {
  const badge = tree(new progress.Badge({
    id: 'badge:1',
    text: '3',
    style: 'accent',
    input: ['mouse'],
  }));
  assert.equal(badge.id, 'badge:1');
  assert.equal(boxText(badge), '3');
  assert.equal(badge.style, 'accent');
  assert.deepEqual(badge.input, ['mouse']);

  const tags = tree(new progress.Tags({
    id: 'tags',
    style: 'base',
    sepStyle: 'sep',
    items: [
      new progress.Tag({ text: 'go', id: 'tag:go' }),
      new progress.Tag({ text: 'ui', style: 'hot' }),
      new progress.Tag({ text: 'cjk 中文' }),
    ],
  }));
  const children = tags.children;
  assert.equal(children.length, 5);
  assert.equal(boxText(children[0]), 'go');
  assert.equal(children[0].style, 'base');
  assert.equal(children[0].id, 'tag:go');
  assert.equal(children[2].style, 'hot');
  assert.equal(builder.displayWidth(rowText(tags)), 14);
});

test('theme variants', () => {
  assert.deepEqual(theme.defaultTheme(), theme.darkTheme());
  assert.notDeepEqual(theme.darkTheme(), theme.lightTheme());
  assert.equal(theme.themeByName('light')[1], true);
  assert.equal(theme.themeByName('neon')[1], false);

  const dark = theme.darkTheme();
  const darkFields = [
    'title', 'accent', 'muted', 'danger', 'success', 'warning', 'border', 'borderFocus',
    'selection', 'selectionText', 'placeholder', 'cursor', 'statusBar', 'tabActive',
    'tabInactive', 'header', 'footer', 'toast', 'overlay', 'backdrop', 'marker',
    'separator', 'zebra',
  ];
  for (const field of darkFields) {
    assert.notEqual(dark[field], '', `DarkTheme.${field}`);
  }
  const light = theme.lightTheme();
  for (const field of ['text', 'accent', 'selection', 'overlay']) {
    assert.notEqual(light[field], '', `LightTheme.${field}`);
  }
});

test('themed constructors fill theme tokens', () => {
  const light = theme.lightTheme();

  const list = theme.themedList(light);
  assert.equal(list.selectedStyle, light.selection);
  assert.equal(list.headerStyle, light.header);
  assert.equal(list.marker, light.marker);
  const virtualList = theme.themedVirtualList(light);
  assert.equal(virtualList.disabledStyle, light.muted);
  assert.equal(virtualList.selectedStyle, light.selection);

  const table = theme.themedTable(light);
  assert.equal(table.headerStyle, light.header);
  assert.equal(table.zebraStyle, light.zebra);
  assert.equal(table.separatorStyle, light.separator);

  const input = theme.themedTextInput(light);
  assert.equal(input.placeholderStyle, light.placeholder);
  assert.equal(input.cursorStyle, light.cursor);
  const area = theme.themedTextArea(light);
  assert.equal(area.style, light.input);
  assert.equal(area.cursorStyle, light.cursor);

  const modal = theme.themedModal(light);
  assert.equal(modal.style, light.borderFocus);
  assert.equal(modal.backdropStyle, light.backdrop);
  const menu = theme.themedMenu(light);
  assert.equal(menu.selectedStyle, light.selection);
  assert.equal(menu.frameStyle, light.border);
  assert.equal(menu.marker, light.marker);

  const bar = theme.themedProgressBar(light);
  assert.equal(bar.style, light.accent);
  assert.equal(bar.trackStyle, light.muted);
  assert.equal(theme.themedSpinner(light).style, light.accent);
  assert.equal(theme.themedBadge(light).style, light.accent);
  const tags = theme.themedTags(light);
  assert.equal(tags.style, light.text);
  assert.equal(tags.sepStyle, light.separator);
});

test('default theme is overridable', () => {
  const previous = theme.defaultTheme();
  theme.setDefaultTheme(theme.lightTheme());
  assert.equal(theme.defaultTheme().name, 'light');
  theme.setDefaultTheme(previous);
  assert.equal(theme.defaultTheme().name, 'dark');
});
