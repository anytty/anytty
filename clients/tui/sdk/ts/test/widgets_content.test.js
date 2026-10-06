'use strict';

// Ports of tui2/sdk/widgets/*_test.go for the content / interaction / form /
// chart half of the widget toolkit. Assertions run on produced box trees and
// pure values, never on rendering.

const { test } = require('node:test');
const assert = require('node:assert/strict');

const builder = require('../src/builder');
const richtext = require('../src/widgets/richtext');
const list = require('../src/widgets/list');
const table = require('../src/widgets/table');
const input = require('../src/widgets/input');
const modal = require('../src/widgets/modal');
const scrollbar = require('../src/widgets/scrollbar');
const mouse = require('../src/widgets/mouse');
const hover = require('../src/widgets/hover');
const contextmenu = require('../src/widgets/contextmenu');
const validate = require('../src/widgets/validate');
const form = require('../src/widgets/form');
const select = require('../src/widgets/select');
const datepicker = require('../src/widgets/datepicker');
const chart = require('../src/widgets/chart');
const format = require('../src/widgets/format');

// ---------------------------------------------------------------- helpers

function findBox(root, id) {
  if (!root) return null;
  if (root.id === id) return root;
  for (const child of root.children || []) {
    const found = findBox(child, id);
    if (found) return found;
  }
  return null;
}

function boxText(b) {
  if (!b || !b.content) return '';
  return b.content.text || '';
}

function rowText(b) {
  if (!b) return '';
  let text = boxText(b);
  for (const child of b.children || []) text += rowText(child);
  return text;
}

function built(widget) {
  return widget.build().build();
}

function childrenOf(box) {
  return box.children || [];
}

// ---------------------------------------------------------------- richtext

test('RichText build merges and styles', () => {
  const root = built(richtext.line(
    richtext.styled('a', 'x'),
    richtext.styled('b', 'x'),
    richtext.styled('', 'z'),
    richtext.styled('c', 'y'),
  ));
  assert.equal(root.children.length, 2);
  assert.equal(boxText(root.children[0]), 'ab');
  assert.equal(root.children[0].style, 'x');
  assert.equal(boxText(root.children[1]), 'c');
  assert.equal(root.children[1].style, 'y');
});

test('RichText width pads and text measures', () => {
  const rich = new richtext.RichText({ spans: [richtext.styled('ab', 'x')], width: 5 });
  assert.equal(rowText(built(rich)), 'ab   ');
  assert.equal(rich.text(), 'ab');
});

test('RichText empty', () => {
  const root = built(richtext.line());
  assert.equal(childrenOf(root).length, 0);
  assert.equal(richtext.line().text(), '');
  const restyled = richtext.line(richtext.styled('hi', 'a'), richtext.styled('!', 'b')).styled('z');
  assert.equal(restyled.text(), 'hi!');
  assert.equal(restyled.spans.length, 1);
  assert.equal(restyled.spans[0].style, 'z');
});

test('WrapText plain', () => {
  const cases = [
    ['simple', 'hello world', 5, ['hello', 'world']],
    ['spaces', 'a b c d', 3, ['a b', 'c d']],
    ['hardbreak', 'abcd ef', 3, ['abc', 'd', 'ef']],
    ['newline', 'ab\ncd', 5, ['ab', 'cd']],
    ['blankline', 'ab\n\ncd', 5, ['ab', '', 'cd']],
    ['exact', 'abcde', 5, ['abcde']],
    ['one', 'a', 1, ['a']],
    ['empty', '', 5, ['']],
    ['leading spaces', '   x', 5, ['x']],
    ['zero width', 'abc', 0, null],
    ['negative width', 'abc', -1, null],
  ];
  for (const [name, text, width, want] of cases) {
    assert.deepEqual(richtext.wrapText(text, width), want, name);
  }
});

test('WrapText wide runes never split', () => {
  assert.deepEqual(richtext.wrapText('你好', 3), ['你', '好']);
  assert.deepEqual(richtext.wrapText('你好', 4), ['你好']);
  assert.deepEqual(richtext.wrapText('a你b', 2), ['a', '你', 'b']);
  for (const line of richtext.wrapText('你好世界', 3)) {
    assert.ok(builder.displayWidth(line) <= 3);
  }
});

test('WrapSpans preserves styles', () => {
  const spans = [{ text: 'ab', style: 'x' }, { text: 'cd', style: 'y' }];
  const got = richtext.wrapSpans(spans, 2);
  assert.equal(got.length, 2);
  assert.equal(got[0].text(), 'ab');
  assert.equal(got[0].spans[0].style, 'x');
  assert.equal(got[1].text(), 'cd');
  assert.equal(got[1].spans[0].style, 'y');
  for (const row of got) assert.equal(row.width, 2);
  assert.equal(richtext.wrapSpans(spans, 0), null);
});

test('WrapSpans splits one run across lines', () => {
  const got = richtext.wrapSpans([{ text: 'abcd', style: 'x' }], 3);
  assert.equal(got.length, 2);
  assert.equal(got[0].text(), 'abc');
  assert.equal(got[1].text(), 'd');
  for (const row of got) {
    assert.equal(row.spans.length, 1);
    assert.equal(row.spans[0].style, 'x');
  }
});

test('RichText wrapSpans marks wrapped', () => {
  const rich = new richtext.RichText({ spans: [richtext.styled('abcdef', 'x')] });
  const rows = rich.wrapSpans(3);
  assert.equal(rows.length, 2);
  assert.ok(rows[0].wrap && rows[1].wrap);
});

test('ParseEmphasis', () => {
  assert.deepEqual(richtext.parseEmphasis('a **b** c'), [
    { text: 'a ', style: '' },
    { text: 'b', style: 'bold' },
    { text: ' c', style: '' },
  ]);
  assert.deepEqual(richtext.parseEmphasis('_em_'), [{ text: 'em', style: 'italic' }]);
  assert.deepEqual(richtext.parseEmphasis('**bold** and _em_'), [
    { text: 'bold', style: 'bold' },
    { text: ' and ', style: '' },
    { text: 'em', style: 'italic' },
  ]);
});

test('ParseEmphasis unclosed and empty stay literal', () => {
  const cases = {
    'a **b': 'a **b',
    '_open': '_open',
    '****': '****',
    '__': '__',
    'a ** b': 'a ** b',
    'a _b c': 'a _b c',
  };
  for (const [input, want] of Object.entries(cases)) {
    const got = richtext.parseEmphasis(input);
    assert.equal(got.length, 1, input);
    assert.equal(got[0].text, want);
    assert.equal(got[0].style, '');
  }
});

test('ParseEmphasis base style composes', () => {
  assert.deepEqual(richtext.parseEmphasisStyled('**x** _y_ z', 'muted'), [
    { text: 'x', style: 'muted;bold' },
    { text: ' ', style: 'muted' },
    { text: 'y', style: 'muted;italic' },
    { text: ' z', style: 'muted' },
  ]);
});

// ---------------------------------------------------------------- list

test('List renders only visible rows', () => {
  const items = [];
  for (let i = 0; i < 100000; i++) items.push(`item-${i}`);
  const l = new list.List({ items, height: 10, width: 20, selected: 50000, follow: true });
  const [start, end] = l.visibleRange();
  assert.equal(end - start, 10);
  const root = built(l);
  assert.equal(root.children.length, 10);
  assert.ok(boxText(root.children[0]).includes('item-49991'));
  const last = boxText(root.children[9]);
  assert.ok(last.includes('item-50000'));
  assert.ok(last.startsWith(list.DEFAULT_LIST_MARKER));
});

test('List helpers', () => {
  const l = new list.List({ items: ['a', 'b', 'c', 'd', 'e'], height: 2 });
  l.move(1);
  assert.equal(l.selected, 1);
  l.move(99);
  assert.equal(l.selected, 4);
  l.pageUp();
  assert.equal(l.selected, 2);
  l.pageDown();
  assert.equal(l.selected, 4);
  l.top();
  assert.equal(l.selected, 0);
  assert.equal(l.offset, 0);
  l.bottom();
  assert.equal(l.selected, 4);
  assert.equal(l.offset, 3);
  l.offset = 100;
  l.selected = 100;
  l.ensureVisible();
  assert.equal(l.selected, 4);
  assert.equal(l.offset, 3);
});

test('List empty and height one', () => {
  const empty = new list.List();
  assert.deepEqual(empty.visibleRange(), [0, 0]);
  empty.move(1);
  empty.pageUp();
  empty.pageDown();
  empty.top();
  empty.bottom();
  assert.equal(empty.selected, 0);
  assert.equal(childrenOf(built(empty)).length, 0);
  const placeholder = built(new list.List({ empty: 'no items', height: 3, width: 10 }));
  assert.equal(childrenOf(placeholder).length, 1);
  assert.equal(boxText(placeholder.children[0]), 'no items  ');

  const one = built(new list.List({ items: ['only'], height: 1, width: 8 }));
  assert.equal(childrenOf(one).length, 1);
  assert.equal(builder.displayWidth(boxText(one.children[0])), 8);
});

test('List header/footer window', () => {
  const l = new list.List({ items: ['a', 'b', 'c', 'd'], height: 4, header: 'HDR', footer: 'FTR', headerStyle: 'h', footerStyle: 'f' });
  assert.deepEqual(l.visibleRange(), [0, 2]);
  const children = built(l).children;
  assert.equal(children.length, 4);
  assert.equal(boxText(children[0]), 'HDR');
  assert.equal(children[0].style, 'h');
  assert.equal(boxText(children[3]), 'FTR');
  assert.equal(children[3].style, 'f');
});

