<script setup lang="ts">
import type { TableColumn } from '@nuxt/ui'
import { getPaginationRowModel } from '@tanstack/table-core'
import { useAuthStore } from '~/stores/auth'
import type { ContactRow } from '~/types'

const UAvatar = resolveComponent('UAvatar')
const UBadge = resolveComponent('UBadge')
const UButton = resolveComponent('UButton')

const { t } = useI18n()
const api = useApi()
const auth = useAuthStore()

interface TableRef {
  tableApi?: {
    getColumn: (id: string) => { getFilterValue: () => unknown, setFilterValue: (v: unknown) => void } | undefined
    getState: () => { pagination: { pageIndex: number, pageSize: number } }
    getFilteredRowModel: () => { rows: unknown[] }
    setPageIndex: (p: number) => void
  }
}
const table = useTemplateRef<TableRef>('table')
const loading = ref(false)
const rows = ref<ContactRow[]>([])
const columnFilters = ref([{ id: 'name', value: '' }])
const pagination = ref({ pageIndex: 0, pageSize: 10 })

async function load() {
  if (!auth.account?.id) return
  loading.value = true
  try {
    // Deriva lista de contatos a partir das conversations (backend ainda não expõe /contacts).
    const convs = await api<Array<{
      id: string
      contactInbox?: { contact?: { id: string, name: string | null, phoneNumber: string | null, waJid: string | null, avatarUrl: string | null, createdAt: string } }
    }>>(`/accounts/${auth.account.id}/conversations`)
    const map = new Map<string, ContactRow>()
    for (const c of convs) {
      const ct = c.contactInbox?.contact
      if (!ct) continue
      if (map.has(ct.id)) continue
      map.set(ct.id, {
        id: ct.id,
        name: ct.name,
        phoneNumber: ct.phoneNumber,
        waJid: ct.waJid,
        avatarUrl: ct.avatarUrl,
        createdAt: ct.createdAt,
        status: 'active'
      })
    }
    rows.value = Array.from(map.values())
  } finally {
    loading.value = false
  }
}

onMounted(load)

const columns: TableColumn<ContactRow>[] = [
  {
    accessorKey: 'name',
    header: t('contacts.columns.name'),
    cell: ({ row }) => h('div', { class: 'flex items-center gap-3' }, [
      h(UAvatar, { src: row.original.avatarUrl ?? undefined, alt: row.original.name ?? '—', size: 'md' }),
      h('div', undefined, [
        h('p', { class: 'font-medium text-highlighted' }, row.original.name ?? '—'),
        h('p', { class: 'text-xs text-muted' }, row.original.waJid ?? '—')
      ])
    ])
  },
  {
    accessorKey: 'phoneNumber',
    header: t('contacts.columns.phone'),
    cell: ({ row }) => row.original.phoneNumber ?? '—'
  },
  {
    accessorKey: 'waJid',
    header: t('contacts.columns.waJid'),
    cell: ({ row }) => row.original.waJid ?? '—'
  },
  {
    accessorKey: 'status',
    header: 'Status',
    cell: ({ row }) => h(UBadge, { variant: 'subtle', color: 'success' }, () => row.original.status)
  },
  {
    accessorKey: 'createdAt',
    header: t('contacts.columns.created'),
    cell: ({ row }) => new Date(row.original.createdAt).toLocaleDateString()
  },
  {
    id: 'actions',
    cell: () => h('div', { class: 'text-right' }, h(UButton, { icon: 'i-lucide-ellipsis-vertical', color: 'neutral', variant: 'ghost', class: 'ml-auto' }))
  }
]

const search = computed<string>({
  get: () => (table.value?.tableApi?.getColumn('name')?.getFilterValue() as string) ?? '',
  set: (v: string) => table.value?.tableApi?.getColumn('name')?.setFilterValue(v || undefined)
})
</script>

<template>
  <UDashboardPanel id="contacts">
    <template #header>
      <UDashboardNavbar :title="t('contacts.title')">
        <template #leading>
          <UDashboardSidebarCollapse />
        </template>
        <template #trailing>
          <UBadge :label="rows.length" variant="subtle" />
        </template>
      </UDashboardNavbar>
    </template>

    <template #body>
      <div class="flex items-center justify-between gap-3 mb-4">
        <UInput
          v-model="search"
          class="max-w-sm"
          icon="i-lucide-search"
          :placeholder="t('contacts.search')"
        />
      </div>

      <div v-if="!loading && !rows.length" class="flex flex-1 items-center justify-center py-24 text-muted text-center">
        <div>
          <UIcon name="i-lucide-users" class="size-12 mx-auto text-dimmed" />
          <p class="mt-2">
            {{ t('contacts.empty') }}
          </p>
        </div>
      </div>

      <UTable
        v-else
        ref="table"
        v-model:column-filters="columnFilters"
        v-model:pagination="pagination"
        :pagination-options="{ getPaginationRowModel: getPaginationRowModel() }"
        class="shrink-0"
        :data="rows"
        :columns="columns"
        :loading="loading"
        :ui="{
          base: 'table-fixed border-separate border-spacing-0',
          thead: '[&>tr]:bg-elevated/50 [&>tr]:after:content-none',
          tbody: '[&>tr]:last:[&>td]:border-b-0',
          th: 'py-2 first:rounded-l-lg last:rounded-r-lg border-y border-default first:border-l last:border-r',
          td: 'border-b border-default',
          separator: 'h-0'
        }"
      />

      <div v-if="rows.length" class="flex items-center justify-end mt-4">
        <UPagination
          :default-page="(table?.tableApi?.getState().pagination.pageIndex ?? 0) + 1"
          :items-per-page="table?.tableApi?.getState().pagination.pageSize ?? 10"
          :total="table?.tableApi?.getFilteredRowModel().rows.length ?? 0"
          @update:page="(p: number) => table?.tableApi?.setPageIndex(p - 1)"
        />
      </div>
    </template>
  </UDashboardPanel>
</template>
