<script setup lang="ts">
import { useContactsStore, type Contact, type ContactMeta } from '~/stores/contacts'
import { useAuthStore } from '~/stores/auth'
import type { FilterQueryPayload } from '~/components/filters/FilterBuilder.vue'

definePageMeta({ layout: 'dashboard' })

const { t } = useI18n()
const errorHandler = useErrorHandler()
const contactsStore = useContactsStore()
const auth = useAuthStore()

const PAGE_SIZE = 15

const searchQuery = ref('')
const selectedIds = ref<string[]>([])
const activeFilter = ref<FilterQueryPayload | null>(null)
let debounceTimer: ReturnType<typeof setTimeout> | null = null

// Server-side search with debounce
watch(searchQuery, () => {
  if (debounceTimer) clearTimeout(debounceTimer)
  debounceTimer = setTimeout(() => {
    activeFilter.value = null
    loadContacts({ page: 1, pageSize: PAGE_SIZE })
  }, 300)
})

// Initial load
onMounted(() => {
  loadContacts({ pageSize: PAGE_SIZE })
})

interface RawContact {
  id: string | number
  accountId?: string | number
  account_id?: string | number
  name?: string | null
  email?: string | null
  phoneNumber?: string | null
  phone_number?: string | null
  identifier?: string | null
  additionalAttributes?: string | Record<string, unknown> | null
  additional_attributes?: string | Record<string, unknown> | null
  avatarUrl?: string | null
  avatar_url?: string | null
  blocked?: boolean
  lastActivityAt?: string | null
  last_activity_at?: string | null
  createdAt?: string
  created_at?: string
  updatedAt?: string
  updated_at?: string
}

interface FilterContactsResponse {
  meta: ContactMeta
  payload: RawContact[]
}

function stringifyAttrs(value: RawContact['additionalAttributes']) {
  if (!value) return null
  return typeof value === 'string' ? value : JSON.stringify(value)
}

function normalizeContact(raw: RawContact): Contact {
  return {
    id: String(raw.id),
    accountId: String(raw.accountId ?? raw.account_id ?? ''),
    name: raw.name ?? null,
    email: raw.email ?? null,
    phoneNumber: raw.phoneNumber ?? raw.phone_number ?? null,
    identifier: raw.identifier ?? null,
    additionalAttributes: stringifyAttrs(raw.additionalAttributes ?? raw.additional_attributes),
    avatarUrl: raw.avatarUrl ?? raw.avatar_url ?? null,
    blocked: raw.blocked ?? false,
    lastActivityAt: raw.lastActivityAt ?? raw.last_activity_at ?? null,
    createdAt: raw.createdAt ?? raw.created_at ?? '',
    updatedAt: raw.updatedAt ?? raw.updated_at ?? ''
  }
}

// Computed
const visibleContactIds = computed(() => contactsStore.list.map(c => c.id))
const isSearchView = computed(() => !!searchQuery.value)
const hasMore = computed(() => {
  const { page, pageSize, total } = contactsStore.meta
  return page * pageSize < total
})

// Filter state
const deleteModalOpen = ref(false)
const filterBuilderOpen = ref(false)
const segmentSaveOpen = ref(false)
const pendingSegmentQuery = ref<FilterQueryPayload | null>(null)
const activeConditionCount = computed(() => activeFilter.value?.conditions.length ?? 0)

async function loadContacts(params: { page?: number, pageSize?: number, append?: boolean } = {}) {
  if (!activeFilter.value) {
    await contactsStore.fetchPage({
      search: searchQuery.value,
      page: params.page,
      pageSize: params.pageSize ?? PAGE_SIZE,
      append: params.append
    })
    return
  }

  const auth = useAuthStore()
  if (!auth.account?.id) return

  contactsStore.loading = true
  try {
    const api = useApi()
    const res = await api<FilterContactsResponse>(
      `/accounts/${auth.account.id}/contacts/filter`,
      {
        method: 'POST',
        body: {
          query: activeFilter.value,
          page: params.page ?? 1,
          per_page: params.pageSize ?? PAGE_SIZE
        }
      }
    )
    const normalized = res.payload.map(normalizeContact)
    contactsStore.setAll(params.append ? [...contactsStore.list, ...normalized] : normalized)
    contactsStore.meta = res.meta
  } finally {
    contactsStore.loading = false
  }
}

