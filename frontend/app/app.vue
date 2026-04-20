<script setup lang="ts">
import * as locales from '@nuxt/ui/locale'

const colorMode = useColorMode()
const { locale } = useI18n()

const color = computed(() => colorMode.value === 'dark' ? '#1b1718' : 'white')

const uiLocaleKey = computed(() => locale.value === 'pt-BR' ? 'pt_br' : 'en')
const uiLocale = computed(() => locales[uiLocaleKey.value as keyof typeof locales])

useHead({
  meta: [
    { charset: 'utf-8' },
    { name: 'viewport', content: 'width=device-width, initial-scale=1' },
    { key: 'theme-color', name: 'theme-color', content: color }
  ],
  link: [
    { rel: 'icon', href: '/favicon.ico' }
  ],
  htmlAttrs: {
    lang: computed(() => locale.value),
    dir: computed(() => uiLocale.value?.dir ?? 'ltr')
  }
})

const title = 'wzap'
const description = 'Atendimento WhatsApp multi-tenant sobre o engine wzap.'

useSeoMeta({
  title,
  description,
  ogTitle: title,
  ogDescription: description,
  twitterCard: 'summary_large_image'
})
</script>

<template>
  <UApp :locale="uiLocale">
    <NuxtLoadingIndicator />

    <NuxtLayout>
      <NuxtPage />
    </NuxtLayout>
  </UApp>
</template>
