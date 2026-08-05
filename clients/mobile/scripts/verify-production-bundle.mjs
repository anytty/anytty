#!/usr/bin/env node

import { mkdtempSync, readdirSync, readFileSync, rmSync, writeFileSync } from 'node:fs'
import { dirname, extname, join, relative, resolve } from 'node:path'
import { tmpdir } from 'node:os'
import { fileURLToPath } from 'node:url'
import { runInNewContext } from 'node:vm'
import { build } from 'vite'
import { parseSync } from 'rolldown/utils'
import { productionRolldownOutput } from '../production-minify.mjs'

const mobileRoot = resolve(dirname(fileURLToPath(import.meta.url)), '..')
const distRoot = join(mobileRoot, 'dist')
const artifactRoot = process.argv[2] ? resolve(process.argv[2]) : distRoot
const forbiddenText = ['camera scan failed', 'pair claim failed', 'terminal-inventory']
const consoleMethods = ['debug', 'error', 'info', 'log', 'trace', 'warn']
const scripts = collectFiles(artifactRoot, '.js')
const styles = collectFiles(artifactRoot, '.css')

if (scripts.length === 0) throw new Error('production bundle contains no JavaScript')
for (const path of scripts) {
  const source = readFileSync(path, 'utf8')
  if (!isVendoredRuntime(path)) assertNoConsoleRuntimeReferences(path, source)
  for (const marker of forbiddenText) {
    if (source.includes(marker)) fail(path, JSON.stringify(marker))
  }
}

function isVendoredRuntime(path) {
  const artifactPath = relative(artifactRoot, path).replaceAll('\\', '/')
  return artifactPath.split('/')[0] === 'vendor' || artifactPath === 'wasm/model/occt-import-js.js'
}
for (const path of styles) {
  if (readFileSync(path, 'utf8').includes('@tailwind')) fail(path, 'raw @tailwind directive')
}
await assertProductionFixtures()

console.log(`Production mobile bundle contract passed (${scripts.length} JavaScript, ${styles.length} CSS files)`)

function collectFiles(directory, extension) {
  return readdirSync(directory, { withFileTypes: true }).flatMap((entry) => {
    const path = join(directory, entry.name)
    if (entry.isDirectory()) return collectFiles(path, extension)
    return entry.isFile() && extname(entry.name) === extension ? [path] : []
  })
}

function fail(path, marker) {
  throw new Error(`production JavaScript contains ${marker}: ${relative(mobileRoot, path)}`)
}

function assertNoConsoleRuntimeReferences(path, source) {
  const reference = findConsoleRuntimeReference(path, source)
  if (reference) fail(path, reference)
}

function findConsoleRuntimeReference(path, source) {
  const parsed = parseSync(path, source, { lang: 'js', sourceType: 'module' })
  if (parsed.errors.length !== 0) throw new Error(`could not parse production JavaScript: ${relative(mobileRoot, path)}`)
  const hasReleaseSink = hasVerifiedReleaseConsoleSink(parsed.program)
  let reference = null
  visit(parsed.program, null, null, (node, parent, key) => {
    if (reference) return
    if (isGlobalConsoleMember(node)) {
      reference = 'global console member access'
      return
    }
    if (isConsoleIdentifier(node) && !hasReleaseSink && !isNonRuntimeIdentifier(node, parent, key)) {
      reference = 'global console reference'
    }
  })
  return reference
}

function isGlobalConsoleMember(node) {
  if (!isMemberExpression(node) || staticMemberName(node) !== 'console') return false
  return node.object?.type === 'Identifier' && ['globalThis', 'self', 'window'].includes(node.object.name)
}

function hasVerifiedReleaseConsoleSink(program) {
  return program.body.some((statement) => {
    if (statement.type !== 'VariableDeclaration' || statement.kind !== 'const' || statement.declarations.length !== 1) return false
    const declaration = statement.declarations[0]
    if (!isConsoleIdentifier(declaration.id)) return false
    const init = declaration.init
    if (init?.type !== 'CallExpression' || init.arguments.length !== 1) return false
    if (!isMemberExpression(init.callee) || staticMemberName(init.callee) !== 'freeze' ||
        init.callee.object?.type !== 'Identifier' || init.callee.object.name !== 'Object') {
      return false
    }
    const object = init.arguments[0]
    if (object?.type !== 'ObjectExpression' || object.properties.length !== consoleMethods.length) return false
    const methods = object.properties.map((property) => {
      const name = property.computed ? null : property.key?.name ?? property.key?.value
      const value = property.value
      const empty = value?.type === 'FunctionExpression' && value.params.length === 0 &&
        value.body?.type === 'BlockStatement' && value.body.body.length === 0
      return empty ? name : null
    }).sort()
    return methods.every((method, index) => method === consoleMethods[index])
  })
}

async function assertProductionFixtures() {
  await assertProductionBuildFixture()

  const rejectedFixtures = [
    'const write = console.error; write(globalThis.__anyttyAliasSideEffect())',
    'const sink = console; sink.info(globalThis.__anyttyObjectAliasSideEffect())',
    'globalThis["console"].warn(globalThis.__anyttyGlobalAliasSideEffect())',
    'window.console.info(globalThis.__anyttyWindowSideEffect())',
    'self["console"].error(globalThis.__anyttySelfSideEffect())',
  ]
  for (const fixture of rejectedFixtures) {
    if (!findConsoleRuntimeReference('console-reference-fixture.js', fixture)) {
      throw new Error('production console gate accepted an indirect console reference')
    }
  }

  const allowedFixture = `
    const words = ['console', 'syntax token'];
    const grammar = { console: words[0] };
    globalThis.__anyttyFixtureKept = words[0] + grammar.console;
  `
  assertNoConsoleRuntimeReferences('console-string-fixture.js', allowedFixture)
}

