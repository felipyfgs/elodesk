<script setup lang="ts">
const props = defineProps<{
  src: string
  fileName: string
  fileSizeLabel?: string | null
}>()

const thumbDataUrl = ref<string | null>(null)
const pageCount = ref<number | null>(null)
const loading = ref(true)
const errored = ref(false)

// Renderiza a primeira página do PDF para um canvas e gera dataURL pra usar
// como thumbnail. PDF.js é carregado dinamicamente pra não inflar o bundle
// inicial — só quando uma mensagem com PDF aparece na tela.
async function renderThumbnail() {
  loading.value = true
  errored.value = false
  try {
    const pdfjs = await import('pdfjs-dist')
    // Worker precisa ser configurado uma vez. PDF.js v5 expõe worker como
    // `pdf.worker.min.mjs` via package exports — Vite resolve via ?url.
    if (!pdfjs.GlobalWorkerOptions.workerSrc) {
      const workerUrl = (await import('pdfjs-dist/build/pdf.worker.min.mjs?url')).default
      pdfjs.GlobalWorkerOptions.workerSrc = workerUrl
    }

    // verbosity: 0 (ERRORS only) silencia warnings benignos como
    // "TT: undefined function: 21" que aparecem quando o PDF usa features
    // de fonte TrueType que o PDF.js não implementa — não afeta render.
    const doc = await pdfjs.getDocument({ url: props.src, isEvalSupported: false, verbosity: 0 }).promise
    pageCount.value = doc.numPages

    const page = await doc.getPage(1)
    // Escala fixa que dá uma imagem nítida sem ser pesada (~ 220px de largura
    // dependendo do tamanho da página). 1.4 é um bom equilíbrio.
    const viewport = page.getViewport({ scale: 1.4 })
    const canvas = document.createElement('canvas')
    const ctx = canvas.getContext('2d')
    if (!ctx) throw new Error('canvas 2d context unavailable')
    canvas.width = viewport.width
    canvas.height = viewport.height

    await page.render({ canvasContext: ctx, viewport, canvas }).promise
    thumbDataUrl.value = canvas.toDataURL('image/png')

    doc.cleanup()
    doc.destroy()
  } catch (err) {
    console.error('[PdfPreview] failed to render', err)
    errored.value = true
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  renderThumbnail()
})

const metaLine = computed(() => {
  const parts: string[] = []
  if (pageCount.value != null) {
    parts.push(pageCount.value === 1 ? '1 página' : `${pageCount.value} páginas`)
  }
  parts.push('PDF')
  if (props.fileSizeLabel) parts.push(props.fileSizeLabel)
  return parts.join(' · ')
})
</script>

<template>
  <a
    :href="src"
    target="_blank"
    rel="noopener noreferrer"
    :title="fileName"
    class="block w-[260px] max-w-full text-current opacity-100 transition-opacity hover:opacity-90"
  >
    <!-- Thumbnail da primeira página -->
    <div class="relative aspect-[4/3] w-full overflow-hidden rounded-md bg-white/90">
      <div v-if="loading" class="absolute inset-0 flex items-center justify-center text-current/60">
        <UIcon name="i-lucide-loader-2" class="size-6 animate-spin" />
      </div>
      <div v-else-if="errored || !thumbDataUrl" class="absolute inset-0 flex flex-col items-center justify-center gap-1 text-current/60">
        <UIcon name="i-lucide-file-text" class="size-10" />
        <span class="text-xs">Pré-visualização indisponível</span>
      </div>
      <img
        v-else
        :src="thumbDataUrl"
        :alt="fileName"
        class="absolute inset-0 h-full w-full object-cover object-top"
        loading="lazy"
      >
    </div>

    <!-- Linha de metadata sem moldura: ícone + nome + páginas/tamanho. Cores
         seguem `currentColor` do balão (texto inverted no outgoing, default
         no incoming). -->
    <div class="mt-1.5 flex items-center gap-2 px-0.5">
      <span class="grid size-8 shrink-0 place-content-center rounded-md bg-white/85">
        <span class="text-[10px] font-bold text-error">PDF</span>
      </span>
      <div class="min-w-0 flex-1">
        <p class="truncate text-xs font-medium">
          {{ fileName }}
        </p>
        <p class="text-[11px] opacity-70">
          {{ metaLine }}
        </p>
      </div>
      <UIcon name="i-lucide-download" class="size-4 shrink-0 opacity-70" />
    </div>
  </a>
</template>
