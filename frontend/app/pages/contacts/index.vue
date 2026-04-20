<script setup lang="ts">
import type { TableColumn } from '@nuxt/ui'
import type { Row } from '@tanstack/table-core'
import { getPaginationRowModel } from '@tanstack/table-core'
import { upperFirst } from 'scule'
import { useContactsStore, type Contact } from '~/stores/contacts'

definePageMeta({ layout: 'dashboard' })

const UAvatar = resolveComponent('UAvatar')
const UButton = resolveComponent('UButton')
const UCheckbox = resolveComponent('UCheckbox')
const UDropdownMenu = resolveComponent('UDropdownMenu')

const { t } = useI18n()
const toast = useToast()
const contactsStore = useContactsStore()
const table = useTemplateRef('table')

const columnFilters = ref([{ id: 'email', value: '' }])
const columnVisibility = ref<Record<string, boolean>>({})
const rowSelection = ref<Record<string, boolean>>({})

const searchQuery = ref('')
let debounceTimer: ReturnType<typeof setTimeout> | null = null

// Server-side search with debounce
watch(searchQuery, (val) => {
  if (debounceTimer) clearTimeout(debounceTimer)
  debounceTimer = setTimeout(() => {
    contactsStore.fetchPage({ search: val, page: 1, pageSize: 25 })
  }, 300)
})

// Initial load
onMounted(() => {
  contactsStore.fetchPage()
})

function getRowItems(row: Row<Contact>) {
  return [
    {
      type: 'label' as const,
      label: 'Actions'
    },
    {
      label: 'View details',
      icon: 'i-lucide-list',
      onSelect() {
        navigateTo(`/contacts/${row.original.id}`)
      }
    },
    {
      type: 'separator' as const
    },
    {
      label: 'Delete contact',
      icon: 'i-lucide-trash',
      color: 'error' as const,
      onSelect() {
        toast.add({ title: 'Contact deleted', description: 'This action is not yet implemented via bulk.', color: 'warning' })
      }
    }
  ]
}

const columns: TableColumn<Contact>[] = [
  {
    id: 'select',
    header: ({ table }) =>
      h(UCheckbox, {
        'modelValue': table.getIsSomePageRowsSelected()
          ? 'indeterminate'
          : table.getIsAllPageRowsSelected(),
        'onUpdate:modelValue': (value: boolean | 'indeterminate') =>
          table.toggleAllPageRowsSelected(!!value),
        'ariaLabel': 'Select all'
      }),
    cell: ({ row }) =>
      h(UCheckbox, {
        'modelValue': row.getIsSelected(),
        'onUpdate:modelValue': (value: boolean | 'indeterminate') => row.toggleSelected(!!value),
        'ariaLabel': 'Select row'
      })
  },
  {
    accessorKey: 'name',
    header: t('contacts.columns.name'),
    cell: ({ row }) => {
      const c = row.original
      const initials = (c.name ?? '?').split(' ').map(w => w[0]).join('').toUpperCase().slice(0, 2)
      return h('div', { class: 'flex items-center gap-3' }, [
        h(UAvatar, { text: initials, size: 'lg' }),
        h('div', undefined, [
          h('p', { class: 'font-medium text-highlighted' }, c.name ?? '—'),
          h('p', { class: 'text-xs text-muted' }, c.email ?? '—')
        ])
      ])
    }
  },
  {
    accessorKey: 'email',
    header: ({ column }) => {
      const isSorted = column.getIsSorted()
      return h(UButton, {
        color: 'neutral',
        variant: 'ghost',
        label: t('contacts.columns.email'),
        icon: isSorted
          ? isSorted === 'asc'
            ? 'i-lucide-arrow-up-narrow-wide'
            : 'i-lucide-arrow-down-wide-narrow'
          : 'i-lucide-arrow-up-down',
        class: '-mx-2.5',
        onClick: () => column.toggleSorting(column.getIsSorted() === 'asc')
      })
    }
  },
  {
    accessorKey: 'phoneNumber',
    header: t('contacts.columns.phone'),
    cell: ({ row }) => row.original.phoneNumber ?? '—'
  },
  {
    accessorKey: 'identifier',
    header: 'Identifier',
    cell: ({ row }) => row.original.identifier ?? '—'
  },
  {
    accessorKey: 'createdAt',
    header: t('contacts.columns.created'),
    cell: ({ row }) => new Date(row.original.createdAt).toLocaleDateString()
  },
  {
    id: 'actions',
    cell: ({ row }) => {
      return h(
        'div',
        { class: 'text-right' },
        h(
          UDropdownMenu,
          {
            content: { align: 'end' },
            items: getRowItems(row)
          },
          () =>
            h(UButton, {
              icon: 'i-lucide-ellipsis-vertical',
              color: 'neutral',
              variant: 'ghost',
              class: 'ml-auto'
            })
        )
      )
    }
  }
]

