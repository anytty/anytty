import assert from 'node:assert/strict';
import {createRequire} from 'node:module';
import {readFile,writeFile,mkdir} from 'node:fs/promises';
import {fileURLToPath,pathToFileURL} from 'node:url';
import path from 'node:path';
const tools=process.argv[2];
if(!tools)throw new Error('Pass the authoring dependency directory.');
const {chromium}=createRequire(path.resolve(tools,'package.json'))('playwright-core');
const out=fileURLToPath(new URL('../docs/assets/brand/motion/rigged/',import.meta.url));
const artifacts=fileURLToPath(new URL('../artifacts/mascot-rigged/',import.meta.url));
await mkdir(artifacts,{recursive:true});
const validation=JSON.parse(await readFile(path.join(out,'rig-validation.json')));
for(const report of validation){assert.equal(report.types.RootBone,1);assert.equal(report.types.Bone,3);assert.equal(report.types.Skin,1);assert.equal(report.types.Weight,165);}
const browser=await chromium.launch({channel:'chrome',headless:true});
const errors=[],requests=[],results=[];
async function pixels(page) {
 return page.evaluate(()=>{
  const source=document.querySelector('#rear'),canvas=document.createElement('canvas');canvas.width=source.width;canvas.height=source.height;
  const ctx=canvas.getContext('2d');ctx.drawImage(source,0,0);
  const {data}=ctx.getImageData(0,0,canvas.width,canvas.height),dpr=devicePixelRatio,pos=window.anyttyRigState.position;
  const alpha=(x,y)=>data[(Math.floor(y*dpr)*canvas.width+Math.floor(x*dpr))*4+3];
  let rootColumns=0,tailPixels=0,whiteTail=0,checksum=0,bodyPixels=0,neckPixels=0,bellyPixels=0;
  for(let x=pos+40;x<pos+120;x++)for(let y=55;y<140;y++){
   const i=(Math.floor(y*dpr)*canvas.width+Math.floor(x*dpr))*4;
   if(data[i+3]>240&&data[i]<20&&data[i+1]<20&&data[i+2]<20){bodyPixels++;if(y>=65&&y<70)neckPixels++;if(y>=128)bellyPixels++;}
  }
  for(let x=pos;x<pos+60;x++){
   if(Array.from({length:9},(_,i)=>alpha(x,124+i)>200).every(Boolean))rootColumns++;
   for(let y=129;y<174;y++){
    const index=(Math.floor(y*dpr)*canvas.width+Math.floor(x*dpr))*4;
    if(data[index+3]>100){tailPixels++;checksum=(checksum+((x-pos)*71+y*127)*data[index+3])%1000000007;if(data[index]>190&&data[index+1]>190&&data[index+2]>190)whiteTail++;}
   }
  }
  return {rootColumns,tailPixels,whiteTail,checksum,bodyPixels,neckPixels,bellyPixels};
 });
}
try {
 for(const viewport of [{width:375,height:900},{width:1280,height:900},{width:844,height:390}]) {
  const page=await browser.newPage({viewport,deviceScaleFactor:2});
  page.on('pageerror',e=>errors.push(e.message));page.on('console',m=>{if(m.type()==='error')errors.push(m.text());});page.on('request',r=>{if(/^https?:/.test(r.url()))requests.push(r.url());});
  await page.goto(pathToFileURL(path.join(out,'index.html')).href);
  await page.waitForFunction(()=>window.anyttyRigState?.ready);
  await page.locator('.scene').scrollIntoViewIfNeeded();
  await page.locator('#motion-toggle').click();
  const samples=[];
  for(const t of [0,.75,1.3,1.8,2.4,3.2,4.2,6.8,7.5,8.3,10,13.9]) {
   await page.evaluate(t=>window.anyttyRigPreview.seek(t),t);
   const p=await pixels(page);assert(p.rootColumns>=6,`Tail contact ${viewport.width}/${t}: ${JSON.stringify(p)}`);assert(p.tailPixels>180,'Tail must remain visible');assert.equal(p.whiteTail,0,'No white matte on tail');samples.push(p.checksum);
   assert(p.bodyPixels>1200,`Body must be opaque: ${JSON.stringify(p)}`);
   assert(p.neckPixels>15&&p.bellyPixels>30,`Neck and belly must be visible: ${JSON.stringify(p)}`);
  }
  assert(new Set(samples).size>=5,'Tail must deform across the cycle');
  assert(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth),'Horizontal overflow');
  for(const dark of [false,true]){await page.locator('#dark').setChecked(dark);await page.locator('.scene').scrollIntoViewIfNeeded();await page.screenshot({path:path.join(artifacts,`${viewport.width}-${dark?'dark':'light'}.png`)});}
  await page.locator('#motion-toggle').click();
  const anchored=await page.evaluate(()=>window.anyttyRigState.position);
  await page.locator('#query').fill('a');await page.waitForTimeout(850);
  const first=await page.evaluate(()=>window.anyttyRigState.position);
  await page.locator('#query').fill('https://anytty.com/session');await page.waitForTimeout(850);
  const second=await page.evaluate(()=>window.anyttyRigState.position);assert.equal(first,anchored,'Focus must not move mascot');assert.equal(second,anchored,'Typing must not move mascot');
  await page.locator('#query').blur();await page.waitForTimeout(200);assert.equal(await page.evaluate(()=>window.anyttyRigState.position),anchored,'Blur must not move mascot');
  await page.locator('#motion-toggle').click();
  const stopped=await page.evaluate(()=>window.anyttyRigState.time);await page.waitForTimeout(200);assert.equal(await page.evaluate(()=>window.anyttyRigState.time),stopped);
  await page.emulateMedia({reducedMotion:'reduce'});await page.waitForTimeout(100);
  const reduced=await page.evaluate(()=>window.anyttyRigState.time);await page.waitForTimeout(200);assert.equal(await page.evaluate(()=>window.anyttyRigState.time),reduced);
  results.push({viewport,sampledFrames:samples.length,tailContact:'pass',opaqueBody:'pass',fixedPerch:'pass',pause:'pass',reducedMotion:'pass'});
  await page.close();
 }
 const page=await browser.newPage({viewport:{width:1280,height:900}});
 page.on('pageerror',e=>errors.push(e.message));
 await page.goto(pathToFileURL(path.join(out,'scenarios.html')).href);
 await page.waitForTimeout(600);
 const tabs=page.locator('[role=tab]');assert.equal(await tabs.count(),9);
 for(let i=0;i<9;i++){await tabs.nth(i).click();await page.waitForTimeout(200);await page.screenshot({path:path.join(artifacts,`scene-${i}.png`)});}
 await page.close();
 assert.deepEqual(errors,[]);assert.deepEqual(requests,[]);
 await writeFile(path.join(artifacts,'report.json'),JSON.stringify({results,sceneCount:9,errors,networkRequests:requests},null,2));
 console.log('PASS: real bone/skin structure; 36 tail-contact and opaque-body frames; desktop/mobile/landscape; fixed perch during focus/typing/blur; pause; reduced motion; nine scenes; offline.');
}finally{await browser.close();}
