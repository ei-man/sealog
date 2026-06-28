<script lang="ts">
  import { onMount } from 'svelte';
  import { Cloud, RefreshCw, Search } from '@lucide/svelte';
  import type { AzureResource, LoadingState, SourceState, TimeRange } from '../lib/types';
  import TimeRangeControl from './TimeRangeControl.svelte';

  let {
    value = $bindable(''),
    error = '',
    showing,
    total,
    connected,
    source,
    timeRange,
    azureResources,
    azureError,
    loading,
    onApply,
    onTimeRangeChange,
    onListResources,
    onSelectResource
  }: {
    value: string;
    error: string;
    showing: number;
    total: number;
    connected: boolean;
    source: SourceState;
    timeRange: TimeRange;
    azureResources: AzureResource[];
    azureError: string;
    loading: LoadingState;
    onApply: (value: string) => void;
    onTimeRangeChange: (timeRange: TimeRange) => void;
    onListResources: () => void;
    onSelectResource: (resourceId: string) => void;
  } = $props();

  let input: HTMLInputElement;

  function handleKeydown(event: KeyboardEvent) {
    if (event.key === 'Enter') {
      onApply(value);
    }
  }

  function handleGlobalKeydown(event: KeyboardEvent) {
    if ((event.metaKey || event.ctrlKey) && event.key.toLowerCase() === 'k') {
      event.preventDefault();
      input?.focus();
      input?.select();
    }
  }

  function resourceLabel(resource: AzureResource): string {
    return `${resource.subscriptionId} / ${resource.resourceGroup} / ${resource.name}`;
  }

  onMount(() => {
    window.addEventListener('keydown', handleGlobalKeydown);
    return () => window.removeEventListener('keydown', handleGlobalKeydown);
  });
</script>

<header class="filter-bar">
  <div class="filter-input-wrap">
    <Search size={16} aria-hidden="true" />
    <input
      bind:this={input}
      bind:value
      class:error={Boolean(error)}
      aria-invalid={Boolean(error)}
      aria-describedby={error ? 'filter-error' : undefined}
      placeholder='level="error" OR message~="timeout|failed"'
      onkeydown={handleKeydown}
    />
  </div>

  <div class="filter-controls">
    <TimeRangeControl {timeRange} onChange={onTimeRangeChange} />

    {#if source.appInsightsEnabled}
      <div class="resource-picker">
        <Cloud size={15} aria-hidden="true" />
        <select
          aria-label="Application Insights resource"
          value={source.selectedResource?.id ?? ''}
          onchange={(event) => onSelectResource((event.currentTarget as HTMLSelectElement).value)}
        >
          <option value="">Select App Insights</option>
          {#each azureResources as resource (resource.id)}
            <option value={resource.id}>{resourceLabel(resource)}</option>
          {/each}
        </select>
        <button class:spinning={loading.active} class="icon-button small" type="button" aria-label="Refresh Application Insights resources" onclick={onListResources}>
          <RefreshCw size={15} aria-hidden="true" />
        </button>
      </div>
    {/if}
  </div>

  <div class="filter-status">
    <span class:offline={!connected} class="connection-dot" title={connected ? 'Connected' : 'Reconnecting'}></span>
    <span>{loading.active ? loading.message || 'Loading' : `showing ${showing.toLocaleString()} / ${total.toLocaleString()}`}</span>
  </div>
</header>

{#if error}
  <div id="filter-error" class="filter-error" role="alert">{error}</div>
{/if}

{#if azureError}
  <div class="filter-error azure-error" role="alert">{azureError}</div>
{/if}
