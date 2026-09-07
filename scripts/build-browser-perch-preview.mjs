import { readFile, writeFile, copyFile, access } from 'node:fs/promises';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const root = fileURLToPath(new URL('../', import.meta.url));
const authoring = process.argv[2];
if (!authoring) throw new Error('Pass the authoring directory containing lucide-static.');
const out = path.join(root, 'artifacts/browser-perch');
await access(path.join(out, 'perch-sprite.png'));
let html = await readFile(path.join(root, 'scripts/templates/browser-perch.html'), 'utf8');
for (const [token, icon] of Object.entries({ REPLAY: 'rotate-ccw', SEARCH: 'search', ARROW: 'arrow-right' })) {
  html = html.replace(`<!-- ICON_${token} -->`, await readFile(path.join(authoring, 'node_modules/lucide-static/icons', `${icon}.svg`), 'utf8'));
}
await copyFile(path.join(authoring, 'node_modules/lucide-static/LICENSE'), path.join(out, 'LUCIDE-LICENSE'));
await writeFile(path.join(out, 'index.html'), html);
console.log(path.join(out, 'index.html'));
