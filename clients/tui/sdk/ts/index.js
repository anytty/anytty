'use strict';

const wire = require('./src/wire');
const core = require('./src/core');
const builder = require('./src/builder');
const widgets = require('./src/widgets');

module.exports = Object.assign({}, wire, core, builder, { wire, core, builder, widgets });
