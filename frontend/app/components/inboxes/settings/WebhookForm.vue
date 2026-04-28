<script setup lang="ts">
const { t } = useI18n()

const webhooks = ref<Array<{ id: string, url: string, events: string[] }>>([])
const showForm = ref(false)

const newWebhook = reactive({
  url: '',
  events: ['message.created']
})

function addWebhook() {
  if (!newWebhook.url) return
  webhooks.value.push({
    id: crypto.randomUUID(),
    url: newWebhook.url,
    events: [...newWebhook.events]
  })
  newWebhook.url = ''
  showForm.value = false
}

function removeWebhook(id: string) {
  webhooks.value = webhooks.value.filter(w => w.id !== id)
}
</script>

<template>
  <div class="flex flex-col gap-4">
    <div class="flex items-center justify-between">
      <p class="text-sm text-muted">
        {{ t('inboxes.webhookDescription') }}
      </p>
      <UButton
        icon="i-lucide-plus"
        size="sm"
        @click="showForm = !showForm"
      >
        {{ t('inboxes.webhookAdd') }}
      </UButton>
    </div>

    <div v-if="showForm" class="rounded-lg border border-[var(--ui-border-accented)] p-4 flex flex-col gap-3">
      <UFormField :label="t('inboxes.webhookUrl')">
        <UInput v-model="newWebhook.url" placeholder="https://example.com/webhook" class="w-full" />
      </UFormField>
      <div class="flex justify-end gap-2">
        <UButton
          variant="ghost"
          color="neutral"
          size="sm"
          @click="showForm = false"
        >
          {{ t('common.cancel') }}
        </UButton>
        <UButton size="sm" @click="addWebhook">
          {{ t('common.create') }}
        </UButton>
      </div>
    </div>

    <div v-if="webhooks.length" class="flex flex-col gap-2">
      <div
        v-for="webhook in webhooks"
        :key="webhook.id"
        class="flex items-center gap-3 rounded-lg bg-[var(--ui-bg-accented)] px-3 py-2"
      >
        <UIcon name="i-lucide-webhook" class="size-4 text-muted shrink-0" />
        <code class="text-xs font-mono truncate flex-1">{{ webhook.url }}</code>
        <UButton
          icon="i-lucide-trash-2"
          variant="ghost"
          color="error"
          size="xs"
          @click="removeWebhook(webhook.id)"
        />
      </div>
    </div>

    <p v-else class="text-sm text-muted text-center py-4">
      {{ t('inboxes.noWebhooks') }}
    </p>
  </div>
</template>
