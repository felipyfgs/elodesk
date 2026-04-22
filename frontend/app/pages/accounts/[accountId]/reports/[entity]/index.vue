<script setup lang="ts">
import EntityList from '~/components/reports/entity/EntityList.vue'
import type { EntityMetric } from '~/types/reports'
import { useApi } from '~/composables/useApi'
import { useAuthStore } from '~/stores/auth'

definePageMeta({ layout: 'dashboard', middleware: 'reports-entity' })

const route = useRoute()
const { t: _t } = useI18n()
const api = useApi()
const auth = useAuthStore()

const entity = computed(() => String(route.params.entity))
const items = ref<EntityMetric[]>([])
const loading = ref(false)

async function load() {
  if (!auth.account?.id) return
  loading.value = true
  try {
    items.value = await api<EntityMetric[]>(`/accounts/${auth.account.id}/reports/${entity.value}`)
  } finally {
    loading.value = false
  }
}

watch(entity, load)
onMounted(load)
</script>

<template>
  <UDashboardPanel :id="`reports-${entity}`">
    <template #header>
      <UDashboardNavbar :title="entity">
        <template #leading>
          <UDashboardSidebarCollapse />
        </template>
      </UDashboardNavbar>
    </template>
    <template #body>
      <div class="max-w-6xl mx-auto w-full">
        <EntityList :items="items" :entity="entity" />
      </div>
    </template>
  </UDashboardPanel>
</template>
