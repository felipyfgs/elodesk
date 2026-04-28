<script setup lang="ts">
import { STATUS_MAP, type Conversation } from '~/stores/conversations'
import { useAuthStore } from '~/stores/auth'
import {
  resolveContactName,
  resolveContactIdentifier,
  resolveContactAvatar
} from '~/utils/chatAdapter'

const props = defineProps<{
  conversation: Conversation
}>()

const emit = defineEmits<{ close: [] }>()

const { t } = useI18n()
const auth = useAuthStore()

const contact = computed(() => props.conversation.meta?.sender ?? null)
const contactName = computed(() => resolveContactName(props.conversation))
const contactIdentifier = computed(() => resolveContactIdentifier(props.conversation))
const contactAvatar = computed(() => resolveContactAvatar(props.conversation))

const statusLabel = computed(() => {
  const keys = ['open', 'resolved', 'pending', 'snoozed']
  return t(`conversations.filters.${keys[props.conversation.status]}`)
})

const statusColor = computed(() => {
  switch (props.conversation.status) {
    case STATUS_MAP.OPEN: return 'success' as const
    case STATUS_MAP.PENDING: return 'warning' as const
    case STATUS_MAP.RESOLVED: return 'info' as const
    case STATUS_MAP.SNOOZED: return 'neutral' as const
    default: return 'neutral' as const
  }
})

const channelBadge = computed(() => {
  const type = props.conversation.inbox?.channelType || props.conversation.meta?.channel || ''
  if (type.includes('Api')) return 'API'
  if (type.includes('Whatsapp')) return 'WA'
  if (type.includes('Twilio')) return 'TW'
  if (type.includes('Sms')) return 'SMS'
  if (type.includes('Email')) return 'EM'
  if (type.includes('Instagram')) return 'IG'
  if (type.includes('Facebook')) return 'FB'
  if (type.includes('Telegram')) return 'TG'
  if (type.includes('Line')) return 'LI'
  if (type.includes('Tiktok')) return 'TK'
  if (type.includes('Twitter')) return 'TW'
  if (type.includes('WebWidget')) return 'WW'
  return type.split('::').pop() || ''
})

// Mostra apenas info que NÃO está visível no header (avatar / nome /
// identifier / badges de conversa+inbox+canal). Telefone/identifier/inbox/
// canal já estão acima — exibir de novo na lista é poluição.
const contactRows = computed(() => {
  const rows: { icon: string, title: string, value: string }[] = []
  const phone = contact.value?.phoneNumber
  const identifier = contactIdentifier.value
  const email = contact.value?.email
  if (email) {
    rows.push({ icon: 'i-lucide-mail', title: t('conversations.detail.email'), value: email })
  }
  if (identifier && identifier !== phone && identifier !== email) {
    rows.push({ icon: 'i-lucide-at-sign', title: t('conversations.detail.identifier'), value: identifier })
  }
  return rows
})

const detailSections = computed(() => [
  { value: 'actions', label: t('conversations.detail.actions'), icon: 'i-lucide-bolt' },
  { value: 'macros', label: t('conversations.detail.macros'), icon: 'i-lucide-command' },
  { value: 'conversationInfo', label: t('conversations.detail.conversationInfo'), icon: 'i-lucide-info' },
  { value: 'contactAttributes', label: t('conversations.detail.contactAttributes'), icon: 'i-lucide-tags' },
  { value: 'contactNotes', label: t('conversations.detail.contactNotes'), icon: 'i-lucide-notebook-pen' },
  { value: 'previousConversations', label: t('conversations.detail.previousConversations'), icon: 'i-lucide-history' },
  { value: 'participants', label: t('conversations.detail.participants'), icon: 'i-lucide-users' },
  { value: 'linkedIssues', label: t('conversations.detail.linkedIssues'), icon: 'i-lucide-git-pull-request' }
])
</script>

