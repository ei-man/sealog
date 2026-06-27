<script lang="ts">
  import { tick } from 'svelte';
  import { settings } from '../lib/stores/settings.svelte';
  import type { LogLine } from '../lib/types';
  import { timestampsSpanMultipleDays } from '../lib/utils';
  import JumpToBottom from './JumpToBottom.svelte';
  import LogRow from './LogRow.svelte';
  import VirtualScroller from './VirtualScroller.svelte';

  const ROW_HEIGHT = 32;

  let {
    lines,
    selectedLineId,
    onSelect
  }: {
    lines: LogLine[];
    selectedLineId: number | null;
    onSelect: (line: LogLine) => void;
  } = $props();

  let scroller: { scrollToBottom: () => Promise<void> } | null = null;
  let draggedField = $state<string | null>(null);
  let resizingField = $state<string | null>(null);
  let atBottom = $state(true);
  let hasNewRowsAwayFromBottom = $state(false);
  let lastLineCount = $state(0);

  const selectedFields = $derived(settings.selectedFields);
  const columnsTemplate = $derived(
    selectedFields.length ? selectedFields.map((field) => `${settings.widthFor(field)}px`).join(' ') : '1fr'
  );
  const tableWidth = $derived(selectedFields.length ? selectedFields.reduce((total, field) => total + settings.widthFor(field), 0) : 480);
  const includeDate = $derived(timestampsSpanMultipleDays(lines));

  $effect(() => {
    const count = lines.length;
    if (count > lastLineCount) {
      if (atBottom) {
        tick().then(() => scroller?.scrollToBottom());
      } else {
        hasNewRowsAwayFromBottom = true;
      }
    }
    lastLineCount = count;
  });

  function handleBottomChange(value: boolean) {
    atBottom = value;
    if (value) hasNewRowsAwayFromBottom = false;
  }

  function jumpToBottom() {
    scroller?.scrollToBottom();
    hasNewRowsAwayFromBottom = false;
  }

  function startResize(event: PointerEvent, field: string) {
    event.preventDefault();
    event.stopPropagation();
    resizingField = field;

    const handle = event.currentTarget as HTMLElement;
    handle.setPointerCapture?.(event.pointerId);
    document.body.classList.add('is-resizing-column');

    const startX = event.clientX;
    const startWidth = settings.widthFor(field);

    function move(moveEvent: PointerEvent) {
      moveEvent.preventDefault();
      settings.setColumnWidth(field, startWidth + moveEvent.clientX - startX);
    }

    function stop(stopEvent?: Event) {
      if (stopEvent) {
        stopEvent.preventDefault();
      }
      handle.releasePointerCapture?.(event.pointerId);
      document.body.classList.remove('is-resizing-column');
      resizingField = null;
      window.removeEventListener('pointermove', move);
      window.removeEventListener('pointerup', stop);
      window.removeEventListener('pointercancel', stop);
      window.removeEventListener('blur', stop);
    }

    window.addEventListener('pointermove', move);
    window.addEventListener('pointerup', stop);
    window.addEventListener('pointercancel', stop);
    window.addEventListener('blur', stop);
  }

  function resizeWithKeyboard(event: KeyboardEvent, field: string) {
    if (event.key !== 'ArrowLeft' && event.key !== 'ArrowRight') return;
    event.preventDefault();
    const delta = event.shiftKey ? 40 : 10;
    settings.setColumnWidth(field, settings.widthFor(field) + (event.key === 'ArrowRight' ? delta : -delta));
  }

  function startColumnDrag(event: PointerEvent, field: string) {
    if (event.button !== 0 || resizingField) return;
    if ((event.target as HTMLElement).closest('.resize-handle')) return;

    event.preventDefault();
    draggedField = field;
    document.body.classList.add('is-dragging-column');

    const header = event.currentTarget as HTMLElement;
    header.setPointerCapture?.(event.pointerId);

    function move(moveEvent: PointerEvent) {
      moveEvent.preventDefault();
      const target = document
        .elementFromPoint(moveEvent.clientX, moveEvent.clientY)
        ?.closest<HTMLElement>('.header-cell[data-field]');

      const targetField = target?.dataset.field;
      if (!targetField || targetField === field) return;

      const rect = target.getBoundingClientRect();
      const placement = moveEvent.clientX > rect.left + rect.width / 2 ? 'after' : 'before';
      settings.moveField(field, targetField, placement);
    }

    function stop(stopEvent?: Event) {
      stopEvent?.preventDefault();
      header.releasePointerCapture?.(event.pointerId);
      document.body.classList.remove('is-dragging-column');
      draggedField = null;
      window.removeEventListener('pointermove', move);
      window.removeEventListener('pointerup', stop);
      window.removeEventListener('pointercancel', stop);
      window.removeEventListener('blur', stop);
    }

    window.addEventListener('pointermove', move);
    window.addEventListener('pointerup', stop);
    window.addEventListener('pointercancel', stop);
    window.addEventListener('blur', stop);
  }

</script>

<section class="log-table" aria-label="Logs">
  <div class="table-x-scroll">
    <div class="table-min" style={`--table-width: ${tableWidth}px; --columns: ${columnsTemplate};`}>
      <div class="log-header" style={`grid-template-columns: ${columnsTemplate};`}>
        {#if selectedFields.length}
          {#each selectedFields as field (field)}
            <div
              class:dragging={draggedField === field}
              class="header-cell"
              data-field={field}
              role="columnheader"
              tabindex="0"
              onpointerdown={(event) => startColumnDrag(event, field)}
              title={field}
            >
              <span>{field}</span>
              <button
                class="resize-handle"
                aria-label={`Resize ${field}`}
                draggable="false"
                tabindex="0"
                type="button"
                ondragstart={(event) => event.preventDefault()}
                onpointerdown={(event) => startResize(event, field)}
                onkeydown={(event) => resizeWithKeyboard(event, field)}
              ></button>
            </div>
          {/each}
        {:else}
          <div class="header-cell empty-header">Select fields to show columns</div>
        {/if}
      </div>

      <VirtualScroller
        bind:this={scroller}
        items={lines}
        rowHeight={ROW_HEIGHT}
        onBottomChange={handleBottomChange}
      >
        {#snippet row(line: LogLine)}
          <LogRow
            {line}
            fields={selectedFields}
            {columnsTemplate}
            includeDate={includeDate}
            selected={line.id === selectedLineId}
            {onSelect}
          />
        {/snippet}
      </VirtualScroller>
    </div>
  </div>

  {#if hasNewRowsAwayFromBottom}
    <JumpToBottom onClick={jumpToBottom} />
  {/if}
</section>
