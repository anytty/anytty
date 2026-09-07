import { createCanvas, loadImage } from '@napi-rs/canvas';
import { readFile, writeFile, mkdir, copyFile } from 'node:fs/promises';
import { createHash } from 'node:crypto';
import { fileURLToPath } from 'node:url';
import path from 'node:path';

const root = fileURLToPath(new URL('../', import.meta.url));
const source = path.join(root, 'docs/assets/brand/motion');
const out = path.join(source, 'rigged');
const tools = process.argv[2];
if (!tools) throw new Error('Pass the authoring dependency directory.');
await mkdir(out, { recursive: true });

// Render the approved vector onto transparent pixels, without the PNG logo's white matte.
const tailSvg = await readFile(path.join(source, 'vector/tail.svg'));
const tailImage = await loadImage(Buffer.from(`<svg xmlns="http://www.w3.org/2000/svg" width="510" height="780" viewBox="0 210 170 260">${tailSvg.toString()}</svg>`));
const canvas = createCanvas(510, 780);
canvas.getContext('2d').drawImage(tailImage, 0, 0);
await writeFile(path.join(out, 'tail-skin.png'), canvas.toBuffer('image/png'));

export function boneRig(scene) {
  // Rigid facial parts retain their exact vector contours and local control nodes.
  // Only the flexible tail requires weighted skeletal deformation.
  scene.bones = [];
  return scene;
}

function skinTail(scene, parent, origin = [149, 444]) {
  const joints = [[149,444], [89,426], [54,369], [66,298], [76,247]];
  let previousAngle = 0;
  const ids = [];
  for (let i = 0; i < joints.length - 1; i++) {
    const [x,y] = joints[i], [nx,ny] = joints[i+1];
    const angle = Math.atan2(ny-y,nx-x)*180/Math.PI;
    const id = `${parent}-joint-${i}`;
    scene.bones.push({id, parent:i ? ids[i-1] : parent, x:i ? 0 : x-origin[0], y:i ? 0 : y-origin[1], length:Math.hypot(nx-x,ny-y), rotation:angle-previousAngle});
    ids.push(id); previousAngle = angle;
  }
  scene.images = [{id:'tail-skin', parent, pngPath:path.join(out,'tail-skin.png'), x:85-origin[0], y:340-origin[1], scale:1/3, mesh:{columns:10,rows:14,bones:ids}}];
  scene.shapes = scene.shapes.filter(s=>s.parent!==parent);
  return ids;
}
const track = (target, property, points) => ({target, property, keyframes:points.map(([frame,value])=>({frame,value,easing:'ease-in-out'}))});
const scene = boneRig(JSON.parse(await readFile(path.join(source,'scene.json'))));
const tailBones = skinTail(scene,'tail');
for (const a of scene.animations) {
  const original = a.tracks.find(t=>t.target==='tail'&&t.property==='rotation');
  if (!original) continue;
  const values = original.keyframes.map(k=>[k.frame,k.value]);
  // Root stays attached; the bend propagates down the chain with progressively later overlap.
  original.keyframes = original.keyframes.map(k=>({...k,value:0}));
  tailBones.forEach((id,i)=>{
    const rest = scene.bones.find(b=>b.id===id).rotation;
    const delay = i*3;
    const keys = values.map(([f,v])=>[Math.min(a.duration,f+delay),rest+v*[0,0.45,0.9,1.15][i]]);
    keys.unshift([0,rest]); keys.push([a.duration,rest]);
    const unique = [...new Map(keys.map(k=>[k[0],k])).values()].sort((a,b)=>a[0]-b[0]);
    a.tracks.push(track(id,'rotation',unique));
  });
}
await writeFile(path.join(out,'scene.json'),JSON.stringify(scene,null,2));
const jobs = [{name:'riv_create',arguments:{outPath:path.join(out,'anytty-mascot.riv'),scene,previewTime:0}}];

