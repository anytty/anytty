import {createRequire} from 'node:module';
import {fileURLToPath} from 'node:url';
import {readFile,writeFile} from 'node:fs/promises';
import path from 'node:path';
const tools=process.argv[2];
if(!tools)throw new Error('Pass the authoring dependency directory.');
const require=createRequire(path.resolve(tools,'package.json'));
const {Client}=require('@modelcontextprotocol/sdk/client/index.js');
const {StdioClientTransport}=require('@modelcontextprotocol/sdk/client/stdio.js');
const client=new Client({name:'anytty-rig-finalize',version:'1'});
const out=fileURLToPath(new URL('../docs/assets/brand/motion/rigged/',import.meta.url));
await client.connect(new StdioClientTransport({command:path.resolve(tools,'node_modules/.bin/rive-mcp')}));
const reports=[];
function varuint(value){const bytes=[];do{const byte=value&127;value>>>=7;bytes.push(byte|(value?128:0));}while(value);return Buffer.from(bytes);}
async function call(name,args){const r=await client.callTool({name,arguments:args});if(r.isError)throw new Error(JSON.stringify(r.content));return r.content.filter(c=>c.type==='text').map(c=>c.text).join('\n');}
try {
 for(const filename of ['anytty-mascot.riv','browser-back.riv']) {
  const rivPath=path.join(out,filename);
  const dump=JSON.parse(await call('riv_dump',{path:rivPath,full:true}));
  const edits=[];
  let file=await readFile(rivPath);
  for(const mesh of dump.objects.filter(o=>o.typeName==='Mesh')) {
   const raw=Buffer.from(mesh.raw.find(p=>p.key===223).value.data);
   // Official Mesh::decodeTriangleIndexBytes reads varuint indices, not uint16 pairs.
   // Normalize only our fresh authoring output; never rewrite unrelated Rive assets.
   const vertices=dump.objects.filter(o=>o.typeName==='MeshVertex');
   const indices=[];
   for(let i=0;i<raw.length;i+=2){const value=raw.readUInt16LE(i);if(value>=vertices.length)throw new Error('Unexpected mesh encoding; rebuild before finalizing.');indices.push(value);}
   const bytes=Buffer.concat(indices.map(varuint));
   const needle=Buffer.concat([varuint(223),varuint(raw.length),raw]);
   const at=file.indexOf(needle);
   if(at<0||file.indexOf(needle,at+1)>=0)throw new Error('Mesh field is not uniquely identifiable.');
   file=Buffer.concat([file.subarray(0,at),varuint(223),varuint(bytes.length),bytes,file.subarray(at+needle.length)]);
  }
  const artboardIndex=dump.objects.find(o=>o.typeName==='Artboard').index;
  for(const weight of dump.objects.filter(o=>o.typeName==='Weight')) {
   const vertex=dump.objects[weight.properties.parentId+artboardIndex];
   const x=vertex.properties.x/3+85,y=vertex.properties.y/3+340;
   if(x>85&&y>400)edits.push({op:'set',index:weight.index,set:{indices:1,values:255}});
  }
  await writeFile(rivPath,file);
  console.log(await call('riv_edit',{path:rivPath,edits}));
  const lint=await call('riv_lint',{path:rivPath});
  reports.push({filename,types:dump.typeCounts,edits:edits.length,lint});
 }
 await writeFile(path.join(out,'rig-validation.json'),JSON.stringify(reports,null,2));
}finally{await client.close();}
