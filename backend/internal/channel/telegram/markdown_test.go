package telegram

import "testing"

func TestMarkdownToHTML_Bold(t *testing.T) {
	got := MarkdownToHTML("**hello**")
	want := "<b>hello</b>"
	if got != want {
		t.Errorf("MarkdownToHTML(%q) = %q, want %q", "**hello**", got, want)
	}
}

func TestMarkdownToHTML_Italic(t *testing.T) {
	got := MarkdownToHTML("*hello*")
	want := "<i>hello</i>"
	if got != want {
		t.Errorf("MarkdownToHTML(%q) = %q, want %q", "*hello*", got, want)
	}
}

func TestMarkdownToHTML_UnderscoreItalic(t *testing.T) {
	got := MarkdownToHTML("_hello_")
	want := "<i>hello</i>"
	if got != want {
		t.Errorf("MarkdownToHTML(%q) = %q, want %q", "_hello_", got, want)
	}
}

func TestMarkdownToHTML_Strikethrough(t *testing.T) {
	got := MarkdownToHTML("~~deleted~~")
	want := "<s>deleted</s>"
	if got != want {
		t.Errorf("MarkdownToHTML(%q) = %q, want %q", "~~deleted~~", got, want)
	}
}

func TestMarkdownToHTML_Code(t *testing.T) {
	got := MarkdownToHTML("`code`")
	want := "<code>code</code>"
	if got != want {
		t.Errorf("MarkdownToHTML(%q) = %q, want %q", "`code`", got, want)
	}
}

func TestMarkdownToHTML_CodeBlock(t *testing.T) {
	got := MarkdownToHTML("```line1\nline2```")
	want := "<pre>line1\nline2</pre>"
	if got != want {
		t.Errorf("MarkdownToHTML = %q, want %q", got, want)
	}
}

func TestMarkdownToHTML_Link(t *testing.T) {
	got := MarkdownToHTML("[click](https://example.com)")
	want := `<a href="https://example.com">click</a>`
	if got != want {
		t.Errorf("MarkdownToHTML = %q, want %q", got, want)
	}
}

func TestMarkdownToHTML_Mixed(t *testing.T) {
	input := "**bold** and *italic* and [link](https://x.com)"
	got := MarkdownToHTML(input)
	if got != "<b>bold</b> and <i>italic</i> and <a href=\"https://x.com\">link</a>" {
		t.Errorf("MarkdownToHTML = %q", got)
	}
}

func TestMarkdownToHTML_ScriptSanitized(t *testing.T) {
	got := MarkdownToHTML("<script>alert(1)</script>")
	if got != "&lt;script&gt;alert(1)&lt;/script&gt;" {
		t.Errorf("MarkdownToHTML should escape script tags, got %q", got)
	}
}

func TestMarkdownToHTML_EscapeHTML(t *testing.T) {
	got := MarkdownToHTML("<b>raw</b>")
	if got != "&lt;b&gt;raw&lt;/b&gt;" {
		t.Errorf("MarkdownToHTML should escape raw HTML, got %q", got)
	}
}

func TestMarkdownToHTML_PlainText(t *testing.T) {
	got := MarkdownToHTML("hello world")
	if got != "hello world" {
		t.Errorf("MarkdownToHTML = %q, want %q", got, "hello world")
	}
}

func TestMarkdownToHTML_Ampersand(t *testing.T) {
	got := MarkdownToHTML("a & b")
	if got != "a &amp; b" {
		t.Errorf("MarkdownToHTML = %q, want %q", got, "a &amp; b")
	}
}
