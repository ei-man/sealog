<script lang="ts">
  import { onMount, untrack } from 'svelte';
  import FilterBar from './components/FilterBar.svelte';
  import FieldSelector from './components/FieldSelector.svelte';
  import LogTable from './components/LogTable.svelte';
  import DetailPanel from './components/DetailPanel.svelte';
  import { evaluateFilter } from './lib/filter/evaluator';
  import { parseQuery, ParseError } from './lib/filter/parser';
  import { logs } from './lib/stores/logs.svelte';
  import { settings } from './lib/stores/settings.svelte';
  import { connectLogs } from './lib/websocket';
  import type { AstNode } from './lib/filter/ast';
  import type { LogLine } from './lib/types';
  import { quoteFilterValue } from './lib/utils';

  let filterText = $state('');
  let filterError = $state('');
  let activeAst = $state<AstNode | null>(null);
  let selectedLineId = $state<number | null>(null);
  let fieldSelectorWidth = $state(250);
  let detailPanelWidth = $state(400);
  let workspaceEl: HTMLElement | undefined = $state();

  const MIN_PANEL_WIDTH = 180;
  const MIN_LOG_TABLE_WIDTH = 300;

  function resizeFieldSelector(delta: number) {
    const totalWidth = workspaceEl?.clientWidth ?? 0;
    const detailWidth = selectedLine ? detailPanelWidth : 0;
    const maxWidth = totalWidth - detailWidth - MIN_LOG_TABLE_WIDTH;
    fieldSelectorWidth = Math.max(MIN_PANEL_WIDTH, Math.min(maxWidth, fieldSelectorWidth + delta));
  }

  function resizeDetailPanel(delta: number) {
    const totalWidth = workspaceEl?.clientWidth ?? 0;
    const maxWidth = totalWidth - fieldSelectorWidth - MIN_LOG_TABLE_WIDTH;
    detailPanelWidth = Math.max(MIN_PANEL_WIDTH, Math.min(maxWidth, detailPanelWidth + delta));
  }

  $effect(() => {
    const fields = logs.fields;
    untrack(() => {
      settings.syncAvailableFields(fields);
    });
  });

  const filteredLines = $derived.by(() => {
    const ast = activeAst;
    if (!ast) return logs.lines;
    return logs.lines.filter((line) => evaluateFilter(ast, line));
  });

  const selectedLine = $derived(
    selectedLineId === null ? null : logs.lines.find((line) => line.id === selectedLineId) ?? null
  );

  function applyFilter(nextFilter: string) {
    const trimmed = nextFilter.trim();
    if (!trimmed) {
      filterText = nextFilter;
      filterError = '';
      activeAst = null;
      return;
    }

    try {
      activeAst = parseQuery(trimmed);
      filterText = nextFilter;
      filterError = '';
    } catch (error) {
      if (error instanceof ParseError) {
        filterError = `${error.message} at position ${error.position + 1}`;
      } else if (error instanceof Error) {
        filterError = error.message;
      } else {
        filterError = 'Invalid filter';
      }
    }
  }

  function selectLine(line: LogLine) {
    selectedLineId = line.id;
  }

  function closeDetails() {
    selectedLineId = null;
  }

  function appendFilter(field: string, value: unknown) {
    const clause = `${field}="${quoteFilterValue(value)}"`;
    const trimmed = filterText.trim();
    const nextFilter = trimmed ? `${trimmed} AND ${clause}` : clause;
    filterText = nextFilter;
    applyFilter(nextFilter);
  }

  function handleKeydown(event: KeyboardEvent) {
    if (event.key === 'Escape' && selectedLineId !== null) {
      selectedLineId = null;
    }
  }

  onMount(() => {
    const disconnect = connectLogs();
    window.addEventListener('keydown', handleKeydown);
    return () => {
      disconnect();
      window.removeEventListener('keydown', handleKeydown);
    };
  });
</script>

<svelte:head>
  <title>Sealog</title>
</svelte:head>

<main class="app-shell">
  <FilterBar
    bind:value={filterText}
    error={filterError}
    showing={filteredLines.length}
    total={logs.lines.length}
    connected={logs.connected}
    onApply={applyFilter}
  />

  <section
    bind:this={workspaceEl}
    class:with-details={Boolean(selectedLine)}
    class="workspace"
    style:--field-selector-width="{fieldSelectorWidth}px"
    style:--detail-panel-width="{detailPanelWidth}px"
  >
    <FieldSelector fields={logs.fields} width={fieldSelectorWidth} onResize={resizeFieldSelector} />
    <LogTable
      lines={filteredLines}
      selectedLineId={selectedLineId}
      onSelect={selectLine}
    />
    {#if selectedLine}
      <DetailPanel line={selectedLine} width={detailPanelWidth} onClose={closeDetails} onFilter={appendFilter} onResize={resizeDetailPanel} />
    {/if}
  </section>
</main>
