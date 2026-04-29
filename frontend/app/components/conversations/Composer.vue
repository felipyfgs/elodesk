<script setup lang="ts">
import type { Conversation } from '~/stores/conversations'
import { useAuthStore } from '~/stores/auth'
import { useCannedResponsesStore } from '~/stores/cannedResponses'
import { useMessagesStore, type Message } from '~/stores/messages'

interface UploadedFile {
  id: string
  file: File
  url?: string
  uploading: boolean
  error?: string
  // Áudio gravado pelo botão `i-lucide-mic` ganha um player inline em vez de
  // virar chip — espelha o Chatwoot. O envio compartilha o mesmo `attachments`
  // pra reaproveitar o pipeline de upload + send; a flag aqui só governa a UI.
  isRecordedAudio?: boolean
}

const props = defineProps<{
  conversation: Conversation
}>()

const { t } = useI18n()
const api = useApi()
const auth = useAuthStore()
const canned = useCannedResponsesStore()
const messages = useMessagesStore()
const toast = useToast()

const reply = ref('')
const sending = ref(false)
const mode = ref<'reply' | 'private'>('reply')
const attachments = ref<UploadedFile[]>([])
const expanded = ref(false)
const cannedOpen = ref(false)
const isRecording = ref(false)
const fileInputRef = ref<HTMLInputElement | null>(null)

// `accept` patterns aplicados dinamicamente conforme a categoria escolhida no
// menu de anexo:
//   - ALL_ACCEPT   → qualquer arquivo (paste, drag-and-drop, "Documento")
//   - MEDIA_ACCEPT → câmera + galeria (image/video)
// Nenhum filtro extra no caminho "Documento" — o agente escolhe o que faz
// sentido. Validação real fica no backend/canal (cada channel rejeita o que
// não suporta).
const ALL_ACCEPT = '*/*'
const MEDIA_ACCEPT = 'image/*,video/*'
const AUDIO_ACCEPT = 'audio/*'

type AttachKind = 'all' | 'document' | 'media' | 'camera' | 'audio'

// Aplica o `accept`/`capture` adequados ao input antes de abrir o seletor.
// Mantemos um único <input type="file"> e mutamos os atributos no momento do
// clique — evita N inputs ocultos no DOM e mantém o handler de change único.
function onAttachKind(kind: AttachKind) {
  const input = fileInputRef.value
  if (!input) return
  switch (kind) {
    case 'document':
      input.accept = ALL_ACCEPT
      input.removeAttribute('capture')
      break
    case 'media':
      input.accept = MEDIA_ACCEPT
      input.removeAttribute('capture')
      break
    case 'camera':
      input.accept = MEDIA_ACCEPT
      // `environment` = câmera traseira; navegadores desktop ignoram o atributo
      // e caem no file picker normal — degrada graciosamente.
      input.setAttribute('capture', 'environment')
      break
    case 'audio':
      input.accept = AUDIO_ACCEPT
      input.removeAttribute('capture')
      break
    default:
      input.accept = ALL_ACCEPT
      input.removeAttribute('capture')
  }
  input.click()
}

// Mapeamento extensão → mime canônico. Browsers (e o OS) frequentemente
// devolvem `file.type` vazio ou genérico (`application/ogg`,
// `application/octet-stream`) para arquivos anexados via clip — o que faz
// `FileTypeFromMime` no backend cair no default e classificar áudio como
// "documento". Este mapa força o mime certo a partir do nome do arquivo.
const EXTENSION_MIME_MAP: Record<string, string> = {
  'ogg': 'audio/ogg', 'oga': 'audio/ogg', 'opus': 'audio/ogg',
  'mp3': 'audio/mpeg', 'm4a': 'audio/mp4', 'aac': 'audio/aac',
  'wav': 'audio/wav', 'flac': 'audio/flac', 'amr': 'audio/amr',
  'mp4': 'video/mp4', 'mov': 'video/quicktime', 'webm': 'video/webm',
  '3gp': 'video/3gpp', 'avi': 'video/x-msvideo', 'mkv': 'video/x-matroska',
  'jpg': 'image/jpeg', 'jpeg': 'image/jpeg', 'png': 'image/png', 'gif': 'image/gif',
  'webp': 'image/webp', 'heic': 'image/heic', 'heif': 'image/heif', 'svg': 'image/svg+xml'
}

function extOf(name: string): string {
  const i = name.lastIndexOf('.')
  return i < 0 ? '' : name.slice(i + 1).toLowerCase()
}