test('List follow=false keeps offset', () => {
  const l = new list.List({ items: ['a', 'b', 'c', 'd', 'e'], height: 2, selected: 4 });
  assert.equal(l.visibleRange()[1], 2);
  l.follow = true;
  assert.equal(l.visibleRange()[0], 3);
});

test('List rowId/style and CJK truncation', () => {
  const l = new list.List({
    items: ['中文中文', 'short'],
    height: 2,
    width: 6,
    selected: 1,
    style: 'base',
    selectedStyle: 'sel',
    rowId: (index) => `row:${index}`,
  });
  const children = built(l).children;
  const cell = boxText(children[0]);
  assert.equal(builder.displayWidth(cell), 6);
  assert.ok(cell.includes('中文'));
  assert.equal(children[0].id, 'row:0');
  assert.deepEqual(children[0].input, ['mouse']);
  assert.ok(boxText(children[1]).startsWith(list.DEFAULT_LIST_MARKER));
  assert.equal(children[1].style, 'sel');
});

test('VirtualList rows and disabled', () => {
  const l = new list.VirtualList({
    id: 'vl',
    rows: [
      { text: 'one', id: 'vl:0', style: 's0' },
      { text: 'two', id: 'vl:1', disabled: true },
      { text: 'three', id: 'vl:2' },
    ],
    height: 2, width: 8, offset: 1, selected: 2, follow: true,
    disabledStyle: 'off', selectedStyle: 'sel',
  });
  assert.deepEqual(l.visibleRange(), [1, 3]);
  const children = built(l).children;
  assert.equal(children.length, 2);
  assert.equal(children[0].style, 'off');
  assert.equal(children[0].id, 'vl:1');
  assert.equal(children[1].style, 'sel');
  assert.ok(boxText(children[1]).startsWith(list.DEFAULT_LIST_MARKER));

  const empty = new list.VirtualList();
  empty.move(3);
  empty.bottom();
  assert.equal(empty.selected, 0);
  assert.equal(built(empty).id, undefined);
});

// ---------------------------------------------------------------- table

test('Table column widths', () => {
  const t = new table.Table({
    columns: [{ title: 'Name' }, { title: 'N', minWidth: 4 }],
    rows: [['alpha', '1'], ['b', '12345']],
  });
  assert.deepEqual(t.columnWidths(), [5, 5]);
  t.columns[0].width = 3;
  assert.deepEqual(t.columnWidths(), [3, 5]);
});

test('Table flex and shrink', () => {
  const flex = new table.Table({
    width: 20,
    columns: [{ title: 'A' }, { title: 'B', flex: 1 }],
    rows: [['x', 'y']],
  });
  let widths = flex.columnWidths();
  assert.equal(widths[0] + widths[1] + 1, 20);
  assert.ok(widths[1] > widths[0]);

  const shrink = new table.Table({
    width: 6,
    columns: [{ title: 'A' }, { title: 'B' }],
    rows: [['aaaa', 'bbbb']],
  });
  widths = shrink.columnWidths();
  assert.equal(widths[0] + widths[1] + 1, 6);
  assert.deepEqual(widths, [4, 1]);
});

test('Table build header selection zebra footer', () => {
  const t = new table.Table({
    id: 'tbl',
    columns: [
      { title: 'Name', width: 4 },
      { title: 'V', width: 3, align: table.ALIGN_RIGHT },
    ],
    rows: [['ab', '7'], ['c', '12'], ['dd', '3']],
    selected: 2,
    zebra: true,
    zebraStyle: 'zebra',
    selectedStyle: 'sel',
    headerStyle: 'hdr',
    footerStyle: 'ftr',
    rule: true,
    footer: ['F', '9'],
    rowId: (index) => `row:${index}`,
  });
  const children = built(t).children;
  assert.equal(children.length, 6);
  const header = children[0].children;
  assert.equal(header.length, 3);
  assert.equal(boxText(header[0]), 'Name');
  assert.equal(boxText(header[2]), '  V');
  assert.equal(header[0].style, 'hdr');
  assert.equal(children[1].style, 'hdr');
  assert.equal(children[2].children[0].style, '');
  assert.equal(children[3].children[0].style, 'zebra');
  assert.equal(children[4].id, 'row:2');
  assert.equal(children[4].children[0].style, 'sel');
  assert.deepEqual(children[4].input, ['mouse']);
  const footer = children[5].children;
  assert.equal(boxText(footer[0]), 'F   ');
  assert.equal(footer[0].style, 'ftr');
});

test('Table cell truncation and alignment', () => {
  const t = new table.Table({
    hideHeader: true,
    columns: [
      { title: 'L', width: 4 },
      { title: 'R', width: 4, align: table.ALIGN_RIGHT },
      { title: 'C', width: 4, align: table.ALIGN_CENTER },
    ],
    rows: [['中文你好', 'ab', 'ab']],
  });
  const row = built(t).children[0].children;
  assert.equal(boxText(row[0]), '中文');
  assert.equal(boxText(row[2]), '  ab');
  assert.equal(boxText(row[4]), ' ab ');
});

test('Table records format', () => {
  const t = new table.Table({
    columns: [
      { title: 'N', format: (value) => `<${value}>` },
      { title: 'D' },
    ],
    records: [[42, 'x'], ['七', 7]],
    rows: [['ignored', 'ignored']],
  });
  assert.equal(t.rowCount(), 2);
  assert.equal(t.cell(0, 0), '<42>');
  assert.equal(t.cell(1, 0), '<七>');
  assert.equal(t.cell(0, 1), 'x');
  assert.equal(t.cell(9, 9), '');
});

test('Table empty and short rows', () => {
  const empty = new table.Table();
  assert.equal(childrenOf(built(empty)).length, 0);
  const short = new table.Table({
    columns: [{ title: 'A', width: 2 }, { title: 'B', width: 2 }],
    rows: [['1']],
    hideHeader: true,
    width: 5,
  });
  const children = built(short).children;
  assert.equal(children.length, 1);
  assert.equal(boxText(children[0].children[2]), '  ');
});

// ---------------------------------------------------------------- input

test('TextInput editing', () => {
  const inRow = new input.TextInput();
  assert.ok(inRow.insertString('hello'));
  assert.equal(inRow.text(), 'hello');
  assert.equal(inRow.cursor, 5);
  assert.ok(inRow.left());
  assert.equal(inRow.cursor, 4);
  assert.ok(inRow.insertRune('X'));
  assert.equal(inRow.text(), 'hellXo');
  assert.equal(inRow.cursor, 5);
  assert.ok(inRow.backspace());
  assert.equal(inRow.text(), 'hello');
  assert.equal(inRow.cursor, 4);
  assert.ok(inRow.delete());
  assert.equal(inRow.text(), 'hell');
  assert.equal(inRow.cursor, 4);
  assert.ok(inRow.home());
  assert.equal(inRow.cursor, 0);
  assert.equal(inRow.home(), false);
  assert.equal(inRow.left(), false);
  assert.ok(inRow.right());
  assert.equal(inRow.cursor, 1);
  assert.ok(inRow.end());
  assert.equal(inRow.cursor, 4);

  const empty = new input.TextInput();
  assert.equal(empty.backspace(), false);
  assert.equal(empty.delete(), false);
  assert.equal(empty.left(), false);
  assert.equal(empty.right(), false);
  assert.equal(empty.deleteToEnd(), false);
});

test('TextInput words and limits', () => {
  const inRow = new input.TextInput({ value: 'foo bar  baz' });
  inRow.end();
  assert.ok(inRow.wordLeft());
  assert.equal(inRow.cursor, 9);
  assert.ok(inRow.wordLeft());
  assert.equal(inRow.cursor, 4);
  assert.ok(inRow.wordRight());
  assert.equal(inRow.cursor, 9);
  inRow.home();
  inRow.end();
  assert.ok(inRow.deleteWordLeft());
  assert.equal(inRow.text(), 'foo bar  ');
  assert.equal(inRow.cursor, 9);

  const limited = new input.TextInput({ maxLen: 3 });
  assert.ok(limited.insertString('abcdef'));
  assert.equal(limited.text(), 'abc');
  assert.equal(limited.insertRune('x'), false);
  assert.equal(limited.insertString('de'), false);
  limited.setValue('toolong');
  assert.equal(limited.text(), 'too');
  assert.equal(limited.cursor, 3);
});

test('TextInput build mask placeholder and cursor', () => {
  const masked = new input.TextInput({ value: 'secret', mask: '•', width: 10, focused: true, cursor: 6 });
  const root = built(masked);
  assert.equal(rowText(root), '••••••    ');
  assert.ok(root.cursor);
  assert.equal(root.cursor.col, 6);
  assert.equal(root.cursor.row, 0);
  assert.equal(root.focused, true);

  const unfocused = new input.TextInput({ value: 'secret', mask: '•', width: 4 });
  assert.equal(rowText(built(unfocused)), '••••');
  assert.equal(built(unfocused).cursor, undefined);

  const placeholder = new input.TextInput({ placeholder: 'name', width: 6, focused: true });
  const pRoot = built(placeholder);
  assert.equal(rowText(pRoot), 'name  ');
  assert.ok(pRoot.cursor);
  assert.equal(pRoot.cursor.col, 0);
});

