<script setup lang="ts">
const { operationLinks, resourceLinks, adminLinks, isSidebarOpen } = useDashboard()
</script>

<template>
  <UDashboardSidebar
    id="dashboard"
    v-model:open="isSidebarOpen"
    collapsible
    resizable
    class="bg-elevated/25"
    :ui="{ footer: 'lg:border-t lg:border-default' }"
  >
    <template #header="{ collapsed }">
      <AccountMenu :collapsed="collapsed" />
    </template>

    <template #default="{ collapsed }">
      <UDashboardSearchButton :collapsed="collapsed" class="bg-transparent ring-default" />

      <div class="flex flex-col gap-1">
        <DashboardSidebarNavGroup
          :items="operationLinks"
          :collapsed="collapsed"
        />

        <DashboardSidebarNavGroup
          :items="resourceLinks"
          :collapsed="collapsed"
        />

        <DashboardSidebarNavGroup
          :items="adminLinks"
          :collapsed="collapsed"
        />
      </div>
    </template>

    <template #footer="{ collapsed }">
      <UserMenu :collapsed="collapsed" />
    </template>
  </UDashboardSidebar>
</template>
