<script setup lang="ts">
import { useAuthStore } from '~/stores/auth'
import { useMessagesStore, type Message } from '~/stores/messages'
import type { Conversation } from '~/stores/conversations'
import { useConversationsStore } from '~/stores/conversations'
import { useAgentsStore } from '~/stores/agents'
import { forwardSelectionModeKey, forwardSelectedIdsKey } from '~/utils/forward'

const props = withDefaults(defineProps<{
  conversation: Conversation
  showBack?: boolean
}>(), {
  showBack: false
})

const emit = defineEmits<{ close: [] }>()

const api = useApi()
const auth = useAuthStore()
const messages = useMessagesStore()
const conversations = useConversationsStore()
const agents = useAgentsStore()

// Estado da sidebar de detalhes — persistido em localStorage e com default
// dependente da viewport (aberta em xl+, fechada em <xl). isCompact decide
// se a sidebar renderiza inline (≥lg) ou dentro do USlideover (<lg).
const { open: detailsOpen } = useDetailsSidebar()
const { isCompact } = useResponsive()

// O USlideover precisa ficar montado (evita race de HMR com o Teleport),
// mas o conteúdo só deve aparecer em viewport compacto. Gate via :open
// — `v-show` no componente não funciona porque o root é um componente
// (DialogRoot) e o conteúdo real fica teleportado pro <body>.
const slideoverOpen = computed({
  get: () => isCompact.value && detailsOpen.value,
  set: (v) => { detailsOpen.value = v }
})

// Forward selection state
const selectionMode = ref(false)
const selectedMessageIds = ref<Set<string>>(new Set())
const forwardModalOpen = ref(false)

provide(forwardSelectionModeKey, selectionMode)
provide(forwardSelectedIdsKey, selectedMessageIds)

const list = computed<Message[]>(() => messages.byConversation[props.conversation.id] ?? [])

// Marca a conversa como lida assim que o agente abre. Optimistic: zera o
// unreadCount localmente (e pina o id em stickyUnreadId pra ela não sumir
// da aba "Não lidas" enquanto ainda estiver aberta) e chama o endpoint que
// atualiza assignee_last_seen_at e dispara conversation.updated para todos
// os clientes. Best-effort: erros de rede são silenciados — o badge volta
// na próxima hidratação.
async function markRead(convId: string) {
  if (!auth.account?.id) return
  conversations.markRead(convId)
  try {
    await api(`/accounts/${auth.account.id}/conversations/${convId}/update_last_seen`, { method: 'POST' })
  } catch {
    // ignore — broadcast realtime will reconcile if/when it succeeds
  }
}

watch(() => props.conversation.id, async (id) => {
  // fetchMessages dedupa contra o hover-prefetch da List: se o bucket já
  // foi populado durante o hover, o click só dispara o markRead e a UI
  // mostra na hora. Passamos freshMs para honrar o TTL do prefetch.
  await messages.fetchMessages(id, { freshMs: 30_000 })
  if (id) markRead(id)
}, { immediate: true })

// Scroll-to-bottom: handled at the outer overflow container instead of
// trusting UChatMessages :auto-scroll. Reactive list updates from the
// realtime store don't reliably trigger Nuxt UI's internal observer when
// the scroll element is the parent overflow div.
const scrollContainerRef = ref<HTMLDivElement | null>(null)
const STICK_THRESHOLD_PX = 80
const stickToBottom = ref(true)

function isNearBottom(el: HTMLElement) {
  return el.scrollHeight - el.scrollTop - el.clientHeight <= STICK_THRESHOLD_PX
}

function scrollToBottom(behavior: ScrollBehavior = 'smooth') {
  const el = scrollContainerRef.value
  if (!el) return
  el.scrollTo({ top: el.scrollHeight, behavior })
}

function onScroll() {
  const el = scrollContainerRef.value
  if (!el) return
  // If the agent reads history (scrolls up), don't yank them back when a
  // new message arrives. Resume auto-stick once they're back near the end.
  stickToBottom.value = isNearBottom(el)
}

// Reset to bottom whenever the active conversation changes.
watch(() => props.conversation.id, () => {
  stickToBottom.value = true
  nextTick(() => scrollToBottom('auto'))
})

// React to new/updated messages. Watching length covers append; watching
// the last id covers the "pending → real" reconciliation in upsert.
watch(
  () => [list.value.length, list.value[list.value.length - 1]?.id] as const,
  () => {
    if (!stickToBottom.value) return
    nextTick(() => scrollToBottom('smooth'))
  }
)

