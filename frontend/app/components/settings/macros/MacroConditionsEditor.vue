<script setup lang="ts">
const props = defineProps<{ modelValue: Record<string, unknown> }>()
const emit = defineEmits<{ 'update:modelValue': [value: Record<string, unknown>] }>()

const text = computed({
  get: () => JSON.stringify(props.modelValue ?? {}, null, 2),
  set: (v: string) => {
    try {
      const parsed = v ? JSON.parse(v) : {}
      emit('update:modelValue', parsed)
    } catch {
      // ignore invalid JSON until fixed
    }
  }
})
</script>

<template>
  <UTextarea
    v-model="text"
    :rows="5"
    class="font-mono text-xs"
    placeholder="{}"
  />
</template>