test('TextInput horizontal scroll', () => {
  const scrolled = new input.TextInput({ value: 'abcdefghij', cursor: 10, width: 5, focused: true });
  const root = built(scrolled);
  assert.equal(rowText(root), 'ghij ');
  assert.equal(root.cursor.col, 4);

  const cjk = new input.TextInput({ value: '中文中文中文', cursor: 6, width: 5, focused: true });
  const cjkRoot = built(cjk);
  assert.equal(rowText(cjkRoot), '中文 ');
  assert.equal(builder.displayWidth(rowText(cjkRoot)), 5);
  assert.equal(cjkRoot.cursor.col, 4);
});

test('TextInput handleKey', () => {
  const inRow = new input.TextInput();
  assert.ok(inRow.handleKey({ key: 'a', char: 'a' }));
  assert.equal(inRow.text(), 'a');
  assert.ok(inRow.handleKey({ key: 'left' }));
  assert.equal(inRow.cursor, 0);
  assert.ok(inRow.handleKey({ key: '中', char: '中' }));
  assert.equal(inRow.text(), '中a');
  assert.equal(inRow.handleKey({ key: 'enter' }), false);
  assert.equal(inRow.handleKey({ key: 'tab' }), false);
  assert.equal(inRow.handleKey(null), false);
  assert.ok(inRow.handleKey({ key: 'ctrl-a' }));
  assert.equal(inRow.cursor, 0);
  assert.ok(inRow.handleKey({ key: 'delete' }));
  assert.equal(inRow.text(), 'a');
});

test('TextArea editing and cursor', () => {
  const area = new input.TextArea();
  assert.ok(area.insertString('ab\r\ncd'));
  assert.equal(area.text(), 'ab\ncd');
  assert.deepEqual(area.rowCol(), [1, 2]);
  assert.ok(area.up());
  assert.deepEqual(area.rowCol(), [0, 2]);
  assert.equal(area.up(), false);
  assert.ok(area.down());
  assert.equal(area.down(), false);
  assert.ok(area.home());
  assert.equal(area.cursor, 3);
  assert.ok(area.end());
  assert.equal(area.cursor, 5);
  assert.ok(area.insertRune('\n'));
  assert.equal(area.text(), 'ab\ncd\n');
  assert.deepEqual(area.rowCol(), [2, 0]);
  assert.ok(area.backspace());
  assert.equal(area.text(), 'ab\ncd');
  assert.equal(area.cursor, 5);
  assert.equal(area.line(0), 'ab');
  assert.equal(area.line(7), '');
});

test('TextArea scroll and cursor protocol', () => {
  const area = new input.TextArea({ value: 'l0\nl1\nl2\nl3\nl4', height: 2, cursor: 14, focused: true });
  const root = built(area);
  const children = root.children;
  assert.equal(children.length, 2);
  assert.equal(rowText(children[0]), 'l3');
  assert.equal(rowText(children[1]), 'l4 ');
  assert.equal(root.cursor.row, 1);
  assert.equal(root.cursor.col, 2);

  const cjk = new input.TextArea({ value: '中文中文\n短', width: 5, focused: true, cursor: 4 });
  const cjkRoot = built(cjk);
  assert.equal(rowText(cjkRoot.children[0]), '中文 ');
  assert.equal(cjkRoot.cursor.col, 4);
});

test('TextArea mask and page keys', () => {
  const masked = new input.TextArea({ value: 'ab\ncd', mask: '*', width: 4, height: 2, focused: true, cursor: 2 });
  const root = built(masked);
  assert.equal(rowText(root.children[0]), '**  ');
  assert.equal(rowText(root.children[1]), '**  ');

  const area = new input.TextArea({ value: 'a\nb\nc\nd\ne\nf', height: 2 });
  area.cursor = 0;
  assert.ok(area.pageDown());
  assert.ok(area.cursor >= 2);
  assert.ok(area.pageUp());
  assert.equal(area.cursor, 0);
  assert.ok(area.handleKey({ key: 'enter' }));
  assert.equal(area.cursor, 1);
  assert.equal(area.handleKey({ key: 'esc' }), false);
});

test('TextArea placeholder empty build', () => {
  const area = new input.TextArea({ placeholder: 'notes', width: 7, focused: true });
  const root = built(area);
  assert.equal(root.children.length, 1);
  assert.equal(rowText(root.children[0]), 'notes  ');
  assert.ok(root.cursor);
});

// ---------------------------------------------------------------- modal

test('Modal position and backdrop', () => {
  const m = new modal.Modal({
    id: 'confirm', title: 'Confirm', width: 20, height: 6,
    center: true, parentWidth: 80, parentHeight: 24,
    backdrop: true, backdropStyle: 'dim',
    rows: [{ text: 'ok', id: 'confirm:ok', input: ['mouse'] }],
  });
  assert.deepEqual(m.position(), [30, 9]);
  const root = built(m);
  const backdrop = findBox(root, 'confirm:backdrop');
  assert.ok(backdrop);
  assert.equal(backdrop.style, 'dim');
  assert.equal(backdrop.size[0], 80);
  assert.equal(backdrop.size[1], 24);
  const layer = findBox(root, 'confirm');
  assert.ok(layer);
  assert.deepEqual(layer.pos, [30, 9]);
  assert.ok(findBox(root, 'confirm:ok'));
});

test('Modal position clamps and offsets', () => {
  const m = new modal.Modal({ width: 60, height: 40, center: true, parentWidth: 40, parentHeight: 20 });
  assert.deepEqual(m.position(), [0, 0]);
  const offset = new modal.Modal({ x: 3, y: 2 });
  assert.deepEqual(offset.position(), [3, 2]);
  assert.equal(findBox(built(offset), 'confirm:backdrop'), null);
});

test('Menu move skips and selects', () => {
  const menu = new modal.Menu({
    items: [
      { id: 'open', label: 'Open', hotkey: 'o' },
      { separator: true },
      { id: 'del', label: 'Delete', hotkey: 'd', disabled: true },
      { id: 'quit', label: 'Quit', hotkey: 'q' },
    ],
    selected: 0,
  });
  menu.move(1);
  assert.equal(menu.selected, 3);
  menu.move(1);
  assert.equal(menu.selected, 3);
  menu.move(-1);
  assert.equal(menu.selected, 0);
  menu.move(-1);
  assert.equal(menu.selected, 0);
  assert.equal(menu.select(1), false);
  assert.equal(menu.select(2), false);
  assert.ok(menu.select(3));
  assert.equal(menu.value(), 'quit');
});

test('Menu hotkey and build', () => {
  const menu = new modal.Menu({
    id: 'menu', title: 'Actions', width: 16, x: 1, y: 1,
    selectedStyle: 'sel', disabledStyle: 'off',
    items: [
      { id: 'open', label: 'Open', hotkey: 'o' },
      { id: 'del', label: 'Delete', hotkey: 'd', disabled: true },
      { id: 'quit', label: 'Quit', hotkey: 'q' },
    ],
  });
  assert.deepEqual(menu.hotkey({ key: 'd', char: 'd' }), { value: '', ok: false });
  const matched = menu.hotkey({ key: 'Q', char: 'Q' });
  assert.deepEqual(matched, { value: 'quit', ok: true });
  assert.equal(menu.selected, 2);
  assert.deepEqual(menu.hotkey({ key: 'x', char: 'x' }), { value: '', ok: false });

  const root = built(menu);
  assert.deepEqual(root.pos, [1, 1]);
  const open = findBox(root, 'open');
  assert.ok(open);
  assert.ok(boxText(open).includes('Open'));
  assert.deepEqual(open.input, ['mouse']);
  const selected = findBox(root, 'quit');
  assert.ok(boxText(selected).startsWith(list.DEFAULT_LIST_MARKER));
  assert.equal(selected.style, 'sel');

  const empty = new modal.Menu();
  empty.move(1);
  assert.equal(empty.value(), '');
  assert.ok(built(empty));
});

// ---------------------------------------------------------------- scrollbar

test('Scrollbar thumb proportional', () => {
  const bar = new scrollbar.Scrollbar({ total: 100, visible: 10, height: 10 });
  assert.deepEqual(bar.thumb(), [0, 1]);
  bar.offset = 90;
  assert.deepEqual(bar.thumb(), [9, 1]);
  bar.offset = 45;
  assert.equal(bar.thumb()[0], 4);

  const half = new scrollbar.Scrollbar({ total: 100, visible: 50, height: 10 });
  assert.deepEqual(half.thumb(), [0, 5]);
});

test('Scrollbar thumb degenerate', () => {
  const fits = new scrollbar.Scrollbar({ total: 5, visible: 10, height: 10 });
  assert.deepEqual(fits.thumb(), [0, 10]);
  assert.equal(fits.offsetAt(7), 0);

  const empty = new scrollbar.Scrollbar({ total: 0, visible: 0, height: 4 });
  assert.deepEqual(empty.thumb(), [0, 4]);

  const zeroVisible = new scrollbar.Scrollbar({ total: 10, visible: 0, height: 10, offset: 5 });
  assert.equal(zeroVisible.thumb()[1], 1);

  const noTrack = new scrollbar.Scrollbar({ total: 10, visible: 3 });
  assert.deepEqual(noTrack.thumb(), [0, 0]);
});

