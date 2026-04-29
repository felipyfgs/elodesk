<script setup lang="ts">
import { format, isToday, formatDistanceToNow } from 'date-fns'
import { ptBR } from 'date-fns/locale'
import type { ContextMenuItem } from '@nuxt/ui'
import type { Conversation } from '~/stores/conversations'
import { useConversationsStore } from '~/stores/conversations'
import { useMessagesStore } from '~/stores/messages'
import { resolveContactName, resolveContactAvatar } from '~/utils/chatAdapter'
import { useLabelsStore } from '~/stores/labels'

const props = defineProps<{
  items: Conversation[]
}>()

const selected = defineModel<Conversation | null>()
const { t } = useI18n()
const convs = useConversationsStore()
const messages = useMessagesStore()
const labelsStore = useLabelsStore()

// Prefetch on-hover: dispara o fetch de mensagens assim que o agente passa o
// mouse na linha. Quando ele clica, a thread já encontra o bucket cacheado e
// renderiza sem flash. Dedup + TTL ficam na store; aqui só conectamos o
// gatilho. Espelha a estratégia do Linear/GitHub no list-detail pattern.
function prefetchMessages(c: Conversation) {
  messages.prefetch(c.id)
}

interface LabelChip { title: string, color: string }

function labelChip(title: string): LabelChip {
  const found = labelsStore.list.find(l => l.title === title)
  return { title, color: found?.color ?? '#94a3b8' }
}

const itemRefs = ref<Record<string, Element | null>>({})
const hoveredId = ref<string | null>(null)

const CHANNEL_ICONS: Record<string, string> = {
  'Channel::Api': 'i-lucide-webhook',
  'Channel::Whatsapp': 'i-simple-icons-whatsapp',
  'Channel::Twilio': 'i-lucide-cloud',
  'Channel::Sms': 'i-lucide-message-square',
  'Channel::Instagram': 'i-simple-icons-instagram',
  'Channel::FacebookPage': 'i-simple-icons-facebook',
  'Channel::Telegram': 'i-simple-icons-telegram',
  'Channel::Line': 'i-simple-icons-line',
  'Channel::Tiktok': 'i-simple-icons-tiktok',
  'Channel::WebWidget': 'i-lucide-globe',
  'Channel::Email': 'i-lucide-mail',
  'Channel::Twitter': 'i-simple-icons-x'
}

function contactName(c: Conversation): string {
  return resolveContactName(c)
}

function contactAvatar(c: Conversation): string | undefined {
  return resolveContactAvatar(c)
}

function unreadCount(c: Conversation): number {
  return c.unreadCount ?? 0
}

function isManuallyUnread(c: Conversation): boolean {
  return convs.manuallyUnread.includes(c.id)
}

// "Não lida" para fins visuais inclui marcação manual (right-click do agente),
// mesmo que unreadCount === 0. Espelha o WhatsApp Web.
function hasUnread(c: Conversation): boolean {
  return unreadCount(c) > 0 || isManuallyUnread(c)
}

function channelIcon(c: Conversation): string {
  return CHANNEL_ICONS[c.inbox?.channelType ?? ''] ?? 'i-lucide-inbox'
}

function lastMessage(c: Conversation): { text: string, icon: string, isPrivate: boolean } {
  const msg = c.lastNonActivityMessage
  if (!msg) return { text: '', icon: 'i-lucide-info', isPrivate: false }

  const isPrivate = !!msg.private
  const isOutgoing = msg.messageType === 1
  const isActivity = msg.messageType === 2

  let icon = 'i-lucide-corner-up-right'
  if (isPrivate) icon = 'i-lucide-lock'
  else if (isActivity) icon = 'i-lucide-info'
  else if (isOutgoing) icon = 'i-lucide-corner-up-right'
  else icon = 'i-lucide-arrow-left'

  let text = msg.content || ''
  if (!text && msg.attachments?.length) text = `[${msg.attachments.length} anexo(s)]`
  if (!text) text = ''

  return { text, icon, isPrivate }
}

