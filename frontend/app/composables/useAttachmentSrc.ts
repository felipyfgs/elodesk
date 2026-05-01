import type { MessageAttachment } from '~/utils/chatAdapter'
import { resolveAttachmentMediaUrl } from '~/utils/attachmentMediaUrl'

/**
 * Resolve a URL do anexo de forma SÍNCRONA — espelha exatamente o Chatwoot:
 * o backend já entrega `attachment.dataUrl` estável (token HMAC permanente).
 *
 * Antes: a URL era fetched via /media-url a cada render, e o token mudava em
 * cada chamada — o navegador via URL diferente toda vez e re-baixava a mídia.
 *
 * Agora: o template faz `<img :src="src.value">` e `src` já vem populado no
 * primeiro render. Como a URL é determinística pra (accountID, attachmentID),
 * o cache HTTP (Cache-Control: max-age=1y, immutable) acerta e dispensa o GET
 * em todas as re-aberturas da conversa.
 *
 * Os refs continuam aqui pra preservar a API consumida pelos templates
 * (`{ src, loading, errored }`), mas `loading` é sempre `false` — não há mais
 * round-trip pra observar.
 */
export function useAttachmentSrc(
  att: MessageAttachment,
  _accountId?: string | number,
) {
  const src = computed<string | null>(() => resolveAttachmentMediaUrl(att))
  const loading = ref(false)
  const errored = ref(false)
  return { src, loading, errored }
}
