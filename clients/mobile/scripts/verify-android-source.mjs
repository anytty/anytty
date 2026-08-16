#!/usr/bin/env node

import { execFileSync } from 'node:child_process'
import { existsSync, readFileSync } from 'node:fs'
import { dirname, join, relative, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

const mobileRoot = resolve(dirname(fileURLToPath(import.meta.url)), '..')
const repoRoot = resolve(mobileRoot, '..', '..')
const androidRoot = join(mobileRoot, 'android')
const appRoot = join(androidRoot, 'app')
const mainRoot = join(appRoot, 'src', 'main')
const javaRoot = join(mainRoot, 'java', 'com', 'anytty', 'app')

function fail(message) {
  throw new Error(`Android source contract failed: ${message}`)
}

function read(path) {
  if (!existsSync(path)) fail(`controlled source is missing: ${relative(repoRoot, path)}`)
  return readFileSync(path, 'utf8')
}

function forbid(text, fragments, label) {
  for (const fragment of fragments) {
    if (text.includes(fragment)) fail(`${label} contains forbidden ${JSON.stringify(fragment)}`)
  }
}

if (existsSync(join(mobileRoot, 'native', 'android'))) fail('duplicate native/android source mirror exists')

for (const path of [
  join(javaRoot, 'AnyTTYDebugLog.kt'),
  join(mainRoot, 'res', 'xml', 'file_paths.xml'),
  join(mobileRoot, 'src', 'nativeDebugLog.ts'),
  join(androidRoot, 'gradle', 'dependency-locks', 'capacitor-cordova-android-plugins.lockfile'),
]) {
  if (existsSync(path)) fail(`deleted security owner returned: ${relative(repoRoot, path)}`)
}

const trackedGradleFiles = execFileSync(
  'git',
  ['ls-files', 'clients/mobile/android/*.gradle', 'clients/mobile/android/**/*.gradle'],
  { cwd: repoRoot, encoding: 'utf8' },
).trim().split('\n').filter(Boolean)
for (const tracked of trackedGradleFiles) {
  const source = read(join(repoRoot, tracked))
  if (/\bflatDir\b/.test(source)) fail(`controlled Gradle source contains flatDir: ${tracked}`)
  if (/implementation\s+fileTree\s*\(/.test(source)) fail(`controlled Gradle source contains a fileTree jar dependency: ${tracked}`)
  forbid(source, ['aaptOptions', "project(':capacitor-cordova-android-plugins')", '1.9.24'], tracked)
}

const manifests = execFileSync(
  'git',
  ['ls-files', 'clients/mobile/android/app/src/**/AndroidManifest.xml'],
  { cwd: repoRoot, encoding: 'utf8' },
).trim().split('\n').filter(Boolean)
for (const tracked of manifests) {
  const source = read(join(repoRoot, tracked))
  const forbidden = [
    'android:debuggable="true"',
    'android:testOnly="true"',
    'android.intent.action.VIEW',
    'android.intent.category.BROWSABLE',
  ]
  if (!tracked.includes('/debug/')) forbidden.push('android:name="androidx.core.content.FileProvider"')
  forbid(source, forbidden, tracked)
}

const capacitor = read(join(mobileRoot, 'capacitor.config.ts'))
forbid(capacitor, ['allowMixedContent: true', "loggingBehavior: 'debug'", "loggingBehavior: 'production'"], 'capacitor.config.ts')
if (!capacitor.includes("appStartPath: '/index.html'")) {
  fail('capacitor.config.ts must use an absolute appStartPath so the local app URL is valid')
}

const activity = read(join(javaRoot, 'MainActivity.java'))
forbid(activity, [
  'MIXED_CONTENT_ALWAYS_ALLOW',
  'MIXED_CONTENT_COMPATIBILITY_MODE',
  'setAllowFileAccess(true)',
  'setAllowContentAccess(true)',
  'setAllowFileAccessFromFileURLs(true)',
  'setAllowUniversalAccessFromFileURLs(true)',
  'setGeolocationEnabled(true)',
], 'MainActivity.java')

const chromeClient = read(join(javaRoot, 'AnyTTYWebChromeClient.java'))
forbid(chromeClient, ['RESOURCE_AUDIO_CAPTURE', 'new Intent(', 'startActivityForResult('], 'AnyTTYWebChromeClient.java')

const bridge = read(join(repoRoot, 'client', 'binding', 'loopback', 'server.go'))
forbid(bridge, ['perOrigin', 'onConnect(false)', 'slotSnapshot', 'getDeclaredMethod(', 'getDeclaredField('], 'bridge transport')

const plugins = read(join(javaRoot, 'NativeConnectionPlugin.kt')) + read(join(javaRoot, 'NativeFilePickerPlugin.kt'))
forbid(plugins, ['android.util.Log', 'exportDebugLogs', 'writeDebugLog'], 'Android plugins')
if (/\bLog\.(?:d|e|i|v|w|wtf)\s*\(/.test(plugins)) fail('Android plugins call Log.*')

const releaseLog = read(join(appRoot, 'src', 'release', 'java', 'com', 'anytty', 'app', 'AnyTTYDebugLog.kt'))
forbid(releaseLog, ['File(', 'Log.', 'Throwable', 'String?', 'message', 'details'], 'release diagnostics')

const debugLog = read(join(appRoot, 'src', 'debug', 'java', 'com', 'anytty', 'app', 'AnyTTYDebugLog.kt'))
forbid(debugLog, ['Throwable', 'Uri', 'message:', 'tag:', 'details:', 'logcat', '<T : Enum'], 'debug diagnostics')

const androidGo = read(join(repoRoot, 'client', 'binding', 'cabi', 'androidlib', 'log_android.go')) +
  read(join(repoRoot, 'client', 'binding', 'cabi', 'androidlib', 'production.go'))
forbid(androidGo, ['slog.', 'log.Printf', 'log.Println'], 'Android Go logging')
for (const required of [
  'cloudTimingPrefix      = []byte("anytty cloud connect ")',
  'cloudFailurePrefix     = []byte("anytty cloud failure ")',
  'webRTCDiagnosticPrefix = []byte("anytty webrtc ")',
  'bytes.HasPrefix(payload, cloudTimingPrefix)',
  'bytes.HasPrefix(payload, cloudFailurePrefix)',
  'bytes.HasPrefix(payload, webRTCDiagnosticPrefix)',
  'C.__android_log_write(C.ANDROID_LOG_INFO, tag, message)',
  'C.CString("AnyTTYCloud")',
]) {
  if (!androidGo.includes(required)) fail(`Android Go timing allowlist is missing ${JSON.stringify(required)}`)
}

const packageJson = JSON.parse(read(join(mobileRoot, 'package.json')))
if ('cap:sync' in (packageJson.scripts ?? {})) fail('legacy cap:sync script remains')

const forbiddenMarkers = ['ANYTTY_ANDROID_GO_TAGS', 'anytty_android_spike', 'createSpike', 'android-spike-daemon']
for (const path of trackedGradleFiles.map((tracked) => join(repoRoot, tracked))) {
  forbid(read(path), forbiddenMarkers, relative(repoRoot, path))
}
forbid(plugins + androidGo, forbiddenMarkers, 'Android production boundary')

console.log('Android source contract passed')
