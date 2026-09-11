package agents

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

// Inject deterministic OS outcomes and a manually advanced retry clock, while
// executing the production adapter's send/drain/event functions in Node. This
// verifies backoff and bounds without assuming CI process scheduling speed.
func TestOpenCodeRetryRecoveryBoundsAndNewState(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("node unavailable")
	}
	dir := t.TempDir()
	faults := `import { EventEmitter } from "node:events";
const fault=globalThis.__anyttyFault;
const Date={now:()=>fault.now};
const setTimeout=(run,delay)=>{const item={run,delay,cancelled:false,unref(){}};fault.timers.push(item);return item;};
const clearTimeout=item=>{if(item)item.cancelled=true;};
const spawn=(command,args,options)=>{
 const child=new EventEmitter();
 child.stdin={on(){},end(data){fault.reports.push(JSON.parse(data));}};
 fault.options.push(options);
 fault.completions.push(()=>{const outcome=fault.outcomes.shift()??7;if(outcome==='EAGAIN')child.emit('error',Object.assign(new Error('temporarily unavailable'),{code:'EAGAIN'}));child.emit('close',outcome==='EAGAIN'?null:outcome);});
 return child;
};`
	source := bytes.Replace(openCodeSource, []byte(`import { spawn } from "node:child_process";`), []byte(faults), 1)
	path := filepath.Join(dir, "fault-plugin.mjs")
	if err = os.WriteFile(path, source, 0600); err != nil {
		t.Fatal(err)
	}
	url, _ := json.Marshal("file://" + path)
	runner := `import assert from 'node:assert/strict';
async function scenario(name,outcomes,newer=false){
 const f=globalThis.__anyttyFault={now:1000,reports:[],options:[],timers:[],completions:[],outcomes:[...outcomes]};
 const {AnyTTYAgents}=await import(` + string(url) + `+'?'+name);
 const plugin=await AnyTTYAgents({directory:'/fixture'});
 await plugin.event({event:{type:'session.created',properties:{info:{id:'s'}}}});
 assert.equal(f.reports.length,1);
 assert.equal(f.completions.length,1,'event callback must resolve before reporter completion');
 if(newer)await plugin.event({event:{type:'session.status',properties:{sessionID:'s',status:{type:'busy'}}}});
 for(let turn=0;turn<30;turn++){
  if(f.completions.length)f.completions.shift()();
  for(let i=0;i<8;i++)await Promise.resolve();
  const timer=f.timers.find(t=>!t.cancelled);
  if(timer){timer.cancelled=true;f.now+=timer.delay;timer.run();}
  for(let i=0;i<8;i++)await Promise.resolve();
 }
 assert.ok(f.options.every(o=>o.timeout===1500&&o.killSignal==='SIGKILL'));
 assert.equal(f.completions.length,0);
 assert.equal(f.timers.filter(t=>!t.cancelled).length,0,'retry loop must terminate');
 return f;
}
const recovered=await scenario('recover',['EAGAIN',7,0]);
assert.deepEqual(recovered.reports.map(r=>r.sequence),[1,1,1]);
assert.equal(recovered.now,1300,'retry backoff changed');
const failed=await scenario('bounded',[7,7,7,7,7]);
assert.equal(failed.reports.length,3,'at most three attempts per complete snapshot');
const replaced=await scenario('newer',[7,0],true);
assert.deepEqual(replaced.reports.map(r=>[r.sequence,r.status]),[[1,'idle'],[2,'working']],'new snapshot must replace old retry');
`
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, node, "--input-type=module", "-e", runner)
	cmd.Env = append(os.Environ(), "ANYTTY_TERMINAL_ID=fixture", "ANYTTY_DAEMON_SOCKET=/fixture.sock")
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("retry adapter: %v %s", err, output)
	}
}
