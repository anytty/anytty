import {readFile,writeFile,copyFile,mkdir} from 'node:fs/promises';
import {fileURLToPath} from 'node:url';
import path from 'node:path';
const root=fileURLToPath(new URL('../',import.meta.url));
const tools=process.argv[2];
if(!tools) throw new Error('Pass the authoring dependency directory.');
const out=path.join(root,'docs/assets/brand/motion/rigged');
const runtime=path.join(tools,'node_modules/@rive-app/webgl2');
await mkdir(path.join(out,'runtime'),{recursive:true});
await copyFile(path.join(runtime,'rive.js'),path.join(out,'runtime/rive.js'));
await copyFile(path.join(tools,'RIVE-LICENSE'),path.join(out,'runtime/RIVE-LICENSE'));
await copyFile(path.join(tools,'node_modules/lucide-static/LICENSE'),path.join(out,'runtime/LUCIDE-LICENSE'));
const data={runtimeVersion:'2.42.0',wasm:(await readFile(path.join(runtime,'rive.wasm'))).toString('base64'),scenarios:JSON.parse(await readFile(path.join(out,'scenarios.json')))};
for(const [key,file]of Object.entries({riv:'anytty-mascot.riv',back:'browser-back.riv',front:'browser-front.riv'})) data[key]=(await readFile(path.join(out,file))).toString('base64');
await writeFile(path.join(out,'preview-data.js'),`window.ANYTTY_MOTION=${JSON.stringify(data)};\n`);
for(const [template,file]of [['mascot-motion.html','scenarios.html'],['browser-hanging.html','index.html']]) {
  let html=await readFile(path.join(root,'scripts/templates',template),'utf8');
  for(const [key,name] of Object.entries({PLAY:'play',PAUSE:'pause',REPLAY:'rotate-ccw',DOWNLOAD:'download',SEARCH:'search',ARROW:'arrow-right'})) html=html.replace(`<!-- ICON_${key} -->`,await readFile(path.join(tools,'node_modules/lucide-static/icons',`${name}.svg`),'utf8'));
  html=html.replaceAll('../mascot.png','../../mascot.png').replace('src="original.png"','src="../../mascot.png"').replaceAll('Sample 03','Sample 04');
  if(file==='index.html') html=html.replace('<!-- MASCOT_REAR -->','<canvas id="rear" aria-hidden="true"></canvas>').replace('<!-- MASCOT_HANDS -->','<canvas id="front" aria-hidden="true"></canvas>')
    .replace('<script src="motion.js"></script>','<script src="runtime/rive.js"></script><script src="preview-data.js"></script><script src="browser.js"></script>')
    .replace('</style>','.mascot{right:0!important;width:100%;height:174px}.mascot canvas{display:block;width:100%;height:100%}.review-link{color:var(--accent);font-size:14px;display:inline-flex;align-items:center;min-height:44px}</style>')
    .replace('<div class="controls">','<div class="controls"><a class="review-link" href="scenarios.html">场景动作</a>');
  else html=html.replace('</header>','<a href="index.html">浏览器悬挂</a></header>');
  await writeFile(path.join(out,file),html);
}
await copyFile(path.join(root,'scripts/templates/browser-rive.js'),path.join(out,'browser.js'));
console.log(path.join(out,'index.html'));
