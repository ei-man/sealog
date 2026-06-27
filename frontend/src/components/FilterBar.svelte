<script lang="ts">
  import { onMount } from 'svelte';
  import { Search } from '@lucide/svelte';

  let {
    value = $bindable(''),
    error = '',
    showing,
    total,
    connected,
    onApply
  }: {
    value: string;
    error: string;
    showing: number;
    total: number;
    connected: boolean;
    onApply: (value: string) => void;
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

  <div class="filter-status">
    <span class:offline={!connected} class="connection-dot" title={connected ? 'Connected' : 'Reconnecting'}></span>
    <span>showing {showing.toLocaleString()} / {total.toLocaleString()}</span>
  </div>
</header>

{#if error}
  <div id="filter-error" class="filter-error" role="alert">{error}</div>
{/if}
