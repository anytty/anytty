import assert from 'node:assert/strict';
import {createRequire} from 'node:module';
import {mkdir} from 'node:fs/promises';
import {dirname,join} from 'node:path';
import {fileURLToPath,pathToFileURL} from 'node:url';
const require=createRequire(import.meta.url);
const {chromium}=require(process.env.PLAYWRIGHT_MODULE||'playwright');
const root=join(dirname(fileURLToPath(import.meta.url)),'..');
const output=join(root,'artifacts/store-screenshots/2026-09-09/google-play');
await mkdir(output,{recursive:true});
const browser=await chromium.launch({channel:'chrome',args:['--allow-file-access-from-files']});
try {
 const page=await browser.newPage({viewport:{width:1920,height:1080},deviceScaleFactor:1});
 for(const [index,slide] of ['devices','terminals','claude','petal'].entries()) {
  await page.goto(`${pathToFileURL(join(root,'docs/assets/brand/store/phone-screenshot.html'))}?slide=${slide}`);
  await page.evaluate(async()=>{await document.fonts.ready;await Promise.all([...document.images].map(image=>image.decode()));});
  const errors=await page.evaluate(()=>[...document.querySelectorAll('section,.brand,.phone,.note')].filter(el=>{const r=el.getBoundingClientRect();return r.right>1920||r.bottom>1080||(!el.classList.contains('phone')&&el.scrollWidth>el.clientWidth+1);}).map(el=>el.className||el.tagName));
  assert.deepEqual(errors,[]);
  const path=join(output,`0${index+1}-${slide}-en.png`);await page.screenshot({path});console.log(path);
 }
} finally {await browser.close();}