test('Scrollbar clamps offset', () => {
  const bar = new scrollbar.Scrollbar({ total: 100, visible: 10, height: 10, offset: 1000 });
  assert.equal(bar.thumb()[0], 9);
  bar.offset = -50;
  assert.equal(bar.thumb()[0], 0);
});

test('Scrollbar offsetAt', () => {
  const bar = new scrollbar.Scrollbar({ total: 100, visible: 10, height: 10 });
  assert.equal(bar.offsetAt(0), 0);
  assert.equal(bar.offsetAt(9), 90);
  assert.equal(bar.offsetAt(100), 90);

  const wide = new scrollbar.Scrollbar({ total: 40, visible: 10, height: 10 });
  assert.deepEqual(wide.thumb(), [0, 2]);
  const mid = wide.offsetAt(5);
  assert.ok(mid > 0 && mid < 30);
});

test('Scrollbar horizontal build', () => {
  const bar = new scrollbar.Scrollbar({ id: 'sb', total: 20, visible: 5, width: 10 });
  assert.equal(bar.isVertical(), false);
  assert.equal(bar.trackLength(), 10);
  const row = built(bar);
  assert.equal(row.children.length, 10);
  assert.equal(boxText(row.children[0]), scrollbar.SCROLLBAR_THUMB_HORIZONTAL);
  assert.deepEqual(bar.thumb(), [0, 2]);
});

test('Scrollbar vertical build caps', () => {
  const bar = new scrollbar.Scrollbar({ id: 'sb', total: 10, visible: 2, height: 5, up: '▲', down: '▼' });
  const children = built(bar).children;
  assert.equal(children.length, 7);
  assert.equal(boxText(children[0]), '▲');
  assert.equal(boxText(children[6]), '▼');
  assert.deepEqual(bar.thumb(), [0, 1]);
});

// ---------------------------------------------------------------- mouse

test('Hit precedence topmost wins', () => {
  const regions = [
    new mouse.HitRegion({ id: 'below', x: 0, y: 0, w: 10, h: 10 }),
    new mouse.HitRegion({ id: 'above', x: 2, y: 2, w: 3, h: 3 }),
  ];
  const region = mouse.hit(regions, 3, 3);
  assert.ok(region);
  assert.equal(region.id, 'above');
  assert.equal(mouse.hit(regions, 0, 0).id, 'below');
  assert.equal(mouse.hit(regions, 20, 20), null);
  assert.deepEqual(mouse.hitId(regions, 3, 3), { id: 'above', ok: true });
});

test('HitRegion unbounded axis', () => {
  const region = new mouse.HitRegion({ id: 'row', x: 0, y: 4, w: 0, h: 1 });
  assert.ok(region.contains(999, 4));
  assert.equal(region.contains(0, 5), false);
});

test('ListRegions and rowAt', () => {
  const l = new list.List({
    items: ['a', 'b', 'c', 'd', 'e'],
    height: 3,
    header: 'H',
    offset: 1,
    rowId: (i) => `row:${String.fromCharCode(97 + i)}`,
  });
  const regions = mouse.listRegions(l, 0, 0, 20);
  assert.equal(regions.length, 2);
  assert.equal(regions[0].y, 1);
  assert.equal(regions[0].data, 1);
  assert.equal(regions[0].id, 'row:b');
  assert.equal(regions[1].y, 2);
  assert.equal(regions[1].data, 2);
  assert.equal(l.rowAt(2), 2);
  assert.equal(l.rowAt(0), null);
  assert.equal(l.rowAt(9), null);
});

test('VirtualListRegions', () => {
  const l = new list.VirtualList({
    rows: [{ text: 'a', id: 'a' }, { text: 'b', id: 'b' }, { text: 'c', id: 'c' }],
    height: 2,
  });
  const regions = mouse.virtualListRegions(l, 3, 5, 8);
  assert.equal(regions.length, 2);
  assert.equal(regions[0].x, 3);
  assert.equal(regions[0].y, 5);
  assert.equal(regions[1].id, 'b');
  assert.equal(regions[1].data, 1);
});

test('TableRegions and rowAt', () => {
  const t = new table.Table({
    columns: [{ title: 'A', width: 2 }, { title: 'B', width: 3 }],
    rows: [['x', 'y'], ['z', 'w']],
    rowId: () => 'r',
  });
  const regions = mouse.tableRegions(t, 0, 0);
  assert.equal(regions.length, 4);
  assert.deepEqual(regions[0].data, [0, 0]);
  assert.deepEqual(regions[1].data, [0, 1]);
  assert.equal(regions[1].x, 3);
  assert.equal(regions[2].y, 2);
  assert.equal(t.rowAt(1), 0);
  assert.equal(t.rowAt(0), null);
  assert.equal(t.rowAt(9), null);
});

test('ApplyWheel clamps', () => {
  assert.equal(mouse.applyWheel(0, 100, 10, 3), 3);
  assert.equal(mouse.applyWheel(5, 100, 10, -100), 0);
  assert.equal(mouse.applyWheel(80, 100, 10, 999), 90);
  assert.equal(mouse.applyWheel(4, 5, 10, 3), 0);
});

test('OnWheel methods', () => {
  const l = new list.List({ items: new Array(50).fill('x'), height: 5 });
  l.onWheel({ delta: 4 });
  assert.equal(l.offset, 4);
  l.onWheel(null);
  assert.equal(l.offset, 4);
  const virtual = new list.VirtualList({ rows: new Array(50).fill({ text: 'x' }), height: 5 });
  virtual.onWheel({ delta: 3 });
  assert.equal(virtual.offset, 3);
  assert.equal(mouse.tableScroll(0, 100, 10, 7), 7);
});

test('Drag rect normalization', () => {
  const cases = [
    ['down-right', 2, 3, 5, 8, { x: 2, y: 3, w: 4, h: 6 }],
    ['up-left', 5, 8, 2, 3, { x: 2, y: 3, w: 4, h: 6 }],
    ['up-right', 5, 3, 2, 8, { x: 2, y: 3, w: 4, h: 6 }],
    ['down-left', 2, 8, 5, 3, { x: 2, y: 3, w: 4, h: 6 }],
  ];
  for (const [name, sx, sy, x, y, want] of cases) {
    const drag = new mouse.Drag();
    drag.begin(sx, sy, 'left');
    drag.update(x, y);
    assert.deepEqual(drag.end(), want, name);
    assert.equal(drag.active, false, name);
  }
  assert.equal(new mouse.Drag().end(), null);
});

test('SelectRange and ListRange', () => {
  assert.deepEqual(mouse.selectRange(9, 2), [2, 9]);
  const l = new list.List({ items: ['a', 'b', 'c', 'd', 'e'], height: 3 });
  assert.deepEqual(mouse.listSelectRange(l, 2, 0), [0, 2]);
  assert.equal(mouse.listSelectRange(l, 5, 7), null);
  const virtual = new list.VirtualList({ rows: new Array(9).fill({ text: 'x' }), height: 4 });
  assert.deepEqual(mouse.virtualListSelectRange(virtual, 0, 3), [0, 3]);
});

test('ClickTracker', () => {
  const base = 0;
  const tracker = new mouse.ClickTracker({ threshold: 400 });
  assert.equal(tracker.click(1, 1, base), 1);
  assert.equal(tracker.click(1, 1, base + 100), 2);
  assert.equal(tracker.click(1, 1, base + 150), 3);
  assert.equal(tracker.click(1, 1, base + 160), 1);
  assert.equal(tracker.click(1, 1, base + 1000), 1);
  assert.equal(tracker.click(1, 1, base + 1100), 2);
  assert.equal(tracker.click(5, 5, base + 1150), 1);
  const zero = new mouse.ClickTracker();
  assert.equal(zero.click(0, 0, base), 1);
  assert.equal(zero.click(0, 0, base + 10), 2);
});

// ---------------------------------------------------------------- hover

test('Hover enter leave and style', () => {
  const h = new hover.Hover({
    style: 'hover',
    regions: [
      new mouse.HitRegion({ id: 'a', x: 0, y: 0, w: 4, h: 1 }),
      new mouse.HitRegion({ id: 'b', x: 4, y: 0, w: 4, h: 1 }),
    ],
  });
  assert.equal(h.update(1, 0), 'a');
  assert.equal(h.styleFor('a'), 'hover');
  assert.equal(h.styleFor('b'), '');
  assert.equal(h.update(5, 0), 'b');
  assert.equal(h.styleFor('a'), '');
  assert.equal(h.update(20, 0), '');
  assert.equal(h.hovered(), '');
  assert.equal(h.styleFor('b'), '');
  h.clear();
  assert.equal(h.currentStyle, '');
});

test('Hover node fallback', () => {
  const h = new hover.Hover({
    node: 'surface',
    style: 'hover',
    regions: [new mouse.HitRegion({ x: 0, y: 0, w: 10, h: 10 })],
  });
  assert.equal(h.update(2, 2), 'surface');
  assert.equal(h.styleFor('surface'), 'hover');
});

// ---------------------------------------------------------------- contextmenu

function contextTestMenu() {
  return new modal.Menu({
    id: 'ctx',
    width: 10,
    items: [{ id: 'copy', label: 'Copy', hotkey: 'c' }, { id: 'paste', label: 'Paste', hotkey: 'p' }],
  });
}