function timeLabel(c: Conversation): string {
  if (!c.lastActivityAt) return ''
  const d = new Date(c.lastActivityAt)
  if (Number.isNaN(d.getTime())) return ''
  if (isToday(d)) return format(d, 'HH:mm')
  return formatDistanceToNow(d, { addSuffix: false, locale: ptBR })
}

function isActive(c: Conversation): boolean {
  return selected.value?.id === c.id
}

function selectConversation(c: Conversation) {
  selected.value = c
}

function rowAriaLabel(c: Conversation): string {
  return t('conversations.list.openConversation', { name: contactName(c), id: c.displayId })
}

// Right-click → "Marcar como não lida" / "Marcar como lida". Toggle baseado no
// estado atual da marcação manual. Marcar como lida aqui só remove a marcação
// manual — não chama o endpoint de update_last_seen porque marcações manuais
// são local-only (igual WhatsApp Web).
function rowMenuItems(c: Conversation): ContextMenuItem[][] {
  const manual = isManuallyUnread(c)
  return [[
    manual
      ? {
          label: t('conversations.list.markAsRead'),
          icon: 'i-lucide-mail-open',
          onSelect: () => {
            const idx = convs.manuallyUnread.indexOf(c.id)
            if (idx >= 0) convs.manuallyUnread.splice(idx, 1)
          }
        }
      : {
          label: t('conversations.list.markAsUnread'),
          icon: 'i-lucide-mail',
          onSelect: () => convs.markAsUnread(c.id)
        }
  ]]
}

watch(selected, () => {
  if (!selected.value) return
  const ref = itemRefs.value[selected.value.id]
  if (ref) ref.scrollIntoView({ block: 'nearest' })
})

defineShortcuts({
  arrowdown: () => {
    const idx = props.items.findIndex(c => c.id === selected.value?.id)
    if (idx === -1) selected.value = props.items[0] ?? null
    else if (idx < props.items.length - 1) selected.value = props.items[idx + 1] ?? null
  },
  arrowup: () => {
    const idx = props.items.findIndex(c => c.id === selected.value?.id)
    if (idx === -1) selected.value = props.items.at(-1) ?? null
    else if (idx > 0) selected.value = props.items[idx - 1] ?? null
  }
})
</script>

