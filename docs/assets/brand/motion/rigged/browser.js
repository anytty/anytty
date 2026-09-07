(async()=>{
  const data=window.ANYTTY_MOTION;
  const binary=value=>Uint8Array.from(atob(value),c=>c.charCodeAt(0));
  rive.RuntimeLoader.setWasmBinary(binary(data.wasm).buffer);
  rive.RuntimeLoader.setWasmFallbackUrl(null);
  const runtime=await rive.RuntimeLoader.awaitInstance();
  const scene=document.querySelector('.scene'),query=document.querySelector('#query');
  const reduce=matchMedia('(prefers-reduced-motion: reduce)');
  const layers=[];
  for(const [id,key] of [['rear','back'],['front','front']]) {
    const canvas=document.querySelector(`#${id}`);
    const file=await runtime.load(binary(data[key]),undefined,false);
    const artboard=file.defaultArtboard();
    const renderer=runtime.makeRenderer(canvas,true);
    const animations=Object.fromEntries(['idle','gaze-x','gaze-y','head-follow','placement'].map(name=>[name,new runtime.LinearAnimationInstance(artboard.animationByName(name),artboard)]));
    layers.push({canvas,file,artboard,renderer,animations});
  }
  let paused=false,visible=true,frame=null,last=0,time=0,activity=0,disposed=false;
  let gaze={x:0,y:0},target={x:0,y:0},head=0,position=0;
  const clamp=(v,min,max)=>Math.max(min,Math.min(max,v));
  const width=()=>scene.getBoundingClientRect().width;
  function caret() {
    const rect=query.getBoundingClientRect(),bounds=scene.getBoundingClientRect();
    const measure=document.createElement('canvas').getContext('2d');
    measure.font=getComputedStyle(query).font;
    const offset=measure.measureText(query.value.slice(0,query.selectionStart??query.value.length)).width-query.scrollLeft;
    const x=rect.left+clamp(offset,0,rect.width);
    target={x:clamp((x-bounds.left-position-77)/65,-1,1),y:.8};
    activity=time+1.4;wake();
  }
  function resetPosition() {position=Math.max(0,width()-156);}
  function resize() {
    for(const {canvas} of layers) {canvas.width=Math.round(width()*devicePixelRatio);canvas.height=Math.round(174*devicePixelRatio);}
    resetPosition();
    if(document.activeElement===query) caret();
    wake();
  }
  function draw(dt) {
    const moving=!paused&&!reduce.matches;
    if(moving) {
      time+=dt;
      gaze.x+=(target.x-gaze.x)*(1-Math.exp(-dt*18));
      gaze.y+=(target.y-gaze.y)*(1-Math.exp(-dt*18));
      head+=(gaze.x-head)*(1-Math.exp(-dt*5));
    }
    for(const {canvas,artboard,renderer,animations:a} of layers) {
      const apply=(name,t)=>{a[name].time=t;a[name].advance(0);a[name].apply(1);};
      apply('idle',reduce.matches?0:time%14);
      apply('gaze-x',reduce.matches?.5:(gaze.x+1)/2);
      apply('gaze-y',reduce.matches?.5:(gaze.y+1)/2);
      apply('head-follow',reduce.matches?.5:(head+1)/2);
      apply('placement',position/1460);
      artboard.advance(0);
      renderer.clear();renderer.save();
      renderer.align(runtime.Fit.fill,runtime.Alignment.topLeft,{minX:0,minY:0,maxX:canvas.width,maxY:canvas.height},{minX:0,minY:0,maxX:width(),maxY:174});
      artboard.draw(renderer);renderer.restore();renderer.flush();
    }
    window.anyttyRigState={time,position,gaze:{...gaze},paused,ready:true};
  }
  function tick(now) {
    frame=null;
    const dt=last?Math.min((now-last)/1000,.05):0;last=now;
    draw(dt);
    if(!paused&&!reduce.matches&&visible&&!document.hidden&&(time<14||time<activity)) frame=runtime.requestAnimationFrame(tick);
    else last=0;
  }
  function wake() {if(!disposed&&frame===null&&visible&&!document.hidden) frame=runtime.requestAnimationFrame(tick);}
  const pointer=event=>{
    if(document.activeElement===query) return;
    const rect=scene.getBoundingClientRect();
    target={x:clamp((event.clientX-rect.left-position-77)/130,-1,1),y:clamp((event.clientY-rect.top-36)/130,-1,1)};
    activity=time+.8;wake();
  };
  document.addEventListener('pointermove',pointer);
  document.addEventListener('pointerdown',pointer);
  for(const name of ['input','keyup','click','select','scroll','focus']) query.addEventListener(name,caret);
  query.addEventListener('blur',()=>{target={x:0,y:0};activity=time+1.5;wake();});
  document.querySelector('#motion-toggle').onclick=()=>{
    paused=!paused;const label=paused?'播放':'暂停';
    const button=document.querySelector('#motion-toggle');button.title=label;button.setAttribute('aria-label',label);
    document.querySelector('#play-icon').hidden=!paused;document.querySelector('#pause-icon').hidden=paused;wake();
  };
  document.querySelector('#replay').onclick=()=>{time=0;activity=0;wake();};
  document.addEventListener('visibilitychange',()=>{last=0;if(document.hidden&&frame!==null){runtime.cancelAnimationFrame(frame);frame=null;}else wake();});
  const intersection=new IntersectionObserver(([entry])=>{visible=entry.isIntersecting;if(!visible&&frame!==null){runtime.cancelAnimationFrame(frame);frame=null;last=0;}else wake();});intersection.observe(scene);
  const observer=new ResizeObserver(resize);observer.observe(scene);
  reduce.addEventListener('change',()=>{time=0;wake();});
  // Deterministic review/QA scrubbing; all deformation remains inside the .riv assets.
  window.anyttyRigPreview={seek:seconds=>new Promise(resolve=>runtime.requestAnimationFrame(()=>{time=clamp(seconds,0,14);draw(0);resolve();}))};
  window.addEventListener('pageshow',event=>{if(event.persisted)location.reload();});
  window.addEventListener('pagehide',()=>{
    disposed=true;if(frame!==null)runtime.cancelAnimationFrame(frame);observer.disconnect();intersection.disconnect();
    for(const l of layers){Object.values(l.animations).forEach(a=>a.delete());l.artboard.delete();l.file.delete();l.renderer.delete();}
  },{once:true});
  resetPosition();resize();
})().catch(error=>{document.querySelector('#result').textContent='动画加载失败';console.error(error);});
