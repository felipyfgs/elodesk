<script setup lang="ts">
import { useInboxesStore, type Inbox } from '~/stores/inboxes'
import { useAuthStore } from '~/stores/auth'

const { t } = useI18n()
const api = useApi()
const rt = useRealtime()
const auth = useAuthStore()
const inboxes = useInboxesStore()

const creating = ref(false)
const newName = ref('')
const credentials = ref<{ inbox: Inbox, identifier: string, apiToken: string, hmacToken: string } | null>(null)
const copiedField = ref<string | null>(null)
const toast = useToast()

function copyToClipboard(text: string, field: string) {
  navigator.clipboard.writeText(text)
  copiedField.value = field
  setTimeout(() => { copiedField.value = null }, 2000)
}

async function loadInboxes() {
  if (!auth.account?.id) return
  inboxes.loading = true
  try {
    const list = await api<Inbox[]>(`/accounts/${auth.account.id}/inboxes`)
    inboxes.setAll(list)
  } finally {
    inboxes.loading = false
  }
}

async function createInbox() {
  if (!auth.account?.id || !newName.value) return
  creating.value = true
  try {
    const res = await api<{ inbox: Inbox, identifier: string, apiToken: string, hmacToken: string }>(
      `/accounts/${auth.account.id}/inboxes`,
      { method: 'POST', body: { name: newName.value, channelType: 'API' } }
    )
    inboxes.upsert(res.inbox)
    newName.value = ''
    credentials.value = res
  } catch {
    toast.add({ title: t('sessions.createFailed'), color: 'error' })
  } finally {
    creating.value = false
  }
}

async function removeInbox(inbox: Inbox) {
  if (!auth.account?.id) return
  await api(`/accounts/${auth.account.id}/inboxes/${inbox.id}`, { method: 'DELETE' })
  inboxes.list = inboxes.list.filter(i => i.id !== inbox.id)
}

onMounted(async () => {
  await loadInboxes()
  if (auth.account?.id) rt.joinAccount(auth.account.id)
})
</script>

<template>
  <UDashboardPanel id="sessions">
    <template #header>
      <UDashboardNavbar :title="t('sessions.title')" :ui="{ right: 'gap-2' }">
        <template #leading>
          <UDashboardSidebarCollapse />
        </template>
        <template #trailing>
          <UBadge :label="inboxes.list.length" variant="subtle" />
        </template>
        <template #right>
          <UInput
            v-model="newName"
            :placeholder="t('sessions.name')"
            icon="i-lucide-webhook"
            size="sm"
            class="w-48"
          />
          <UButton
            :loading="creating"
            :disabled="!newName"
            icon="i-lucide-plus"
            :label="t('sessions.new')"
            @click="createInbox"
          />
        </template>
      </UDashboardNavbar>
    </template>

    <template #body>
      <div v-if="!inboxes.list.length" class="flex flex-1 items-center justify-center py-24 text-muted">
        <div class="text-center">
          <UIcon name="i-lucide-webhook" class="size-12 mx-auto text-dimmed" />
          <p class="mt-2">
            {{ t('sessions.empty') }}
          </p>
        </div>
      </div>

      <UPageGrid v-else class="lg:grid-cols-2 xl:grid-cols-3 gap-4">
        <UPageCard
          v-for="inbox in inboxes.list"
          :key="inbox.id"
          variant="subtle"
          :ui="{ container: 'gap-3', wrapper: 'items-start', leading: 'p-2.5 rounded-full bg-primary/10 ring ring-inset ring-primary/25' }"
          icon="i-lucide-webhook"
        >
          <template #title>
            <div class="flex items-center gap-2">
              <span class="font-medium">{{ inbox.name }}</span>
              <UBadge color="success" variant="subtle" size="xs">
                API
              </UBadge>
            </div>
          </template>
          <template #description>
            <span v-if="inbox.channelApi?.identifier" class="text-xs text-muted font-mono">
              {{ inbox.channelApi.identifier }}
            </span>
          </template>

          <div class="flex flex-wrap gap-1 mt-3">
            <UButton
              size="xs"
              color="error"
              variant="soft"
              icon="i-lucide-trash"
              :label="t('sessions.actions.delete')"
              @click="removeInbox(inbox)"
            />
          </div>
        </UPageCard>
      </UPageGrid>

      <UModal v-model:open="!!credentials" :close="() => credentials = null">
        <template #content>
          <div v-if="credentials" class="p-6 flex flex-col gap-4">
            <h2 class="text-lg font-semibold">
              {{ t('sessions.credentials.title') }}
            </h2>
            <p class="text-sm text-muted">
              {{ t('sessions.credentials.warning') }}
            </p>

            <div class="space-y-3">
              <div>
                <label class="text-xs font-medium text-muted">Identifier</label>
                <div class="flex items-center gap-2 mt-1">
                  <code class="flex-1 rounded bg-elevated px-3 py-1.5 text-sm font-mono break-all">
                    {{ credentials.identifier }}
                  </code>
                  <UButton
                    size="xs"
                    icon="i-lucide-copy"
                    :color="copiedField === 'identifier' ? 'success' : 'neutral'"
                    @click="copyToClipboard(credentials.identifier, 'identifier')"
                  />
                </div>
              </div>

              <div>
                <label class="text-xs font-medium text-muted">API Token</label>
                <div class="flex items-center gap-2 mt-1">
                  <code class="flex-1 rounded bg-elevated px-3 py-1.5 text-sm font-mono break-all">
                    {{ credentials.apiToken }}
                  </code>
                  <UButton
                    size="xs"
                    icon="i-lucide-copy"
                    :color="copiedField === 'apiToken' ? 'success' : 'neutral'"
                    @click="copyToClipboard(credentials.apiToken, 'apiToken')"
                  />
                </div>
              </div>

              <div>
                <label class="text-xs font-medium text-muted">HMAC Token</label>
                <div class="flex items-center gap-2 mt-1">
                  <code class="flex-1 rounded bg-elevated px-3 py-1.5 text-sm font-mono break-all">
                    {{ credentials.hmacToken }}
                  </code>
                  <UButton
                    size="xs"
                    icon="i-lucide-copy"
                    :color="copiedField === 'hmacToken' ? 'success' : 'neutral'"
                    @click="copyToClipboard(credentials.hmacToken, 'hmacToken')"
                  />
                </div>
              </div>
            </div>

            <div class="flex justify-end mt-2">
              <UButton label="Close" @click="credentials = null" />
            </div>
          </div>
        </template>
      </UModal>
    </template>
  </UDashboardPanel>
</template>