<template>
  <!--
    Lista segue o padrão visual do template oficial Nuxt UI Dashboard
    (_refs/dashboard/InboxList.vue): faixa lateral 2px (`border-l-2`) que
    transita de cor no hover/selected, em vez de uma borda completa
    arredondada. `divide-y` substitui o gap com bordas finas entre itens —
    igual ao template, mais limpo visualmente.

    transition-colors em vez de transition genérico: a animação fica só
    nas cores (border + bg), sem afetar layout/transform — o efeito é
    instantâneo perceptualmente mas suaviza o hover/click.
  -->
  <ul
    class="min-h-0 flex-1 divide-y divide-default overflow-y-auto"
    role="listbox"
    :aria-label="t('conversations.list.ariaLabel')"
  >
    <li
      v-for="c in items"
      :key="c.id"
      :ref="(el) => { itemRefs[c.id] = el as Element | null }"
    >
      <UContextMenu :items="rowMenuItems(c)">
        <div
          class="group relative grid cursor-pointer grid-cols-[auto_minmax(0,1fr)] items-center gap-3 border-l-2 px-3 py-3 transition-colors outline-none"
          :class="[
            isActive(c)
              ? 'border-primary bg-primary/10'
              : 'border-bg hover:border-primary hover:bg-primary/5 focus-visible:border-primary focus-visible:bg-primary/5'
          ]"
          role="option"
          tabindex="0"
          :aria-selected="isActive(c)"
          :aria-label="rowAriaLabel(c)"
          @click="selectConversation(c)"
          @keydown.enter.prevent="selectConversation(c)"
          @keydown.space.prevent="selectConversation(c)"
          @mouseenter="hoveredId = c.id; prefetchMessages(c)"
          @mouseleave="hoveredId = null"
          @focus="prefetchMessages(c)"
        >
        <div class="relative size-8 shrink-0">
          <UAvatar
            :alt="contactName(c)"
            :src="contactAvatar(c)"
            size="md"
            class="size-8"
          />
          <span class="absolute -bottom-1 -right-1 flex size-5 items-center justify-center rounded-md bg-default ring ring-default">
            <UIcon :name="channelIcon(c)" class="size-3 text-muted" />
          </span>

          <label
            v-if="hoveredId === c.id || convs.selection.includes(c.id)"
            class="absolute inset-0 z-10 flex cursor-pointer items-center justify-center rounded-full bg-default/55 backdrop-blur-[1px]"
            @click.stop
          >
            <UCheckbox
              :model-value="convs.selection.includes(c.id)"
              :aria-label="t('conversations.list.selectConversation', { id: c.displayId })"
              :ui="{
                root: 'size-4 items-center justify-center',
                container: 'flex size-4 items-center justify-center',
                base: 'size-4 rounded-[3px] bg-default shadow-sm ring ring-default',
                wrapper: 'hidden'
              }"
              @update:model-value="() => convs.toggleSelection(c.id)"
            />
          </label>
        </div>

        <div class="min-w-0">
          <div class="flex min-w-0 items-start gap-2">
            <div class="min-w-0 flex-1">
              <h4
                class="truncate text-sm leading-5"
                :class="hasUnread(c) ? 'font-semibold text-highlighted' : 'font-medium text-default'"
              >
                {{ contactName(c) }}
              </h4>
              <div class="mt-0.5 flex min-w-0 items-center gap-1.5 text-[11px] text-muted">
                <span class="shrink-0 font-medium">#{{ c.displayId }}</span>
                <span class="text-dimmed">/</span>
                <span class="truncate">{{ c.inbox?.name || t('conversations.detail.noInbox') }}</span>
              </div>
            </div>

            <div class="flex shrink-0 flex-col items-end gap-1">
              <span class="text-[11px] text-muted">
                {{ timeLabel(c) }}
              </span>
              <span
                v-if="unreadCount(c) > 0"
                class="inline-flex h-[18px] min-w-[18px] items-center justify-center rounded-full bg-primary px-1 text-[10px] font-semibold text-inverted"
                :aria-label="t('conversations.list.unreadCount', { count: unreadCount(c) })"
              >
                {{ unreadCount(c) > 99 ? '99+' : unreadCount(c) }}
              </span>
              <!--
                Marcação manual via right-click ("Marcar como não lida") — sem
                contagem real, mostra apenas um dot indicador (igual ao
                comportamento do WhatsApp Web). Só aparece quando não há
                unreadCount real, pra não duplicar com o badge numérico.
              -->
              <span
                v-else-if="isManuallyUnread(c)"
                class="size-2 rounded-full bg-primary"
                :aria-label="t('conversations.list.markedUnread')"
              />
            </div>
          </div>

          <div v-if="lastMessage(c).text" class="mt-2 flex min-w-0 items-center gap-1.5">
            <UIcon
              :name="lastMessage(c).icon"
              class="size-3.5 shrink-0"
              :class="lastMessage(c).isPrivate ? 'text-warning' : 'text-dimmed'"
            />
            <p
              class="min-w-0 flex-1 truncate text-xs"
              :class="hasUnread(c) ? 'text-default font-medium' : 'text-muted'"
            >
              {{ lastMessage(c).text }}
            </p>
          </div>
          <p
            v-else
            class="mt-2 truncate text-xs italic text-dimmed"
          >
            {{ t('conversations.list.noPreview') }}
          </p>

          <div v-if="c.labels?.length" class="mt-2 flex flex-wrap items-center gap-1">
            <span
              v-for="title in c.labels.slice(0, 2)"
              :key="title"
              class="inline-flex max-w-[7rem] items-center truncate rounded-md px-2 py-0.5 text-[10px] font-medium"
              :style="{ backgroundColor: `${labelChip(title).color}20`, color: labelChip(title).color }"
            >
              {{ labelChip(title).title }}
            </span>
            <span v-if="c.labels.length > 2" class="text-[10px] text-muted">
              +{{ c.labels.length - 2 }}
            </span>
          </div>
        </div>
        </div>
      </UContextMenu>
    </li>
  </ul>
</template>
