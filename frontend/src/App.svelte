<script lang="ts">
  import { onMount, untrack } from 'svelte';
  import FilterBar from './components/FilterBar.svelte';
  import FieldSelector from './components/FieldSelector.svelte';
  import LogTable from './components/LogTable.svelte';
  import DetailPanel from './components/DetailPanel.svelte';
  import { logs } from './lib/stores/logs.svelte';
  import { settings } from './lib/stores/settings.svelte';
  import { connectLogs, type LogConnection } from './lib/websocket';
  import type { LogLine, TimeRange } from './lib/types';
  import { quoteFilterValue } from './lib/utils';

  let filterText = $state('');
  let selectedLineId = $state<number | null>(null);
  let fieldSelectorWidth = $state(250);
  let detailPanelWidth = $state(400);
  let workspaceEl: HTMLElement | undefined = $state();
  let connection: LogConnection | null = null;
  let requestedAzureResources = $state(false);
  let lastServerQuery = $state('');

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

  $effect(() => {
    const query = logs.source.query;
    untrack(() => {
      if (query !== lastServerQuery) {
        lastServerQuery = query;
        filterText = query;
      }
    });
  });

  $effect(() => {
    if (logs.connected && logs.source.appInsightsEnabled && !requestedAzureResources) {
      requestedAzureResources = true;
      connection?.send({ type: 'azure.resources.list' });
    }
  });

  const selectedLine = $derived(
    selectedLineId === null ? null : logs.lines.find((line) => line.id === selectedLineId) ?? null
  );

  function applyFilter(nextFilter: string) {
    filterText = nextFilter;
    connection?.send({ type: 'query.apply', query: nextFilter });
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

  function setTimeRange(timeRange: TimeRange) {
    connection?.send({ type: 'timeRange.set', timeRange });
  }

  function listAzureResources() {
    requestedAzureResources = true;
    connection?.send({ type: 'azure.resources.list' });
  }

  function selectAzureResource(resourceId: string) {
    connection?.send({ type: 'azure.resource.select', resourceId });
  }

  function handleKeydown(event: KeyboardEvent) {
    if (event.key === 'Escape' && selectedLineId !== null) {
      selectedLineId = null;
    }
  }

  onMount(() => {
    connection = connectLogs();
    window.addEventListener('keydown', handleKeydown);
    return () => {
      connection?.disconnect();
      connection = null;
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
    error={logs.queryError}
    showing={logs.lines.length}
    total={logs.total}
    connected={logs.connected}
    source={logs.source}
    timeRange={logs.timeRange}
    azureResources={logs.azureResources}
    azureError={logs.azureError}
    loading={logs.loading}
    onApply={applyFilter}
    onTimeRangeChange={setTimeRange}
    onListResources={listAzureResources}
    onSelectResource={selectAzureResource}
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
      lines={logs.lines}
      selectedLineId={selectedLineId}
      onSelect={selectLine}
    />
    {#if selectedLine}
      <DetailPanel line={selectedLine} width={detailPanelWidth} onClose={closeDetails} onFilter={appendFilter} onResize={resizeDetailPanel} />
    {/if}
  </section>
</main>
