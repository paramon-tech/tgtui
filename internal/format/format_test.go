package format

import (
	"strings"
	"testing"

	"github.com/gotd/td/tg"
)

func TestRenderStyledText_Bold(t *testing.T) {
	text := "Hello World"
	entities := []tg.MessageEntityClass{
		&tg.MessageEntityBold{Offset: 0, Length: 5},
	}
	result := RenderStyledText(text, entities)
	if !strings.Contains(result, "\x1b[1mHello\x1b[0m") {
		t.Errorf("Expected bold ANSI wrapping 'Hello', got: %q", result)
	}
	if !strings.Contains(result, " World") {
		t.Errorf("Expected ' World' unstyled, got: %q", result)
	}
}

func TestRenderStyledText_NoEntities(t *testing.T) {
	text := "Hello\nWorld"
	result := RenderStyledText(text, nil)
	if result != "Hello World" {
		t.Errorf("Expected 'Hello World', got: %q", result)
	}
}

func TestRenderStyledText_Cyrillic(t *testing.T) {
	text := "Привет мир"
	entities := []tg.MessageEntityClass{
		&tg.MessageEntityBold{Offset: 0, Length: 6},
	}
	result := RenderStyledText(text, entities)
	if !strings.Contains(result, "\x1b[1mПривет\x1b[0m") {
		t.Errorf("Expected bold ANSI wrapping 'Привет', got: %q", result)
	}
}

func TestRenderStyledText_TextURL(t *testing.T) {
	text := "Click here for info"
	entities := []tg.MessageEntityClass{
		&tg.MessageEntityTextURL{Offset: 6, Length: 4, URL: "https://example.com"},
	}
	result := RenderStyledText(text, entities)
	if !strings.Contains(result, "\x1b[") {
		t.Errorf("Expected ANSI codes, got: %q", result)
	}
	if !strings.Contains(result, "here") {
		t.Errorf("Expected 'here' in output, got: %q", result)
	}
	if !strings.Contains(result, "example.com") {
		t.Errorf("Expected appended URL, got: %q", result)
	}
}

func TestRenderStyledText_MultipleEntities(t *testing.T) {
	text := "Bold and italic text"
	entities := []tg.MessageEntityClass{
		&tg.MessageEntityBold{Offset: 0, Length: 4},
		&tg.MessageEntityItalic{Offset: 9, Length: 6},
	}
	result := RenderStyledText(text, entities)
	if !strings.Contains(result, "\x1b[1mBold\x1b[0m") {
		t.Errorf("Expected bold 'Bold', got: %q", result)
	}
	if !strings.Contains(result, "\x1b[3mitalic\x1b[0m") {
		t.Errorf("Expected italic 'italic', got: %q", result)
	}
}

func TestRenderStyledText_URL(t *testing.T) {
	text := "Visit https://example.com today"
	entities := []tg.MessageEntityClass{
		&tg.MessageEntityURL{Offset: 6, Length: 19},
	}
	result := RenderStyledText(text, entities)
	// Should wrap the entire URL in one ANSI sequence, not per-character
	if !strings.Contains(result, "https://example.com\x1b[0m") {
		t.Errorf("Expected URL wrapped as one segment, got: %q", result)
	}
}

func TestRenderStyledText_Emoji(t *testing.T) {
	// Test with emoji (BMP + variation selector)
	text := "❗️Hello"
	entities := []tg.MessageEntityClass{
		&tg.MessageEntityBold{Offset: 2, Length: 5}, // "Hello" starts after ❗️ (2 UTF-16 units)
	}
	result := RenderStyledText(text, entities)
	if !strings.Contains(result, "\x1b[1mHello\x1b[0m") {
		t.Errorf("Expected bold 'Hello' after emoji, got: %q", result)
	}
}
