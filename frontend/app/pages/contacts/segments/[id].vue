<script setup lang="ts">
import { useSavedFiltersStore } from '~/stores/savedFilters'

definePageMeta({ layout: 'dashboard' })

const route = useRoute()
const router = useRouter()
const store = useSavedFiltersStore()

const filterId = route.params.id as string

onMounted(() => {
  const filter = store.list.find(f => f.id === filterId)
  if (!filter) {
    router.replace('/contacts')
  }
})
</script>

<template>
  <NuxtPage />
</template>
