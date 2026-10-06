'use strict';

// Rich text: styled runs, wrapping and a tiny emphasis grammar.
// Port of tui2/sdk/widgets/richtext.go.

const { box, text, displayWidth, cellWidth } = require('../builder');

// styled returns a Span of text rendered with style.
function styled(value, style) {
  return { text: value, style: style };
}

// line is the variadic constructor of a RichText from spans.
function line(...spans) {
  return new RichText({ spans });
}

// mergeSpans coalesces adjacent runs with the same style and drops empty runs.
function mergeSpans(spans) {
  const out = [];
  for (const span of spans) {
    const value = span.text == null ? '' : span.text;
    if (value === '') continue;
    const style = span.style == null ? '' : span.style;
    if (out.length > 0 && out[out.length - 1].style === style) {
      out[out.length - 1].text += value;
      continue;
    }
    out.push({ text: value, style });
  }
  return out;
}

function spansText(spans) {
  let out = '';
  for (const span of spans) out += span.text == null ? '' : span.text;
  return out;
}

// RichText is a single row of styled runs.
class RichText {
  constructor(options = {}) {
    this.spans = (options.spans || []).map((span) => ({
      text: span.text == null ? '' : span.text,
      style: span.style == null ? '' : span.style,
    }));
    this.width = options.width || 0;
    this.wrap = Boolean(options.wrap);
  }

  text() {
    return spansText(this.spans);
  }

  // build renders the spans as a one-row box tree: one text box per merged
  // run, plus a trailing unstyled pad box when Width was declared.
  build() {
    const row = box('row').height(1);
    const spans = mergeSpans(this.spans);
    let used = 0;
    for (const span of spans) {
      const node = text(span.text);
      if (span.style !== '') node.style(span.style);
      row.child(node);
      used += displayWidth(span.text);
    }
    const pad = this.width - used;
    if (pad > 0) row.child(text(' '.repeat(pad)));
    return row;
  }

  // styled returns the RichText re-styled as a single span.
  styled(style) {
    return new RichText({ spans: [styled(this.text(), style)], width: this.width, wrap: this.wrap });
  }

  // wrapSpans wraps to width and marks every resulting row as wrapped.
  wrapSpans(width) {
    const wrapped = wrapSpans(this.spans, width);
    if (wrapped == null) return null;
    for (const row of wrapped) row.wrap = true;
    return wrapped;
  }
}

// wrapText hard-wraps s into lines of at most width display cells. A width <= 0
// yields null.
function wrapText(s, width) {
  if (width <= 0) return null;
  const runes = Array.from(s);
  const indexLines = wrapIndexes(runes, width);
  return indexLines.map((line) => line.map((i) => runes[i]).join(''));
}

// wrapSpans wraps styled runs into lines of at most width display cells. A
// width <= 0 yields null.
function wrapSpans(spans, width) {
  if (width <= 0) return null;
  const runes = [];
  const styles = [];
  for (const span of spans) {
    const value = span.text == null ? '' : span.text;
    const style = span.style == null ? '' : span.style;
    for (const ch of value) {
      runes.push(ch);
      styles.push(style);
    }
  }
  const indexLines = wrapIndexes(runes, width);
  return indexLines.map((line) => new RichText({ spans: groupRunes(runes, styles, line), width }));
}

// groupRunes builds spans from the rune indexes of one wrapped line, merging
// adjacent runes that share a style.
function groupRunes(runes, styles, indexes) {
  const out = [];
  for (const i of indexes) {
    if (out.length > 0 && out[out.length - 1].style === styles[i]) {
      out[out.length - 1].text += runes[i];
      continue;
    }
    out.push({ text: runes[i], style: styles[i] });
  }
  return out;
}

// wrapIndexes is the shared wrap core: every returned line is a list of rune
// indexes into the source. A wide rune is never split.
function wrapIndexes(runes, width) {
  const lines = [];
  let cur = [];
  let curWidth = 0;
  const flush = (force) => {
    while (cur.length > 0 && runes[cur[cur.length - 1]] === ' ') cur.pop();
    if (cur.length > 0 || force) lines.push(cur);
    cur = [];
    curWidth = 0;
  };
  for (let i = 0; i < runes.length; i++) {
    const ch = runes[i];
    if (ch === '\n') {
      flush(true);
      continue;
    }
    if (ch === ' ' && curWidth === 0) continue;
    const rw = cellWidth(ch);
    if (curWidth + rw > width) {
      if (ch === ' ') {
        flush(false);
        continue;
      }
      const last = lastSpaceIndex(cur, runes);
      if (last >= 0) {
        const tail = cur.slice(last + 1);
        cur = cur.slice(0, last);
        flush(false);
        let tailWidth = 0;
        for (const index of tail) tailWidth += cellWidth(runes[index]);
        if (tailWidth + rw <= width) {
          cur = tail;
          curWidth = tailWidth;
          cur.push(i);
          curWidth += rw;
          continue;
        }
        cur = [i];
        curWidth = rw;
        flush(false);
        continue;
      }
      flush(false);
    }
    cur.push(i);
    curWidth += rw;
  }
  flush(lines.length === 0);
  return lines;
}

function lastSpaceIndex(line, runes) {
  for (let i = line.length - 1; i >= 0; i--) {
    if (runes[line[i]] === ' ') return i;
  }
  return -1;
}

// styleHasSegment reports whether style already carries segment as a
// semicolon-separated piece.
function styleHasSegment(style, segment) {
  if (style === '') return false;
  for (const part of style.split(';')) {
    if (part.trim() === segment) return true;
  }
  return false;
}

function withBold(style) {
  if (styleHasSegment(style, 'bold')) return style;
  if (style === '') return 'bold';
  return style + ';bold';
}

// withItalic appends the italic attribute unless the style already carries it.
function withItalic(style) {
  if (styleHasSegment(style, 'italic')) return style;
  if (style === '') return 'italic';
  return style + ';italic';
}

// parseEmphasis parses "**bold**" and "_em_" into spans.
function parseEmphasis(s) {
  return parseEmphasisStyled(s, '');
}

// parseEmphasisStyled is parseEmphasis with a base style composed onto every
// span; emphasis attributes are appended to it.
function parseEmphasisStyled(s, base) {
  const out = [];
  let plain = '';
  const flushPlain = () => {
    if (plain.length > 0) {
      out.push({ text: plain, style: base });
      plain = '';
    }
  };
  for (let i = 0; i < s.length;) {
    if (s.startsWith('**', i)) {
      const found = s.indexOf('**', i + 2);
      const end = found < 0 ? -1 : found - (i + 2);
      if (end > 0) {
        flushPlain();
        out.push({ text: s.slice(i + 2, i + 2 + end), style: withBold(base) });
        i += 2 + end + 2;
        continue;
      }
    }
    if (s[i] === '_') {
      const found = s.indexOf('_', i + 1);
      const end = found < 0 ? -1 : found - (i + 1);
      if (end > 0) {
        flushPlain();
        out.push({ text: s.slice(i + 1, i + 1 + end), style: withItalic(base) });
        i += 1 + end + 1;
        continue;
      }
    }
    plain += s[i];
    i++;
  }
  flushPlain();
  return out;
}

module.exports = {
  RichText,
  styled,
  line,
  mergeSpans,
  wrapText,
  wrapSpans,
  parseEmphasis,
  parseEmphasisStyled,
  withItalic,
};
