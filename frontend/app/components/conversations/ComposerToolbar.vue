<script setup lang="ts">
import EmojiPicker from 'vue3-emoji-picker'
import 'vue3-emoji-picker/css'
import type { DropdownMenuItem } from '@nuxt/ui'

interface EmojiPickerSelectEvent { i: string, n: string[], r: string, t: string, u: string }

type RichEditor = {
  insertAtCursor: (text: string) => void
  replaceSlashCommand: (content: string) => void
  focus: () => void
  toggleBold: () => void
  toggleItalic: () => void
  toggleStrike: () => void
  toggleCode: () => void
  toggleBulletList: () => void
  toggleOrderedList: () => void
  toggleBlockquote: () => void
  undo: () => void
  redo: () => void
  setLink: (href: string) => void
  isActive: (mark: string) => boolean
}

// Espelha o type AttachKind do Composer.vue — fonte autoritativa lá. Mantemos
// um literal aqui pra evitar import circular do componente filho-pai.
type AttachKind = 'all' | 'document' | 'media' | 'camera'

const props = defineProps<{
  richEditor: RichEditor | null
  mode: 'reply' | 'private'
  expanded: boolean
  sending: boolean
  disabled: boolean
  cannedSearch: string
  channelType?: string
}>()

const emit = defineEmits<{
  'submit': []
  'attach': [kind: AttachKind]
  'record': []
  'update:expanded': [value: boolean]
  'cannedSelect': [content: string]
  'emojiSelect': [event: EmojiPickerSelectEvent]
}>()

const { t } = useI18n()
const colorMode = useColorMode()

const cannedOpen = defineModel<boolean>('cannedOpen', { default: false })
const emojiOpen = ref(false)

const emojiTheme = computed<'dark' | 'light'>(() =>
  colorMode.value === 'dark' ? 'dark' : 'light'
)

const emojiGroupNames = computed(() => ({
  smileys_people: t('conversations.compose.emojiGroups.smileysPeople'),
  animals_nature: t('conversations.compose.emojiGroups.animalsNature'),
  food_drink: t('conversations.compose.emojiGroups.foodDrink'),
  activities: t('conversations.compose.emojiGroups.activities'),
  travel_places: t('conversations.compose.emojiGroups.travelPlaces'),
  objects: t('conversations.compose.emojiGroups.objects'),
  symbols: t('conversations.compose.emojiGroups.symbols'),
  flags: t('conversations.compose.emojiGroups.flags'),
  recent: t('conversations.compose.emojiGroups.recent')
}))

function onEmojiSelect(event: EmojiPickerSelectEvent) {
  emit('emojiSelect', event)
  emojiOpen.value = false
}

function promptLink() {
  const href = window.prompt(t('conversations.compose.linkPrompt'), '')
  if (href === null) return
  props.richEditor?.setLink(href)
}

function toggleExpanded() {
  emit('update:expanded', !props.expanded)
}

// Itens do menu de anexo. Estilo WhatsApp: agrupa "mídia universal" no topo
// (cobre todo canal que aceita anexo) e poderá ganhar uma seção condicional
// por canal no futuro (ex.: Contato/Enquete só em WhatsApp). Hoje só os
// universais — evita itens disabled/coming-soon na UI.
//
// O `onSelect` emite o tipo pra cima; o pai (Composer) decide o accept e
// dispara o file picker. Item "Áudio" abre o file picker de arquivos de áudio
// existentes no computador — gravação ao vivo continua acessível pelo botão
// `i-lucide-mic` separado fora do menu (paridade WhatsApp Web: clip = anexo
// existente, microfone = grava agora).
const attachItems = computed<DropdownMenuItem[][]>(() => {
  const universal: DropdownMenuItem[] = [
    {
      label: t('conversations.compose.attachMenu.document'),
      icon: 'i-lucide-file-text',
      onSelect: () => emit('attach', 'document')
    },
    {
      label: t('conversations.compose.attachMenu.media'),
      icon: 'i-lucide-image',
      onSelect: () => emit('attach', 'media')
    },
    {
      label: t('conversations.compose.attachMenu.camera'),
      icon: 'i-lucide-camera',
      onSelect: () => emit('attach', 'camera')
    },
    {
      label: t('conversations.compose.attachMenu.audio'),
      icon: 'i-lucide-music',
      onSelect: () => emit('attach', 'audio')
    }
  ]
  return [universal]
})

