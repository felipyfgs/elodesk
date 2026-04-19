package telegram

import (
	"strings"

	"github.com/microcosm-cc/bluemonday"
)

var htmlPolicy *bluemonday.Policy

func init() {
	p := bluemonday.NewPolicy()
	p.AllowElements("b", "strong", "i", "em", "u", "s", "strike", "del", "code", "pre", "a")
	p.AllowAttrs("href").OnElements("a")
	htmlPolicy = p
}

func MarkdownToHTML(md string) string {
	var b strings.Builder
	b.Grow(len(md) + 16)

	runes := []rune(md)
	i := 0
	n := len(runes)

	for i < n {
		ch := runes[i]

		if ch == '\\' && i+1 < n {
			next := runes[i+1]
			if next == '*' || next == '_' || next == '`' || next == '[' || next == ']' || next == '~' {
				b.WriteRune(next)
				i += 2
				continue
			}
		}

		if ch == '\\' {
			b.WriteRune(ch)
			i++
			continue
		}

		if ch == '*' && i+1 < n && runes[i+1] == '*' {
			end := findClosing(runes, i+2, "**")
			if end != -1 {
				b.WriteString("<b>")
				b.WriteString(MarkdownToHTML(string(runes[i+2 : end])))
				b.WriteString("</b>")
				i = end + 2
				continue
			}
		}

		if ch == '*' && i+1 < n {
			end := findClosing(runes, i+1, "*")
			if end != -1 {
				b.WriteString("<i>")
				b.WriteString(MarkdownToHTML(string(runes[i+1 : end])))
				b.WriteString("</i>")
				i = end + 1
				continue
			}
		}

		if ch == '_' && i+1 < n {
			end := findClosing(runes, i+1, "_")
			if end != -1 {
				b.WriteString("<i>")
				b.WriteString(MarkdownToHTML(string(runes[i+1 : end])))
				b.WriteString("</i>")
				i = end + 1
				continue
			}
		}

		if ch == '~' && i+1 < n && runes[i+1] == '~' {
			end := findClosing(runes, i+2, "~~")
			if end != -1 {
				b.WriteString("<s>")
				b.WriteString(MarkdownToHTML(string(runes[i+2 : end])))
				b.WriteString("</s>")
				i = end + 2
				continue
			}
		}

		if ch == '`' && i+2 < n && runes[i+1] == '`' && runes[i+2] == '`' {
			end := findClosing(runes, i+3, "```")
			if end != -1 {
				b.WriteString("<pre>")
				b.WriteString(escapeHTML(string(runes[i+3 : end])))
				b.WriteString("</pre>")
				i = end + 3
				continue
			}
		}

		if ch == '`' && i+1 < n {
			end := findClosing(runes, i+1, "`")
			if end != -1 {
				b.WriteString("<code>")
				b.WriteString(escapeHTML(string(runes[i+1 : end])))
				b.WriteString("</code>")
				i = end + 1
				continue
			}
		}

		if ch == '[' {
			linkEnd := strings.Index(string(runes[i:]), "](")
			if linkEnd != -1 {
				remaining := string(runes[i+linkEnd+2:])
				urlEnd := strings.Index(remaining, ")")
				if urlEnd != -1 {
					title := string(runes[i+1 : i+linkEnd])
					url := remaining[:urlEnd]
					b.WriteString(`<a href="`)
					b.WriteString(escapeAttr(url))
					b.WriteString(`">`)
					b.WriteString(MarkdownToHTML(title))
					b.WriteString("</a>")
					i = i + linkEnd + 2 + urlEnd + 1
					continue
				}
			}
		}

		if ch == '<' {
			b.WriteString("&lt;")
			i++
			continue
		}
		if ch == '>' {
			b.WriteString("&gt;")
			i++
			continue
		}
		if ch == '&' {
			b.WriteString("&amp;")
			i++
			continue
		}

		b.WriteRune(ch)
		i++
	}

	return htmlPolicy.Sanitize(b.String())
}

func findClosing(runes []rune, start int, marker string) int {
	markerRunes := []rune(marker)
	ml := len(markerRunes)
	i := start
	n := len(runes)
	for i <= n-ml {
		if runes[i] == markerRunes[0] {
			match := true
			for j := 1; j < ml; j++ {
				if runes[i+j] != markerRunes[j] {
					match = false
					break
				}
			}
			if match {
				return i
			}
		}
		i++
	}
	return -1
}

func escapeHTML(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	return s
}

func escapeAttr(s string) string {
	s = escapeHTML(s)
	s = strings.ReplaceAll(s, `"`, "&quot;")
	return s
}
