'use strict';

const wire = require('./src/wire');
const core = require('./src/core');
const builder = require('./src/builder');

module.exports = Object.assign({}, wire, core, builder, { wire, core, builder });
