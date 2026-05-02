<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { usePipelinesStore, type Pipeline } from '~/stores/pipelines'

definePageMeta({ layout: 'dashboard' })

const { t } = useI18n()
const pipelinesStore = usePipelinesStore()
const errorHandler = useErrorHandler()
const route = useRoute()

const aid = computed(() => route.params.accountId as string)

const templatesOpen = ref(false)
const fetching = ref(false)

async function fetchAll() {
  fetching.value = true
  try {
    await pipelinesStore.fetchAll()
  } catch (err) {
    errorHandler.handle(err, { title: t('pipelines.fetchFailed'), onRetry: fetchAll })
  } finally {
    fetching.value = false
  }
}

onMounted(fetchAll)

function onCreated(pipeline: Pipeline) {
  void navigateTo(`/accounts/${aid.value}/pipelines/${pipeline.id}`)
}

const list = computed<Pipeline[]>(() => pipelinesStore.activeList)
</script>

<template>
  <UDashboardPanel id="pipelines" :ui="{ body: 'lg:py-8' }">
    <template #header>
      <UDashboardNavbar :title="t('pipelines.title')">
        <template #leading>
          <UDashboardSidebarCollapse />
        </template>
        <template #right>
          <UButton
            icon="i-lucide-plus"
            @click="templatesOpen = true"
          >
            {{ t('pipelines.new') }}
          </UButton>
        </template>
      </UDashboardNavbar>
    </template>

    <template #body>
      <div class="max-w-6xl mx-auto w-full flex flex-col gap-4 sm:gap-6 lg:gap-8">
        <div v-if="fetching" class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
          <USkeleton v-for="i in 3" :key="i" class="h-28 w-full" />
        </div>

        <div
          v-else-if="!list.length"
          class="flex flex-col items-center text-center gap-2 py-12"
        >
          <UIcon name="i-lucide-kanban-square" class="size-10 text-muted" />
          <p class="text-sm text-muted">
            {{ t('pipelines.empty') }}
          </p>
          <UButton
            variant="ghost"
            icon="i-lucide-plus"
            @click="templatesOpen = true"
          >
            {{ t('pipelines.new') }}
          </UButton>
        </div>

        <div v-else class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
          <NuxtLink
            v-for="pipeline in list"
            :key="pipeline.id"
            :to="`/accounts/${aid}/pipelines/${pipeline.id}`"
            class="block"
          >
            <UCard class="hover:shadow-md transition-shadow h-full">
              <div class="flex items-start gap-3">
                <UIcon
                  :name="pipeline.icon || 'i-lucide-kanban-square'"
                  class="size-6 shrink-0"
                  :style="{ color: pipeline.color }"
                />
                <div class="min-w-0 flex-1">
                  <h3 class="text-sm font-semibold text-default truncate">
                    {{ pipeline.name }}
                  </h3>
                  <p v-if="pipeline.description" class="text-xs text-muted line-clamp-2 mt-1">
                    {{ pipeline.description }}
                  </p>
                  <div class="flex flex-wrap gap-1 mt-2">
                    <UBadge
                      v-for="stage in (pipeline.stages ?? []).slice(0, 4)"
                      :key="stage.id"
                      size="sm"
                      variant="subtle"
                      color="neutral"
                    >
                      {{ stage.name }}
                    </UBadge>
                    <UBadge
                      v-if="(pipeline.stages?.length ?? 0) > 4"
                      size="sm"
                      variant="subtle"
                      color="neutral"
                    >
                      +{{ (pipeline.stages?.length ?? 0) - 4 }}
                    </UBadge>
                  </div>
                </div>
              </div>
            </UCard>
          </NuxtLink>
        </div>

        <PipelinesTemplatesModal
          v-model:open="templatesOpen"
          @created="onCreated"
        />
      </div>
    </template>
  </UDashboardPanel>
</template>
