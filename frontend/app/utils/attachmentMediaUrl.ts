// Resolução SÍNCRONA da URL de mídia. Espelha o Chatwoot: o backend já entrega
// `attachment.dataUrl` byte-a-byte estável (token HMAC permanente, espelhando
// o ActiveStorage signed_id). O cliente só precisa preferir essa URL ao
// fileUrl externo — qualquer round-trip extra a `/media-url` é eliminado.
//
// O cache HTTP do navegador (Cache-Control: max-age=1y, immutable em
// PublicAttachmentDownload) garante que re-aberturas da conversa não disparem
// novo download — a URL é determinística pra (accountID, attachmentID).
//
// Áudio recebe `?t=<timestamp>` opcional pra forçar re-leitura de metadata
// (mesmo padrão do Chatwoot timeStampAppendedURL). Imagens NUNCA recebem
// timestamp — destruiria o cache.

interface AttachmentLike {
  id?: number | string
  dataUrl?: string | null
  fileUrl?: string | null
  path?: string | null
}

// resolveAttachmentMediaUrl prefere, em ordem:
//   1. dataUrl injetada pelo backend (estável, com token HMAC permanente)
//   2. fileUrl (CDN externa do Meta/Telegram quando o blob ainda não chegou
//      ao MinIO — também estável pelo lado do canal)
// Retorna null quando não há URL pública resolvível (ex.: composer preview
// sem id ainda persistido — caller usa blob fetch autenticado).
export function resolveAttachmentMediaUrl(att: AttachmentLike): string | null {
  if (att.dataUrl) return att.dataUrl
  if (att.fileUrl) return att.fileUrl
  return null
}

// timeStampAppendedURL anexa `?t=<now>` à URL — usado APENAS pelo player de
// áudio pra forçar re-leitura de metadata quando o `<source>` re-monta.
// Imagens/vídeos NÃO devem passar por aqui: o cache HTTP do navegador é o que
// economiza o re-download.
export function timeStampAppendedURL(url: string): string {
  try {
    const u = new URL(url)
    if (!u.searchParams.has('t')) {
      u.searchParams.append('t', String(Date.now()))
    }
    return u.toString()
  } catch {
    return url
  }
}
