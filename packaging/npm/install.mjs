import { spawnSync } from 'node:child_process'
import { createHash } from 'node:crypto'
import { chmod, copyFile, mkdir, mkdtemp, readFile, rm, writeFile } from 'node:fs/promises'
import { tmpdir } from 'node:os'
import { dirname, join } from 'node:path'
import { fileURLToPath } from 'node:url'
import { releaseTarget } from './platform.mjs'

const packageRoot = dirname(fileURLToPath(import.meta.url))
const packageJson = JSON.parse(await readFile(join(packageRoot, 'package.json'), 'utf8'))
const tag = `v${packageJson.version}`
const { os, arch, extension } = releaseTarget()
const archiveBase = `anytty-${tag}-${os}-${arch}`
const archiveName = `${archiveBase}${extension}`
const releaseBase = (process.env.ANYTTY_RELEASE_BASE_URL || `https://github.com/anytty/anytty/releases/download/${tag}`).replace(/\/$/, '')
const workDir = await mkdtemp(join(tmpdir(), 'anytty-npm-'))

async function download(url) {
  const response = await fetch(url, { redirect: 'follow' })
  if (!response.ok) throw new Error(`Download failed (${response.status}): ${url}`)
  return Buffer.from(await response.arrayBuffer())
}

try {
  const [archive, checksumFile] = await Promise.all([
    download(`${releaseBase}/${archiveName}`),
    download(`${releaseBase}/SHA256SUMS`),
  ])
  const checksumLine = checksumFile.toString('utf8').split(/\r?\n/).find((line) => line.trim().endsWith(` ${archiveName}`))
  if (!checksumLine) throw new Error(`Checksum not found for ${archiveName}`)
  const expected = checksumLine.trim().split(/\s+/)[0].toLowerCase()
  const actual = createHash('sha256').update(archive).digest('hex')
  if (actual !== expected) throw new Error(`Checksum verification failed for ${archiveName}`)

  const archivePath = join(workDir, archiveName)
  await writeFile(archivePath, archive)
  const extraction = os === 'windows'
    ? spawnSync('powershell.exe', ['-NoProfile', '-NonInteractive', '-Command', 'Expand-Archive -LiteralPath $args[0] -DestinationPath $args[1] -Force', archivePath, workDir], { stdio: 'inherit' })
    : spawnSync('tar', ['-xzf', archivePath, '-C', workDir], { stdio: 'inherit' })
  if (extraction.error) throw extraction.error
  if (extraction.status !== 0) throw new Error(`Archive extraction failed with exit code ${extraction.status}`)

  const executableName = os === 'windows' ? 'anytty.exe' : 'anytty'
  const source = join(workDir, archiveBase, executableName)
  const destination = join(packageRoot, 'vendor', executableName)
  await mkdir(dirname(destination), { recursive: true })
  await copyFile(source, destination)
  if (os !== 'windows') await chmod(destination, 0o755)
  console.log(`Installed AnyTTY ${packageJson.version} for ${os}/${arch}`)
} finally {
  await rm(workDir, { recursive: true, force: true })
}
