<script lang="ts">
  import type { LogLine } from '../lib/types';
  import { formatCellValue, isLevelField, levelTone, levelToneFromValue, MESSAGE_FIELDS, TIMESTAMP_FIELDS, valueToText } from '../lib/utils';

  let {
    line,
    fields,
    columnsTemplate,
    selected,
    includeDate,
    onSelect
  }: {
    line: LogLine;
    fields: string[];
    columnsTemplate: string;
    selected: boolean;
    includeDate: boolean;
    onSelect: (line: LogLine) => void;
  } = $props();

  const tone = $derived(levelTone(line));
  const rawText = $derived(valueToText(line.fields.raw));

  function cellClass(field: string): string {
    if (MESSAGE_FIELDS.has(field)) return 'log-cell message-cell';
    if (TIMESTAMP_FIELDS.has(field)) return 'log-cell meta-cell timestamp-cell';
    if (isLevelField(field)) return 'log-cell level-cell';
    return 'log-cell meta-cell';
  }

  function handleKeydown(event: KeyboardEvent) {
    if (event.key === 'Enter' || event.key === ' ') {
      event.preventDefault();
      onSelect(line);
    }
  }
</script>

<div
  class:selected
  class={`log-row tone-${tone}`}
  role="button"
  tabindex="0"
  style={`grid-template-columns: ${columnsTemplate};`}
  onclick={() => onSelect(line)}
  onkeydown={handleKeydown}
>
  {#if line.raw}
    <div class="log-cell raw-cell" title={rawText}>{rawText}</div>
  {:else if fields.length}
    {#each fields as field (field)}
      {@const value = formatCellValue(line.fields[field], field, includeDate)}
      <div class={cellClass(field)} title={value}>
        {#if isLevelField(field) && value}
          <span class={`level-badge badge-${levelToneFromValue(line.fields[field])}`}>{value}</span>
        {:else}
          {value}
        {/if}
      </div>
    {/each}
  {:else}
    <div class="log-cell empty-row"></div>
  {/if}
</div>
