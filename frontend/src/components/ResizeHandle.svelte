<script lang="ts">
  let {
    position,
    onResize
  }: {
    position: 'left' | 'right';
    onResize: (delta: number) => void;
  } = $props();

  let dragging = $state(false);
  let startX = 0;

  function handlePointerDown(event: PointerEvent) {
    event.preventDefault();
    dragging = true;
    startX = event.clientX;
    document.body.classList.add('is-resizing-panel');
    (event.target as HTMLElement).setPointerCapture(event.pointerId);
  }

  function handlePointerMove(event: PointerEvent) {
    if (!dragging) return;
    const delta = event.clientX - startX;
    startX = event.clientX;
    onResize(position === 'left' ? delta : -delta);
  }

  function handlePointerUp(event: PointerEvent) {
    if (!dragging) return;
    dragging = false;
    document.body.classList.remove('is-resizing-panel');
    (event.target as HTMLElement).releasePointerCapture(event.pointerId);
  }
</script>

<div
  class="panel-resize-handle {position}"
  role="separator"
  aria-orientation="vertical"
  onpointerdown={handlePointerDown}
  onpointermove={handlePointerMove}
  onpointerup={handlePointerUp}
  onpointercancel={handlePointerUp}
></div>
