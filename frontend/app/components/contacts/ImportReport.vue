<script setup lang="ts">
interface ImportError {
  row: number
  reason: string
}

interface ImportResult {
  inserted: number
  updated: number
  totalRows: number
  errors?: ImportError[]
}

const props = defineProps<{
  result: ImportResult | null
}>()

void props

function downloadErrorsCSV() {
  if (!props.result?.errors?.length) return
  const header = 'row,reason\n'
  const rows = props.result.errors.map(e => `${e.row},"${e.reason}"`).join('\n')
  const blob = new Blob([header + rows], { type: 'text/csv' })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = 'import-errors.csv'
  a.click()
  URL.revokeObjectURL(url)
}
</script>

<template>
  <div v-if="result" class="space-y-4">
    <div class="grid grid-cols-3 gap-4">
      <UPageCard variant="outline">
        <p class="text-2xl font-bold text-success">
          {{ result.inserted }}
        </p>
        <p class="text-sm text-muted">
          Inserted
        </p>
      </UPageCard>
      <UPageCard variant="outline">
        <p class="text-2xl font-bold text-info">
          {{ result.updated }}
        </p>
        <p class="text-sm text-muted">
          Updated
        </p>
      </UPageCard>
      <UPageCard variant="outline">
        <p class="text-2xl font-bold" :class="result.errors?.length ? 'text-error' : 'text-success'">
          {{ result.errors?.length ?? 0 }}
        </p>
        <p class="text-sm text-muted">
          Errors
        </p>
      </UPageCard>
    </div>

    <div v-if="result.errors?.length" class="space-y-2">
      <div class="flex items-center justify-between">
        <h3 class="text-sm font-medium">
          Import errors
        </h3>
        <UButton
          label="Download errors CSV"
          icon="i-lucide-download"
          size="xs"
          color="neutral"
          variant="ghost"
          @click="downloadErrorsCSV"
        />
      </div>
      <UTable
        :data="result.errors"
        :columns="[
          { accessorKey: 'row', header: 'Row' },
          { accessorKey: 'reason', header: 'Reason' }
        ]"
        :ui="{
          base: 'table-fixed border-separate border-spacing-0',
          thead: '[&>tr]:bg-elevated/50 [&>tr]:after:content-none',
          th: 'py-2 first:rounded-l-lg last:rounded-r-lg border-y border-default first:border-l last:border-r',
          td: 'border-b border-default',
          separator: 'h-0'
        }"
      />
    </div>
  </div>
</template>