// Selection handlers
function toggleContact(payload: { id: string, value: boolean }) {
  if (payload.value) {
    if (!selectedIds.value.includes(payload.id)) {
      selectedIds.value = [...selectedIds.value, payload.id]
    }
  } else {
    selectedIds.value = selectedIds.value.filter(id => id !== payload.id)
  }
}

function selectAllVisible() {
  const currentSet = new Set(selectedIds.value)
  visibleContactIds.value.forEach(id => currentSet.add(id))
  selectedIds.value = Array.from(currentSet)
}

function clearSelection() {
  selectedIds.value = []
}

// Contact actions
async function handleUpdateContact(payload: { id: string, name?: string, email?: string, phone_number?: string, additional_attributes?: Record<string, unknown> }) {
  try {
    await contactsStore.update(payload.id, payload)
    errorHandler.success(t('contacts.card.updateSuccess'))
  } catch (err) {
    errorHandler.handle(err, {
      title: t('contacts.card.updateError')
    })
  }
}

async function handleDeleteContact(id: string) {
  try {
    await contactsStore.remove(id)
    errorHandler.success(t('contacts.deleteSuccess'))
  } catch (err) {
    errorHandler.handle(err, {
      title: t('contacts.deleteError')
    })
  }
}

function handleShowDetails(id: string) {
  navigateTo(`/accounts/${auth.account?.id}/contacts/${id}`)
}

// Filter handlers
function openFilterBuilder() {
  filterBuilderOpen.value = true
}

async function applyFilter(payload: FilterQueryPayload) {
  activeFilter.value = payload
  await loadContacts({ page: 1, pageSize: PAGE_SIZE })
}

function onSaveSegment(payload: FilterQueryPayload) {
  pendingSegmentQuery.value = payload
  segmentSaveOpen.value = true
}

// Bulk actions
function onDeleteFromToolbar() {
  if (!selectedIds.value.length) return
  deleteModalOpen.value = true
}

function onBulkDeleted() {
  clearSelection()
  loadContacts({ page: 1, pageSize: PAGE_SIZE })
}

// Pagination
function handlePageChange(page: number) {
  loadContacts({ page, pageSize: PAGE_SIZE })
}

// Infinite scroll (for search)
async function loadMore() {
  if (!hasMore.value || contactsStore.loading) return
  const nextPage = contactsStore.meta.page + 1
  await loadContacts({
    page: nextPage,
    pageSize: PAGE_SIZE,
    append: true
  })
}

// Import/Add handlers
function handleImport() {
  navigateTo(`/accounts/${auth.account?.id}/contacts/import`)
}

const addModalOpen = ref(false)
function handleAdd() {
  addModalOpen.value = true
}

const sendMessageOpen = ref(false)
function handleSendMessage() {
  sendMessageOpen.value = true
}

function handleSearch(event: Event) {
  searchQuery.value = (event.target as HTMLInputElement).value
}

function escapeCsv(value: string | null): string {
  if (!value) return ''
  // RFC 4180: wrap in quotes and double internal quotes
  if (value.includes('"') || value.includes(',') || value.includes('\n') || value.includes('\r')) {
    return `"${value.replace(/"/g, '""')}"`
  }
  return value
}

function handleExportCSV() {
  const contacts = contactsStore.list
  if (!contacts.length) return
  const header = 'name,email,phone_number\n'
  const rows = contacts.map(c =>
    `${escapeCsv(c.name)},${escapeCsv(c.email)},${escapeCsv(c.phoneNumber)}`
  ).join('\n')
  const blob = new Blob([header + rows], { type: 'text/csv' })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = 'contacts.csv'
  a.click()
  URL.revokeObjectURL(url)
}

const headerMenuItems = computed(() => [
  [
    { label: t('contacts.add'), icon: 'i-lucide-plus', onSelect: handleAdd },
    { label: t('contacts.import'), icon: 'i-lucide-download', onSelect: handleImport },
    { label: t('contacts.export'), icon: 'i-lucide-upload', onSelect: handleExportCSV }
  ]
])
</script>

