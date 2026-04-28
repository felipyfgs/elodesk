import type { MessageAttachment } from '~/utils/chatAdapter'

// Cache global de URLs assinadas por (accountId, attachmentId). Evita assinar
// o mesmo objeto N vezes ao re-renderizar a thread. As URLs do MinIO valem 15
// min — re-assinamos quando expira (cache miss volta naturalmente após reload).
const signedCache = new Map<string, string>()

function cacheKey(accountId: string | number, id: number): string {
  return `${accountId}:${id}`
}

/**
 * Resolve a URL do anexo em ordem de preferência:
 *   1. fileUrl (URL externa direta — CDN do Meta/Telegram)
 *   2. fileKey + id → signed URL via /attachments/:id/signed-url
 *
 * Retorna um ref reativo. Útil pro template fazer `<img :src="src">` sem
 * lidar com promises.
 */
export function useAttachmentSrc(
  att: MessageAttachment,
  accountId: string | number | undefined
) {
  const src = ref<string | null>(att.fileUrl ?? null)
  const loading = ref(false)
  const errored = ref(false)

  async function resolve() {
    if (src.value) return // já temos URL externa
    if (!att.id || !att.path || !accountId) return

    const key = cacheKey(accountId, att.id)
    const cached = signedCache.get(key)
    if (cached) {
      src.value = cached
      return
    }

    loading.value = true
    try {
      const api = useApi()
      // useApi normaliza snake_case → camelCase nas respostas.
      const res = await api<{ downloadUrl: string }>(
        `/accounts/${accountId}/attachments/${att.id}/signed-url`
      )
      if (res.downloadUrl) {
        signedCache.set(key, res.downloadUrl)
        src.value = res.downloadUrl
      }
    } catch {
      errored.value = true
    } finally {
      loading.value = false
    }
  }

  resolve()

  return { src, loading, errored }
}
