export function releaseTarget(platform = process.platform, architecture = process.arch) {
  const operatingSystems = { darwin: 'darwin', linux: 'linux', win32: 'windows' }
  const architectures = { x64: 'amd64', arm64: 'arm64' }
  const os = operatingSystems[platform]
  const arch = architectures[architecture]
  if (!os || !arch) {
    throw new Error(`Unsupported platform: ${platform}/${architecture}`)
  }
  return { os, arch, extension: os === 'windows' ? '.zip' : '.tar.gz' }
}
