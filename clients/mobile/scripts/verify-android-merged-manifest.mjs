#!/usr/bin/env node

import { readFileSync, statSync } from 'node:fs'
import { resolve } from 'node:path'

const manifestPath = process.argv[2] ? resolve(process.argv[2]) : null
const resourceTablePath = process.argv[3] ? resolve(process.argv[3]) : null
if (!manifestPath) {
  console.error(`usage: ${process.argv[1]} MERGED_MANIFEST [AAPT2_RESOURCE_TABLE]`)
  process.exit(2)
}

function fail(message) {
  throw new Error(`Android merged manifest contract failed: ${message}`)
}

if (!statSync(manifestPath).isFile()) fail(`not a file: ${manifestPath}`)
const manifest = readFileSync(manifestPath, 'utf8')
const resourceTable = resourceTablePath ? readFileSync(resourceTablePath, 'utf8') : null
const application = manifest.match(/<application\b[\s\S]*?>/)?.[0]
if (!application) fail('application element is missing')

const requiredApplicationAttributes = new Map([
  ['android:allowBackup', 'false'],
  ['android:fullBackupContent', resourceReference('backup_rules')],
  ['android:dataExtractionRules', resourceReference('data_extraction_rules')],
  ['android:usesCleartextTraffic', 'false'],
  ['android:networkSecurityConfig', resourceReference('network_security_config')],
])
for (const [name, expected] of requiredApplicationAttributes) {
  const actual = attribute(application, name)
  if (actual !== expected) fail(`${name} must be ${JSON.stringify(expected)}, got ${JSON.stringify(actual)}`)
}

if (attribute(application, 'android:debuggable') === 'true') fail('release application is debuggable')
if (attribute(application, 'android:testOnly') === 'true') fail('release application is testOnly')
if (/android\.intent\.action\.VIEW|android\.intent\.category\.BROWSABLE|<data\b/.test(manifest)) {
  fail('release manifest contains a deep-link entry')
}

const providers = componentBlocks('provider')
for (const provider of providers) {
  const name = attribute(provider.tag, 'android:name')
  const authority = attribute(provider.tag, 'android:authorities')
  if (name === 'androidx.core.content.FileProvider' || authority?.endsWith('.debugprovider')) {
    fail(`release manifest contains forbidden FileProvider: ${JSON.stringify({ name, authority })}`)
  }
}
const startupProviders = providers.filter(({ tag }) =>
  attribute(tag, 'android:name') === 'androidx.startup.InitializationProvider')
if (startupProviders.length !== 1) {
  fail(`release manifest must contain exactly one InitializationProvider, got ${startupProviders.length}`)
}
const startupProvider = startupProviders[0]
if (attribute(startupProvider.tag, 'android:exported') !== 'false') {
  fail('InitializationProvider must not be exported')
}
const lifecycleInitializers = [...startupProvider.body.matchAll(/<meta-data\b[^>]*>/g)].filter((match) =>
  attribute(match[0], 'android:name') === 'androidx.lifecycle.ProcessLifecycleInitializer' &&
  attribute(match[0], 'android:value') === 'androidx.startup')
if (lifecycleInitializers.length !== 1) {
  fail(`InitializationProvider must contain exactly one ProcessLifecycleInitializer, got ${lifecycleInitializers.length}`)
}

const profileReceivers = componentBlocks('receiver').filter(({ tag }) =>
  attribute(tag, 'android:name') === 'androidx.profileinstaller.ProfileInstallReceiver')
if (profileReceivers.length !== 1) {
  fail(`release manifest must contain exactly one ProfileInstallReceiver, got ${profileReceivers.length}`)
}
const profileReceiver = profileReceivers[0].tag
if (attribute(profileReceiver, 'android:exported') !== 'true' ||
    attribute(profileReceiver, 'android:permission') !== 'android.permission.DUMP') {
  fail('ProfileInstallReceiver must be exported only behind android.permission.DUMP')
}

const exported = []
for (const match of manifest.matchAll(/<(activity|activity-alias|service|receiver|provider)\b[^>]*>/g)) {
  if (attribute(match[0], 'android:exported') === 'true') {
    exported.push({ type: match[1], name: attribute(match[0], 'android:name') })
  }
}
const expectedExported = exported.length === 2 &&
  exported.some(({ type, name }) => type === 'activity' && isMainActivity(name)) &&
  exported.some(({ type, name }) => type === 'receiver' && name === 'androidx.profileinstaller.ProfileInstallReceiver')
if (!expectedExported) {
  fail(`unexpected exported components: ${JSON.stringify(exported)}`)
}
if (!manifest.includes('android.intent.action.MAIN') || !manifest.includes('android.intent.category.LAUNCHER')) {
  fail('launcher activity contract is missing')
}

console.log(`Android merged manifest contract passed: ${manifestPath}`)

function attribute(tag, name) {
  return tag.match(new RegExp(`(?:^|\\s)${escapeRegExp(name)}="([^"]*)"`))?.[1] ?? null
}

function isMainActivity(name) {
  return name === '.MainActivity' || name === 'com.anytty.app.MainActivity'
}

function componentBlocks(name) {
  const blocks = []
  const components = new RegExp(`<${name}\\b[^>]*?(?:\\/>|>[\\s\\S]*?<\\/${name}>)`, 'g')
  for (const match of manifest.matchAll(components)) {
    const tag = match[0].match(new RegExp(`^<${name}\\b[^>]*>`))[0]
    blocks.push({ tag, body: tag.endsWith('/>') ? '' : match[0] })
  }
  return blocks
}

function resourceReference(name) {
  if (resourceTable === null) return `@xml/${name}`
  const matches = [...resourceTable.matchAll(new RegExp(`^\\s*resource (0x[0-9a-fA-F]+) xml/${escapeRegExp(name)}\\s*$`, 'gm'))]
  if (matches.length !== 1) fail(`resource table must contain exactly one xml/${name}, got ${matches.length}`)
  return `@ref/${matches[0][1].toLowerCase()}`
}

function escapeRegExp(value) {
  return value.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
}
