<script lang="ts" generics="T">
  import { onMount, tick, type Snippet } from 'svelte';

  type VisibleItem<T> = {
    item: T;
    index: number;
  };

  let {
    items,
    rowHeight,
    overscan = 10,
    onBottomChange,
    row
  }: {
    items: T[];
    rowHeight: number;
    overscan?: number;
    onBottomChange?: (atBottom: boolean) => void;
    row: Snippet<[T, number]>;
  } = $props();

  let viewport: HTMLDivElement;
  let height = $state(0);
  let scrollTop = $state(0);

  const totalHeight = $derived(items.length * rowHeight);
  const startIndex = $derived(Math.max(0, Math.floor(scrollTop / rowHeight) - overscan));
  const visibleCount = $derived(Math.ceil(height / rowHeight) + overscan * 2);
  const endIndex = $derived(Math.min(items.length, startIndex + visibleCount));
  const visibleItems = $derived.by((): VisibleItem<T>[] =>
    items.slice(startIndex, endIndex).map((item, offset) => ({ item, index: startIndex + offset }))
  );
  const offsetY = $derived(startIndex * rowHeight);
  const atBottom = $derived(totalHeight <= height || scrollTop + height >= totalHeight - 4);

  $effect(() => {
    onBottomChange?.(atBottom);
  });

  function handleScroll() {
    scrollTop = viewport?.scrollTop ?? 0;
  }

  export async function scrollToBottom() {
    await tick();
    if (!viewport) return;
    viewport.scrollTop = Math.max(0, viewport.scrollHeight - viewport.clientHeight);
    scrollTop = viewport.scrollTop;
  }

  onMount(() => {
    const updateHeight = () => {
      height = viewport?.clientHeight ?? 0;
      scrollTop = viewport?.scrollTop ?? 0;
    };
    updateHeight();
    const observer = new ResizeObserver(updateHeight);
    if (viewport) observer.observe(viewport);
    return () => observer.disconnect();
  });
</script>

<div class="virtual-scroller" bind:this={viewport} onscroll={handleScroll}>
  <div class="virtual-spacer" style={`height: ${totalHeight}px;`}>
    <div class="virtual-window" style={`transform: translateY(${offsetY}px);`}>
      {#each visibleItems as visible (visible.index)}
        {@render row(visible.item, visible.index)}
      {/each}
    </div>
  </div>
</div>