// Decide o mime efetivo do arquivo: prefere o `file.type` do browser quando
// já é específico (image/*, audio/*, video/*); caso contrário deriva do
// nome. Mantém o type do recorder (`audio/ogg;codecs=opus`) intacto.
function effectiveMime(file: File): string {
  const t = file.type
  if (t && (t.startsWith('image/') || t.startsWith('audio/') || t.startsWith('video/'))) {
    return t
  }
  const fromExt = EXTENSION_MIME_MAP[extOf(file.name)]
  if (fromExt) return fromExt
  return t || 'application/octet-stream'
}

function normalizeFile(file: File): File {
  const target = effectiveMime(file)
  if (target === file.type) return file
  return new File([file], file.name, { type: target, lastModified: file.lastModified })
}

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
const richEditorRef = ref<RichEditor | null>(null)

const maxChars = computed(() => {
  const channelType = props.conversation.inbox?.channelType
  switch (channelType) {
    case 'Channel::Whatsapp': return 4096
    case 'Channel::Sms': return 160
    default: return 0
  }
})

const charCount = computed(() => reply.value.length)
const charExceeded = computed(() => maxChars.value > 0 && charCount.value > maxChars.value)

const promptPlaceholder = computed(() => {
  // Sem placeholder quando gravando ou com voice note no rascunho — esses
  // fluxos não aceitam texto (áudio do mic vai sem legenda), então o
  // "Digite uma mensagem..." só polui a UI do recorder.
  if (isRecording.value || hasRecordedAudio.value) return ''
  return mode.value === 'private'
    ? t('conversations.compose.privatePlaceholder')
    : t('conversations.compose.placeholder')
})

const composerShellClass = computed(() => [
  'mx-auto w-full rounded-lg px-3 py-2 shadow-lg ring transition-colors',
  expanded.value ? 'max-w-3xl flex h-[80vh] min-h-0 flex-col' : 'max-w-5xl xl:max-w-6xl 2xl:max-w-7xl',
  mode.value === 'private'
    ? 'bg-warning/5 ring-warning/25'
    : 'bg-elevated/85 ring-default'
])

function closeExpanded() {
  expanded.value = false
}

watch(expanded, (val) => {
  if (val) {
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        closeExpanded()
        window.removeEventListener('keydown', onKey)
      }
    }
    window.addEventListener('keydown', onKey)
  }
})

const slashMatch = computed(() => {
  const m = reply.value.match(/(?:^|\s)\/(\w*)$/)
  return m ? m[1] ?? '' : null
})

const cannedSearch = computed(() => slashMatch.value ?? '')

watch(slashMatch, (val) => {
  if (val !== null && canned.list.length) {
    cannedOpen.value = true
  } else if (val === null) {
    cannedOpen.value = false
  }
})

function handleCannedSelect(content: string) {
  richEditorRef.value?.replaceSlashCommand(content)
  cannedOpen.value = false
}

function onEmojiSelect(event: { i: string }) {
  richEditorRef.value?.insertAtCursor(event.i)
}

function onRecorded(file: File) {
  isRecording.value = false
  uploadFile(file, { isRecordedAudio: true })
}

// Espelha o Chatwoot (`AttachmentsPreview.vue`): áudios gravados saem da lista
// de chips e ganham um player inline. O envio compartilha o mesmo array
// `attachments` — só a renderização é dividida.
const recordedAudioAttachments = computed(() => attachments.value.filter(a => a.isRecordedAudio))
const nonRecordedAudioAttachments = computed(() => attachments.value.filter(a => !a.isRecordedAudio))
// Áudio gravado pelo botão do mic vai sem legenda — diferente de imagem/vídeo
// que aceitam caption. Enquanto o agente está gravando ou tem voice note no
// rascunho, o editor de texto fica trancado pra evitar texto que não seria
// enviado junto.
const hasRecordedAudio = computed(() => recordedAudioAttachments.value.length > 0)
const composeLocked = computed(() => sending.value || isRecording.value || hasRecordedAudio.value)

function onRecorderError(reason: 'permissionDenied' | 'unavailable' | 'unsupported') {
  isRecording.value = false
  const keys = { permissionDenied: 'voicePermissionDenied', unavailable: 'voiceUnavailable', unsupported: 'voiceUnsupported' }
  toast.add({ title: t(`conversations.compose.${keys[reason]}`), color: reason === 'unsupported' ? 'warning' : 'error' })
}

function buildInReplyTo() {
  const target = messages.replyingTo[props.conversation.id] ?? null
  if (!target) return undefined
  const isAgent = target.sender?.type === 'user' || target.senderType === 'USER'
  return { id: target.id, content: target.content ?? '', author: (isAgent ? auth.user?.name : props.conversation.meta?.sender?.name) ?? '' }
}

