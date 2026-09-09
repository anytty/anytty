(() => {
  const tail = document.querySelector('#lower-tail');
  const head = document.querySelector('#original-head');
  const toggle = document.querySelector('#motion-toggle');
  const replay = document.querySelector('#replay');
  const reduced = matchMedia('(prefers-reduced-motion: reduce)');
  const eyes = ['left', 'right'].map((side, i) => ({
    lid: document.querySelector(`#eyelid-${side}`),
    pupil: document.querySelector(`#pupil-${side}`),
    x: i ? 354 : 180, y: i ? 99 : 173,
  }));
  let elapsed = 0, last = 0, raf = 0, paused = false, onScreen = true, target = null;
  let gazeX = 0, gazeY = 0;
  const pulse = (t, start, length) => {
    const phase = Math.max(0, Math.min(1, (t - start) / length));
    return Math.sin(phase * Math.PI) ** 2;
  };
  const clamp = (value, limit) => Math.max(-limit, Math.min(limit, value));
  function pose(t, dt = 0) {
    const active = !reduced.matches;
    const envelope = pulse(t, .2, 4.6) + .6 * pulse(t, 7.8, 3.6);
    const angle = active ? 5 * Math.sin(t * Math.PI * 1.25) * envelope : 0;
    tail.setAttribute('transform', `translate(44 118) rotate(${-105 + angle}) scale(.20) translate(-149 -444)`);
    const glance = pulse(t, 2.8, 3.8);
    let x = -12 * glance + 9 * pulse(t, 8.2, 2.8);
    let y = 5 * glance;
    if (target && performance.now() < target.until) { x = target.x; y = target.y; }
    const blend = 1 - Math.exp(-dt / .10);
    gazeX = active ? gazeX + (x - gazeX) * blend : 0;
    gazeY = active ? gazeY + (y - gazeY) * blend : 0;
    for (let i = 0; i < eyes.length; i++) {
      const eye = eyes[i];
      const blink = active ? Math.max(pulse(t, 1.45 + i * .025, .26), pulse(t, 6.5 + i * .025, .30), pulse(t, 10.8 + i * .025, .24)) : 0;
      eye.lid.setAttribute('transform', `translate(${eye.x} ${eye.y}) scale(1 ${1 - blink * .96}) translate(${-eye.x} ${-eye.y})`);
      eye.pupil.setAttribute('transform', `translate(${gazeX} ${gazeY})`);
    }
  }
  function stop() { cancelAnimationFrame(raf); raf = 0; last = 0; }
  function tick(now) {
    raf = 0;
    const dt = last ? Math.min((now - last) / 1000, .05) : 0;
    last = now; elapsed = (elapsed + dt) % 14;
    pose(elapsed, dt);
    raf = requestAnimationFrame(tick);
  }
  function sync() {
    stop();
    const still = paused || reduced.matches;
    toggle.disabled = reduced.matches;
    replay.disabled = reduced.matches;
    toggle.title = still ? '播放' : '暂停';
    toggle.setAttribute('aria-label', toggle.title);
    document.querySelector('#pause-icon').hidden = still;
    document.querySelector('#play-icon').hidden = !still;
    if (reduced.matches) { elapsed = 0; target = null; pose(0); }
    if (!still && onScreen && !document.hidden) raf = requestAnimationFrame(tick);
  }
  function lookAt(event) {
    if (paused || reduced.matches || !onScreen || document.hidden || event.target.closest('.controls')) return;
    const matrix = head.getScreenCTM();
    if (!matrix) return;
    const point = new DOMPoint(event.clientX, event.clientY).matrixTransform(matrix.inverse());
    target = { x: clamp((point.x - 267) / 12, 16), y: clamp((point.y - 136) / 12, 10), until: performance.now() + (event.pointerType === 'touch' ? 1800 : 2500) };
  }
  toggle.addEventListener('click', () => { paused = !paused; sync(); });
  replay.addEventListener('click', () => { elapsed = 0; gazeX = 0; gazeY = 0; target = null; paused = false; pose(0); sync(); });
  document.addEventListener('pointermove', lookAt, { passive: true });
  document.addEventListener('pointerdown', lookAt, { passive: true });
  document.documentElement.addEventListener('pointerleave', () => { target = null; });
  document.addEventListener('visibilitychange', sync);
  reduced.addEventListener('change', sync);
  const observer = new IntersectionObserver(entries => { onScreen = entries[0].isIntersecting; sync(); });
  observer.observe(document.querySelector('.scene'));
  window.addEventListener('pagehide', stop);
  window.addEventListener('pageshow', sync);
  pose(0); sync();
})();
