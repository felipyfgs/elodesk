<script setup lang="ts">
const props = defineProps<{
  visible?: boolean
  search: string
}>()

const model = defineModel<boolean>()

const emit = defineEmits<{
  select: [name: string]
}>()

// Agents would come from the agents store (Fase 4), for now use a simple approach
// const auth = useAuthStore()

// Placeholder — will be populated from agents endpoint in Fase 4
const agents = ref<{ id: string, name: string }[]>([])

const filtered = computed(() => {
  if (!props.search) return agents.value.slice(0, 8)
  const lower = props.search.toLowerCase()
  return agents.value
    .filter(a => a.name.toLowerCase().includes(lower))
    .slice(0, 8)
})

function handleSelect(agent: { name: string }) {
  emit('select', agent.name)
}

function handleKeydown(e: KeyboardEvent) {
  if (e.key === 'Escape') model.value = false
}
</script>

<template>
  <Teleport to="body">
    <div
      v-if="visible"
      class="fixed z-50 bg-popover border border-default rounded-lg shadow-lg py-1 max-h-60 overflow-y-auto w-56"
      tabindex="-1"
      @keydown="handleKeydown"
    >
      <p class="px-3 py-1 text-xs text-dimmed font-medium uppercase tracking-wider">
        Mentions
      </p>
      <button
        v-for="agent in filtered"
        :key="agent.id"
        type="button"
        class="w-full text-left px-3 py-2 text-sm hover:bg-elevated transition-colors flex items-center gap-2"
        @click="handleSelect(agent)"
      >
        <UAvatar :alt="agent.name" size="xs" />
        <span>{{ agent.name }}</span>
      </button>
      <p v-if="!filtered.length" class="px-3 py-2 text-sm text-muted">
        No agents found
      </p>
    </div>
  </Teleport>
</template>