async function send() {
  const text = reply.value.trim()
  const readyAttachments = attachments.value.filter(a => a.url && !a.error)
  if (!auth.account?.id || (!text && !readyAttachments.length) || charExceeded.value) return
  sending.value = true

  const isPrivate = mode.value === 'private'
  const now = new Date().toISOString()
  const inReplyTo = buildInReplyTo()

  // Espelha o WhatsApp: cada anexo vira uma mensagem própria. Cada item
  // recebe seu próprio echo_id pra reconciliar otimista ↔ real um a um.
  // Sem anexos, manda 1 mensagem só de texto.
  type SendUnit = { echoId: string, tmpId: string, text: string, attachment: UploadedFile | null, isFirst: boolean }
  const units: SendUnit[] = []
  // Áudio gravado pelo mic do navegador segue sem legenda — caption só vai
  // pra primeira attachment que não seja voice note. Se só tem voice notes
  // e o agente tinha texto digitado, manda o texto como mensagem própria
  // antes dos áudios pra não perder a fala.
  const captionTargetIdx = readyAttachments.findIndex(a => !a.isRecordedAudio)
  const needsTextUnit = !!text && captionTargetIdx === -1 && readyAttachments.length > 0
  if (readyAttachments.length === 0) {
    const echoId = crypto.randomUUID()
    units.push({ echoId, tmpId: `tmp:${echoId}`, text, attachment: null, isFirst: true })
  } else {
    if (needsTextUnit) {
      const echoId = crypto.randomUUID()
      units.push({ echoId, tmpId: `tmp:${echoId}`, text, attachment: null, isFirst: true })
    }
    readyAttachments.forEach((a, idx) => {
      const echoId = crypto.randomUUID()
      const carriesCaption = !needsTextUnit && idx === captionTargetIdx
      units.push({
        echoId,
        tmpId: `tmp:${echoId}`,
        text: carriesCaption ? text : '',
        attachment: a,
        isFirst: !needsTextUnit && idx === 0
      })
    })
  }

  for (const u of units) {
    const contentAttrs: Record<string, unknown> = { format: 'markdown', echo_id: u.echoId }
    // in_reply_to fica só na primeira mensagem do batch — uma resposta lógica
    // não vira N respostas só porque o agente anexou vários arquivos.
    if (u.isFirst && inReplyTo) contentAttrs.in_reply_to = inReplyTo

    messages.upsert({
      id: u.tmpId,
      echoId: u.echoId,
      conversationId: props.conversation.id,
      inboxId: props.conversation.inboxId,
      accountId: props.conversation.accountId,
      content: u.text,
      contentType: u.attachment ? (u.attachment.file.type || 'file') : 'text',
      messageType: 1,
      senderType: 'USER',
      senderId: auth.user?.id ?? null,
      sourceId: null,
      private: isPrivate,
      status: 'sending',
      contentAttributes: contentAttrs,
      attachments: u.attachment
        ? [{
            id: -1, messageId: -1, fileType: u.attachment.file.type || 'file',
            fileKey: u.attachment.url, contentType: u.attachment.file.type, size: u.attachment.file.size, createdAt: now
          }]
        : [],
      createdAt: now,
      updatedAt: now
    } satisfies Message)
  }

  try {
    // Sequencial pra preservar ordem cronológica. Se um POST falhar, as
    // unidades anteriores já viraram mensagens reais; só removemos a tmp
    // que falhou e abortamos as restantes.
    for (const u of units) {
      const body: Record<string, unknown> = {
        message: u.text,
        echo_id: u.echoId,
        private: isPrivate,
        content_attributes: u.isFirst && inReplyTo
          ? { format: 'markdown', in_reply_to: inReplyTo }
          : { format: 'markdown' }
      }
      if (u.attachment) {
        body.attachments = [{
          file_key: u.attachment.url!,
          file_name: u.attachment.file.name,
          file_type: u.attachment.file.type,
          size: u.attachment.file.size
        }]
      }
      try {
        const res = await api<Message>(`/accounts/${auth.account.id}/conversations/${props.conversation.id}/messages`, { method: 'POST', body })
        if (res?.id) messages.upsert({ ...res, echoId: res.echoId ?? u.echoId })
      } catch (err) {
        messages.remove(u.tmpId)
        throw err
      }
    }
    reply.value = ''
    messages.clearReplyTarget(props.conversation.id)
    attachments.value = []
  } finally {
    sending.value = false
    // Re-foca o editor após enviar para que o agente possa digitar a
    // próxima mensagem sem clicar no input. O `setContent('')` que o
    // RichTextComposer faz no watcher de modelValue (após reply.value = '')
    // remove o foco; precisamos esperar o ciclo Vue propagar a mudança
    // antes de re-focar, daí o nextTick.
    await nextTick()
    richEditorRef.value?.focus()
  }
}