test('ContextMenu open clamps inside parent', () => {
  const menu = new contextmenu.ContextMenu({ menu: contextTestMenu(), parentWidth: 20, parentHeight: 10 });
  const [x, y] = menu.open(18, 8);
  assert.ok(menu.visible);
  assert.deepEqual([x, y], [10, 6]);
  assert.equal(menu.anchorX, x);
  assert.equal(menu.anchorY, y);
  const box = built(menu);
  assert.deepEqual(box.pos, [10, 6]);
});

test('ContextMenu open without parent', () => {
  const menu = new contextmenu.ContextMenu({ menu: contextTestMenu() });
  assert.deepEqual(menu.open(100, 50), [100, 50]);
});

test('ContextMenu margin and negative', () => {
  const menu = new contextmenu.ContextMenu({ menu: contextTestMenu(), parentWidth: 12, parentHeight: 6, margin: 1 });
  assert.deepEqual(menu.open(11, 5), [1, 1]);
  menu.margin = 0;
  assert.equal(menu.open(-4, -4)[0], 0);
});

test('ContextMenu close and height', () => {
  const menu = new contextmenu.ContextMenu({ menu: contextTestMenu() });
  menu.open(3, 4);
  assert.equal(menu.menuHeight(), 4);
  menu.close();
  assert.equal(menu.visible, false);
  const box = built(menu);
  assert.equal(box.visible, false);
  assert.equal(box.children, undefined);
});

test('ContextMenu keyboard drives embedded menu', () => {
  const menu = new contextmenu.ContextMenu({ menu: contextTestMenu() });
  menu.open(0, 0);
  menu.move(1);
  assert.equal(menu.selected, 1);
  assert.deepEqual(menu.hotkey({ key: 'c' }), { value: 'copy', ok: true });
  assert.equal(menu.value(), 'copy');
});

// ---------------------------------------------------------------- validate

test('Required validator', () => {
  const v = validate.required('required');
  assert.equal(v(''), 'required');
  assert.equal(v('   '), 'required');
  assert.equal(v(' ok '), '');
});

test('MinMaxLen validators', () => {
  const min = validate.minLen(3, 'short');
  assert.equal(min(''), '');
  assert.equal(min('ab'), 'short');
  assert.equal(min('abc'), '');
  assert.equal(min('中中中'), '');
  assert.equal(validate.minLen(0, 'short')('x'), '');

  const max = validate.maxLen(3, 'long');
  assert.equal(max(''), '');
  assert.equal(max('abcd'), 'long');
  assert.equal(max('abc'), '');
  assert.equal(validate.maxLen(0, 'long')('anything'), '');
});

test('Pattern validator', () => {
  const digits = validate.pattern('^\\d+$', 'digits only');
  assert.equal(digits(''), '');
  assert.equal(digits('123'), '');
  assert.equal(digits('12a'), 'digits only');

  const broken = validate.pattern('[', 'bad pattern');
  assert.equal(broken(''), '');
  assert.equal(broken('x'), 'bad pattern');
});

test('Email validator', () => {
  const v = validate.email('bad email');
  const cases = {
    '': '',
    'a@b.co': '',
    'first.last+tag@x.io': '',
    'no-at-sign': 'bad email',
    'a@b': 'bad email',
    'a@b.': 'bad email',
    'a b@c.d': 'bad email',
  };
  for (const [value, want] of Object.entries(cases)) {
    assert.equal(v(value), want, value);
  }
});

test('IntRange validator', () => {
  const v = validate.intRange(1, 10, 'out of range');
  const cases = {
    '': '',
    '1': '',
    '10': '',
    ' 5 ': '',
    '0': 'out of range',
    '11': 'out of range',
    x: 'out of range',
    '1.5': 'out of range',
  };
  for (const [value, want] of Object.entries(cases)) {
    assert.equal(v(value), want, value);
  }
});

test('OneOf validator', () => {
  const v = validate.oneOf(['red', 'green', 'blue'], 'pick one');
  assert.equal(v(''), '');
  assert.equal(v('green'), '');
  assert.equal(v('yellow'), 'pick one');
});

test('All and Custom validators', () => {
  const v = validate.all(validate.required('required'), validate.minLen(3, 'short'), validate.email('bad email'));
  assert.equal(v(''), 'required');
  assert.equal(v('ab'), 'short');
  assert.equal(v('not-an-email'), 'bad email');
  assert.equal(v('a@b.co'), '');

  const withNil = validate.all(null, validate.required('required'));
  assert.equal(withNil(''), 'required');
  assert.equal(validate.all()('anything'), '');
  assert.equal(validate.custom(null)('x'), '');
  assert.equal(validate.custom(() => 'nope')('x'), 'nope');
});

// ---------------------------------------------------------------- form

function newTestForm() {
  return new form.Form({
    id: 'login',
    width: 30,
    fields: [
      new form.Field({ id: 'user', label: 'User', input: new input.TextInput({ id: 'user' }), required: true }),
      new form.Field({ id: 'skip', label: 'Skip', input: new input.TextInput({ id: 'skip' }), disabled: true }),
      new form.Field({ id: 'pass', label: 'Pass', input: new input.TextInput({ id: 'pass' }), validate: validate.minLen(4, 'too short') }),
    ],
  });
}

test('Form navigation skips disabled and readonly', () => {
  const f = newTestForm();
  assert.equal(f.focusId(), 'user');
  f.next();
  assert.equal(f.focusId(), 'pass');
  f.next();
  assert.equal(f.focusId(), 'user');
  f.prev();
  assert.equal(f.focusId(), 'pass');
  f.fields[2].readonly = true;
  assert.equal(f.focusId(), 'pass');
  f.prev();
  assert.equal(f.focusId(), 'user');
});

test('Form navigation all-disabled is stable', () => {
  const f = new form.Form({ fields: [new form.Field({ id: 'a', disabled: true }), new form.Field({ id: 'b', readonly: true })] });
  f.next();
  assert.equal(f.focusId(), 'a');
  const empty = new form.Form();
  empty.next();
  empty.prev();
  assert.equal(empty.focusId(), '');
});

test('Form values and input', () => {
  const f = newTestForm();
  f.setValues({ user: 'ada', pass: 'hunter2' });
  const values = f.values();
  assert.equal(values.user, 'ada');
  assert.equal(values.pass, 'hunter2');
  assert.ok(f.input('user'));
  assert.equal(f.input('user').text(), 'ada');
  assert.equal(f.input('missing'), null);
  f.input('user').insertRune('!');
  assert.equal(f.values().user, 'ada!');
});

test('Form validate aggregation', () => {
  const f = new form.Form({
    fields: [
      new form.Field({ id: 'user', input: new input.TextInput(), required: true }),
      new form.Field({ id: 'pass', input: new input.TextInput(), validate: validate.minLen(4, 'too short') }),
    ],
  });
  assert.equal(f.validate(), false);
  assert.equal(f.fields[0].error, form.FORM_REQUIRED_MESSAGE);
  assert.equal(f.fields[1].error, '');
  f.setValues({ user: 'ada', pass: 'x' });
  assert.equal(f.validate(), false);
  assert.equal(f.fields[0].error, '');
  assert.equal(f.fields[1].error, 'too short');
  f.setValues({ pass: 'longenough' });
  assert.equal(f.validate(), true);
  assert.equal(f.fields[0].error, '');
  assert.equal(f.fields[1].error, '');

  const overrides = new form.Form({
    fields: [new form.Field({ input: new input.TextInput(), required: true, validate: () => 'custom' })],
  });
  overrides.validate();
  assert.equal(overrides.fields[0].error, form.FORM_REQUIRED_MESSAGE);
});

test('Form handleKey routing', () => {
  const f = newTestForm();
  assert.equal(f.handleKey('unknown', { key: 'a', char: 'a' }), false);
  assert.equal(f.input('user').text(), '');
  assert.ok(f.handleKey('user', { key: 'a', char: 'a' }));
  assert.equal(f.input('user').text(), 'a');
  assert.ok(f.handleKey('user', { key: 'tab' }));
  assert.equal(f.focusId(), 'pass');
  assert.ok(f.handleKey('', { key: 'shift-tab' }));
  assert.equal(f.focusId(), 'user');
  assert.ok(f.handleKey('user', { key: 'enter' }));
  assert.equal(f.focusId(), 'pass');
  assert.equal(f.handleKey('user', null), false);
});

test('Form validateOnChange', () => {
  const f = new form.Form({
    validateOnChange: true,
    fields: [
      new form.Field({ id: 'email', input: new input.TextInput({ id: 'email' }), required: true, validate: validate.email('bad email') }),
    ],
  });
  f.fields[0].error = form.FORM_REQUIRED_MESSAGE;
  f.handleKey('email', { key: 'a', char: 'a' });
  assert.equal(f.fields[0].error, 'bad email');

  f.input('email').setValue('a@b.co');
  f.fields[0].error = 'stale';
  f.handleKey('email', { key: 'x', char: 'x' });
  assert.equal(f.fields[0].error, '');

  f.input('email').setValue('ab');
  f.fields[0].error = 'stale';
  f.handleKey('email', { key: 'backspace' });
  assert.equal(f.fields[0].error, 'bad email');
});

