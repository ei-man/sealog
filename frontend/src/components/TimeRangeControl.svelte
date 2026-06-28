<script lang="ts">
  import { Clock } from '@lucide/svelte';
  import type { TimeRange, TimeRangePreset } from '../lib/types';

  let {
    timeRange,
    onChange
  }: {
    timeRange: TimeRange;
    onChange: (timeRange: TimeRange) => void;
  } = $props();

  const presets: { value: TimeRangePreset; label: string }[] = [
    { value: '15m', label: '15m' },
    { value: '30m', label: '30m' },
    { value: '1h', label: '1h' },
    { value: '3h', label: '3h' },
    { value: '6h', label: '6h' },
    { value: '12h', label: '12h' },
    { value: '24h', label: '24h' },
    { value: '3d', label: '3d' },
    { value: 'custom', label: 'Custom' }
  ];

  let preset = $state<TimeRangePreset>('1h');
  let customStart = $state('');
  let customEnd = $state('');

  $effect(() => {
    preset = timeRange.preset;
    customStart = toLocalInput(timeRange.start);
    customEnd = toLocalInput(timeRange.end);
  });

  function handlePresetChange() {
    if (preset !== 'custom') {
      onChange({ preset });
      return;
    }

    const end = new Date();
    const start = new Date(end.getTime() - 60 * 60 * 1000);
    customStart = toLocalInput(start.toISOString());
    customEnd = toLocalInput(end.toISOString());
    emitCustom();
  }

  function emitCustom() {
    if (!customStart || !customEnd) return;
    onChange({
      preset: 'custom',
      start: new Date(customStart).toISOString(),
      end: new Date(customEnd).toISOString()
    });
  }

  function toLocalInput(value: string | undefined): string {
    if (!value) return '';
    const date = new Date(value);
    if (Number.isNaN(date.getTime())) return '';
    const year = date.getFullYear();
    const month = String(date.getMonth() + 1).padStart(2, '0');
    const day = String(date.getDate()).padStart(2, '0');
    const hours = String(date.getHours()).padStart(2, '0');
    const minutes = String(date.getMinutes()).padStart(2, '0');
    return `${year}-${month}-${day}T${hours}:${minutes}`;
  }
</script>

<div class="time-range-control">
  <Clock size={15} aria-hidden="true" />
  <select bind:value={preset} aria-label="Time range" onchange={handlePresetChange}>
    {#each presets as option}
      <option value={option.value}>{option.label}</option>
    {/each}
  </select>

  {#if preset === 'custom'}
    <input bind:value={customStart} aria-label="Start time" type="datetime-local" onchange={emitCustom} />
    <input bind:value={customEnd} aria-label="End time" type="datetime-local" onchange={emitCustom} />
  {/if}
</div>