async function uploadFile(rawFile: File, opts?: { isRecordedAudio?: boolean }) {
  const file = normalizeFile(rawFile)
  const id = crypto.randomUUID()
  attachments.value.push({ id, file, uploading: true, isRecordedAudio: opts?.isRecordedAudio })
  try {
    const form = new FormData()
    form.append('file', file, file.name)
    const res = await api<{ path: string }>(`/accounts/${auth.account!.id}/uploads`, { method: 'POST', body: form })
    const idx = attachments.value.findIndex(a => a.id === id)
    if (idx >= 0) Object.assign(attachments.value[idx]!, { url: res.path, uploading: false })
  } catch (err) {
    console.error('[Composer] upload failed', err)
    const idx = attachments.value.findIndex(a => a.id === id)
    if (idx >= 0) Object.assign(attachments.value[idx]!, { uploading: false, error: 'Upload failed' })
  }
}

function onFileInputChange(e: Event) {
  const input = e.target as HTMLInputElement
  for (const file of input.files ? Array.from(input.files) : []) uploadFile(file)
  input.value = ''
}

onMounted(async () => {
  if (!auth.account?.id || canned.list.length) return
  try {
    const res = await api<{ payload: { id: string, shortCode: string, content: string }[] }>(`/accounts/${auth.account.id}/canned_responses`)
    const now = new Date().toISOString()
    for (const item of res.payload ?? []) {
      canned.upsert({ id: item.id, accountId: auth.account!.id, shortCode: item.shortCode, content: item.content, createdAt: now, updatedAt: now })
    }
  } catch { /* ignore */ }
})
</script>

<template>
  <!--
    `pb-[max(0.375rem,env(safe-area-inset-bottom))]` reserva o espaço da
    home-indicator no iOS Safari quando o composer está colado no rodapé;
    `pb-1.5` (0.375rem) é o padding base usado em telas sem notch.
  -->
  <div
    :class="expanded
      ? 'fixed inset-0 z-40 flex flex-col items-center justify-center bg-default/70 px-4 py-6 backdrop-blur'
      : 'shrink-0 border-t border-default/80 bg-default/95 px-2 pt-1.5 pb-[max(0.375rem,env(safe-area-inset-bottom))] sm:px-4'"
    @click.self="expanded && closeExpanded()"
  >
    <ClientOnly>
      <ConversationsRichTextComposer
        ref="richEditorRef"
        v-model="reply"
        :placeholder="promptPlaceholder"
        :disabled="composeLocked"
        :ui-class="composerShellClass"
        @submit="send"
        @file="uploadFile"
      >
        <template #header>
          <ConversationsComposerHeader
            v-model:mode="mode"
            :conversation="conversation"
            :char-count="charCount"
            :max-chars="maxChars"
          />

          <!--
            Voice note do mic vai sem legenda (não é imagem com caption), então
            o editor de texto fica desabilitado enquanto a gravação rola; a
            UI do recorder mora aqui no slot do header.
          -->
          <div v-if="isRecording" class="mb-2">
            <ConversationsAudioRecorder
              @recorded="onRecorded"
              @canceled="isRecording = false"
              @error="onRecorderError"
            />
          </div>

          <!--
            Áudios gravados pelo recorder ganham um player inline (estilo
            Chatwoot) — agente revisa antes de enviar. Áudios anexados via
            picker (kind 'audio' do dropdown) seguem como chip normal.
          -->
          <ConversationsRecordedAudioPreview
            v-for="att in recordedAudioAttachments"
            :key="att.id"
            :attachment="att"
            class="mb-2"
            @remove="(id: string) => attachments = attachments.filter(a => a.id !== id)"
          />

          <ConversationsComposerAttachmentsList
            :attachments="nonRecordedAudioAttachments"
            @remove="(id: string) => attachments = attachments.filter(a => a.id !== id)"
          />

          <input
            ref="fileInputRef"
            type="file"
            multiple
            :accept="ALL_ACCEPT"
            class="hidden"
            @change="onFileInputChange"
          >
        </template>

        <template #footer>
          <ConversationsComposerToolbar
            v-model:expanded="expanded"
            v-model:canned-open="cannedOpen"
            :rich-editor="richEditorRef"
            :mode="mode"
            :sending="sending"
            :disabled="charExceeded || (!reply.trim() && !attachments.length)"
            :canned-search="cannedSearch"
            :channel-type="conversation.inbox?.channelType"
            @submit="send"
            @attach="onAttachKind"
            @record="isRecording = true"
            @canned-select="handleCannedSelect"
            @emoji-select="onEmojiSelect"
          />
        </template>
      </ConversationsRichTextComposer>
    </ClientOnly>
  </div>
</template>