test('Form and Field build', () => {
  const f = newTestForm();
  f.fields[0].error = 'user is required';
  f.fields[0].help = 'your login';
  const root = built(f);
  assert.equal(root.id, 'login');
  assert.equal(root.children.length, f.fields.length);
  const field = root.children[0];
  assert.equal(boxText(field.children[0]), 'User *');
  assert.equal(boxText(field.children[2]), 'user is required');

  f.fields[0].error = '';
  const helpRoot = built(f);
  assert.equal(boxText(helpRoot.children[0].children[2]), 'your login');

  const bare = new form.Field();
  assert.equal(childrenOf(built(bare)).length, 0);
});

test('Form error style propagation', () => {
  const f = new form.Form({
    errorStyle: 'danger',
    fields: [new form.Field({ id: 'a', error: 'boom', input: new input.TextInput() })],
  });
  const field = built(f).children[0];
  assert.equal(field.children[1].style, 'danger');
});

test('Form width propagates to input', () => {
  const f = new form.Form({
    id: 'f',
    width: 20,
    fields: [new form.Field({ id: 'a', input: new input.TextInput({ id: 'a', width: 20 }) })],
  });
  const root = built(f);
  const inputBox = findBox(root, 'a');
  assert.ok(inputBox);
  assert.equal(inputBox.size[0], 20);
  assert.equal(builder.displayWidth(rowText(inputBox)), 20);
});

// ---------------------------------------------------------------- select

function testSelect() {
  return new select.Select({
    id: 'color',
    label: 'Color',
    width: 20,
    options: [
      { value: 'red', label: 'Red' },
      { value: 'green', label: 'Green', disabled: true },
      { value: 'grey', label: 'Grey' },
      { value: 'blue', label: 'Blue' },
    ],
  });
}

test('Select selected', () => {
  const sel = testSelect();
  assert.equal(sel.selected().ok, false);
  assert.equal(sel.index(), -1);
  assert.ok(sel.selectIndex(3));
  assert.equal(sel.value, 'blue');
  const selected = sel.selected();
  assert.ok(selected.ok);
  assert.equal(selected.option.value, 'blue');
  assert.equal(sel.selectIndex(1), false);
  assert.equal(sel.selectIndex(4), false);
  assert.equal(sel.selectIndex(-1), false);
  sel.value = 'green';
  assert.equal(sel.selected().ok, false);
});

test('Select move skips disabled', () => {
  const sel = testSelect();
  sel.selectIndex(0);
  sel.move(1);
  assert.equal(sel.value, 'grey');
  sel.move(1);
  assert.equal(sel.value, 'blue');
  sel.move(1);
  assert.equal(sel.value, 'blue');
  sel.move(-1);
  assert.equal(sel.value, 'grey');
  sel.move(-3);
  assert.equal(sel.value, 'red');

  const unselected = testSelect();
  unselected.move(1);
  assert.equal(unselected.value, 'red');
  const back = testSelect();
  back.move(-1);
  assert.equal(back.value, 'blue');
  const empty = new select.Select();
  empty.move(1);
  assert.equal(empty.value, '');
  empty.move(0);
});

test('Select typeahead', () => {
  const sel = testSelect();
  sel.selectIndex(0);
  assert.ok(sel.typeahead('gr'));
  assert.equal(sel.value, 'grey');
  assert.ok(sel.typeahead('b'));
  assert.equal(sel.value, 'blue');
  assert.ok(sel.typeahead('Re'));
  assert.equal(sel.value, 'red');
  assert.equal(sel.typeahead('zzz'), false);
  assert.equal(sel.typeahead(''), false);
  assert.equal(new select.Select().typeahead('a'), false);

  const values = new select.Select({ options: [{ value: 'alpha' }, { value: 'beta', label: 'B' }] });
  assert.ok(values.typeahead('be'));
  assert.equal(values.value, 'beta');
});

test('Select build closed', () => {
  const sel = testSelect();
  sel.value = 'blue';
  const root = built(sel);
  assert.equal(root.id, 'color');
  assert.equal(boxText(root.children[0]), 'Color');
  const row = root.children[1];
  const text = rowText(row);
  assert.equal(builder.displayWidth(text), 20);
  assert.ok(text.startsWith('Blue'));
  assert.ok(text.trimEnd().endsWith('▾'));

  const empty = new select.Select({ placeholder: 'choose', indicator: '!' });
  assert.equal(rowText(built(empty)), 'choose!');
});

test('Select build dropdown', () => {
  const sel = testSelect();
  sel.open = true;
  sel.value = 'grey';
  const root = built(sel);
  assert.equal(root.children.length, sel.options.length);
  const selected = rowText(root.children[2]);
  assert.ok(selected.startsWith(list.DEFAULT_LIST_MARKER));
  const disabled = root.children[1];
  assert.ok(boxText(disabled).startsWith('  '));

  sel.disabledStyle = 'muted';
  const dropdown = sel.buildDropdown().build();
  assert.equal(dropdown.children[1].style, 'muted');
});

test('Select unicode labels', () => {
  const sel = new select.Select({
    options: [
      { value: 'cn', label: '中文' },
      { value: 'jp', label: '日本語' },
    ],
    placeholder: '选择',
  });
  assert.ok(sel.typeahead('中'));
  assert.equal(sel.value, 'cn');
  sel.value = 'jp';
  const text = rowText(built(sel));
  assert.equal(builder.displayWidth(text), builder.displayWidth('日本語 ▾'));
});

// ---------------------------------------------------------------- datepicker

test('Date daysInMonth and isLeap', () => {
  const cases = [
    [2023, 1, 31, false],
    [2023, 2, 28, false],
    [2024, 2, 29, true],
    [1900, 2, 28, false],
    [2000, 2, 29, true],
    [2023, 4, 30, false],
    [2023, 12, 31, false],
  ];
  for (const [year, month, days, leap] of cases) {
    const date = new datepicker.Date({ year, month, day: 1 });
    assert.equal(date.daysInMonth(), days, `${year}-${month}`);
    assert.equal(date.isLeap(), leap, String(year));
  }
  assert.equal(new datepicker.Date({ year: 2024, month: 13, day: 1 }).daysInMonth(), 0);
});

test('Date addDays across boundaries', () => {
  const cases = [
    [new datepicker.Date({ year: 2024, month: 2, day: 28 }), 1, new datepicker.Date({ year: 2024, month: 2, day: 29 })],
    [new datepicker.Date({ year: 2024, month: 2, day: 29 }), 1, new datepicker.Date({ year: 2024, month: 3, day: 1 })],
    [new datepicker.Date({ year: 2023, month: 2, day: 28 }), 1, new datepicker.Date({ year: 2023, month: 3, day: 1 })],
    [new datepicker.Date({ year: 2023, month: 12, day: 31 }), 1, new datepicker.Date({ year: 2024, month: 1, day: 1 })],
    [new datepicker.Date({ year: 2024, month: 1, day: 1 }), -1, new datepicker.Date({ year: 2023, month: 12, day: 31 })],
    [new datepicker.Date({ year: 2024, month: 3, day: 1 }), -1, new datepicker.Date({ year: 2024, month: 2, day: 29 })],
    [new datepicker.Date({ year: 2024, month: 1, day: 1 }), 60, new datepicker.Date({ year: 2024, month: 3, day: 1 })],
    [new datepicker.Date({ year: 2024, month: 1, day: 1 }), 0, new datepicker.Date({ year: 2024, month: 1, day: 1 })],
  ];
  for (const [start, days, want] of cases) {
    assert.ok(start.addDays(days).equals(want), `${start} + ${days}`);
  }
  const zero = new datepicker.Date();
  assert.ok(zero.addDays(1).equals(zero));
  assert.ok(zero.addMonths(1).equals(zero));
});

test('Date addMonths clamps', () => {
  const cases = [
    [new datepicker.Date({ year: 2024, month: 1, day: 31 }), 1, new datepicker.Date({ year: 2024, month: 2, day: 29 })],
    [new datepicker.Date({ year: 2023, month: 1, day: 31 }), 1, new datepicker.Date({ year: 2023, month: 2, day: 28 })],
    [new datepicker.Date({ year: 2024, month: 1, day: 15 }), 1, new datepicker.Date({ year: 2024, month: 2, day: 15 })],
    [new datepicker.Date({ year: 2024, month: 12, day: 15 }), 1, new datepicker.Date({ year: 2025, month: 1, day: 15 })],
    [new datepicker.Date({ year: 2024, month: 1, day: 15 }), -1, new datepicker.Date({ year: 2023, month: 12, day: 15 })],
    [new datepicker.Date({ year: 2024, month: 1, day: 15 }), 12, new datepicker.Date({ year: 2025, month: 1, day: 15 })],
    [new datepicker.Date({ year: 2024, month: 1, day: 15 }), -13, new datepicker.Date({ year: 2022, month: 12, day: 15 })],
  ];
  for (const [start, months, want] of cases) {
    assert.ok(start.addMonths(months).equals(want), `${start} + ${months} months`);
  }
});

test('Date compare weekday string', () => {
  const a = new datepicker.Date({ year: 2024, month: 1, day: 1 });
  const b = new datepicker.Date({ year: 2024, month: 1, day: 2 });
  assert.equal(a.compare(b), -1);
  assert.equal(b.compare(a), 1);
  assert.equal(a.compare(a), 0);
  assert.ok(a.before(b));
  assert.equal(a.after(b), false);
  assert.ok(a.equals(new datepicker.Date({ year: 2024, month: 1, day: 1 })));
  assert.equal(a.weekdayName(), 'Monday');
  assert.equal(new datepicker.Date({ year: 2024, month: 2, day: 29 }).weekdayName(), 'Thursday');
  assert.equal(b.toString(), '2024-01-02');
  assert.equal(new datepicker.Date().toString(), '');
});

