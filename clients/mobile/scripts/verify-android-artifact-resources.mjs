#!/usr/bin/env node

import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'

if (process.argv.length !== 6) {
  console.error(`usage: ${process.argv[1]} NETWORK_SECURITY_CONFIG BACKUP_RULES DATA_EXTRACTION_RULES INDEX_HTML`)
  process.exit(2)
}

const [networkPath, backupPath, extractionPath, indexPath] = process.argv.slice(2).map((path) => resolve(path))
const network = normalizedXml(readFileSync(networkPath, 'utf8'))
const backup = normalizedXml(readFileSync(backupPath, 'utf8'))
const extraction = normalizedXml(readFileSync(extractionPath, 'utf8'))
const index = readFileSync(indexPath, 'utf8')
const backupDomains = ['root', 'file', 'database', 'sharedpref', 'external', 'device_root', 'device_file', 'device_database', 'device_sharedpref']
const expectedCsp = "default-src 'none'; script-src 'self'; style-src 'self' 'unsafe-inline'; font-src 'self'; img-src 'self' data: blob:; media-src 'self' data: blob:; worker-src 'self'; child-src 'self'; connect-src 'self' ws://127.0.0.1:*; object-src 'none'; base-uri 'none'; frame-src 'none'; form-action 'none'; manifest-src 'none'"

function fail(message) {
  throw new Error(`Android artifact resource contract failed: ${message}`)
}

function normalizedXml(source) {
  if (source.trimStart().startsWith('<')) return source
  const roots = []
  const stack = []
  for (const [index, line] of source.split(/\r?\n/).entries()) {
    if (line.trim() === '') continue
    const indent = line.match(/^\s*/)[0].length
    const element = line.trim().match(/^E: ([A-Za-z][A-Za-z0-9-]*) \(line=\d+\)$/)
    if (element) {
      while (stack.length > 0 && stack.at(-1).indent >= indent) stack.pop()
      const node = { name: element[1], attributes: {}, text: '', children: [], indent }
      if (stack.length === 0) roots.push(node)
      else stack.at(-1).children.push(node)
      stack.push(node)
      continue
    }
    const attribute = line.trim().match(/^A: ([A-Za-z_:][A-Za-z0-9_.:-]*)=(?:"([^"]*)"|([^\s]+))(?: \(Raw:.*\))?$/)
    if (attribute && stack.length > 0 && indent > stack.at(-1).indent) {
      stack.at(-1).attributes[attribute[1]] = attribute[2] ?? attribute[3]
      continue
    }
    const text = line.trim().match(/^T: '([^']*)'$/)
    if (text && stack.length > 0 && indent > stack.at(-1).indent) {
      stack.at(-1).text += text[1]
      continue
    }
    fail(`unrecognized aapt2 xmltree line ${index + 1}: ${line.trim()}`)
  }
  if (roots.length !== 1) fail(`aapt2 xmltree must contain exactly one root, got ${roots.length}`)
  return serialize(roots[0])
}

function serialize(node) {
  const attrs = Object.entries(node.attributes).map(([name, value]) => ` ${name}="${escapeXml(value)}"`).join('')
  return `<${node.name}${attrs}>${escapeXml(node.text)}${node.children.map(serialize).join('')}</${node.name}>`
}

function escapeXml(value) {
  return value.replaceAll('&', '&amp;').replaceAll('"', '&quot;').replaceAll('<', '&lt;').replaceAll('>', '&gt;')
}

assertOnlyTags(network, ['network-security-config', 'base-config', 'domain-config', 'domain'], 'network security config')
const bases = elements(network, 'base-config')
if (bases.length !== 1 || bases[0].attributes.cleartextTrafficPermitted !== 'false') {
  fail('network base-config must deny cleartext')
}
const domainConfigs = elements(network, 'domain-config')
if (domainConfigs.length !== 1 || domainConfigs[0].attributes.cleartextTrafficPermitted !== 'true') {
  fail('network domain-config must be the only cleartext exception')
}
const domains = elements(network, 'domain').map(({ attributes, body }) => ({
  name: body.trim(),
  includeSubdomains: attributes.includeSubdomains,
}))
const expectedDomains = [
  { name: '127.0.0.1', includeSubdomains: 'false' },
  { name: 'localhost', includeSubdomains: 'false' },
]
domains.sort((left, right) => left.name.localeCompare(right.name))
if (JSON.stringify(domains) !== JSON.stringify(expectedDomains)) fail(`unexpected cleartext domains: ${JSON.stringify(domains)}`)

assertOnlyTags(backup, ['full-backup-content', 'exclude'], 'full backup rules')
assertExcludes(backup, backupDomains, 'full backup rules')
assertOnlyTags(extraction, ['data-extraction-rules', 'cloud-backup', 'device-transfer', 'exclude'], 'data extraction rules')
for (const section of ['cloud-backup', 'device-transfer']) {
  const body = elements(extraction, section)
  if (body.length !== 1) fail(`${section} must occur exactly once`)
  assertExcludes(body[0].body, backupDomains, section)
}

const charsetTag = '<meta charset="UTF-8" />'
const cspTag = `<meta http-equiv="Content-Security-Policy" content="${expectedCsp}" />`
const charsetOffset = index.indexOf(charsetTag)
if (charsetOffset < 0) fail('UTF-8 charset meta is missing')
const afterCharset = index.slice(charsetOffset + charsetTag.length)
if (!/^\s*<meta http-equiv="Content-Security-Policy"/.test(afterCharset)) fail('CSP does not immediately follow charset')
if (!afterCharset.trimStart().startsWith(cspTag)) fail('CSP value is not the exact Android contract')

console.log('Android artifact resource contract passed')

function elements(xml, name) {
  const result = []
  const paired = new RegExp(`<${name}(?=\\s|>)([^>]*)>([\\s\\S]*?)<\\/${name}>`, 'g')
  for (const match of xml.matchAll(paired)) result.push({ attributes: attributes(match[1]), body: match[2] })
  const selfClosing = new RegExp(`<${name}(?=\\s|\\/>)([^>]*)\\/>`, 'g')
  for (const match of xml.matchAll(selfClosing)) result.push({ attributes: attributes(match[1]), body: '' })
  return result
}

function attributes(source) {
  const result = {}
  for (const match of source.matchAll(/([A-Za-z_:][A-Za-z0-9_.:-]*)="([^"]*)"/g)) result[match[1]] = match[2]
  return result
}

function assertOnlyTags(xml, allowed, label) {
  const actual = [...xml.matchAll(/<\/?([A-Za-z][A-Za-z0-9-]*)\b/g)].map((match) => match[1])
  for (const name of actual) if (!allowed.includes(name)) fail(`${label} contains unexpected tag ${name}`)
}

function assertExcludes(xml, expectedDomains, label) {
  if (/<include\b/.test(xml)) fail(`${label} contains an include`)
  const excludes = elements(xml, 'exclude').map(({ attributes }) => `${attributes.domain}\0${attributes.path}`).sort()
  const expected = expectedDomains.map((domain) => `${domain}\0.`).sort()
  if (JSON.stringify(excludes) !== JSON.stringify(expected)) fail(`${label} exclusions are incomplete or contain extras`)
}
