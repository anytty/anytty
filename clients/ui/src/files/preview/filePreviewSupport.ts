const supportedPreviewExtensions = new Set([
  '3dm', '3ds', '3mf', 'aac', 'amf', 'avif', 'bash', 'bdl', 'bmp', 'brep', 'bundle',
  'c', 'cc', 'cjs', 'cpp', 'cs', 'css', 'csv', 'dae', 'diff', 'doc', 'docm', 'docx',
  'dot', 'dotm', 'dotx', 'fbx', 'flac', 'fods', 'gif', 'glb', 'gltf', 'go', 'gv',
  'h', 'hcl', 'heic', 'heif', 'hpp', 'htm', 'html', 'http', 'ico', 'ifc', 'iges',
  'igs', 'ini', 'ipynb', 'java', 'jpeg', 'jpg', 'js', 'json', 'json5', 'jsonc',
  'jxl', 'jsx', 'kmz', 'kt', 'log', 'm3u8', 'm4a', 'markdown', 'md', 'mid', 'midi',
  'mjs', 'mp3', 'mp4', 'mpeg', 'numbers', 'obj', 'odp', 'ods', 'odt', 'oga', 'ogg',
  'opus', 'patch', 'pcd', 'pdf', 'php', 'ply', 'png', 'proto', 'py', 'rb', 'react',
  'rs', 'rtf', 'sh', 'sql', 'step', 'stl', 'stp', 'svg', 'swift', 'tex', 'tif',
  'tiff', 'toml', 'ts', 'tsv', 'tsx', 'txt', 'usd', 'usda', 'usdc', 'usdz', 'vrml',
  'vtk', 'vtp', 'vue', 'wav', 'weba', 'webm', 'webp', 'wrl', 'xls', 'xlsb', 'xlsm',
  'xlsx', 'xlt', 'xltm', 'xltx', 'xml', 'xyz', 'yaml', 'yml',
])

export function isKnownUnsupportedPreviewPath(path: string): boolean {
  const name = path.slice(path.lastIndexOf('/') + 1)
  const dot = name.lastIndexOf('.')
  if (dot < 1 || dot === name.length - 1) return false
  return !supportedPreviewExtensions.has(name.slice(dot + 1).toLowerCase())
}
