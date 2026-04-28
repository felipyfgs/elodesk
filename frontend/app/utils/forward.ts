import type { InjectionKey, Ref } from 'vue'

// Forward selection mode flag injected into MessageBubble.
// Provide `ref(false)` in Thread, inject with default `ref(false)` in children.
export const forwardSelectionModeKey: InjectionKey<Ref<boolean>> = Symbol('forward-selection-mode')

// Selected message IDs set injected into MessageBubble.
export const forwardSelectedIdsKey: InjectionKey<Ref<Set<string>>> = Symbol('forward-selected-ids')

// Backend enum: 0=image, 1=audio, 2=video, 3=file
export type FileTypeSpan = 'image' | 'audio' | 'video' | 'file'
