import { defineNuxtConfig } from 'nuxt/config'

export default defineNuxtConfig({
  modules: [
    '@nuxt/eslint',
    '@nuxt/ui',
    '@vueuse/nuxt',
    '@pinia/nuxt',
    '@nuxtjs/i18n'
  ],

  ssr: false,

  devtools: {
    enabled: true
  },

  css: ['~/assets/css/main.css'],

  runtimeConfig: {
    public: {
      apiUrl: '',
      wsUrl: ''
    }
  },

  compatibilityDate: '2024-07-11',

  vite: {
    optimizeDeps: {
      include: [
        'zod',
        'zod/v4',
        'date-fns',
        'date-fns/locale',
        'pinia',
        'reka-ui',
        '@unovis/vue',
        '@unovis/ts',
        'vue3-emoji-picker',
        'libphonenumber-js',
        'libphonenumber-js/mobile/examples',
        '@tiptap/vue-3',
        '@tiptap/starter-kit',
        '@tiptap/extension-link',
        'tiptap-markdown',
        'wavesurfer.js',
        'wavesurfer.js/dist/plugins/record.esm.js',
        'vue-audio-visual',
        'markdown-it',
        'dompurify',
        'pdfjs-dist'
      ]
    }
  },

  // Workaround for Nuxt 4.4.2 duplicate useAppConfig warning (nuxt/nuxt#34812)
  hooks: {
    'nitro:config'(nitroConfig) {
      const imports = (nitroConfig as { imports?: { imports?: Array<{ name?: string }> } }).imports
      if (imports?.imports) {
        imports.imports = imports.imports.filter(i => i?.name !== 'useAppConfig')
      }
    }
  },

  eslint: {
    config: {
      stylistic: {
        commaDangle: 'never',
        braceStyle: '1tbs'
      }
    }
  },

  i18n: {
    strategy: 'no_prefix',
    defaultLocale: 'pt-BR',
    locales: [
      { code: 'pt-BR', name: 'Português (Brasil)', file: 'pt-BR.json' },
      { code: 'en', name: 'English', file: 'en.json' }
    ],
    detectBrowserLanguage: {
      useCookie: true,
      cookieKey: 'i18n_redirected',
      redirectOn: 'root',
      fallbackLocale: 'pt-BR'
    },
    bundle: {
      optimizeTranslationDirective: false
    }
  },

  pinia: {
    storesDirs: []
  }
})