// `channelType` ainda não é usado para itens condicionais — fica ancorado aqui
// pra quando entrarem opções WhatsApp-only (Contato, Enquete). Suprimimos o
// warning de unused enquanto a infra está pronta mas a UX final em discussão.
void props.channelType
</script>

<template>
  <div class="flex w-full min-w-0 items-center gap-1.5">
    <div class="flex min-w-0 flex-1 items-center gap-0.5">
      <UPopover :content="{ align: 'end' }">
        <UTooltip :text="t('conversations.compose.format')">
          <UButton
            icon="i-lucide-type"
            color="neutral"
            variant="ghost"
            size="xs"
            class="hidden sm:inline-flex"
            :aria-label="t('conversations.compose.format')"
          />
        </UTooltip>

        <template #content>
          <div class="flex items-center gap-0.5 p-1">
            <UTooltip :text="t('conversations.compose.toolbar.bold')">
              <UButton
                icon="i-lucide-bold"
                color="neutral"
                :variant="richEditor?.isActive('bold') ? 'soft' : 'ghost'"
                size="xs"
                :aria-label="t('conversations.compose.toolbar.bold')"
                @click="richEditor?.toggleBold()"
              />
            </UTooltip>
            <UTooltip :text="t('conversations.compose.toolbar.italic')">
              <UButton
                icon="i-lucide-italic"
                color="neutral"
                :variant="richEditor?.isActive('italic') ? 'soft' : 'ghost'"
                size="xs"
                :aria-label="t('conversations.compose.toolbar.italic')"
                @click="richEditor?.toggleItalic()"
              />
            </UTooltip>
            <UTooltip :text="t('conversations.compose.toolbar.strike')">
              <UButton
                icon="i-lucide-strikethrough"
                color="neutral"
                :variant="richEditor?.isActive('strike') ? 'soft' : 'ghost'"
                size="xs"
                :aria-label="t('conversations.compose.toolbar.strike')"
                @click="richEditor?.toggleStrike()"
              />
            </UTooltip>
            <UTooltip :text="t('conversations.compose.toolbar.link')">
              <UButton
                icon="i-lucide-link"
                color="neutral"
                :variant="richEditor?.isActive('link') ? 'soft' : 'ghost'"
                size="xs"
                :aria-label="t('conversations.compose.toolbar.link')"
                @click="promptLink"
              />
            </UTooltip>
            <UTooltip :text="t('conversations.compose.toolbar.undo')">
              <UButton
                icon="i-lucide-undo-2"
                color="neutral"
                variant="ghost"
                size="xs"
                :aria-label="t('conversations.compose.toolbar.undo')"
                @click="richEditor?.undo()"
              />
            </UTooltip>
            <UTooltip :text="t('conversations.compose.toolbar.redo')">
              <UButton
                icon="i-lucide-redo-2"
                color="neutral"
                variant="ghost"
                size="xs"
                :aria-label="t('conversations.compose.toolbar.redo')"
                @click="richEditor?.redo()"
              />
            </UTooltip>
            <UTooltip :text="t('conversations.compose.toolbar.list')">
              <UButton
                icon="i-lucide-list"
                color="neutral"
                :variant="richEditor?.isActive('bulletList') ? 'soft' : 'ghost'"
                size="xs"
                :aria-label="t('conversations.compose.toolbar.list')"
                @click="richEditor?.toggleBulletList()"
              />
            </UTooltip>
            <UTooltip :text="t('conversations.compose.toolbar.orderedList')">
              <UButton
                icon="i-lucide-list-ordered"
                color="neutral"
                :variant="richEditor?.isActive('orderedList') ? 'soft' : 'ghost'"
                size="xs"
                :aria-label="t('conversations.compose.toolbar.orderedList')"
                @click="richEditor?.toggleOrderedList()"
              />
            </UTooltip>
            <UTooltip :text="t('conversations.compose.toolbar.blockquote')">
              <UButton
                icon="i-lucide-quote"
                color="neutral"
                :variant="richEditor?.isActive('blockquote') ? 'soft' : 'ghost'"
                size="xs"
                :aria-label="t('conversations.compose.toolbar.blockquote')"
                @click="richEditor?.toggleBlockquote()"
              />
            </UTooltip>
            <UTooltip :text="t('conversations.compose.toolbar.code')">
              <UButton
                icon="i-lucide-code-2"
                color="neutral"
                :variant="richEditor?.isActive('code') ? 'soft' : 'ghost'"
                size="xs"
                :aria-label="t('conversations.compose.toolbar.code')"
                @click="richEditor?.toggleCode()"
              />
            </UTooltip>
          </div>
        </template>
      </UPopover>

      <UPopover v-model:open="emojiOpen" :content="{ align: 'end', side: 'top' }">
        <UTooltip :text="t('conversations.compose.emoji')">
          <UButton
            icon="i-lucide-smile"
            color="neutral"
            variant="ghost"
            size="xs"
            :aria-label="t('conversations.compose.emoji')"
          />
        </UTooltip>
        <template #content>
          <EmojiPicker
            :theme="emojiTheme"
            :native="true"
            :group-names="emojiGroupNames"
            :hide-group-icons="false"
            @select="onEmojiSelect"
          />
        </template>
      </UPopover>
      <UDropdownMenu :items="attachItems" :content="{ align: 'center', side: 'top' }">
        <UTooltip :text="t('conversations.compose.attach')">
          <UButton
            icon="i-lucide-paperclip"
            color="neutral"
            variant="ghost"
            size="xs"
            :aria-label="t('conversations.compose.attach')"
          />
        </UTooltip>
      </UDropdownMenu>
      <UTooltip :text="t('conversations.compose.voice')">
        <UButton
          icon="i-lucide-mic"
          color="neutral"
          variant="ghost"
          size="xs"
          :aria-label="t('conversations.compose.voice')"
          @click="emit('record')"
        />
      </UTooltip>
      <UPopover v-model:open="cannedOpen" :content="{ align: 'end', side: 'top' }">
        <UTooltip :text="t('conversations.compose.canned')">
          <UButton
            icon="i-lucide-quote"
            color="neutral"
            variant="ghost"
            size="xs"
            class="hidden md:inline-flex"
            :aria-label="t('conversations.compose.canned')"
          />
        </UTooltip>
        <template #content>
          <ConversationsCannedResponsePicker
            :search="cannedSearch"
            @select="(content: string) => emit('cannedSelect', content)"
          />
        </template>
      </UPopover>
      <UTooltip :text="expanded ? t('conversations.compose.collapse') : t('conversations.compose.expand')">
        <UButton
          :icon="expanded ? 'i-lucide-minimize-2' : 'i-lucide-maximize-2'"
          color="neutral"
          variant="ghost"
          size="xs"
          class="hidden md:inline-flex"
          :aria-label="expanded ? t('conversations.compose.collapse') : t('conversations.compose.expand')"
          @click="toggleExpanded"
        />
      </UTooltip>
    </div>

    <UButton
      :label="t('conversations.compose.send')"
      trailing-icon="i-lucide-corner-down-left"
      :color="mode === 'private' ? 'warning' : 'primary'"
      size="sm"
      class="shrink-0"
      :loading="sending"
      :disabled="disabled"
      @click="emit('submit')"
    />
  </div>
</template>
