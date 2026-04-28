<script setup lang="ts">
const emit = defineEmits<{
  'file-selected': [file: File]
}>()

const { t } = useI18n()
const isDragging = ref(false)
const fileInput = ref<HTMLInputElement | null>(null)

const MAX_SIZE = 10 * 1024 * 1024 // 10 MB

function handleDrop(e: DragEvent) {
  isDragging.value = false
  const file = e.dataTransfer?.files[0]
  if (file && file.name.endsWith('.csv') && file.size <= MAX_SIZE) {
    emit('file-selected', file)
  }
}

function handleFileSelect(e: Event) {
  const input = e.target as HTMLInputElement
  const file = input.files?.[0]
  if (file) {
    emit('file-selected', file)
  }
}
</script>

<template>
  <div
    class="border-2 border-dashed rounded-xl p-12 text-center transition-colors cursor-pointer"
    :class="isDragging ? 'border-primary bg-primary/5' : 'border-[var(--ui-border)] hover:border-primary/50'"
    @dragover.prevent="isDragging = true"
    @dragleave.prevent="isDragging = false"
    @drop.prevent="handleDrop"
    @click="fileInput?.click()"
  >
    <input
      ref="fileInput"
      type="file"
      accept=".csv"
      class="hidden"
      @change="handleFileSelect"
    >
    <UIcon name="i-lucide-upload-cloud" class="size-12 mx-auto text-dimmed mb-4" />
    <p class="text-sm font-medium">
      {{ t('contacts.importDrop') }}
    </p>
    <p class="text-xs text-muted mt-1">
      CSV, max 10 MB
    </p>
  </div>
</template>
