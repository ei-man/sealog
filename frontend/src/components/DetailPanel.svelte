<script lang="ts">
  import { Check, Copy, Filter, Search, X } from '@lucide/svelte';
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

  const properties = $derived.by(() => {
    const needle = search.trim().toLocaleLowerCase();
    return Object.entries(line.fields).filter(([key, value]) => {
      if (!needle) return true;
      return key.toLocaleLowerCase().includes(needle) || valueToText(value).toLocaleLowerCase().includes(needle);
    });
  });

  function displayValue(value: unknown, field: string): string {
    const text = valueToText(value);
    if (expanded.has(field)) return text;
    return text;
  }

  function isComplex(value: unknown): boolean {
    return typeof value === 'object' && value !== null;
  }

  function toggleExpanded(field: string) {
    const next = new Set(expanded);
    if (next.has(field)) next.delete(field);
    else next.add(field);
    expanded = next;
  }

  async function copyValue(field: string, value: unknown) {
    await navigator.clipboard.writeText(valueToText(value));
    copiedField = field;
    window.setTimeout(() => {
      if (copiedField === field) copiedField = null;
    }, 1200);
  }
</script>

<aside class="detail-panel" aria-label="Log entry details" style:width="{width}px">
  <div class="detail-header">
    <h2>Log entry details</h2>
    <button class="icon-button" type="button" aria-label="Close details" onclick={onClose}>
      <X size={18} aria-hidden="true" />
    </button>
  </div>

  <label class="detail-search">
    <Search size={15} aria-hidden="true" />
    <input bind:value={search} placeholder="Filter properties" />
  </label>

  <div class="detail-section-title">Properties</div>
  <div class="property-table" role="table" aria-label="Log properties">
    <div class="property-header" role="row">
      <div role="columnheader">Name</div>
      <div role="columnheader">Value</div>
    </div>

    {#each properties as [field, value] (field)}
      <div class="property-row" role="row">
        <div class="property-name" role="cell" title={field}>{field}</div>
        <div class="property-value-wrap" role="cell">
          <button
            class:expanded={expanded.has(field)}
            class:complex={isComplex(value)}
            class="property-value"
            type="button"
            title="Copy value"
            onclick={() => copyValue(field, value)}
            ondblclick={() => toggleExpanded(field)}
          >
            {displayValue(value, field)}
          </button>
          <button class="icon-button tiny" type="button" aria-label={`Copy ${field}`} onclick={() => copyValue(field, value)}>
            {#if copiedField === field}
              <Check size={14} aria-hidden="true" />
            {:else}
              <Copy size={14} aria-hidden="true" />
            {/if}
          </button>
          <button class="icon-button tiny" type="button" aria-label={`Filter by ${field}`} onclick={() => onFilter(field, value)}>
            <Filter size={14} aria-hidden="true" />
          </button>
          {#if copiedField === field}
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
