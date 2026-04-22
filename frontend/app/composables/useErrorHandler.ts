import type { FetchError } from 'ofetch'

export interface ErrorHandlerOptions {
  title?: string
  description?: string
  icon?: string
  duration?: number
  silent?: boolean
  onRetry?: () => void | Promise<void>
}

interface ApiErrorResponse {
  data?: {
    message?: string
    error?: string
  }
  response?: {
    _data?: {
      message?: string
      error?: string
    }
  }
}

export const useErrorHandler = () => {
  const toast = useToast()
  const { t } = useI18n()

  const extractErrorMessage = (error: unknown): string => {
    if (!error) return t('common.error')

    const err = error as ApiErrorResponse & FetchError

    return (
      err.data?.message
      || err.data?.error
      || err.response?._data?.message
      || err.response?._data?.error
      || err.message
      || t('common.error')
    )
  }

  const handle = (error: unknown, options: ErrorHandlerOptions = {}) => {
    if (options.silent) return

    const message = extractErrorMessage(error)
    const title = options.title || t('common.error')
    const description = options.description || message

    const toastConfig: Parameters<typeof toast.add>[0] = {
      title,
      description,
      color: 'error',
      icon: options.icon || 'i-lucide-alert-circle',
      duration: options.duration
    }

    if (options.onRetry) {
      toastConfig.actions = [{
        label: t('common.retry'),
        color: 'error',
        variant: 'ghost',
        onClick: async () => {
          try {
            await options.onRetry?.()
          } catch (retryError) {
            handle(retryError, { ...options, onRetry: undefined })
          }
        }
      }]
    }

    toast.add(toastConfig)

    if (import.meta.dev) {
      console.error('[useErrorHandler]', error)
    }
  }

  const success = (title: string, description?: string) => {
    toast.add({
      title,
      description,
      color: 'success',
      icon: 'i-lucide-check-circle'
    })
  }

  const warning = (title: string, description?: string) => {
    toast.add({
      title,
      description,
      color: 'warning',
      icon: 'i-lucide-alert-triangle'
    })
  }

  const info = (title: string, description?: string) => {
    toast.add({
      title,
      description,
      color: 'info',
      icon: 'i-lucide-info'
    })
  }

  return {
    handle,
    success,
    warning,
    info,
    extractErrorMessage
  }
}