// Cancel selection mode and clear
function cancelSelection() {
  selectionMode.value = false
  selectedMessageIds.value = new Set()
}

// Handle Esc key to exit selection mode
function onKeydown(e: KeyboardEvent) {
  if (e.key === 'Escape' && selectionMode.value) {
    cancelSelection()
  }
}

// Handler for when forward completes (success)
function onForwardComplete() {
  forwardModalOpen.value = false
  cancelSelection()
}

// When the modal is dismissed without submitting (cancel/click-outside),
// exit selection mode too.
watch(forwardModalOpen, (open) => {
  if (!open && selectionMode.value) {
    cancelSelection()
  }
})

const selectedMessages = computed<Message[]>(() =>
  list.value.filter(m => selectedMessageIds.value.has(String(m.id)))
)

onMounted(() => {
  if (!agents.items.length) {
    agents.fetch().catch((err) => {
      console.error('[ConversationThread] failed to fetch agents', err)
    })
  }
  document.addEventListener('keydown', onKeydown)
})

onUnmounted(() => {
  document.removeEventListener('keydown', onKeydown)
})
</script>

<template>
  <UDashboardPanel id="conversations-thread" class="min-w-0 flex-1">
    <div class="flex min-h-0 flex-1 bg-default">
      <section class="flex min-w-0 flex-1 flex-col bg-default">
        <ConversationsThreadHeader
          v-model:details-open="detailsOpen"
          :conversation="conversation"
          :show-back="showBack"
          @back="emit('close')"
        />

        <div class="flex min-h-0 flex-1 flex-col bg-default">
          <div
            ref="scrollContainerRef"
            class="min-h-0 flex-1 overflow-y-auto px-3 sm:px-4"
            @scroll.passive="onScroll"
          >
            <!--
              Padrão "stick to bottom" usado por WhatsApp/iMessage: quando há
              poucas mensagens, elas grudam acima do composer; quando o
              conteúdo excede o viewport, o scroll funciona normal e o final
              continua próximo do composer (auto-stick em scrollToBottom).

              `min-h-full + flex flex-col + justify-end` cresce o wrapper até
              a altura do scroller e empurra os filhos pro fundo. Sem isso, o
              fluxo normal de bloco renderiza as bolhas a partir do topo,
              deixando um vazio enorme entre a primeira mensagem e o input.
            -->
            <div class="flex min-h-full flex-col justify-end">
              <ConversationsMessageList :messages="list" :conversation="conversation" />
            </div>
          </div>

          <ConversationsSelectionToolbar
            v-if="selectionMode"
            :count="selectedMessageIds.size"
            @cancel="cancelSelection"
            @forward="forwardModalOpen = true"
          />
          <ConversationsComposer v-else :conversation="conversation" />
        </div>
      </section>

      <!--
        Inline: só em ≥lg quando há espaço pras 3 colunas (lista|thread|side).
        Largura fixa para não competir com a Thread; em xl ganha mais 32px.
      -->
      <aside
        v-if="detailsOpen && !isCompact"
        class="hidden w-72 shrink-0 flex-col border-l border-default bg-default lg:flex xl:w-80"
      >
        <ConversationsSidebar
          :conversation="conversation"
          @close="detailsOpen = false"
        />
      </aside>
    </div>

    <!--
      Compact (mobile + tablet): a sidebar vira slideover lateral acionado
      pelo botão "i" do ThreadHeader. Sempre montado para evitar HMR race
      conditions com Teleport — quando Vite hot-replaces e o slideover está
      aberto, unmount via v-if corrompe a árvore DOM do Teleport causando
      "can't access property 'parentNode', node is null". O gate visual é
      feito pelo :open via slideoverOpen (computed que respeita isCompact).
    -->
    <USlideover
      v-model:open="slideoverOpen"
      :ui="{ content: 'w-full sm:max-w-sm' }"
    >
      <template #content>
        <ConversationsSidebar
          :conversation="conversation"
          @close="detailsOpen = false"
        />
      </template>
    </USlideover>

    <ConversationsForwardModal
      v-if="selectionMode"
      v-model:open="forwardModalOpen"
      :message-ids="[...selectedMessageIds]"
      :selected-messages="selectedMessages"
      @done="onForwardComplete"
    />
  </UDashboardPanel>
</template>