const selectedIds = computed(() => Object.keys(rowSelection.value))
const pagination = ref({ pageIndex: 0, pageSize: 25 })

function clearSelection() {
  rowSelection.value = {}
}
</script>

<template>
  <UDashboardPanel id="contacts">
    <template #header>
      <UDashboardNavbar :title="t('contacts.title')">
        <template #leading>
          <UDashboardSidebarCollapse />
        </template>

        <template #right>
          <ContactsBulkToolbar
            v-if="selectedIds.length"
            :selected-ids="selectedIds"
            @clear-selection="clearSelection"
          />

          <ContactsDeleteModal :count="selectedIds.length">
            <UButton
              v-if="selectedIds.length"
              label="Delete"
              color="error"
              variant="subtle"
              icon="i-lucide-trash"
            >
              <template #trailing>
                <UKbd>{{ selectedIds.length }}</UKbd>
              </template>
            </UButton>
          </ContactsDeleteModal>

          <UButton
            :label="t('contacts.import')"
            icon="i-lucide-upload"
            color="neutral"
            variant="outline"
            to="/contacts/import"
          />

          <ContactsAddModal />
        </template>
      </UDashboardNavbar>
    </template>

    <template #body>
      <div class="flex flex-wrap items-center justify-between gap-1.5">
        <UInput
          v-model="searchQuery"
          class="max-w-sm"
          icon="i-lucide-search"
          :placeholder="t('contacts.search')"
        />

        <div class="flex flex-wrap items-center gap-1.5">
          <UDropdownMenu
            :items="
              table?.tableApi
                ?.getAllColumns()
                .filter((column: any) => column.getCanHide())
                .map((column: any) => ({
                  label: upperFirst(column.id),
                  type: 'checkbox' as const,
                  checked: column.getIsVisible(),
                  onUpdateChecked(checked: boolean) {
                    table?.tableApi?.getColumn(column.id)?.toggleVisibility(!!checked)
                  },
                  onSelect(e?: Event) {
                    e?.preventDefault()
                  }
                }))
            "
            :content="{ align: 'end' }"
          >
            <UButton
              label="Display"
              color="neutral"
              variant="outline"
              trailing-icon="i-lucide-settings-2"
            />
          </UDropdownMenu>
        </div>
      </div>

      <div v-if="!contactsStore.loading && !contactsStore.list.length" class="flex flex-1 items-center justify-center py-24 text-muted text-center">
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
        v-model:column-visibility="columnVisibility"
        v-model:row-selection="rowSelection"
        v-model:pagination="pagination"
        :pagination-options="{ getPaginationRowModel: getPaginationRowModel() }"
        class="shrink-0"
        :data="contactsStore.list"
        :columns="columns"
        :loading="contactsStore.loading"
        :ui="{
          base: 'table-fixed border-separate border-spacing-0',
          thead: '[&>tr]:bg-elevated/50 [&>tr]:after:content-none',
          tbody: '[&>tr]:last:[&>td]:border-b-0',
          th: 'py-2 first:rounded-l-lg last:rounded-r-lg border-y border-default first:border-l last:border-r',
          td: 'border-b border-default',
          separator: 'h-0'
        }"
      />

      <div class="flex items-center justify-between gap-3 border-t border-default pt-4 mt-auto">
        <div class="text-sm text-muted">
          {{ selectedIds.length || 0 }} of
          {{ contactsStore.meta.total }} contact(s) total.
        </div>

        <div class="flex items-center gap-1.5">
          <UPagination
            :default-page="(table?.tableApi?.getState().pagination.pageIndex || 0) + 1"
            :items-per-page="table?.tableApi?.getState().pagination.pageSize"
            :total="contactsStore.meta.total"
            @update:page="(p: number) => table?.tableApi?.setPageIndex(p - 1)"
          />
        </div>
      </div>
    </template>
  </UDashboardPanel>
</template>