test('ParseDate', () => {
  assert.ok(datepicker.parseDate('2024-02-29').equals(new datepicker.Date({ year: 2024, month: 2, day: 29 })));
  const bad = ['', '2024-2-9', '2024-02-30', '2023-02-29', '2023-13-01', '2024-00-10', '2024-01-32', 'not-a-date', '2024-01-01T00:00:00Z'];
  for (const value of bad) {
    assert.throws(() => datepicker.parseDate(value), undefined, value);
  }
});

test('Calendar grid alignment', () => {
  const cases = [
    [new datepicker.Date({ year: 2024, month: 1, day: 1 }), new datepicker.Date({ year: 2024, month: 1, day: 1 }), new datepicker.Date({ year: 2024, month: 2, day: 11 })],
    [new datepicker.Date({ year: 2024, month: 2, day: 1 }), new datepicker.Date({ year: 2024, month: 1, day: 29 }), new datepicker.Date({ year: 2024, month: 3, day: 10 })],
    [new datepicker.Date({ year: 2023, month: 12, day: 1 }), new datepicker.Date({ year: 2023, month: 11, day: 27 }), new datepicker.Date({ year: 2024, month: 1, day: 7 })],
    [new datepicker.Date({ year: 2024, month: 9, day: 1 }), new datepicker.Date({ year: 2024, month: 8, day: 26 }), new datepicker.Date({ year: 2024, month: 10, day: 6 })],
  ];
  for (const [month, firstCell, lastCell] of cases) {
    const grid = new datepicker.Calendar({ month }).grid();
    assert.equal(grid.length, 6);
    for (const rowDates of grid) assert.equal(rowDates.length, 7);
    assert.ok(grid[0][0].equals(firstCell), month.toString());
    assert.ok(grid[5][6].equals(lastCell), month.toString());
    for (const rowDates of grid) {
      assert.equal(rowDates[0].weekdayName(), 'Monday');
      for (let col = 1; col < 7; col++) {
        assert.ok(rowDates[col].equals(rowDates[col - 1].addDays(1)));
      }
    }
  }
  assert.equal(new datepicker.Calendar().grid(), null);
});

test('Calendar moveCursor', () => {
  const cal = new datepicker.Calendar({
    month: new datepicker.Date({ year: 2024, month: 1, day: 1 }),
    selected: new datepicker.Date({ year: 2024, month: 1, day: 31 }),
  });
  assert.ok(cal.moveCursor(1));
  assert.ok(cal.selected.equals(new datepicker.Date({ year: 2024, month: 2, day: 1 })));
  assert.equal(cal.month.month, 2);
  assert.equal(cal.month.day, 1);
  assert.ok(cal.moveCursor(-1));
  assert.ok(cal.selected.equals(new datepicker.Date({ year: 2024, month: 1, day: 31 })));
  assert.equal(cal.month.month, 1);
  assert.equal(cal.moveCursor(0), false);
  const cursorless = new datepicker.Calendar({ month: new datepicker.Date({ year: 2024, month: 3, day: 1 }) });
  cursorless.moveCursor(7);
  assert.ok(cursorless.selected.equals(new datepicker.Date({ year: 2024, month: 3, day: 8 })));
});

test('Calendar build', () => {
  const cal = new datepicker.Calendar({
    id: 'cal',
    month: new datepicker.Date({ year: 2024, month: 2, day: 1 }),
    selected: new datepicker.Date({ year: 2024, month: 2, day: 14 }),
    marked: new Map([['2024-02-20', '★']]),
    today: new datepicker.Date({ year: 2024, month: 2, day: 1 }),
    selectedStyle: 'reverse',
    markedStyle: 'accent',
    todayStyle: 'bold',
  });
  const root = built(cal);
  assert.equal(root.id, 'cal');
  assert.equal(boxText(root.children[0]), 'February 2024');
  const weekdays = rowText(root.children[1]);
  assert.equal(builder.displayWidth(weekdays), 21);
  assert.ok(weekdays.trim().startsWith('Mo'));
  assert.ok(weekdays.trim().endsWith('Su'));
  const grid = root.children.slice(2);
  assert.equal(grid.length, 6);
  let rowWidth = -1;
  for (const row of grid) {
    const width = builder.displayWidth(rowText(row));
    if (rowWidth === -1) rowWidth = width;
    assert.equal(width, rowWidth);
  }
  assert.equal(rowWidth, 21);

  let marked = '';
  let selected = '';
  for (const row of root.children) {
    for (const cell of row.children || []) {
      if (cell.style === 'accent') marked = cell.style;
      if (cell.style === 'reverse') selected = boxText(cell);
    }
  }
  assert.equal(marked, 'accent');
  assert.notEqual(selected, '');

  const unset = new datepicker.Calendar({ month: new datepicker.Date({ year: 2024, month: 2, day: 1 }) });
  assert.equal(built(unset).children.length, 8);
});

test('DayValidator', () => {
  const v = datepicker.dayValidator(
    new datepicker.Date({ year: 2024, month: 1, day: 1 }),
    new datepicker.Date({ year: 2024, month: 12, day: 31 }),
    'bad range',
  );
  assert.equal(v(''), '');
  assert.equal(v('2024-06-15'), '');
  assert.equal(v('2023-12-31'), 'bad range');
  assert.equal(v('2025-01-01'), 'bad range');
  assert.equal(v('2024-02-30'), 'bad range');
  const openMin = datepicker.dayValidator(new datepicker.Date(), new datepicker.Date({ year: 2024, month: 12, day: 31 }), 'bad range');
  assert.equal(openMin('1900-01-01'), '');
  const openMax = datepicker.dayValidator(new datepicker.Date({ year: 2024, month: 1, day: 1 }), new datepicker.Date(), 'bad range');
  assert.equal(openMax('2999-12-31'), '');
});

test('DateField', () => {
  const inputBox = new input.TextInput({ id: 'due', placeholder: 'YYYY-MM-DD' });
  const field = datepicker.newDateField('due', 'Due', inputBox);
  assert.equal(field.id, 'due');
  assert.equal(field.label, 'Due');
  assert.ok(field.validate);
  assert.equal(field.validate('2024-02-29'), '');
  assert.equal(field.validate('2024-02-30'), datepicker.DATE_FIELD_MESSAGE);
  assert.equal(field.date(), null);
  inputBox.setValue('2024-07-04');
  const date = field.date();
  assert.ok(date);
  assert.ok(date.equals(new datepicker.Date({ year: 2024, month: 7, day: 4 })));

  field.required = true;
  const f = new form.Form({ fields: [field] });
  inputBox.setValue('');
  assert.equal(f.validate(), false);
  assert.equal(f.fields[0].error, form.FORM_REQUIRED_MESSAGE);
  inputBox.setValue('2024-02-30');
  assert.equal(f.validate(), false);
  assert.equal(f.fields[0].error, datepicker.DATE_FIELD_MESSAGE);
});

test('Calendar CJK marked width stable', () => {
  const cal = new datepicker.Calendar({
    month: new datepicker.Date({ year: 2024, month: 2, day: 1 }),
    marked: new Map([['2024-02-01', '春节'], ['2024-02-02', '★']]),
  });
  const root = built(cal);
  const widths = root.children.slice(2).map((row) => builder.displayWidth(rowText(row)));
  for (const width of widths) assert.equal(width, widths[0]);
});

// ---------------------------------------------------------------- chart

test('Sparkline glyph mapping', () => {
  assert.equal(new chart.Sparkline({ values: [0, 1, 2, 3, 4, 5, 6, 7] }).line(), '▁▂▃▄▅▆▇█');
  assert.equal(new chart.Sparkline({ values: [5, 5, 5] }).line(), '▄▄▄');
  assert.equal(new chart.Sparkline({ values: [42] }).line(), '▄');
  assert.equal(new chart.Sparkline().line(), '');
  assert.equal(new chart.Sparkline({ values: [0, 5, 10], min: 0, max: 10 }).line(), '▁▅█');
  assert.equal(new chart.Sparkline({ values: [0, 0, 10, 10], width: 2 }).line(), '▁█');
  const sparkBuilt = built(new chart.Sparkline({ values: [0, 7], style: 'accent' }));
  assert.equal(boxText(sparkBuilt), '▁█');
  assert.equal(sparkBuilt.style, 'accent');
});

test('BarChart vertical dimensions', () => {
  const c = new chart.BarChart({ values: [1, 2, 3, 4], width: 7, style: 'bar', labelStyle: 'lbl', labels: ['a', 'b', 'c', 'd'] });
  assert.equal(c.maxValue(), 4);
  const lines = c.verticalLines();
  assert.equal(lines.length, 2);
  assert.equal(lines[0], '▂ ▄ ▆ █');
  assert.equal(lines[1], 'a b c d');
  const children = built(c).children;
  assert.equal(children.length, 2);
  assert.equal(boxText(children[0].children[0]), '▂');
  assert.equal(children[0].children[0].style, 'bar');
  assert.equal(boxText(children[1].children[0]), 'a');
  assert.equal(children[1].children[0].style, 'lbl');
});

