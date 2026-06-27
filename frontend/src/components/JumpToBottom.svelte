<script lang="ts">
  import { ArrowDown } from '@lucide/svelte';

  let { onClick }: { onClick: () => void } = $props();
</script>

<button class="jump-to-bottom" type="button" onclick={onClick}>
  <ArrowDown size={16} aria-hidden="true" />
  <span>Jump to latest</span>
</button>
