// Union · unionize.run
// Tiny progressive-enhancement script — the page works without it.

(function () {
  // Subtle parallax wobble on the crest as the cursor moves in the hero.
  const crest = document.querySelector('.crest');
  const stage = document.querySelector('.crest-stage');
  if (!crest || !stage || window.matchMedia('(prefers-reduced-motion: reduce)').matches) return;

  let raf = 0;
  let targetX = 0;
  let targetY = 0;
  let currentX = 0;
  let currentY = 0;
  let hovering = false;

  function onMove(e) {
    const rect = stage.getBoundingClientRect();
    const cx = rect.left + rect.width / 2;
    const cy = rect.top + rect.height / 2;
    targetX = Math.max(-1, Math.min(1, (e.clientX - cx) / (rect.width / 2)));
    targetY = Math.max(-1, Math.min(1, (e.clientY - cy) / (rect.height / 2)));
    hovering = true;
    if (!raf) raf = requestAnimationFrame(tick);
  }

  function onLeave() {
    hovering = false;
    targetX = 0;
    targetY = 0;
    if (!raf) raf = requestAnimationFrame(tick);
  }

  function tick() {
    currentX += (targetX - currentX) * 0.08;
    currentY += (targetY - currentY) * 0.08;
    const rx = currentY * -3.2; // tilt away from cursor
    const ry = currentX * 3.2;
    crest.style.transform = `perspective(900px) rotateX(${rx}deg) rotateY(${ry}deg)`;
    const rest = Math.abs(targetX - currentX) + Math.abs(targetY - currentY);
    if (rest > 0.002 || hovering) {
      raf = requestAnimationFrame(tick);
    } else {
      raf = 0;
      crest.style.transform = '';
    }
  }

  stage.addEventListener('pointermove', onMove);
  stage.addEventListener('pointerleave', onLeave);
})();
