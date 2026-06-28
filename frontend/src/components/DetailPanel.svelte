<script lang="ts">
  import { Check, ChevronDown, ChevronRight, Copy, Filter, Search, X } from '@lucide/svelte';
  import type { LogLine } from '../lib/types';
  import { valueToText } from '../lib/utils';
  import ResizeHandle from './ResizeHandle.svelte';

  let {
    line,
    width,
    onClose,
    onFilter,
    onResize
  }: {
    line: LogLine;
    width: number;
    onClose: () => void;
    onFilter: (field: string, value: unknown) => void;
    onResize: (delta: number) => void;
  } = $props();

  let search = $state('');
  let copiedField = $state<string | null>(null);
  let expanded = $state<Set<string>>(new Set());
  let propertyNameWidth = $state(180);

  const MIN_PROPERTY_NAME_WIDTH = 96;
  const MIN_PROPERTY_VALUE_WIDTH = 180;

  type PropertyRow = {
    path: string;
    name: string;
    value: unknown;
    depth: number;
    expandable: boolean;
    expanded: boolean;
    filterable: boolean;
  };

  const properties = $derived.by<PropertyRow[]>(() => {
    const needle = search.trim().toLocaleLowerCase();
    return buildRows(line.fields, '', 0, needle);
  });

  function buildRows(value: unknown, prefix: string, depth: number, needle: string): PropertyRow[] {
    if (!isRecord(value)) return [];

    const rows: PropertyRow[] = [];
    for (const [key, item] of Object.entries(value)) {
      const path = prefix ? `${prefix}.${key}` : key;
      const expandable = isRecord(item);
      if (needle && !matchesTree(path, item, needle)) continue;

      rows.push({
        path,
        name: key,
        value: item,
        depth,
        expandable,
        expanded: expanded.has(path),
        filterable: !expandable
      });

      if (expandable && (expanded.has(path) || needle)) {
        rows.push(...buildRows(item, path, depth + 1, needle));
      }
    }
    return rows;
  }

  function matchesTree(path: string, value: unknown, needle: string): boolean {
    if (path.toLocaleLowerCase().includes(needle)) return true;
    if (!isRecord(value)) return valueToText(value).toLocaleLowerCase().includes(needle);
    return Object.entries(value).some(([key, item]) => matchesTree(`${path}.${key}`, item, needle));
  }

  function isRecord(value: unknown): value is Record<string, unknown> {
    return typeof value === 'object' && value !== null && !Array.isArray(value);
  }

  function displayValue(row: PropertyRow): string {
    if (row.expandable) {
      const count = Object.keys(row.value as Record<string, unknown>).length;
      return `{${count}}`;
    }
    return valueToText(row.value);
  }

  function toggleExpanded(field: string) {
    const next = new Set(expanded);
    if (next.has(field)) next.delete(field);
    else next.add(field);
    expanded = next;
  }

  function maxPropertyNameWidth(): number {
    return Math.max(MIN_PROPERTY_NAME_WIDTH, width - MIN_PROPERTY_VALUE_WIDTH);
  }

  function setPropertyNameWidth(width: number) {
    propertyNameWidth = Math.max(MIN_PROPERTY_NAME_WIDTH, Math.min(maxPropertyNameWidth(), Math.round(width)));
  }

  function startPropertyNameResize(event: PointerEvent) {
    event.preventDefault();
    event.stopPropagation();

    const handle = event.currentTarget as HTMLElement;
    handle.setPointerCapture?.(event.pointerId);
    document.body.classList.add('is-resizing-column');

    const startX = event.clientX;
    const startWidth = propertyNameWidth;

    function move(moveEvent: PointerEvent) {
      moveEvent.preventDefault();
      setPropertyNameWidth(startWidth + moveEvent.clientX - startX);
    }

    function stop(stopEvent?: Event) {
      stopEvent?.preventDefault();
      handle.releasePointerCapture?.(event.pointerId);
      document.body.classList.remove('is-resizing-column');
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

  function resizePropertyNameWithKeyboard(event: KeyboardEvent) {
    if (event.key !== 'ArrowLeft' && event.key !== 'ArrowRight') return;
    event.preventDefault();
    const delta = event.shiftKey ? 40 : 10;
    setPropertyNameWidth(propertyNameWidth + (event.key === 'ArrowRight' ? delta : -delta));
  }

  async function copyValue(field: string, value: unknown) {
    await navigator.clipboard.writeText(valueToText(value));
    copiedField = field;
    window.setTimeout(() => {
      if (copiedField === field) copiedField = null;
    }, 1200);
  }

  async function copyEntry() {
    await navigator.clipboard.writeText(JSON.stringify(line.fields, null, 2));
    copiedField = '__entry';
    window.setTimeout(() => {
      if (copiedField === '__entry') copiedField = null;
    }, 1200);
  }
</script>

<aside class="detail-panel" aria-label="Log entry details" style:width="{width}px">
  <div class="detail-header">
    <h2>Log entry details</h2>
    <div class="detail-actions">
      <button class="icon-button" type="button" aria-label="Copy full entry" onclick={copyEntry}>
        {#if copiedField === '__entry'}
          <Check size={17} aria-hidden="true" />
        {:else}
          <Copy size={17} aria-hidden="true" />
        {/if}
      </button>
      <button class="icon-button" type="button" aria-label="Close details" onclick={onClose}>
        <X size={18} aria-hidden="true" />
      </button>
    </div>
  </div>

  <label class="detail-search">
    <Search size={15} aria-hidden="true" />
    <input bind:value={search} placeholder="Filter properties" />
  </label>

  <div class="detail-section-title">Properties</div>
  <div class="property-table" role="table" aria-label="Log properties" style={`--property-name-width: ${propertyNameWidth}px;`}>
    <div class="property-header" role="row">
      <div class="property-name-header" role="columnheader">
        <span>Name</span>
        <button
          class="property-column-resize-handle"
          type="button"
          aria-label="Resize property name column"
          onpointerdown={startPropertyNameResize}
          onkeydown={resizePropertyNameWithKeyboard}
        ></button>
      </div>
      <div role="columnheader">Value</div>
    </div>

    {#each properties as row (row.path)}
      <div class="property-row" role="row">
        <div class="property-name" role="cell" title={row.path} style={`--depth: ${row.depth};`}>
          {#if row.expandable}
            <button class="tree-toggle" type="button" aria-label={`${row.expanded ? 'Collapse' : 'Expand'} ${row.path}`} onclick={() => toggleExpanded(row.path)}>
              {#if row.expanded || search}
                <ChevronDown size={13} aria-hidden="true" />
              {:else}
                <ChevronRight size={13} aria-hidden="true" />
              {/if}
            </button>
          {:else}
            <span class="tree-spacer"></span>
          {/if}
          <span>{row.name}</span>
        </div>
        <div class="property-value-wrap" role="cell">
          <div
            class:expanded={row.expandable && row.expanded}
            class:complex={row.expandable}
            class="property-value"
            title={displayValue(row)}
          >
            {displayValue(row)}
          </div>
          <button class="icon-button tiny property-action" type="button" aria-label={`Copy ${row.path}`} onclick={() => copyValue(row.path, row.value)}>
            {#if copiedField === row.path}
              <Check size={14} aria-hidden="true" />
            {:else}
              <Copy size={14} aria-hidden="true" />
            {/if}
          </button>
          {#if row.filterable}
            <button class="icon-button tiny property-action" type="button" aria-label={`Filter by ${row.path}`} onclick={() => onFilter(row.path, row.value)}>
              <Filter size={14} aria-hidden="true" />
            </button>
          {:else}
            <span class="property-action-placeholder"></span>
          {/if}
          {#if copiedField === row.path}
            <span class="copied-tip">Copied</span>
          {/if}
        </div>
      </div>
    {/each}

    {#if !properties.length}
      <div class="empty-properties">No matching properties</div>
    {/if}
  </div>
  <ResizeHandle position="right" {onResize} />
</aside>
