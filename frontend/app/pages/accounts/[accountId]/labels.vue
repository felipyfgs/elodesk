<script setup lang="ts">
definePageMeta({ layout: 'dashboard' })

const { t } = useI18n()
</script>

<template>
  <UDashboardPanel id="labels" :ui="{ body: 'lg:py-8' }">
    <template #header>
      <UDashboardNavbar :title="t('labels.title')">
        <template #leading>
          <UDashboardSidebarCollapse />
        </template>
      </UDashboardNavbar>
    </template>

    <template #body>
      <div class="flex flex-col gap-4 sm:gap-6 lg:gap-8 max-w-6xl mx-auto w-full">
        <LabelsManager />
      </div>
    </template>
  </UDashboardPanel>
</template>
