'use strict';

const { ProgressBar, Spinner, Badge, Tags } = require('./progress');

class Theme {
  constructor(options = {}) {
    this.name = options.name || '';
    this.text = options.text || '';
    this.title = options.title || '';
    this.accent = options.accent || '';
    this.muted = options.muted || '';
    this.success = options.success || '';
    this.warning = options.warning || '';
    this.danger = options.danger || '';
    this.border = options.border || '';
    this.borderFocus = options.borderFocus || '';
    this.selection = options.selection || '';
    this.selectionText = options.selectionText || '';
    this.input = options.input || '';
    this.placeholder = options.placeholder || '';
    this.cursor = options.cursor || '';
    this.statusBar = options.statusBar || '';
    this.tabActive = options.tabActive || '';
    this.tabInactive = options.tabInactive || '';
    this.header = options.header || '';
    this.footer = options.footer || '';
    this.toast = options.toast || '';
    this.overlay = options.overlay || '';
    this.backdrop = options.backdrop || '';
    this.marker = options.marker || '';
    this.separator = options.separator || '';
    this.zebra = options.zebra || '';
  }
}

function darkTheme() {
  return new Theme({
    name: 'dark',
    text: '',
    title: 'chrome',
    accent: 'accent',
    muted: 'muted',
    success: 'success',
    warning: 'warning',
    danger: 'danger',
    border: 'border',
    borderFocus: 'border_focus',
    selection: 'selection',
    selectionText: 'selection',
    input: '',
    placeholder: 'muted',
    cursor: 'reverse',
    statusBar: 'status',
    tabActive: 'tab_active',
    tabInactive: 'tab_inactive',
    header: 'status',
    footer: 'muted',
    toast: 'overlay',
    overlay: 'overlay',
    backdrop: 'dim',
    marker: 'accent',
    separator: 'muted',
    zebra: 'muted',
  });
}

function lightTheme() {
  return new Theme({
    name: 'light',
    text: 'fg:#24242a',
    title: 'fg:#f7f6fa;bg:#6d5ae6;bold',
    accent: 'fg:#6d3bd4;bold',
    muted: 'fg:#77737f',
    success: 'fg:#1d8a4a;bold',
    warning: 'fg:#a96b00;bold',
    danger: 'fg:#c02b3a;bold',
    border: 'fg:#b9b4c4',
    borderFocus: 'fg:#6d3bd4',
    selection: 'fg:#24242a;bg:#d9d1f5',
    selectionText: 'fg:#24242a;bg:#d9d1f5',
    input: 'fg:#24242a',
    placeholder: 'fg:#9a96a4',
    cursor: 'reverse',
    statusBar: 'fg:#3a3743;bg:#e4e1ec',
    tabActive: 'fg:#ffffff;bg:#6d5ae6;bold',
    tabInactive: 'fg:#77737f',
    header: 'fg:#3a3743;bg:#eceaf2;bold',
    footer: 'fg:#77737f',
    toast: 'fg:#24242a;bg:#fff2c2',
    overlay: 'fg:#24242a;bg:#f7f6fa',
    backdrop: 'dim',
    marker: 'fg:#6d3bd4;bold',
    separator: 'fg:#b9b4c4',
    zebra: 'fg:#77737f',
  });
}

let defaultThemeValue = darkTheme();

function defaultTheme() {
  return defaultThemeValue;
}

function setDefaultTheme(theme) {
  defaultThemeValue = theme;
  return theme;
}

function themeByName(name) {
  if (!name || name === 'dark') return [darkTheme(), true];
  if (name === 'light') return [lightTheme(), true];
  return [new Theme(), false];
}

function themed(modulePath, exportName, fields) {
  try {
    const moduleExports = require(modulePath);
    if (moduleExports && typeof moduleExports[exportName] === 'function') {
      return Object.assign(new moduleExports[exportName](), fields);
    }
  } catch (err) {
    return Object.assign({}, fields);
  }
  return Object.assign({}, fields);
}

function themedList(theme) {
  return themed('./list', 'List', {
    style: theme.text,
    selectedStyle: theme.selection,
    headerStyle: theme.header,
    footerStyle: theme.footer,
    emptyStyle: theme.muted,
    marker: theme.marker,
  });
}

function themedVirtualList(theme) {
  return themed('./list', 'VirtualList', {
    style: theme.text,
    selectedStyle: theme.selection,
    disabledStyle: theme.muted,
    headerStyle: theme.header,
    footerStyle: theme.footer,
    emptyStyle: theme.muted,
    marker: theme.marker,
  });
}

function themedTable(theme) {
  return themed('./table', 'Table', {
    style: theme.text,
    headerStyle: theme.header,
    selectedStyle: theme.selection,
    zebraStyle: theme.zebra,
    footerStyle: theme.footer,
    separatorStyle: theme.separator,
  });
}

function themedTextInput(theme) {
  return themed('./input', 'TextInput', {
    style: theme.input,
    placeholderStyle: theme.placeholder,
    cursorStyle: theme.cursor,
  });
}

function themedTextArea(theme) {
  return themed('./input', 'TextArea', {
    style: theme.input,
    placeholderStyle: theme.placeholder,
    cursorStyle: theme.cursor,
  });
}

function themedModal(theme) {
  return themed('./modal', 'Modal', {
    style: theme.borderFocus,
    backdropStyle: theme.backdrop,
  });
}

function themedMenu(theme) {
  return themed('./modal', 'Menu', {
    style: theme.text,
    selectedStyle: theme.selection,
    disabledStyle: theme.muted,
    separatorStyle: theme.separator,
    frameStyle: theme.border,
    marker: theme.marker,
  });
}

function themedProgressBar(theme) {
  return new ProgressBar({
    style: theme.accent,
    trackStyle: theme.muted,
    labelStyle: theme.text,
    percentStyle: theme.muted,
  });
}

function themedSpinner(theme) {
  return new Spinner({ style: theme.accent, labelStyle: theme.text });
}

function themedBadge(theme) {
  return new Badge({ style: theme.accent });
}

function themedTags(theme) {
  return new Tags({ style: theme.text, sepStyle: theme.separator });
}

module.exports = {
  Theme,
  darkTheme,
  lightTheme,
  defaultTheme,
  setDefaultTheme,
  themeByName,
  themedList,
  themedVirtualList,
  themedTable,
  themedTextInput,
  themedTextArea,
  themedModal,
  themedMenu,
  themedProgressBar,
  themedSpinner,
  themedBadge,
  themedTags,
};
