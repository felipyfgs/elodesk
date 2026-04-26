<script setup lang="ts">
import { Editor, EditorContent } from '@tiptap/vue-3'
import StarterKit from '@tiptap/starter-kit'
import Link from '@tiptap/extension-link'
import { Markdown } from 'tiptap-markdown'

const props = defineProps<{
  modelValue: string
  placeholder?: string
  disabled?: boolean
  uiClass?: string | (string | null | undefined | false)[]
}>()

const emit = defineEmits<{
  'update:modelValue': [value: string]
  'submit': []
}>()

const editor = shallowRef<Editor | null>(null)

function getMarkdown(ed: Editor): string {
  const storage = ed.storage as unknown as { markdown?: { getMarkdown?: () => string } }
  return storage.markdown?.getMarkdown?.() ?? ed.getText()
}

onMounted(() => {
  const instance = new Editor({
    content: props.modelValue,
    editable: !props.disabled,
    extensions: [
      StarterKit.configure({ heading: false, horizontalRule: false, blockquote: false, link: false }),
      Link.configure({ openOnClick: false, HTMLAttributes: { rel: 'noopener noreferrer', target: '_blank' } }),
      Markdown.configure({ transformPastedText: true, transformCopiedText: true, breaks: true })
    ],
    editorProps: {
      attributes: {
        class: 'min-h-10 w-full text-sm leading-5 focus:outline-none'
      },
      handleKeyDown(_view, event) {
        if (event.key === 'Enter' && !event.shiftKey) {
          event.preventDefault()
          emit('submit')
          return true
        }
        return false
      }
    },
    onUpdate({ editor: ed }) {
      emit('update:modelValue', getMarkdown(ed as Editor))
    }
  })
  editor.value = instance
})

onBeforeUnmount(() => {
  editor.value?.destroy()
  editor.value = null
})

watch(() => props.modelValue, (val) => {
  const ed = editor.value
  if (!ed) return
  const current = getMarkdown(ed)
  if (val !== current) {
    ed.commands.setContent(val || '', { emitUpdate: false })
  }
})

watch(() => props.disabled, (val) => {
  editor.value?.setEditable(!val)
})

function insertAtCursor(text: string) {
  editor.value?.chain().focus().insertContent(text).run()
}

function replaceSlashCommand(content: string) {
  const ed = editor.value
  if (!ed) return
  const { state } = ed
  const { from } = state.selection
  const before = state.doc.textBetween(0, from, '\n', '\n')
  const match = before.match(/(^|\s)\/(\w*)$/)
  if (!match) {
    insertAtCursor(content)
    return
  }
  const slashLength = match[0].length - (match[1]?.length ?? 0)
  ed.chain().focus().deleteRange({ from: from - slashLength, to: from }).insertContent(content).run()
}

const isActive = (mark: string) => !!editor.value?.isActive(mark)

function toggleBold() {
  editor.value?.chain().focus().toggleBold().run()
}
function toggleItalic() {
  editor.value?.chain().focus().toggleItalic().run()
}
function toggleCode() {
  editor.value?.chain().focus().toggleCode().run()
}
function toggleBulletList() {
  editor.value?.chain().focus().toggleBulletList().run()
}
function toggleOrderedList() {
  editor.value?.chain().focus().toggleOrderedList().run()
}
function undo() {
  editor.value?.chain().focus().undo().run()
}
function redo() {
  editor.value?.chain().focus().redo().run()
}

function setLink(href: string) {
  const ed = editor.value
  if (!ed) return
  if (!href) {
    ed.chain().focus().unsetLink().run()
  } else {
    ed.chain().focus().extendMarkRange('link').setLink({ href }).run()
  }
}

function focus() {
  editor.value?.commands.focus()
}

const isEmpty = computed(() => !props.modelValue.trim())

defineExpose({
  insertAtCursor,
  replaceSlashCommand,
  focus,
  toggleBold,
  toggleItalic,
  toggleCode,
  toggleBulletList,
  toggleOrderedList,
  undo,
  redo,
  setLink,
  isActive
})
</script>

<template>
  <div :class="['flex min-h-0 flex-col gap-1.5', uiClass]">
    <slot name="header" />
    <div class="relative min-h-0 flex-1">
      <EditorContent
        :editor="editor ?? undefined"
        class="prose-sm h-full max-w-none overflow-y-auto"
      />
      <span
        v-if="isEmpty && placeholder"
        class="pointer-events-none absolute inset-0 text-sm leading-5 text-dimmed"
      >
        {{ placeholder }}
      </span>
    </div>
    <slot name="footer" />
  </div>
</template>

<style>
.ProseMirror p.is-editor-empty:first-child::before {
  content: attr(data-placeholder);
  color: var(--ui-text-dimmed);
  float: left;
  height: 0;
  pointer-events: none;
}
.ProseMirror {
  outline: none;
}
.ProseMirror p {
  margin: 0;
}
.ProseMirror ul, .ProseMirror ol {
  margin: 0.25rem 0;
  padding-left: 1.5rem;
}
.ProseMirror code {
  background: var(--ui-bg-elevated);
  padding: 0 0.25rem;
  border-radius: 0.25rem;
  font-size: 0.85em;
}
</style>
