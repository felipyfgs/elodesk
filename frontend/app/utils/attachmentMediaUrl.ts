interface AttachmentLike {
  id?: number | string
  dataUrl?: string | null
  fileUrl?: string | null
  path?: string | null
}

export function resolveAttachmentMediaUrl(att: AttachmentLike): string | null {
  if (att.dataUrl) return att.dataUrl
  if (att.fileUrl) return att.fileUrl
  return null
}

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
