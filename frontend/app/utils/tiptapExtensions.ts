import Italic from '@tiptap/extension-italic'
import Blockquote from '@tiptap/extension-blockquote'

export const ItalicUnderscore = Italic.extend({
  addStorage() {
    return {
      ...this.parent?.(),
      markdown: {
        serialize: {
          open: '_',
          close: '_',
          mixable: true,
          expelEnclosingWhitespace: true
        },
        parse: { setup() { } }
      }
    }
  }
})

export const BlockquoteExt = Blockquote
