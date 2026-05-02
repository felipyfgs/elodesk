import type { MessageAttachment } from '~/utils/chatAdapter'
import { resolveAttachmentMediaUrl } from '~/utils/attachmentMediaUrl'

export function useAttachmentSrc(
  att: MessageAttachment,
  _accountId?: string | number
) {
  const src = computed<string | null>(() => resolveAttachmentMediaUrl(att))
  const isLoading = ref(false)
  const errored = ref(false)
  return { src, isLoading, errored }
}