<template>
  <UDashboardPanel id="contacts">
    <template #header>
      <UDashboardNavbar :title="t('contacts.title')">
        <template #leading>
          <UDashboardSidebarCollapse />
        </template>

        <template #right>
          <UInput
            :model-value="searchQuery"
            icon="i-lucide-search"
            :placeholder="t('contacts.search')"
            size="sm"
            class="w-64 hidden sm:block"
            @input="handleSearch"
          />

          <div class="relative">
            <UButton
              icon="i-lucide-list-filter"
              color="neutral"
              variant="ghost"
              size="sm"
              square
              :aria-label="t('contacts.filters.button')"
              @click="openFilterBuilder"
            />
            <span
              v-if="activeConditionCount > 0"
              class="absolute top-1 right-1 size-1.5 rounded-full bg-primary"
            />
          </div>

          <UDropdownMenu :items="headerMenuItems">
            <UButton
              icon="i-lucide-ellipsis-vertical"
              color="neutral"
              variant="ghost"
              size="sm"
              square
              :aria-label="t('contacts.actionsLabel')"
            />
          </UDropdownMenu>

          <div class="w-px h-4 bg-accented mx-1" />

          <UButton
            :label="t('contacts.sendMessage')"
            icon="i-lucide-message-square-plus"
            size="sm"
            @click="handleSendMessage"
          />
        </template>
      </UDashboardNavbar>

      <UDashboardToolbar class="sm:hidden">
        <template #left>
          <UInput
            :model-value="searchQuery"
            icon="i-lucide-search"
            :placeholder="t('contacts.search')"
            class="w-full"
            @input="handleSearch"
          />
        </template>
      </UDashboardToolbar>

      <ContactsBulkToolbar
        v-if="selectedIds.length"
        :selected-ids="selectedIds"
        :visible-ids="visibleContactIds"
        :total-count="contactsStore.meta.total"
        @select-all="selectAllVisible"
        @clear-selection="clearSelection"
        @delete-request="onDeleteFromToolbar"
      />
    </template>

    <template #body>
      <div class="max-w-6xl mx-auto w-full">
        <!-- Empty state -->
        <div
          v-if="!contactsStore.loading && !contactsStore.list.length"
          class="flex flex-1 items-center justify-center py-24 text-muted text-center"
        >
          <div>
            <UIcon name="i-lucide-users" class="size-12 mx-auto text-dimmed" />
            <p class="mt-2">
              {{ t('contacts.empty') }}
            </p>
          </div>
        </div>

        <!-- Loading state -->
        <div
          v-else-if="contactsStore.loading && !contactsStore.list.length"
          class="flex items-center justify-center py-12"
        >
          <UIcon name="i-lucide-loader-circle" class="size-8 animate-spin text-primary" />
        </div>

        <!-- Contacts list -->
        <ContactsCardList
          v-else
          :contacts="contactsStore.list"
          :selected-contact-ids="selectedIds"
          :loading="contactsStore.loading"
          @toggle-contact="toggleContact"
          @update-contact="handleUpdateContact"
          @delete-contact="handleDeleteContact"
          @show-details="handleShowDetails"
        />

        <!-- Infinite scroll trigger (only in search) -->
        <div
          v-if="isSearchView && hasMore && !contactsStore.loading"
          class="flex justify-center py-4"
        >
          <UButton
            :label="t('contacts.loadMore', 'Carregar mais')"
            variant="outline"
            @click="loadMore"
          />
        </div>
      </div>

      <!-- Modals (inside #body so they don't render as DashboardPanel default slot content; UModal portals to document body) -->
      <ContactsAddModal v-model:open="addModalOpen" />

      <ContactsSendMessageModal v-model:open="sendMessageOpen" />

      <ContactsDeleteModal
        v-model:open="deleteModalOpen"
        :count="selectedIds.length"
        :ids="selectedIds"
        @deleted="onBulkDeleted"
      />

      <FiltersFilterBuilder
        v-model="filterBuilderOpen"
        filter-type="contact"
        :initial-query="activeFilter"
        @apply="applyFilter"
        @save-segment="onSaveSegment"
      />

      <ContactsContactSegmentSaveModal
        v-model="segmentSaveOpen"
        :query="pendingSegmentQuery"
      />
    </template>

    <template v-if="!isSearchView && contactsStore.list.length > 0" #footer>
      <ContactsPaginationFooter
        :current-page="contactsStore.meta.page"
        :total-items="contactsStore.meta.total"
        :items-per-page="contactsStore.meta.pageSize"
        @update:page="handlePageChange"
      />
    </template>
  </UDashboardPanel>
</template>
