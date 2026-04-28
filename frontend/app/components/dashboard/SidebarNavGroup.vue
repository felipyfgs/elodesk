<script setup lang="ts">
import type { SidebarItem } from '~/composables/useDashboard'

defineProps<{
  items: SidebarItem[]
  collapsed?: boolean
}>()

const route = useRoute()
const sectionKey = computed(() => route.path.split('/').slice(0, 4).join('/'))
</script>

<template>
  <UNavigationMenu
    :key="sectionKey"
    :collapsed="collapsed"
    :items="(items as any)"
    orientation="vertical"
    tooltip
    popover
    :ui="{
      label: 'mt-3 mb-0.5 px-2 flex items-center gap-1.5 text-[11px] font-medium uppercase tracking-wider text-muted'
    }"
  >
    <template #colored-leading="{ item }">
      <span
        class="size-2.5 rounded-full ring-1 ring-default shrink-0"
        :style="{ backgroundColor: ((item as { meta?: { color?: string } }).meta?.color) ?? '#888' }"
      />
    </template>
  </UNavigationMenu>
</template>
