export const modelMimeTypesByExtension: Record<string, string> = {
  '3ds': 'application/x-3ds',
  '3mf': 'model/3mf',
  amf: 'application/x-amf',
  dae: 'model/vnd.collada+xml',
  fbx: 'application/vnd.autodesk.fbx',
  glb: 'model/gltf-binary',
  gltf: 'model/gltf+json',
  obj: 'model/obj',
  pcd: 'application/vnd.pointcloud',
  ply: 'model/ply',
  stl: 'model/stl',
  vox: 'application/x-vox',
  vrml: 'model/vrml',
  vtk: 'model/vtk',
  wrl: 'model/vrml',
  xyz: 'chemical/x-xyz',
}

const modelMimeTypes = new Set(Object.values(modelMimeTypesByExtension))

export function isModelPreviewFile(name: string, mimeType: string): boolean {
  const normalizedMime = mimeType.trim().toLowerCase()
  return extension(name) in modelMimeTypesByExtension ||
    normalizedMime.startsWith('model/') ||
    modelMimeTypes.has(normalizedMime) ||
    normalizedMime === 'application/sla'
}

export function modelPreviewMimeType(name: string, mimeType: string): string {
  const normalizedMime = mimeType.trim()
  if (normalizedMime) return normalizedMime
  return modelMimeTypesByExtension[extension(name)] ?? 'application/octet-stream'
}

export function modelPreviewFormatLabel(name: string): string {
  const ext = extension(name)
  return ext ? `${ext.toUpperCase()} Model` : '3D Model'
}

export function extension(name: string): string {
  const base = basename(name).toLowerCase()
  const dot = base.lastIndexOf('.')
  return dot >= 0 ? base.slice(dot + 1) : base
}

function basename(path: string): string {
  const normalized = path.replace(/\/+$/, '')
  const index = normalized.lastIndexOf('/')
  return index >= 0 ? normalized.slice(index + 1) : normalized
}
