import { build } from 'esbuild';
import { mkdir, writeFile, copyFile } from 'node:fs/promises';
import { dirname, join } from 'node:path';
import { fileURLToPath } from 'node:url';

const root = join(dirname(fileURLToPath(import.meta.url)), '..');
const output = join(root, 'clients/flutter/assets/file-viewer');
const result = await build({
  absWorkingDir: root,
  entryPoints: ['clients/flutter/file-viewer/main.tsx'],
  bundle: true,
  write: false,
  format: 'iife',
  platform: 'browser',
  target: ['chrome110', 'safari16'],
  minify: true,
  legalComments: 'inline',
  define: { 'process.env.NODE_ENV': '"production"' },
});
const script = result.outputFiles[0].text;
// No networking or native bridge is exposed to documents. The nonce is used
// only by bundled application code; untrusted HTML lives in a sandboxed frame.
const html = `<!doctype html><html><head><meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<meta http-equiv="Content-Security-Policy" content="default-src 'none'; script-src 'nonce-__ANYTTY_NONCE__'; style-src 'unsafe-inline'; img-src data: blob:; font-src data:; connect-src 'none'; frame-src about:; worker-src blob:; base-uri 'none'; form-action 'none'">
<title>AnyTTY file preview</title><style>html,body,#root{margin:0;width:100%;height:100%;overflow:hidden}body{font-family:system-ui,sans-serif}iframe{display:block;width:100%;height:100%;border:0;background:white}
*{box-sizing:border-box;letter-spacing:0}.document{height:100%;display:flex;flex-direction:column;--surface:#f6f8fa;--text:#182a30;--border:#cbd8de;background:var(--surface);color:var(--text)}.document[data-theme=dark]{--surface:#101416;--text:#f4f8f7;--border:#364248}.tools,.search{display:flex;align-items:center;min-height:48px;flex-shrink:0;border-bottom:1px solid var(--border);padding:0 8px;gap:4px}.tool-space{flex:1}.zoom{min-width:48px;text-align:center;font-size:13px}.document button{display:grid;place-items:center;flex-shrink:0;width:48px;height:48px;padding:0;border:0;border-radius:4px;background:transparent;color:inherit;cursor:pointer}.document button:disabled{opacity:.4;cursor:default}.document button:active:not(:disabled){background:#20b89b22}.document button:focus-visible,.search input:focus-visible{outline:2px solid #087e9c;outline-offset:-2px}.search{flex-wrap:wrap;padding:8px}.search input{flex:1;min-width:120px;height:48px;border:1px solid var(--border);border-radius:4px;background:var(--surface);color:var(--text);font:16px system-ui;padding:0 8px}.search output{font-size:13px}.search button{width:44px}
</style>
</head><body><div id="root"></div><script id="preview-data" type="application/json">__ANYTTY_PREVIEW_DATA__</script><script nonce="__ANYTTY_NONCE__">${script}</script></body></html>`;
await mkdir(output, { recursive: true });
await writeFile(join(output, 'viewer.html'), html);
await copyFile(join(root, 'clients/web/public/third-party/NPM_NOTICES.txt'), join(output, 'NPM_NOTICES.txt'));
console.log(`Built offline mobile file viewer: ${(Buffer.byteLength(html) / 1024 / 1024).toFixed(2)} MB`);