<template>
  <!--
    Sidebar é um wrapper "burro" com flex-col cobrindo 100% do container.
    O caller (Thread.vue) decide o envólucro: <aside> com largura fixa em
    ≥lg, ou USlideover.content em <lg. Manter a estrutura interna idêntica
    nos dois modos garante que o conteúdo (header + scroll) renderize
    consistente em desktop e mobile.
  -->
  <div class="flex h-full w-full min-w-0 flex-col bg-default">
    <div class="flex h-(--ui-header-height) shrink-0 items-center justify-between border-b border-default px-4">
      <h2 class="text-sm font-semibold text-highlighted">
        {{ t('conversations.detail.contacts') }}
      </h2>
      <div class="flex items-center gap-1">
        <UTooltip v-if="contact?.id && auth.account?.id" :text="t('contactDetail.viewContact')">
          <UButton
            :to="`/accounts/${auth.account.id}/contacts/${contact.id}`"
            icon="i-lucide-external-link"
            color="neutral"
            variant="ghost"
            size="xs"
            :aria-label="t('contactDetail.viewContact')"
          />
        </UTooltip>
        <UButton
          icon="i-lucide-x"
          color="neutral"
          variant="ghost"
          size="xs"
          :aria-label="t('common.close')"
          @click="emit('close')"
        />
      </div>
    </div>

    <div class="min-h-0 flex-1 overflow-y-auto">
      <section class="border-b border-default px-4 py-3">
        <div class="flex flex-col items-center text-center">
          <div class="relative">
            <UAvatar
              :alt="contactName"
              :src="contactAvatar"
              size="xl"
            />
            <span class="absolute -bottom-1 -right-1 rounded-md bg-elevated px-1 py-0.5 text-[10px] font-semibold text-muted ring ring-default">
              {{ channelBadge }}
            </span>
          </div>

          <p class="mt-2 truncate max-w-full text-base font-semibold text-highlighted">
            {{ contactName }}
          </p>
          <p v-if="contactIdentifier" class="truncate max-w-full text-xs text-muted">
            {{ contactIdentifier }}
          </p>

          <div class="mt-2 flex flex-wrap items-center justify-center gap-1.5">
            <UBadge
              :label="statusLabel"
              :color="statusColor"
              variant="subtle"
              size="xs"
            />
            <UBadge
              :label="`#${conversation.displayId}`"
              color="neutral"
              variant="soft"
              size="xs"
            />
            <UBadge
              v-if="conversation.inbox?.name"
              :label="conversation.inbox.name"
              color="neutral"
              variant="soft"
              size="xs"
              class="max-w-full truncate"
            />
          </div>
        </div>

        <div class="mt-3 grid grid-cols-4 gap-1.5">
          <UTooltip :text="t('conversations.detail.message')">
            <UButton
              icon="i-lucide-message-circle"
              color="neutral"
              variant="soft"
              size="xs"
              block
              :aria-label="t('conversations.detail.message')"
            />
          </UTooltip>
          <UTooltip :text="t('common.edit')">
            <UButton
              icon="i-lucide-pencil"
              color="neutral"
              variant="soft"
              size="xs"
              block
              :aria-label="t('common.edit')"
            />
          </UTooltip>
          <UTooltip :text="t('conversations.compose.voice')">
            <UButton
              icon="i-lucide-mic"
              color="neutral"
              variant="soft"
              size="xs"
              block
              :aria-label="t('conversations.compose.voice')"
            />
          </UTooltip>
          <UTooltip :text="t('common.delete')">
            <UButton
              icon="i-lucide-trash-2"
              color="error"
              variant="soft"
              size="xs"
              block
              :aria-label="t('common.delete')"
            />
          </UTooltip>
        </div>

        <dl v-if="contactRows.length" class="mt-3 space-y-1.5">
          <div
            v-for="row in contactRows"
            :key="`${row.icon}-${row.title}`"
            class="flex min-w-0 items-center gap-2"
          >
            <UIcon :name="row.icon" class="size-3.5 shrink-0 text-dimmed" />
            <span class="truncate text-xs text-muted" :title="row.value">{{ row.value }}</span>
          </div>
        </dl>
      </section>

      <UAccordion
        :items="detailSections"
        type="multiple"
        :default-value="['actions']"
        :ui="{
          root: 'space-y-2 p-2',
          item: 'overflow-hidden rounded-md border border-default bg-default last:border-b',
          trigger: 'px-3 py-3 text-sm hover:bg-elevated/60',
          leadingIcon: 'size-4 text-muted',
          body: 'px-3 pb-3 text-sm'
        }"
      >
        <template #trailing="{ open }">
          <UIcon
            name="i-lucide-plus"
            :class="['ms-auto size-4 shrink-0 text-muted transition-transform', open ? 'rotate-45' : '']"
          />
        </template>

        <template #body="{ item }">
          <ConversationsSidebarActions
            v-if="item.value === 'actions'"
            :conversation="conversation"
          />

          <div v-else class="flex items-center gap-2 rounded-md bg-elevated/50 px-3 py-2 text-sm text-muted">
            <UIcon name="i-lucide-circle-dashed" class="size-4 shrink-0 text-dimmed" />
            {{ t('conversations.detail.emptySection') }}
          </div>
        </template>
      </UAccordion>
    </div>
  </div>
</template>