test('BarChart vertical partial and height', () => {
  assert.equal(new chart.BarChart({ values: [0.5], max: 1, width: 1 }).verticalLines()[0], '▄');
  let lines = new chart.BarChart({ values: [4], max: 4, width: 1, height: 2 }).verticalLines();
  assert.deepEqual(lines, ['█', '█']);
  lines = new chart.BarChart({ values: [2], max: 4, width: 1, height: 2 }).verticalLines();
  assert.deepEqual(lines, [' ', '█']);
  assert.ok(new chart.BarChart({ values: [1, 2, 3, 4], width: 3 }).verticalLines()[0].length > 0);
  assert.equal(new chart.BarChart().verticalLines(), null);
});

test('BarChart horizontal', () => {
  let lines = new chart.BarChart({ values: [50, 100], width: 10, horizontal: true }).horizontalLines();
  assert.deepEqual(lines, ['█████', '██████████']);
  lines = new chart.BarChart({ values: [50, 100], width: 10, horizontal: true, labels: ['a', 'bb'], labelStyle: 'lbl' }).horizontalLines();
  assert.equal(lines[0], 'a  ████');
  assert.equal(lines[1], 'bb ███████');
  const capped = new chart.BarChart({ values: [1, 2, 3], height: 2, horizontal: true }).horizontalLines();
  assert.equal(capped.length, 2);
  const blank = new chart.BarChart({ values: [-1, 0], width: 4, horizontal: true }).horizontalLines();
  assert.equal(blank[0].trim(), '');
  assert.equal(blank[1].trim(), '');
});

test('BarChart selected style', () => {
  const c = new chart.BarChart({ values: [1, 1], width: 3, style: 'base', selected: 1, selectedStyle: 'hot' });
  const children = built(c).children[0].children;
  assert.equal(children[0].style, 'base');
  assert.equal(children[2].style, 'hot');
});

test('Heatmap shade ramp and ragged', () => {
  const heat = new chart.Heatmap({ values: [[0, 1], [2, 3]] });
  assert.equal(heat.shades, null);
  let grid = heat.grid();
  assert.deepEqual(grid, [' ░', '▒█']);
  assert.equal(heat.level(-100), 0);
  assert.equal(heat.level(100), 5);
  assert.equal(new chart.Heatmap({ values: [[5, 5]] }).level(5), 3);

  const ragged = new chart.Heatmap({
    values: [[1, 2, 3], [4]],
    colLabels: ['x', 'y', 'z'],
    rowLabels: ['r1', 'r2'],
  });
  assert.equal(ragged.cols(), 3);
  assert.equal(ragged.rows(), 2);
  assert.equal(ragged.cell(1, 1), ' ');
  grid = ragged.grid();
  assert.equal(grid.length, 3);
  assert.ok(grid[1].startsWith('r1 '));

  const custom = new chart.Heatmap({ values: [[1], [2]], shades: ['a', 'b', 'c'] }).grid();
  assert.deepEqual(custom, ['a', 'c']);
  assert.equal(new chart.Heatmap().grid(), null);

  const heatBuilt = built(new chart.Heatmap({ values: [[1]], style: 'heat', labelStyle: 'lbl', colLabels: ['c'] }));
  assert.equal(heatBuilt.children.length, 2);
  assert.equal(heatBuilt.children[0].style, 'lbl');
  assert.equal(heatBuilt.children[1].style, 'heat');
});

test('Meter and Gauge', () => {
  const meter = new chart.Meter({ value: 3, max: 10, width: 20, style: 'fill', trackStyle: 'track', showValue: true, label: 'cpu ' });
  assert.equal(meter.fraction(), 0.3);
  assert.equal((meter.barText().match(/█/g) || []).length, 6);
  assert.equal(Array.from(meter.barText()).length, 20);
  assert.equal(meter.valueText(), '3/10');
  const children = built(meter).children;
  assert.equal(children.length, 4);
  assert.equal(new chart.Gauge({ value: 1, max: 0 }).fraction(), 0);
  assert.equal(new chart.Meter({ value: 20, max: 10 }).fraction(), 1);
  assert.equal(new chart.Meter({ value: 1, max: 2 }).valueText(), '');
});

test('Legend', () => {
  const legend = new chart.Legend({
    style: 'muted',
    items: [
      { label: 'go', color: 'accent', marker: 'x' },
      { label: 'ui' },
    ],
  });
  assert.equal(legend.text(), 'x go ■ ui');
  const children = built(legend).children;
  assert.equal(children.length, 5);
  assert.equal(boxText(children[0]), 'x');
  assert.equal(children[0].style, 'accent');
  assert.equal(boxText(children[1]), ' go');
  assert.equal(children[1].style, 'muted');
  assert.equal(boxText(children[2]), ' ');
  assert.equal(boxText(children[3]), '■');
  assert.equal(children[3].style, undefined);
  assert.equal(new chart.Legend({ separator: ' | ', items: [{ label: 'a' }] }).text(), '■ a');
  assert.equal(new chart.Legend({ items: [{ label: 'a' }, { label: 'b' }], separator: '|' }).text(), '■ a|■ b');
});

test('Chart plain text is ANSI-free', () => {
  const texts = [
    new chart.Sparkline({ values: [1, 2, 3] }).line(),
    new chart.BarChart({ values: [1, 2], width: 3, horizontal: true }).horizontalLines().join('\n'),
    new chart.Heatmap({ values: [[1, 2]] }).grid().join('\n'),
    new chart.Legend({ items: [{ label: 'x' }] }).text(),
  ];
  for (const text of texts) {
    assert.equal(text.includes('\x1b'), false);
    assert.ok(builder.displayWidth(text) >= 0);
  }
});

// ---------------------------------------------------------------- format

test('FormatBytes', () => {
  const cases = [
    [0, '0 B'],
    [1, '1 B'],
    [512, '512 B'],
    [1023, '1023 B'],
    [1024, '1 KB'],
    [1536, '1.5 KB'],
    [1024 * 1024, '1 MB'],
    [3 * 1024 * 1024, '3 MB'],
    [1024 * 1024 * 1024, '1 GB'],
    [5 * 1024 * 1024 * 1024 + 512 * 1024 * 1024, '5.5 GB'],
    [-2048, '-2 KB'],
    [9223372036854775807, '8 EB'],
  ];
  for (const [input, want] of cases) {
    assert.equal(format.formatBytes(input), want, String(input));
  }
});

test('FormatCount', () => {
  const cases = [
    [0, '0'],
    [7, '7'],
    [999, '999'],
    [1000, '1,000'],
    [1234, '1,234'],
    [1234567, '1,234,567'],
    [9007199254740991, '9,007,199,254,740,991'],
    [-1000, '-1,000'],
    [-1, '-1'],
  ];
  for (const [input, want] of cases) {
    assert.equal(format.formatCount(input), want, String(input));
  }
});

test('FormatDuration boundaries', () => {
  const cases = [
    [0, '0s'],
    [500, '500ns'],
    [999, '999ns'],
    [1500, '2µs'],
    [250000, '250µs'],
    [1500000, '1.5ms'],
    [250000000, '250ms'],
    [1000000000, '1s'],
    [1500000000, '1.5s'],
    [30e9, '30s'],
    [90e9, '1m30s'],
    [60e9, '1m'],
    [3600e9, '1h'],
    [3661e9, '1h1m1s'],
    [3605e9, '1h5s'],
    [-90e9, '-1m30s'],
  ];
  for (const [input, want] of cases) {
    assert.equal(format.formatDuration(input), want, String(input));
  }
});

test('FormatPercent and Float', () => {
  const percents = [
    [0, 0, '0%'],
    [0.125, 1, '12.5%'],
    [1, 0, '100%'],
    [-0.5, 0, '-50%'],
    [1 / 3, 2, '33.33%'],
    [0.5, -1, '50%'],
  ];
  for (const [f, digits, want] of percents) {
    assert.equal(format.formatPercent(f, digits), want);
  }
  assert.equal(format.formatFloat(3.14159, 2, 0), '3.14');
  assert.equal(format.formatFloat(3.5, 0, 6), '     4');
});

test('FormatNonFinite', () => {
  assert.equal(format.formatPercent(NaN, 1), 'NaN%');
  assert.equal(format.formatPercent(Infinity, 0), '+Inf%');
  assert.equal(format.formatFloat(NaN, 2, 0), 'NaN');
  assert.equal(format.padLeft('x', 4), '   x');
  assert.equal(format.padRight('x', 4), 'x   ');
  assert.equal(format.padLeft('toolong', 2), 'toolong');
});

test('ScaleValue', () => {
  const cases = [
    [0, 0, ''],
    [999, 999, ''],
    [1500, 1.5, 'K'],
    [2000000, 2, 'M'],
    [1e9, 1, 'G'],
    [1e12, 1, 'T'],
    [1e15, 1, 'P'],
    [1e18, 1, 'E'],
    [-1500, -1.5, 'K'],
  ];
  for (const [input, wantValue, wantSuffix] of cases) {
    assert.deepEqual(format.scaleValue(input), [wantValue, wantSuffix]);
  }
  const nan = format.scaleValue(NaN);
  assert.ok(Number.isNaN(nan[0]));
  assert.equal(nan[1], '');
  const inf = format.scaleValue(Infinity);
  assert.equal(inf[0], Infinity);
  assert.equal(inf[1], '');
});
