import Italic from '@tiptap/extension-italic'
import Blockquote from '@tiptap/extension-blockquote'

// Italic em markdown padrão pode ser `*X*` ou `_X_`. Por padrão o
// tiptap-markdown serializa como `*X*`, mas isso colide com o dialeto WA
// (onde `*X*` é bold). Configuramos pra preferir `_X_`, que é interpretado
// como italic em AMBOS os formatos — assim o display do elodesk e o WA
// renderizam consistentemente.
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
        parse: { setup() { /* default markdown-it italic já cobre _X_ */ } }
      }
    }
  }
})

// Blockquote padrão do StarterKit já serializa como `> texto`. Re-exportamos
// pra ficar explícito que estamos habilitando.
export const BlockquoteExt = Blockquote
