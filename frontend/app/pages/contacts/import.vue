<script setup lang="ts">
import { useAuthStore } from '~/stores/auth'

definePageMeta({ layout: 'dashboard' })

const { t } = useI18n()
const api = useApi()
const auth = useAuthStore()
const toast = useToast()

const currentStep = ref<'upload' | 'preview' | 'result'>('upload')
const file = ref<File | null>(null)
const preview = ref<string[][]>([])
const importing = ref(false)
const importResult = ref<{
  inserted: number
  updated: number
  totalRows: number
  errors?: Array<{ row: number, reason: string }>
} | null>(null)

function onFileSelected(f: File) {
  file.value = f
  parsePreview(f)
}

async function parsePreview(f: File) {
  const text = await f.text()
  const lines = text.split('\n').filter(l => l.trim())
  if (lines.length < 2) return

  const firstLine = lines[0]
  if (!firstLine) return
  const header = parseCSVLine(firstLine)
  const dataLines = lines.slice(1, 11) // First 10 data rows
  preview.value = [header, ...dataLines.map(parseCSVLine)]
  currentStep.value = 'preview'
}

function parseCSVLine(line: string): string[] {
  const result: string[] = []
  let current = ''
  let inQuotes = false
  for (const char of line) {
    if (char === '"') {
      inQuotes = !inQuotes
    } else if (char === ',' && !inQuotes) {
      result.push(current.trim())
      current = ''
    } else {
      current += char
    }
  }
  result.push(current.trim())
  return result
}

async function doImport() {
  if (!file.value || !auth.account?.id) return
  importing.value = true
  try {
    const formData = new FormData()
    formData.append('file', file.value)
    const res = await api<{
      inserted: number
      updated: number
      totalRows: number
      errors?: Array<{ row: number, reason: string }>
    }>(`/accounts/${auth.account.id}/contacts/import`, {
      method: 'POST',
      body: file.value
    })
    importResult.value = res
    currentStep.value = 'result'
    toast.add({ title: 'Import complete', description: `${res.inserted} inserted, ${res.updated} updated`, color: 'success' })
  } catch {
    toast.add({ title: 'Import failed', color: 'error' })
  } finally {
    importing.value = false
  }
}

function reset() {
  currentStep.value = 'upload'
  file.value = null
  preview.value = []
  importResult.value = null
}
</script>

<template>
  <UDashboardPanel id="contacts-import">
    <template #header>
      <UDashboardNavbar :title="t('contacts.import')">
        <template #leading>
          <UButton
            icon="i-lucide-arrow-left"
            color="neutral"
            variant="ghost"
            to="/contacts"
          />
        </template>
      </UDashboardNavbar>
    </template>

    <template #body>
      <div class="max-w-3xl mx-auto space-y-6">
        <!-- Step 1: Upload -->
        <div v-if="currentStep === 'upload'">
          <ContactsImportDropzone @file-selected="onFileSelected" />
        </div>

        <!-- Step 2: Preview -->
        <div v-else-if="currentStep === 'preview'" class="space-y-4">
          <div class="flex items-center justify-between">
            <h3 class="text-sm font-medium">
              Preview (first 10 rows)
            </h3>
            <UButton
              color="neutral"
              variant="ghost"
              size="xs"
              @click="reset"
            >
              {{ t('common.cancel') }}
            </UButton>
          </div>

          <UTable
            :data="preview.slice(1).map(row => {
              const obj: Record<string, string> = {}
              const hdr = preview[0]
              if (hdr) hdr.forEach((h, i) => { obj[h] = row[i] ?? '' })
              return obj
            })"
            :columns="(preview[0] ?? []).map(h => ({ accessorKey: h, header: h }))"
            :ui="{
              base: 'table-fixed border-separate border-spacing-0',
              thead: '[&>tr]:bg-elevated/50 [&>tr]:after:content-none',
              th: 'py-2 first:rounded-l-lg last:rounded-r-lg border-y border-default first:border-l last:border-r',
              td: 'border-b border-default',
              separator: 'h-0'
            }"
          />

          <div class="flex justify-end">
            <UButton :loading="importing" @click="doImport">
              {{ t('contacts.import') }}
            </UButton>
          </div>
        </div>

        <!-- Step 3: Result -->
        <div v-else-if="currentStep === 'result'">
          <ContactsImportReport :result="importResult" />

          <div class="flex justify-end gap-2 mt-6">
            <UButton color="neutral" variant="ghost" @click="reset">
              Import more
            </UButton>
            <UButton to="/contacts">
              Back to contacts
            </UButton>
          </div>
        </div>
      </div>
    </template>
  </UDashboardPanel>
</template>
