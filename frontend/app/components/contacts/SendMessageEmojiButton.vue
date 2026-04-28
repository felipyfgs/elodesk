<script setup lang="ts">
import EmojiPicker from 'vue3-emoji-picker'
import 'vue3-emoji-picker/css'

interface EmojiPickerSelectEvent { i: string, n: string[], r: string, t: string, u: string }

const emit = defineEmits<{
  select: [emoji: string]
}>()

const { t } = useI18n()
const colorMode = useColorMode()

const emojiTheme = computed<'dark' | 'light'>(() =>
  colorMode.value === 'dark' ? 'dark' : 'light'
)

const emojiStaticTexts = computed(() => ({
  placeholder: t('contactsSendMessage.emoji.searchPlaceholder')
}))

const emojiGroupNames = computed(() => ({
  smileys_people: t('contactsSendMessage.emoji.groups.smileysPeople'),
  animals_nature: t('contactsSendMessage.emoji.groups.animalsNature'),
  food_drink: t('contactsSendMessage.emoji.groups.foodDrink'),
  activities: t('contactsSendMessage.emoji.groups.activities'),
  travel_places: t('contactsSendMessage.emoji.groups.travelPlaces'),
  objects: t('contactsSendMessage.emoji.groups.objects'),
  symbols: t('contactsSendMessage.emoji.groups.symbols'),
  flags: t('contactsSendMessage.emoji.groups.flags'),
  recent: t('contactsSendMessage.emoji.groups.recent')
}))

function onEmojiSelect(event: EmojiPickerSelectEvent) {
  emit('select', event.i)
}
</script>

<template>
  <UPopover :content="{ align: 'start', side: 'top', sideOffset: 8 }" :ui="{ content: 'p-0 overflow-hidden' }">
    <UButton
      icon="i-lucide-smile"
      color="neutral"
      variant="ghost"
      size="sm"
      square
      :aria-label="t('contactsSendMessage.emojiAria')"
    />
    <template #content>
      <ClientOnly>
        <EmojiPicker
          :native="true"
          :hide-search="false"
          :disable-sticky-group-names="true"
          :disable-skin-tones="true"
          :theme="emojiTheme"
          :static-texts="emojiStaticTexts"
          :group-names="emojiGroupNames"
          @select="onEmojiSelect"
        />
      </ClientOnly>
    </template>
  </UPopover>
</template>