async function assertProductionBuildFixture() {
  const directMarker = '__anyttyDirectConsoleSideEffect'
  const computedMarker = '__anyttyComputedConsoleSideEffect'
  const methodAliasMarker = '__anyttyMethodAliasSideEffect'
  const objectAliasMarker = '__anyttyObjectAliasSideEffect'
  const fixtureRoot = mkdtempSync(join(tmpdir(), 'anytty-production-build-'))
  const entry = join(fixtureRoot, 'fixture.js')
  const outDir = join(fixtureRoot, 'dist')
  writeFileSync(entry, `
    globalThis.__anyttyFixtureKept = true;
    console.error(globalThis.${directMarker}());
    console['warn'](globalThis.${computedMarker}());
    const write = console.error;
    const sink = console;
    globalThis.__anyttyRunConsoleAliases = () => {
      write(globalThis.${methodAliasMarker}());
      sink.info(globalThis.${objectAliasMarker}());
    };
  `)

  try {
    await build({
      configFile: false,
      root: fixtureRoot,
      publicDir: false,
      logLevel: 'silent',
      build: {
        emptyOutDir: true,
        minify: 'oxc',
        outDir,
        rolldownOptions: {
          input: entry,
          output: {
            ...productionRolldownOutput(),
            entryFileNames: 'fixture.js',
            format: 'iife',
            name: 'AnyTTYProductionConsoleFixture',
          },
        },
      },
    })
    const outputPath = join(outDir, 'fixture.js')
    const output = readFileSync(outputPath, 'utf8')
    if (output.includes(directMarker) || output.includes(computedMarker)) {
      throw new Error('Vite production build retained direct console argument evaluation')
    }
    assertNoConsoleRuntimeReferences(outputPath, output)

    let directSideEffects = 0
    let aliasSideEffects = 0
    let globalConsoleCalls = 0
    const globalConsole = Object.fromEntries(consoleMethods.map((method) => [method, () => { globalConsoleCalls += 1 }]))
    const sandbox = {
      console: globalConsole,
      [directMarker]: () => { directSideEffects += 1 },
      [computedMarker]: () => { directSideEffects += 1 },
      [methodAliasMarker]: () => { aliasSideEffects += 1 },
      [objectAliasMarker]: () => { aliasSideEffects += 1 },
    }
    sandbox.globalThis = sandbox
    runInNewContext(output, sandbox)
    if (directSideEffects !== 0) throw new Error('Vite production build evaluated a dropped console argument')
    if (typeof sandbox.__anyttyRunConsoleAliases !== 'function') {
      throw new Error('Vite production build fixture did not retain the dependency alias path')
    }
    sandbox.__anyttyRunConsoleAliases()
    if (aliasSideEffects !== 2 || globalConsoleCalls !== 0) {
      throw new Error('Vite production build dependency alias reached the global console')
    }
  } finally {
    rmSync(fixtureRoot, { force: true, recursive: true })
  }
}

function isMemberExpression(node) {
  return node?.type === 'MemberExpression' ||
    node?.type === 'StaticMemberExpression' ||
    node?.type === 'ComputedMemberExpression'
}

function staticMemberName(node) {
  if (!isMemberExpression(node)) return null
  const property = node.type === 'ComputedMemberExpression' ? node.expression ?? node.property : node.property
  if (node.computed || node.type === 'ComputedMemberExpression') {
    return property?.type === 'Literal' || property?.type === 'StringLiteral' ? property.value : null
  }
  return property?.name ?? null
}

function isConsoleIdentifier(node) {
  return node?.type === 'Identifier' && node.name === 'console'
}

function isNonRuntimeIdentifier(node, parent, key) {
  if (!parent) return false
  if (isMemberExpression(parent) && key === 'property' && !parent.computed && parent.type !== 'ComputedMemberExpression') {
    return true
  }
  if ((parent.type === 'Property' || parent.type === 'ObjectProperty' || parent.type === 'MethodDefinition' ||
      parent.type === 'PropertyDefinition') && key === 'key' && !parent.computed) {
    return true
  }
  if (parent.type === 'VariableDeclarator' && key === 'id') return true
  if ((parent.type === 'FunctionDeclaration' || parent.type === 'FunctionExpression' || parent.type === 'ClassDeclaration' ||
      parent.type === 'ClassExpression') && key === 'id') {
    return true
  }
  if ((parent.type === 'FunctionDeclaration' || parent.type === 'FunctionExpression' || parent.type === 'ArrowFunctionExpression') &&
      key === 'params') {
    return true
  }
  if (parent.type.startsWith?.('Import') || parent.type === 'ExportSpecifier') return true
  return (parent.type === 'LabeledStatement' || parent.type === 'BreakStatement' || parent.type === 'ContinueStatement') && key === 'label'
}

function visit(value, parent, key, callback) {
  if (Array.isArray(value)) {
    value.forEach((entry) => visit(entry, parent, key, callback))
    return
  }
  if (value === null || typeof value !== 'object') return
  if (typeof value.type === 'string') callback(value, parent, key)
  for (const [childKey, child] of Object.entries(value)) visit(child, value, childKey, callback)
}
