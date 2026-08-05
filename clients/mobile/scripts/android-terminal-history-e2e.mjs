#!/usr/bin/env node

import { execFileSync } from 'node:child_process'
import { createHash } from 'node:crypto'
import { mkdirSync, readFileSync, writeFileSync } from 'node:fs'
import { basename, dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

const packageName = 'com.anytty.app'
const scriptDir = dirname(fileURLToPath(import.meta.url))
const mobileDir = resolve(scriptDir, '..')
const repoRoot = resolve(mobileDir, '../..')
const defaultApk = resolve(mobileDir, 'android/app/build/outputs/apk/debug/app-debug.apk')
const args = parseArgs(process.argv.slice(2))
if (args.help) {
  process.stdout.write(`Usage: npm run e2e:android:history -- [options]\n\n` +
    `  --serial <adb-serial>       Select a device when more than one is connected\n` +
    `  --apk <path>                APK to verify and install\n` +
    `  --artifacts <directory>     Screenshot, log, and report output directory\n` +
    `  --terminal-id <id>          Select a specific existing terminal\n` +
    `  --pairing-file <path>       Import a one-time pairing URI on a clean emulator\n` +
    `  --bootstrap-local-pairing   Create an in-memory Cloud pairing URI with the local daemon\n` +
    `  --skip-install              Run against the currently installed debug app\n`)
  process.exit(0)
}
const apk = resolve(args.apk ?? defaultApk)
assert(!(args.pairingFile && args.bootstrapLocalPairing), '--pairing-file and --bootstrap-local-pairing are mutually exclusive')
const pairingCode = args.bootstrapLocalPairing
  ? createLocalPairingCode(args.terminalId)
  : args.pairingFile ? readFileSync(resolve(args.pairingFile), 'utf8').trim() : ''
assert(!(args.pairingFile || args.bootstrapLocalPairing) || pairingCode, 'Pairing code source returned empty')
const stamp = new Date().toISOString().replaceAll(':', '-').replaceAll('.', '-')
const artifactDir = resolve(args.artifacts ?? resolve(repoRoot, 'test-results', `android-terminal-history-${stamp}`))
mkdirSync(artifactDir, { recursive: true })

const serial = selectDevice(args.serial)
const report = {
  startedAt: new Date().toISOString(),
  serial,
  apk,
  apkSha256: sha256(apk),
  artifactDir,
  steps: [],
}
let cdp
let forwardedPort

async function run() {
try {
  verifyCloudDefaults(apk)
  const packageBefore = packageInfo(serial)
  assert(packageBefore.firstInstallTime, `${packageName} must already be installed so app data can be preserved`)
  const dataBefore = appDataInventory(serial)
  step('pre-install', { package: packageBefore, dataFiles: dataBefore.length })

  if (!args.skipInstall) {
    const installOutput = adb(serial, ['install', '-r', '-t', apk])
    assert(/Success/i.test(installOutput), `adb install did not report success: ${installOutput}`)
    const packageAfter = packageInfo(serial)
    const dataAfter = appDataInventory(serial)
    assert(packageAfter.firstInstallTime === packageBefore.firstInstallTime, 'firstInstallTime changed; app data was not preserved')
    assert(dataBefore.every((file) => dataAfter.includes(file)), 'one or more pre-existing app data files disappeared after install')
    step('install', { output: installOutput.trim(), package: packageAfter, dataFiles: dataAfter.length })
  }

  adb(serial, ['shell', 'am', 'force-stop', packageName])
  adb(serial, ['shell', 'am', 'start', '-W', '-n', `${packageName}/.MainActivity`])
  const pid = await waitFor(async () => adb(serial, ['shell', 'pidof', packageName], { allowFailure: true }).trim() || null, 20_000, 'app process')
  const socket = await waitFor(async () => webViewSocket(serial, pid), 20_000, 'WebView debug socket')
  forwardedPort = Number(adb(serial, ['forward', 'tcp:0', `localabstract:${socket}`]).trim())
  assert(Number.isFinite(forwardedPort) && forwardedPort > 0, 'adb did not allocate a CDP forwarding port')
  cdp = await connectWebView(forwardedPort)
  await cdp.send('Page.enable')
  await cdp.send('Runtime.enable')
  await waitFor(async () => await evaluate('document.readyState') === 'complete' ? true : null, 20_000, 'WebView document')
  step('webview-connected', { pid, socket, forwardedPort })

  await openFirstAvailableTerminal(args.terminalId, pairingCode)
  await waitForTerminalReady()
  const terminalIdentity = await evaluate(`(() => {
    const root = document.querySelector('[data-testid="anytty-terminal"]')
    return root ? { machineId: root.getAttribute('data-machine-id'), terminalId: root.getAttribute('data-terminal-id') } : null
  })()`)
  step('terminal-open', terminalIdentity)
  await capture('01-initial-terminal')

  const stressCommand = 'time python scripts/generate_terminal_stress.py --lines 100000'
  const snapshotsBeforeStress = eventCount(readDebugLog(serial), 'session.snapshot_received')
  await sendTerminalCommand(stressCommand)
  await waitForLogCount(serial, 'session.snapshot_received', snapshotsBeforeStress + 1, 120_000)
  await waitForDiagnosticQuiet(serial, 'session.snapshot_received', 4_000, 180_000)
  assert(!(await historyLoadingVisible()), 'history loading indicator appeared during stress output')
  await capture('02-stress-complete')
  step('stress-command', { command: stressCommand })

  await leaveAndReenterTerminal(terminalIdentity.terminalId)
  await waitForTerminalReady()
  const reentryCanvas = await canvasMetrics()
  assertCanvasVisible(reentryCanvas, 'first re-entry')
  await capture('03-first-reentry')
  step('first-reentry', { canvas: reentryCanvas, viewport: await viewportState() })

  const beforeHistoryLogs = readDebugLog(serial)
  const beforeHistoryCounts = diagnosticCounts(beforeHistoryLogs)
  const integerScrollErrorsBefore = textCount(beforeHistoryLogs, 'This API only accepts integers')
  const historyStart = eventCount(beforeHistoryLogs, 'session.scrollback_load_success')
  const firstPageStagedStart = eventCount(beforeHistoryLogs, 'xterm.history_first_page_staged')
  const pages = []
  for (let page = 1; page <= 3; page += 1) {
    await pullUntilHistoryPage(serial, historyStart + page)
    if (page === 1) {
      await waitForLogCount(serial, 'xterm.history_first_page_staged', firstPageStagedStart + 1, 30_000)
    }
    await waitFor(async () => !(await historyLoadingVisible()) ? true : null, 15_000, `history loader page ${page}`)
    const state = await viewportState()
    const canvas = await canvasMetrics()
    const frameHold = await evaluate(`(() => {
      const overlay = document.querySelector('[data-anytty-terminal-frame-hold]')
      if (!overlay) return { present: false, canvasTopOffsets: [] }
      const overlayTop = overlay.getBoundingClientRect().top
      return {
        present: true,
        canvasTopOffsets: Array.from(overlay.querySelectorAll('canvas'), (canvas) =>
          canvas.getBoundingClientRect().top - overlayTop),
      }
    })()`)
    if (page === 1) {
      assert(state?.atBottom, 'first history gesture moved the viewport before the staged page was ready')
      assert(Number(state?.scrollHeight) > Number(state?.clientHeight), 'first history page was not staged behind the live viewport')
      assert(frameHold.present, 'the live terminal frame was not retained over the staged first history page')
      assert(frameHold.canvasTopOffsets.length > 0, 'the retained live frame has no canvas layers')
      assert(frameHold.canvasTopOffsets.every((offset) => Math.abs(offset) < 1),
        `retained canvas layers shifted vertically: ${JSON.stringify(frameHold.canvasTopOffsets)}`)
    } else {
      assert(state && !state.atBottom, `history page ${page} left the viewport at the bottom`)
      assert(!frameHold.present, `the live terminal frame still covered history page ${page}`)
    }
    assertCanvasVisible(canvas, `history page ${page}`)
    pages.push({ page, viewport: state, frameHold, canvas })
    await capture(`04-history-page-${page}`)
  }
  step('history-pages', { pages })

  const frozenBefore = await viewportState()
  const deferredBefore = eventCount(readDebugLog(serial), 'xterm.history_live_update_deferred')
  sendTerminalCommandOutsideApp(terminalIdentity.terminalId, "printf '\\n__ANYTTY_LIVE_DURING_HISTORY__\\n'")
  await waitForLogCount(serial, 'xterm.history_live_update_deferred', deferredBefore + 1, 30_000)
  await sleep(1_000)
  const frozenAfter = await viewportState()
  assert(frozenBefore && frozenAfter && Math.abs(frozenAfter.viewportY - frozenBefore.viewportY) <= 1,
    `live output moved frozen viewport from ${frozenBefore?.viewportY} to ${frozenAfter?.viewportY}`)
  const frozenLogs = readDebugLog(serial)
  const frozenWrites = diagnosticDetails(frozenLogs, 'xterm.write_enqueue')
    .slice(beforeHistoryCounts['xterm.write_enqueue'] ?? 0)
  assert(!frozenWrites.some((entry) => entry.reason === 'snapshot_full_text' || entry.reason === 'snapshot_recovery'),
    `live snapshot replayed the full terminal while history was frozen: ${JSON.stringify(frozenWrites)}`)
  await capture('05-live-output-while-frozen')
  step('frozen-live-output', { before: frozenBefore, after: frozenAfter })

  const resumeBefore = eventCount(readDebugLog(serial), 'xterm.history_resume_live')
  const historyRequestsBeforeResume = eventCount(readDebugLog(serial), 'session.scrollback_load_start')
  await swipeUntilBottom()
  await waitForLogCount(serial, 'xterm.history_resume_live', resumeBefore + 1, 30_000)
  await sleep(1_000)
  const resumeAfter = eventCount(readDebugLog(serial), 'xterm.history_resume_live')
  assert(resumeAfter === resumeBefore + 1, `history resumed ${resumeAfter - resumeBefore} times instead of once`)
  assert(eventCount(readDebugLog(serial), 'session.scrollback_load_start') === historyRequestsBeforeResume,
    'returning to the frozen tail unexpectedly requested another history page')
  const bottomState = await viewportState()
  assert(bottomState?.atBottom, 'viewport did not return to the live bottom')
  await capture('06-returned-to-bottom')
  step('resume-live', { viewport: bottomState, resumeEvents: resumeAfter - resumeBefore })

  await leaveAndReenterTerminal(terminalIdentity.terminalId)
  await waitForTerminalReady()
  const finalCanvas = await canvasMetrics()
  assertCanvasVisible(finalCanvas, 'second re-entry')
  assert(!(await historyLoadingVisible()), 'history loading indicator remained visible after second re-entry')
  await capture('07-second-reentry')

  const finalLogs = readDebugLog(serial)
  const afterCounts = diagnosticCounts(finalLogs)
  const loadStarts = diagnosticDetails(finalLogs, 'session.scrollback_load_start').slice(beforeHistoryCounts['session.scrollback_load_start'] ?? 0)
  const loadSuccesses = diagnosticDetails(finalLogs, 'session.scrollback_load_success').slice(beforeHistoryCounts['session.scrollback_load_success'] ?? 0)
  const prefetchConsumptions = diagnosticDetails(finalLogs, 'session.scrollback_prefetch_consumed')
    .slice(beforeHistoryCounts['session.scrollback_prefetch_consumed'] ?? 0)
  const firstPageStaged = diagnosticDetails(finalLogs, 'xterm.history_first_page_staged')
    .slice(beforeHistoryCounts['xterm.history_first_page_staged'] ?? 0)
  const historyEntries = diagnosticDetails(finalLogs, 'xterm.history_enter')
    .slice(beforeHistoryCounts['xterm.history_enter'] ?? 0)
  assert(loadStarts.length >= 3, `expected at least three history requests, got ${loadStarts.length}`)
  const renderedCols = Number(diagnosticDetails(beforeHistoryLogs, 'session.snapshot_received').at(-1)?.cols)
  assert(Number(renderedCols) > 0 && loadStarts.every((entry) => Number(entry.cols) === Number(renderedCols)),
    `history cols did not match the local xterm: rendered=${renderedCols} requests=${JSON.stringify(loadStarts)}`)
  assert(loadStarts.filter((entry) => Number(entry.offset) === 0).length === 1, 'history unexpectedly repeated the latest offset-zero request')
  assert(prefetchConsumptions.length >= 1 && loadSuccesses[0]?.prefetched === true,
    `first history page did not consume the idle prefetch: ${JSON.stringify({ prefetchConsumptions, firstLoad: loadSuccesses[0] })}`)
  assert(firstPageStaged.length >= 1 && firstPageStaged[0]?.atBottom === true,
    `first history page was not staged at the live bottom: ${JSON.stringify(firstPageStaged)}`)
  assert(historyEntries.length >= 1,
    'a later gesture did not transition the staged page into history mode')
  assert(loadSuccesses.length >= 3 && loadSuccesses[0]?.operation === 'replace' && loadSuccesses.slice(1, 3).every((entry) => entry.operation === 'prepend'),
    `history operations were not replace followed by prepends: ${JSON.stringify(loadSuccesses)}`)
  for (let index = 1; index < Math.min(3, loadSuccesses.length); index += 1) {
    const previousTotal = Number(loadSuccesses[index - 1].totalRows)
    assert(Number(loadStarts[index].offset) === previousTotal &&
      Number(loadSuccesses[index].totalRows) === previousTotal + Number(loadSuccesses[index].loadedRows),
    `history pagination was not contiguous: ${JSON.stringify({ starts: loadStarts.slice(0, 3), successes: loadSuccesses.slice(0, 3) })}`)
  }
  assert(Math.max(...loadSuccesses.map((entry) => Number(entry.logicalTotalRows ?? 0))) >= 100000,
    `stress command did not produce at least 100000 logical rows: ${JSON.stringify(loadSuccesses)}`)
  assert((afterCounts['session.scrollback_load_failed'] ?? 0) === (beforeHistoryCounts['session.scrollback_load_failed'] ?? 0),
    'a history request failed during the E2E run')
  assert(textCount(finalLogs, 'This API only accepts integers') === integerScrollErrorsBefore,
    'xterm received a non-integer scrollLines delta during touch scrolling')
  assert(/session\.connection_info .*"path":"hub"/.test(finalLogs), 'Cloud hub connection was not confirmed in diagnostics')
  writeFileSync(resolve(artifactDir, 'anytty-debug.log'), finalLogs)
  step('final', { canvas: finalCanvas, viewport: await viewportState(), diagnosticCounts: afterCounts })

  report.completedAt = new Date().toISOString()
  report.status = 'passed'
  writeReport()
  process.stdout.write(`${JSON.stringify(report, null, 2)}\n`)
} catch (error) {
  report.completedAt = new Date().toISOString()
  report.status = 'failed'
  report.error = error instanceof Error ? error.stack ?? error.message : String(error)
  try {
    writeFileSync(resolve(artifactDir, 'anytty-debug.log'), readDebugLog(serial))
    if (cdp) await capture('failure')
  } catch {}
  writeReport()
  process.stderr.write(`${report.error}\nArtifacts: ${artifactDir}\n`)
  process.exitCode = 1
} finally {
  cdp?.close()
  if (forwardedPort) adb(serial, ['forward', '--remove', `tcp:${forwardedPort}`], { allowFailure: true })
}
}

function parseArgs(values) {
  const parsed = {}
  for (let index = 0; index < values.length; index += 1) {
    const value = values[index]
    if (value === '--skip-install') parsed.skipInstall = true
    else if (value === '--bootstrap-local-pairing') parsed.bootstrapLocalPairing = true
    else if (value === '--help' || value === '-h') parsed.help = true
    else if (value === '--apk') parsed.apk = values[++index]
    else if (value === '--artifacts') parsed.artifacts = values[++index]
    else if (value === '--serial') parsed.serial = values[++index]
    else if (value === '--terminal-id') parsed.terminalId = values[++index]
    else if (value === '--pairing-file') parsed.pairingFile = values[++index]
    else throw new Error(`Unknown argument: ${value}`)
  }
  return parsed
}

function selectDevice(requested) {
  const lines = adb(null, ['devices']).split('\n').slice(1).map((line) => line.trim()).filter(Boolean)
  const devices = lines.filter((line) => line.endsWith('\tdevice')).map((line) => line.split('\t')[0])
  if (requested) {
    assert(devices.includes(requested), `ADB device ${requested} is not connected`)
    return requested
  }
  assert(devices.length === 1, `Expected exactly one connected ADB device, found ${devices.length}`)
  return devices[0]
}

function adb(serialValue, command, options = {}) {
  const full = serialValue ? ['-s', serialValue, ...command] : command
  try {
    return execFileSync('adb', full, { encoding: 'utf8', maxBuffer: 64 * 1024 * 1024 }).replaceAll('\r', '')
  } catch (error) {
    if (options.allowFailure) return error.stdout?.toString() ?? ''
    throw error
  }
}

function packageInfo(serialValue) {
  const output = adb(serialValue, ['shell', 'dumpsys', 'package', packageName], { allowFailure: true })
  return {
    firstInstallTime: /firstInstallTime=([^\n]+)/.exec(output)?.[1]?.trim() ?? '',
    lastUpdateTime: /lastUpdateTime=([^\n]+)/.exec(output)?.[1]?.trim() ?? '',
    versionName: /versionName=([^\s]+)/.exec(output)?.[1] ?? '',
  }
}

function appDataInventory(serialValue) {
  const output = adb(serialValue, ['shell', 'run-as', packageName, 'find', '.', '-type', 'f'], { allowFailure: true })
  return output.split('\n').map((line) => line.trim()).filter(Boolean).sort()
}

function verifyCloudDefaults(apkPath) {
  const dexFiles = execFileSync('unzip', ['-Z1', apkPath], { encoding: 'utf8' })
    .split('\n').filter((name) => /^classes.*\.dex$/.test(name))
  const found = new Set()
  for (const dex of dexFiles) {
    const bytes = execFileSync('unzip', ['-p', apkPath, dex], { maxBuffer: 128 * 1024 * 1024 })
    const text = bytes.toString('latin1')
    if (text.includes('cloud.anytty.com:443')) found.add('cloud.anytty.com:443')
    if (text.includes('cloud.anytty.com')) found.add('cloud.anytty.com')
  }
  assert(found.has('cloud.anytty.com:443') && found.has('cloud.anytty.com'), 'APK DEX is missing default Cloud controller values')
  step('apk-defaults', { values: [...found] })
}

function createLocalPairingCode(terminalId) {
  assert(terminalId, '--bootstrap-local-pairing requires --terminal-id')
  const cli = resolve(repoRoot, 'anytty')
  return execFileSync(cli, [
    'pair', 'create', '--text', '--route', 'cloud', '--terminal', terminalId,
    '--label', 'Android emulator history E2E', '--ttl', '10m', '--grant-ttl', '24h',
  ], { encoding: 'utf8' }).trim()
}

function webViewSocket(serialValue, pid) {
  const output = adb(serialValue, ['shell', 'cat', '/proc/net/unix'], { allowFailure: true })
  const match = output.split('\n').find((line) => line.includes(`@webview_devtools_remote_${pid}`))
  return match?.match(/@(webview_devtools_remote_\d+)/)?.[1] ?? null
}

async function connectWebView(port) {
  const targets = await waitFor(async () => {
    try {
      const response = await fetch(`http://127.0.0.1:${port}/json`)
      return response.ok ? await response.json() : null
    } catch {
      return null
    }
  }, 20_000, 'CDP target list')
  const target = targets.find((entry) => entry.type === 'page' && entry.webSocketDebuggerUrl)
  assert(target, 'No debuggable WebView page target was found')
  const url = new URL(target.webSocketDebuggerUrl)
  url.hostname = '127.0.0.1'
  url.port = String(port)
  return await CdpConnection.open(url.toString())
}

class CdpConnection {
  static async open(url) {
    const socket = new WebSocket(url)
    await new Promise((resolveOpen, reject) => {
      socket.addEventListener('open', resolveOpen, { once: true })
      socket.addEventListener('error', reject, { once: true })
    })
    return new CdpConnection(socket)
  }

  constructor(socket) {
    this.socket = socket
    this.nextId = 1
    this.pending = new Map()
    socket.addEventListener('message', (event) => {
      const message = JSON.parse(String(event.data))
      if (!message.id) return
      const pending = this.pending.get(message.id)
      if (!pending) return
      this.pending.delete(message.id)
      if (message.error) pending.reject(new Error(`${pending.method}: ${message.error.message}`))
      else pending.resolve(message.result)
    })
  }

  send(method, params = {}) {
    const id = this.nextId++
    return new Promise((resolveSend, reject) => {
      this.pending.set(id, { method, resolve: resolveSend, reject })
      this.socket.send(JSON.stringify({ id, method, params }))
    })
  }

  close() { this.socket.close() }
}

async function evaluate(expression) {
  const result = await cdp.send('Runtime.evaluate', { expression, awaitPromise: true, returnByValue: true })
  if (result.exceptionDetails) throw new Error(result.exceptionDetails.text ?? 'Runtime.evaluate failed')
  return result.result?.value
}

async function openFirstAvailableTerminal(terminalId, pairingCodeValue = '') {
  await waitFor(async () => await evaluate(`(() => Boolean(
    document.querySelector('[data-testid="anytty-terminal"]') ||
    document.querySelector('[data-testid="anytty-terminal-list"]') ||
    document.querySelector('[data-testid="anytty-first-use"]') ||
    document.querySelector('[data-testid="anytty-app-home"] ul > li')
  ))()`), 20_000, 'stable app view')
  if (await evaluate("Boolean(document.querySelector('[data-testid=\"anytty-terminal\"]'))")) return
  if (pairingCodeValue && await evaluate("Boolean(document.querySelector('[data-testid=\"anytty-first-use\"]'))")) {
    await importPairingCode(pairingCodeValue)
  }
  if (await evaluate("Boolean(document.querySelector('[data-testid=\"anytty-app-home\"]'))")) {
    const opened = await evaluate(`(() => {
      const button = document.querySelector('[data-testid="anytty-app-home"] ul > li > div > button:first-of-type')
      if (!button) return false
      button.click()
      return true
    })()`)
    assert(opened, 'No saved Cloud machine was available on the app home screen')
  }
  const machineView = await waitFor(async () => await evaluate(`(() => {
    if (document.querySelector('[data-testid="anytty-terminal-list"]')) return 'terminal-list'
    if (document.querySelector('[data-testid="anytty-verification-gate"]') || document.querySelector('[data-testid="anytty-pair-sheet"]')) return 'verification'
    return ''
  })()`), 45_000, 'terminal list or verification gate')
  if (machineView === 'verification') {
    assert(pairingCodeValue, 'Cloud machine requires verification; no pairing code was provided')
    await importPairingCode(pairingCodeValue)
  }
  const selector = terminalId
    ? `[data-testid="anytty-terminal-list"] li[data-terminal-id=${JSON.stringify(terminalId)}] button:first-of-type`
    : '[data-testid="anytty-terminal-list"] li[data-terminal-id] button:first-of-type'
  const terminalState = await waitFor(async () => await evaluate(`(() => {
    if (document.querySelector(${JSON.stringify(selector)})) return { state: 'ready' }
    const verification = document.querySelector('[data-testid="anytty-verification-gate"]')
    const pairing = document.querySelector('[data-testid="anytty-pair-sheet"]')
    if (verification || pairing) {
      return {
        state: 'verification',
        message: (pairing || verification)?.textContent?.replace(/\\s+/g, ' ').trim() || '',
      }
    }
    return null
  })()`), 60_000, terminalId ? `terminal ${terminalId}` : 'an available terminal')
  assert(terminalState?.state === 'ready', `Cloud credentials did not connect after pairing: ${terminalState?.message || 'verification is still required'}`)
  const opened = await evaluate(`(() => { const button = document.querySelector(${JSON.stringify(selector)}); if (!button) return false; button.click(); return true })()`)
  assert(opened, terminalId ? `Terminal ${terminalId} was not found` : 'No terminal was available')
}

async function importPairingCode(pairingCodeValue) {
  const opened = await evaluate(`(() => {
    if (document.querySelector('[data-testid="anytty-pair-sheet"]')) return true
    const root = document.querySelector('[data-testid="anytty-first-use"]') || document.querySelector('[data-testid="anytty-verification-gate"]')
    const button = root?.querySelector('button')
    if (!button) return false
    button.click()
    return true
  })()`)
  assert(opened, 'Could not open the pairing sheet')
  await waitFor(async () => await evaluate("Boolean(document.querySelector('[data-testid=\"anytty-pair-sheet\"]'))") ? true : null, 10_000, 'pairing sheet')
  const submitted = await evaluate(`(() => {
    const sheet = document.querySelector('[data-testid="anytty-pair-sheet"]')
    const textarea = sheet?.querySelector('textarea')
    if (!textarea) return false
    const setter = Object.getOwnPropertyDescriptor(HTMLTextAreaElement.prototype, 'value')?.set
    setter?.call(textarea, ${JSON.stringify(pairingCodeValue)})
    textarea.dispatchEvent(new Event('input', { bubbles: true }))
    const button = sheet?.querySelector('form button[type="submit"]')
    if (!button) return false
    button.click()
    return true
  })()`)
  assert(submitted, 'Could not submit the pairing code')
  await waitFor(async () => {
    const state = await evaluate(`(() => {
      const sheet = document.querySelector('[data-testid="anytty-pair-sheet"]')
      const error = [...(sheet?.querySelectorAll('p') ?? [])].find((node) => node.className.includes('text-red'))?.textContent
      if (!sheet && document.querySelector('[data-testid="anytty-terminal-list"]')) return { ready: true }
      if (!sheet && document.querySelector('[data-testid="anytty-app-home"] ul > li')) return { ready: true }
      return { ready: false, error: error || '' }
    })()`)
    if (state?.error) throw new Error(`Pairing failed: ${state.error}`)
    return state?.ready ? true : null
  }, 60_000, 'Cloud pairing import')
  step('pairing-import', { source: 'one-time code' })
}

async function waitForTerminalReady() {
  await waitFor(async () => {
    const connected = await evaluate(`(() => {
    const terminal = document.querySelector('[data-testid="anytty-terminal"]')
    const canvas = terminal?.querySelector('.xterm-screen canvas')
    return terminal?.getAttribute('data-channel-state') === 'open' &&
      canvas && canvas.width > 0 && canvas.height > 0 ? true : null
    })()`)
    if (!connected) return null
    const metrics = await canvasMetrics()
    return metrics.some((entry) =>
      entry.width > 0 && entry.height > 0 && entry.contrastingPixels > 20 &&
      entry.uniqueColors > 1 && entry.dataUrlLength > 1000) ? true : null
  }, 60_000, 'visible connected terminal canvas')
}

async function leaveAndReenterTerminal(terminalId) {
  const left = await evaluate(`(() => {
    const button = document.querySelector('[data-testid="anytty-terminal-header"] button:first-of-type')
    if (!button) return false
    button.click()
    return true
  })()`)
  assert(left, 'Could not leave the terminal page')
  await waitFor(async () => await evaluate("Boolean(document.querySelector('[data-testid=\"anytty-terminal-list\"]'))") ? true : null, 20_000, 'terminal list after exit')
  await openFirstAvailableTerminal(terminalId)
}

async function sendTerminalCommand(command) {
  const focused = await evaluate(`(() => {
    const textarea = document.querySelector('[data-testid="anytty-terminal"] .xterm-helper-textarea')
    if (!textarea) return false
    textarea.focus()
    return document.activeElement === textarea
  })()`)
  assert(focused, 'xterm input textarea could not be focused')
  await cdp.send('Input.insertText', { text: command })
  await cdp.send('Input.dispatchKeyEvent', { type: 'rawKeyDown', key: 'Enter', code: 'Enter', windowsVirtualKeyCode: 13, nativeVirtualKeyCode: 13 })
  await cdp.send('Input.dispatchKeyEvent', { type: 'keyUp', key: 'Enter', code: 'Enter', windowsVirtualKeyCode: 13, nativeVirtualKeyCode: 13 })
}

function sendTerminalCommandOutsideApp(terminalId, command) {
  const cli = resolve(repoRoot, 'anytty')
  const output = execFileSync(cli, [
    'terminal', 'send', `local:${terminalId}`, '--literal', command, '--enter', '--json',
  ], { encoding: 'utf8' })
  assert(/"kind":"terminal_input_sent"/.test(output), `daemon-side terminal input failed: ${output}`)
}

async function pullUntilHistoryPage(serialValue, targetCount) {
  for (let attempt = 0; attempt < 40; attempt += 1) {
    if (eventCount(readDebugLog(serialValue), 'session.scrollback_load_success') >= targetCount) return
    await swipeTerminal('older')
    await sleep(120)
  }
  await waitForLogCount(serialValue, 'session.scrollback_load_success', targetCount, 20_000)
}

async function swipeUntilBottom() {
  const initial = await viewportState()
  assert(initial, 'terminal viewport is unavailable while returning to the bottom')
  const initialRemaining = Math.max(0, initial.scrollHeight - initial.clientHeight - initial.scrollTop)
  const maxAttempts = Math.min(240, Math.max(50, Math.ceil(initialRemaining / Math.max(1, initial.clientHeight) * 2) + 20))
  let previousRemaining = initialRemaining
  let stalledAttempts = 0
  for (let attempt = 0; attempt < maxAttempts; attempt += 1) {
    const state = await viewportState()
    if (state?.atBottom) return
    const remaining = Math.max(0, (state?.scrollHeight ?? 0) - (state?.clientHeight ?? 0) - (state?.scrollTop ?? 0))
    stalledAttempts = remaining >= previousRemaining - 1 ? stalledAttempts + 1 : 0
    assert(stalledAttempts < 12, `Terminal viewport stopped moving with ${Math.round(remaining)}px remaining`)
    previousRemaining = remaining
    await swipeTerminal('newer')
    await sleep(80)
  }
  const final = await viewportState()
  throw new Error(`Could not return to the terminal bottom with touch gestures: ${JSON.stringify({ initialRemaining, maxAttempts, final })}`)
}

async function swipeTerminal(direction) {
  const rect = await evaluate(`(() => {
    const element = document.querySelector('[data-testid="anytty-terminal"] .xterm-screen')
    if (!element) return null
    const rect = element.getBoundingClientRect()
    return { left: rect.left, top: rect.top, width: rect.width, height: rect.height }
  })()`)
  assert(rect, 'xterm screen rectangle is unavailable')
  const x = rect.left + rect.width * 0.5
  const fromY = rect.top + rect.height * (direction === 'older' ? 0.25 : 0.8)
  const toY = rect.top + rect.height * (direction === 'older' ? 0.8 : 0.25)
  await cdp.send('Input.dispatchTouchEvent', { type: 'touchStart', touchPoints: [{ x, y: fromY, radiusX: 3, radiusY: 3, force: 1 }] })
  for (let index = 1; index <= 10; index += 1) {
    const y = fromY + (toY - fromY) * (index / 10)
    await cdp.send('Input.dispatchTouchEvent', { type: 'touchMove', touchPoints: [{ x, y, radiusX: 3, radiusY: 3, force: 1 }] })
    await sleep(18)
  }
  await cdp.send('Input.dispatchTouchEvent', { type: 'touchEnd', touchPoints: [] })
}

async function viewportState() {
  return await evaluate(`(() => {
    const terminal = document.querySelector('[data-testid="anytty-terminal"]')
    const viewport = terminal?.querySelector('.xterm-viewport')
    if (!viewport) return null
    const cell = terminal.querySelector('.xterm-rows > div')
    const screen = terminal.querySelector('.xterm-screen')
    const measure = terminal.querySelector('.xterm-char-measure-element')
    const rowHeight = cell?.getBoundingClientRect().height || 1
    const cellWidth = measure?.getBoundingClientRect().width || 0
    const cols = cellWidth > 0 ? Math.round(screen.getBoundingClientRect().width / cellWidth) : 0
    const viewportY = Math.round(viewport.scrollTop / rowHeight)
    const maxViewportY = Math.max(0, Math.round((viewport.scrollHeight - viewport.clientHeight) / rowHeight))
    return {
      scrollTop: viewport.scrollTop,
      scrollHeight: viewport.scrollHeight,
      clientHeight: viewport.clientHeight,
      rowHeight,
      cellWidth,
      cols,
      viewportY,
      maxViewportY,
      atBottom: viewportY >= maxViewportY,
    }
  })()`)
}

async function canvasMetrics() {
  return await evaluate(`(() => {
    const canvases = [...document.querySelectorAll('[data-testid="anytty-terminal"] .xterm-screen canvas')]
    return canvases.map((canvas) => {
      const context = canvas.getContext('2d')
      const pixels = context?.getImageData(0, 0, canvas.width, canvas.height).data
      let contrastingPixels = 0
      const colors = new Set()
      if (pixels) {
        const stride = Math.max(4, Math.floor(pixels.length / 200000 / 4) * 4)
        const background = [pixels[0], pixels[1], pixels[2], pixels[3]]
        for (let index = 0; index < pixels.length; index += stride) {
          colors.add([pixels[index], pixels[index + 1], pixels[index + 2], pixels[index + 3]].join(','))
          const difference = Math.abs(pixels[index] - background[0]) + Math.abs(pixels[index + 1] - background[1]) + Math.abs(pixels[index + 2] - background[2])
          if (pixels[index + 3] > 0 && difference > 24) contrastingPixels += 1
        }
      }
      return { width: canvas.width, height: canvas.height, contrastingPixels, uniqueColors: colors.size, dataUrlLength: canvas.toDataURL().length }
    })
  })()`)
}

function assertCanvasVisible(metrics, label) {
  assert(Array.isArray(metrics) && metrics.some((entry) => entry.width > 0 && entry.height > 0 && entry.contrastingPixels > 20 && entry.uniqueColors > 1 && entry.dataUrlLength > 1000), `${label} terminal canvas is blank`)
}

async function historyLoadingVisible() {
  return await evaluate("Boolean(document.querySelector('[data-testid=\"anytty-history-loading\"]'))")
}

async function capture(name) {
  const cdpImage = await cdp.send('Page.captureScreenshot', { format: 'png', fromSurface: true })
  writeFileSync(resolve(artifactDir, `${name}-cdp.png`), Buffer.from(cdpImage.data, 'base64'))
  const deviceImage = execFileSync('adb', ['-s', serial, 'exec-out', 'screencap', '-p'], { maxBuffer: 32 * 1024 * 1024 })
  writeFileSync(resolve(artifactDir, `${name}-device.png`), deviceImage)
}

function readDebugLog(serialValue) {
  return adb(serialValue, ['exec-out', 'run-as', packageName, 'cat', 'cache/anytty-debug-logs/anytty-debug.log'], { allowFailure: true })
}

function diagnosticCounts(log) {
  const counts = {}
  for (const event of diagnosticEvents(log)) counts[event.name] = (counts[event.name] ?? 0) + 1
  return counts
}

function diagnosticDetails(log, name) {
  return diagnosticEvents(log).filter((event) => event.name === name).map((event) => event.details)
}

function diagnosticEvents(log) {
  const events = []
  for (const line of log.split('\n')) {
    const match = /\/TerminalJS:\s+(\S+)\s+(\{.*\})\s*$/.exec(line)
    if (!match) continue
    try { events.push({ name: match[1], details: JSON.parse(match[2]) }) } catch {}
  }
  return events
}

function eventCount(log, name) {
  return diagnosticEvents(log).filter((event) => event.name === name).length
}

function textCount(text, value) {
  return text.split(value).length - 1
}

async function waitForLogCount(serialValue, name, count, timeoutMs) {
  return await waitFor(async () => {
    const current = eventCount(readDebugLog(serialValue), name)
    return current >= count ? current : null
  }, timeoutMs, `${name} count ${count}`)
}

async function waitForDiagnosticQuiet(serialValue, name, quietMs, timeoutMs) {
  const startedAt = Date.now()
  let lastCount = eventCount(readDebugLog(serialValue), name)
  let lastChangeAt = Date.now()
  while (Date.now() - startedAt < timeoutMs) {
    await sleep(500)
    const current = eventCount(readDebugLog(serialValue), name)
    if (current !== lastCount) {
      lastCount = current
      lastChangeAt = Date.now()
    }
    if (current > 0 && Date.now() - lastChangeAt >= quietMs) return current
  }
  throw new Error(`${name} did not become quiet within ${timeoutMs}ms`)
}

async function waitFor(check, timeoutMs, label) {
  const startedAt = Date.now()
  let lastError
  while (Date.now() - startedAt < timeoutMs) {
    try {
      const value = await check()
      if (value) return value
    } catch (error) {
      lastError = error
      if (error instanceof Error && /requires verification|credentials did not connect/.test(error.message)) throw error
    }
    await sleep(250)
  }
  throw new Error(`Timed out waiting for ${label}${lastError ? `: ${lastError}` : ''}`)
}

function step(name, details = {}) {
  report.steps.push({ name, at: new Date().toISOString(), ...details })
}

function writeReport() {
  writeFileSync(resolve(artifactDir, 'report.json'), `${JSON.stringify(report, null, 2)}\n`)
}

function sha256(file) {
  return createHash('sha256').update(readFileSync(file)).digest('hex')
}

function assert(condition, message) {
  if (!condition) throw new Error(message)
}

function sleep(ms) {
  return new Promise((resolveSleep) => setTimeout(resolveSleep, ms))
}

await run()
