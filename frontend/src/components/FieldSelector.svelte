<script lang="ts">
  import { Search } from '@lucide/svelte';
  import { settings } from '../lib/stores/settings.svelte';
  import ResizeHandle from './ResizeHandle.svelte';

  let { fields, width, onResize }: { fields: string[]; width: number; onResize: (delta: number) => void } = $props();

  let search = $state('');

  const selectedFields = $derived.by(() => {
    const needle = search.trim().toLocaleLowerCase();
    return settings.selectedFields.filter((field) => fields.includes(field) && field.toLocaleLowerCase().includes(needle));
  });

  const unselectedFields = $derived.by(() => {
    const selected = new Set(settings.selectedFields);
    const needle = search.trim().toLocaleLowerCase();
    return fields
      .filter((field) => !selected.has(field) && field.toLocaleLowerCase().includes(needle))
      .sort((a, b) => a.localeCompare(b));
  });
</script>

<aside class="field-selector" aria-label="Field selector" style:width="{width}px">
  <div class="panel-header">Fields</div>
  <label class="sidebar-search">
    <Search size={15} aria-hidden="true" />
    <input bind:value={search} placeholder="Filter fields" />
  </label>

  <div class="field-list">
    {#if selectedFields.length}
      <div class="field-group-label">Selected</div>
      {#each selectedFields as field (field)}
        <label class="field-row selected">
          <input type="checkbox" checked onchange={() => settings.toggleField(field)} />
          <span title={field}>{field}</span>
        </label>
      {/each}
    {/if}

    {#if unselectedFields.length}
      <div class="field-group-label">Available</div>
      {#each unselectedFields as field (field)}
        <label class="field-row">
          <input type="checkbox" onchange={() => settings.toggleField(field)} />
          <span title={field}>{field}</span>
        </label>
      {/each}
    {/if}

    {#if !fields.length}
      <div class="empty-panel">Waiting for log fields</div>
    {:else if !selectedFields.length && !unselectedFields.length}
      <div class="empty-panel">No matching fields</div>
    {/if}
  </div>
  <ResizeHandle position="left" {onResize} />
</aside>
