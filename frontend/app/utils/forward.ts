import type { InjectionKey, Ref } from 'vue'

export const forwardSelectionModeKey: InjectionKey<Ref<boolean>> = Symbol('forward-selection-mode')

export const forwardSelectedIdsKey: InjectionKey<Ref<Set<string>>> = Symbol('forward-selected-ids')

export type FileTypeSpan = 'image' | 'audio' | 'video' | 'file'
