package agents

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestOpenCodeExecutablePluginCallback(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("node unavailable")
	}
	dir := t.TempDir()
	log := filepath.Join(dir, "reports.jsonl")
	logJSON, _ := json.Marshal(log)
	receiver := filepath.Join(dir, "receiver")
	// Use a lightweight real executable to isolate adapter IO from receiver VM startup.
	quotedLog := "'" + strings.ReplaceAll(log, "'", "'\"'\"'") + "'"
	code := "#!/bin/sh\ncat >> " + quotedLog + "\nprintf '\\n' >> " + quotedLog + "\n"
	if err = os.WriteFile(log, nil, 0600); err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(receiver, []byte(code), 0700); err != nil {
		t.Fatal(err)
	}
	exeJSON, _ := json.Marshal(receiver)
	source := bytes.Replace(openCodeSource, []byte(`"__ANYTTY_EXECUTABLE__"`), exeJSON, 1)
	// Observe the real OS process without changing production timeout behavior.
	instrumentation := `import { spawn as nativeSpawn } from "node:child_process";
 const spawn=(command,args,options)=>{
  const started=Date.now();
  console.error(JSON.stringify({stage:"spawn-request",command,args,timeout:options.timeout}));
  const child=nativeSpawn(command,args,{...options,stdio:["pipe","ignore","pipe"]});
  child.on("spawn",()=>console.error(JSON.stringify({stage:"spawned",pid:child.pid,elapsed:Date.now()-started})));
  child.on("error",error=>console.error(JSON.stringify({stage:"spawn-error",code:error.code,message:error.message,elapsed:Date.now()-started})));
  child.on("exit",(code,signal)=>console.error(JSON.stringify({stage:"exit",code,signal,elapsed:Date.now()-started})));
  child.on("close",(code,signal)=>console.error(JSON.stringify({stage:"close",code,signal,elapsed:Date.now()-started})));
  child.stderr.on("data",data=>console.error("receiver stderr: "+data.toString()));
  child.stdin.on("error",error=>console.error(JSON.stringify({stage:"stdin-error",code:error.code,message:error.message})));
  return child;
 };`
	source = bytes.Replace(source, []byte(`import { spawn } from "node:child_process";`), []byte(instrumentation), 1)

	plugin := filepath.Join(dir, "plugin.mjs")
	if err = os.WriteFile(plugin, source, 0600); err != nil {
		t.Fatal(err)
	}
	pluginJSON, _ := json.Marshal("file://" + plugin)
	runner := `import {readFile} from 'node:fs/promises';
import {watch} from 'node:fs';
const {AnyTTYAgents}=await import(` + string(pluginJSON) + `);
const plugin=await AnyTTYAgents({directory:'/project'});
const events=[
 {type:'session.created',properties:{info:{id:'s',title:'Review'}}},
 {type:'session.status',properties:{sessionID:'s',status:{type:'busy'}}},
 {type:'permission.asked',properties:{sessionID:'s',id:'p1'}},
 {type:'permission.asked',properties:{sessionID:'s',id:'p2'}},
 {type:'permission.replied',properties:{sessionID:'s',requestID:'p1'}},
 {type:'permission.replied',properties:{sessionID:'s',requestID:'p2'}},
 {type:'session.deleted',properties:{info:{id:'s'}}},
];
const logPath = ` + string(logJSON) + `;
const waitForSequence = (expected) => new Promise((resolve, reject) => {
 let checking = false, dirty = false;
 const watcher = watch(logPath, () => { void check(); });
 const timer = setTimeout(async () => {
  watcher.close();
  reject(new Error('report sequence ' + expected + ' missing; captured=' + await readFile(logPath,'utf8')));
 }, 5000);
 async function check() {
  if (checking) { dirty = true; return; }
  checking = true;
  try {
   do { dirty = false;
   const reports = (await readFile(logPath,'utf8')).trim().split('\n').filter(Boolean).map(line => JSON.parse(line));
   if (reports.some(report => report.sequence === expected)) { clearTimeout(timer); watcher.close(); resolve(); }
   } while (dirty);
  } catch {} finally { checking = false; }
 }
 void check();
});
for(let i=0;i<events.length;i++) {
 await plugin.event({event:events[i]});
 await waitForSequence(i+1);
}
// A burst is allowed to coalesce intermediate callbacks. The final complete
// snapshot must retain all answered permissions and the final idle state.
for(let i=0;i<100;i++) {
 await plugin.event({event:{type:'permission.asked',properties:{sessionID:'burst-session',id:'burst-'+i}}});
 await plugin.event({event:{type:'session.status',properties:{sessionID:'burst-session',status:{type:'busy'}}}});
 await plugin.event({event:{type:'permission.replied',properties:{sessionID:'burst-session',requestID:'burst-'+i}}});
}
await plugin.event({event:{type:'session.idle',properties:{sessionID:'burst-session'}}});
await waitForSequence(308);
`
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, node, "--input-type=module", "-e", runner)
	cmd.Env = append(os.Environ(), "ANYTTY_TERMINAL_ID=t", "ANYTTY_DAEMON_SOCKET=/test.sock")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("plugin callback: %v %s", err, out)
	}
	data, _ := os.ReadFile(log)
	s := &Store{}
	want := []string{"idle", "working", "blocked", "blocked", "blocked", "working", "exited"}
	var lastSequence uint64
	var last Record
	for i, line := range bytes.Split(bytes.TrimSpace(data), []byte{'\n'}) {
		e, err := DecodeOpenCode(line, Context{TerminalID: "t", DaemonSocket: "/test.sock"})
		if err != nil {
			t.Fatal(err)
		}
		if i < 7 && e.Sequence != uint64(i+1) {
			t.Fatal("ordered lifecycle reports missing")
		}
		if i >= 7 && e.Sequence <= lastSequence {
			t.Fatal("coalesced callbacks reordered")
		}
		lastSequence = e.Sequence
		r, _, err := s.Apply(*e, time.Now())
		last = r
		if err != nil || i < 7 && r.Status != want[i] {
			t.Fatalf("event %d: %+v %v", i, r, err)
		}
	}
	if lastSequence != 308 || last.Status != "idle" || len(last.PendingPermissions) != 0 {
		t.Fatalf("burst lost final complete state: %+v", last)
	}
	if lines := bytes.Count(bytes.TrimSpace(data), []byte{'\n'}) + 1; lines >= 308 {
		t.Fatal("burst did not exercise complete-state coalescing")
	}
}