const browser = {artboard:{name:'AnyTTY Browser',width:1600,height:174}, groups:[
  {id:'character',x:0,y:0},
  {id:'tail',parent:'character',x:44,y:118,rotation:-105,scaleX:.2,scaleY:.2},
  {id:'body',parent:'character',x:7,y:0,scaleX:.28,scaleY:.28},
  {id:'head',parent:'character',x:77,y:64.68,scaleX:.28,scaleY:.28},
  ...['left','right'].flatMap((side,i)=>[
    {id:`eye-${side}`,parent:'head',x:i?104:-70,y:i?-132:-58},
    {id:`pupil-${side}`,parent:`eye-${side}`,x:0,y:0},
    {id:`foot-${side}`,parent:'character',x:i?96:70,y:126,rotation:i?40:-40,scaleX:i?.23:-.23,scaleY:.23},
    {id:`grip-${side}`,parent:'character',x:i?108:45,y:69,rotation:i?-90:90,scaleX:.17,scaleY:.17},
  ]),
],shapes:[],animations:[]};
async function contour(name,parent,origin,extra={}) {
  const part = JSON.parse(await readFile(path.join(source,'vector',`${name}.json`)));
  browser.shapes.push(...part.shapes.map(s=>({...s,id:`${parent}-${s.id}`,parent,x:s.x-origin[0],y:s.y-origin[1],...extra})));
}
const bodyContour = JSON.parse(await readFile(path.join(out,'body-contour.json')));
browser.shapes.push(...bodyContour.shapes.map(shape=>({...shape,parent:'body'})));
for (const side of ['left','right']) await contour('free-foot',`foot-${side}`,[26,8]);
await contour('head','head',[250,231]);
for (const [side,origin] of [['left',[180,173]],['right',[354,99]]]) {
  await contour(`eye-${side}`,`eye-${side}`,origin);
  await contour(`pupil-${side}`,`pupil-${side}`,origin,{clipBy:`eye-${side}-eye-${side}-p0`});
}
await contour('nostrils','head',[250,231]);
for(const [side,x] of [['left',181],['right',355]]) await contour(`hand-${side}`,`grip-${side}`,[x,341]);
boneRig(browser);
const browserTail = skinTail(browser,'tail');
const idle={name:'idle',fps:60,duration:840,loop:'loop',tracks:[]};
for(let i=0;i<browserTail.length;i++) {
  const id=browserTail[i],rest=browser.bones.find(b=>b.id===id).rotation;
  const amplitude=[0,7,14,19][i],delay=i*5;
  idle.tracks.push(track(id,'rotation',[[0,rest],[40+delay,rest],[75+delay,rest+amplitude],[110+delay,rest-amplitude*.6],[150+delay,rest+amplitude*.22],[200+delay,rest],[350+delay,rest],[393+delay,rest-amplitude*.7],[441+delay,rest+amplitude*.4],[510+delay,rest],[840,rest]]));
}
for(const [side,offset] of [['left',0],['right',1]]) {
  idle.tracks.push(track(`eye-${side}`,'scaleY',[[0,1],...[140,360,678].flatMap(f=>[[f+offset,1],[f+5+offset,.035],[f+8+offset,.035],[f+17+offset,1]]),[840,1]]));
  idle.tracks.push(track(`foot-${side}`,'rotation',[[0,side==='left'?-40:40],[250,side==='left'?-40:40],[265,side==='left'?-45:43],[290,side==='left'?-40:40],[840,side==='left'?-40:40]]));
}
browser.animations.push(idle);
for(const axis of ['x','y']) {
  browser.animations.push({name:`gaze-${axis}`,fps:60,duration:60,loop:'oneshot',tracks:['left','right'].map(side=>track(`pupil-${side}`,axis,[[0,-12],[30,0],[60,12]]))});
}
browser.animations.push({name:'head-follow',fps:60,duration:60,loop:'oneshot',tracks:[track('head','rotation',[[0,-3],[30,0],[60,3]])]});
// Scrubbed only on layout changes to keep the mascot anchored at the field's right edge.
browser.animations.push({name:'placement',fps:60,duration:60,loop:'oneshot',tracks:[{target:'character',property:'x',keyframes:[{frame:0,value:0},{frame:60,value:1460,easing:'linear'}]}]});
browser.stateMachine={name:'Browser',inputs:[],states:[{name:'idle',animation:'idle'}],transitions:[{from:'entry',to:'idle'}]};
for (const layer of ['back','front']) {
  const layerScene=structuredClone(browser);
  layerScene.shapes=layerScene.shapes.filter(s=>s.parent.startsWith('grip-')===(layer==='front'));
  if(layer==='front') layerScene.images=[];
  await writeFile(path.join(out,`browser-${layer}.json`),JSON.stringify(layerScene,null,2));
  jobs.push({name:'riv_create',arguments:{outPath:path.join(out,`browser-${layer}.riv`),scene:layerScene,previewTime:0}});
}
await writeFile(path.join(out,'create-jobs.json'),JSON.stringify(jobs,null,2));
await copyFile(path.join(source,'scenarios.json'),path.join(out,'scenarios.json'));
await writeFile(path.join(out,'source.json'),JSON.stringify({revision:4,status:'review-sample',source:'../scene.json',sourceSha256:createHash('sha256').update(await readFile(path.join(source,'scene.json'))).digest('hex'),rig:'Four-bone, distance-weighted tail mesh; independent Rive control nodes for rigid facial parts and limbs',tailTexture:'3x transparent rasterization of the existing traced vector; not the white-matted logo PNG',browser:'Fixed right-edge perch; original opaque body contour behind the input; gaze only follows caret/pointer, without horizontal travel or stepping',bodyContourSha256:createHash('sha256').update(await readFile(path.join(out,'body-contour.json'))).digest('hex'),appIntegrated:false},null,2));
console.log(path.join(out,'create-jobs.json'));
