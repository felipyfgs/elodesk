<script setup lang="ts">
withDefaults(defineProps<{
  url?: string | null
  name?: string | null
  size?: '3xs' | '2xs' | 'xs' | 'sm' | 'md' | 'lg' | 'xl' | '2xl' | '3xl'
  isGroup?: boolean
}>(), {
  size: 'md'
})

function initials(name?: string | null): string {
  if (!name) return '?'
  return name.split(/\s+/).map(w => w[0]).join('').slice(0, 2).toUpperCase()
}

function bgColor(name?: string | null): string {
  if (!name) return 'bg-neutral-400'
  const colors = [
    'bg-red-500', 'bg-orange-500', 'bg-amber-500', 'bg-yellow-500',
    'bg-lime-500', 'bg-green-500', 'bg-emerald-500', 'bg-teal-500',
    'bg-cyan-500', 'bg-sky-500', 'bg-blue-500', 'bg-indigo-500',
    'bg-violet-500', 'bg-purple-500', 'bg-fuchsia-500', 'bg-pink-500',
    'bg-rose-500'
  ]
  let hash = 0
  for (let i = 0; i < name.length; i++) {
    hash = name.charCodeAt(i) + ((hash << 5) - hash)
  }
  const idx = Math.abs(hash) % colors.length
  return colors[idx] || 'bg-neutral-400'
}
</script>

<template>
  <div class="relative inline-flex shrink-0">
    <UAvatar
      :src="url || ''"
      :alt="name ?? ''"
      :text="initials(name)"
      :size="size"
      :ui="{
        fallback: bgColor(name) + ' text-white text-xs font-medium'
      }"
    />
    <span
      v-if="isGroup"
      class="absolute -bottom-0.5 -right-0.5 flex size-4 items-center justify-center rounded-full bg-primary ring-2 ring-default"
    >
      <UIcon name="i-lucide-users" class="size-2.5 text-inverted" />
    </span>
  </div>
</template>
