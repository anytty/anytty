import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import { fileViewerRenderers } from '@file-viewer/vite-plugin'

export default defineConfig(({ command, mode }) => ({
  plugins: [react(), command === 'build' && mode !== 'test' && fileViewerRenderers({ formats: ['md', 'markdown', 'txt', 'json', 'diff', 'patch', 'pdf', 'doc', 'docx', 'rtf', 'odt', 'xlsx', 'xls', 'csv', 'tsv', 'ods', 'mp4', 'webm', 'm3u8', 'glb', 'gltf', 'obj', 'stl', 'ply', 'fbx', 'dae', '3ds', '3mf', 'amf', 'pcd', 'wrl', 'vrml', 'xyz', 'vtk'], inject: false, copyAssets: true, chunkStrategy: 'none' })],
  build: {
    outDir: 'dist',
    emptyOutDir: true,
  },
}))
