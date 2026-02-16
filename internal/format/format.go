package format

import (
	"fmt"
	"sort"
	"strings"
	"unicode/utf16"

	"github.com/charmbracelet/lipgloss"
	"github.com/gotd/td/tg"
)

type entityInfo struct {
	ent     tg.MessageEntityClass
	endByte int
}

// ANSI SGR codes
const (
	ansiReset         = "\x1b[0m"
	ansiBold          = "1"
	ansiItalic        = "3"
	ansiUnderline     = "4"
	ansiStrikethrough = "9"
	ansiReverse       = "7"
)

func ansiFg(r, g, b int) string {
	return fmt.Sprintf("38;2;%d;%d;%d", r, g, b)
}

func ansiBg(r, g, b int) string {
	return fmt.Sprintf("48;2;%d;%d;%d", r, g, b)
}

func ansiWrap(text string, codes []string) string {
	if len(codes) == 0 {
		return text
	}
	return "\x1b[" + strings.Join(codes, ";") + "m" + text + ansiReset
}

// RenderStyledText applies Telegram entity formatting to message text,
// returning a styled string suitable for terminal display.
// Newlines are replaced with spaces so each message occupies one visual line.
func RenderStyledText(text string, entities []tg.MessageEntityClass) string {
	return renderStyledText(text, entities, false)
}

// RenderStyledTextMultiline applies Telegram entity formatting and preserves
// newlines. Long lines are word-wrapped to the given width (ANSI-aware).
func RenderStyledTextMultiline(text string, entities []tg.MessageEntityClass, width int) string {
	result := renderStyledText(text, entities, true)
	if width > 0 {
		return lipgloss.NewStyle().Width(width).Render(result)
	}
	return result
}

func renderStyledText(text string, entities []tg.MessageEntityClass, preserveNewlines bool) string {
	if len(entities) == 0 {
		if preserveNewlines {
			return text
		}
		return strings.ReplaceAll(text, "\n", " ")
	}

	utf16Units := utf16.Encode([]rune(text))
	utf16ToByteOffset := buildUTF16ToByteMap(text)

	type boundary struct {
		bytePos int
		isStart bool
		entity  tg.MessageEntityClass
	}

	var bounds []boundary
	for _, ent := range entities {
		offset := entityOffset(ent)
		length := entityLength(ent)
		end := offset + length

		if offset < 0 {
			offset = 0
		}
		if end > len(utf16Units) {
			end = len(utf16Units)
		}
		if offset >= end {
			continue
		}

		startByte := utf16ToByteOffset[offset]
		endByte := utf16ToByteOffset[end]

		bounds = append(bounds,
			boundary{bytePos: startByte, isStart: true, entity: ent},
			boundary{bytePos: endByte, isStart: false, entity: ent},
		)
	}

	sort.Slice(bounds, func(i, j int) bool {
		if bounds[i].bytePos != bounds[j].bytePos {
			return bounds[i].bytePos < bounds[j].bytePos
		}
		return !bounds[i].isStart && bounds[j].isStart
	})

	splitPoints := []int{0}
	seen := map[int]bool{0: true}
	for _, b := range bounds {
		if !seen[b.bytePos] {
			splitPoints = append(splitPoints, b.bytePos)
			seen[b.bytePos] = true
		}
	}
	if !seen[len(text)] {
		splitPoints = append(splitPoints, len(text))
	}
	sort.Ints(splitPoints)

	var active []entityInfo

	startMap := map[int][]entityInfo{}
	endMap := map[int][]tg.MessageEntityClass{}
	for _, ent := range entities {
		offset := entityOffset(ent)
		length := entityLength(ent)
		end := offset + length
		if offset < 0 {
			offset = 0
		}
		if end > len(utf16Units) {
			end = len(utf16Units)
		}
		if offset >= end {
			continue
		}
		startByte := utf16ToByteOffset[offset]
		endByte := utf16ToByteOffset[end]
		startMap[startByte] = append(startMap[startByte], entityInfo{ent: ent, endByte: endByte})
		endMap[endByte] = append(endMap[endByte], ent)
	}

	var result strings.Builder

	for i := 0; i < len(splitPoints)-1; i++ {
		pos := splitPoints[i]
		nextPos := splitPoints[i+1]

		if ends, ok := endMap[pos]; ok {
			for _, e := range ends {
				for j := 0; j < len(active); j++ {
					if active[j].ent == e {
						active = append(active[:j], active[j+1:]...)
						j--
					}
				}
			}
		}

		if starts, ok := startMap[pos]; ok {
			active = append(active, starts...)
		}

		segment := text[pos:nextPos]
		if !preserveNewlines {
			segment = strings.ReplaceAll(segment, "\n", " ")
		}

		if len(active) == 0 {
			result.WriteString(segment)
			continue
		}

		codes, suffix := buildANSICodes(active)
		result.WriteString(ansiWrap(segment, codes))
		if suffix != "" {
			result.WriteString(suffix)
		}
	}

	return result.String()
}

func buildANSICodes(active []entityInfo) ([]string, string) {
	var codes []string
	var suffix string
	isBlockquote := false

	for _, a := range active {
		switch e := a.ent.(type) {
		case *tg.MessageEntityBold:
			codes = append(codes, ansiBold)
		case *tg.MessageEntityItalic:
			codes = append(codes, ansiItalic)
		case *tg.MessageEntityUnderline:
			codes = append(codes, ansiUnderline)
		case *tg.MessageEntityStrike:
			codes = append(codes, ansiStrikethrough)
		case *tg.MessageEntityCode:
			codes = append(codes, ansiFg(255, 158, 100), ansiBg(26, 27, 38))
		case *tg.MessageEntityPre:
			codes = append(codes, ansiFg(169, 177, 214), ansiBg(26, 27, 38))
		case *tg.MessageEntityURL:
			codes = append(codes, ansiUnderline, ansiFg(122, 162, 247))
		case *tg.MessageEntityTextURL:
			codes = append(codes, ansiUnderline, ansiFg(122, 162, 247))
			if e.URL != "" {
				suffix = ansiWrap(" ("+e.URL+")", []string{ansiFg(86, 95, 137)})
			}
		case *tg.MessageEntityEmail:
			codes = append(codes, ansiUnderline, ansiFg(122, 162, 247))
		case *tg.MessageEntityMention:
			codes = append(codes, ansiFg(187, 154, 247))
		case *tg.MessageEntityMentionName:
			codes = append(codes, ansiFg(187, 154, 247))
		case *tg.MessageEntityHashtag:
			codes = append(codes, ansiFg(125, 207, 255))
		case *tg.MessageEntityBotCommand:
			codes = append(codes, ansiFg(125, 207, 255))
		case *tg.MessageEntityCashtag:
			codes = append(codes, ansiFg(125, 207, 255))
		case *tg.MessageEntitySpoiler:
			codes = append(codes, ansiReverse)
		case *tg.MessageEntityBlockquote:
			isBlockquote = true
		case *tg.MessageEntityPhone:
			codes = append(codes, ansiFg(122, 162, 247))
		default:
			_ = e
		}
	}

	if isBlockquote {
		codes = append(codes, ansiFg(169, 177, 214))
	}

	return codes, suffix
}

// buildUTF16ToByteMap builds a mapping from UTF-16 code unit index to byte offset in the Go string.
func buildUTF16ToByteMap(s string) []int {
	runes := []rune(s)
	utf16Len := 0
	for _, r := range runes {
		if r >= 0x10000 {
			utf16Len += 2
		} else {
			utf16Len++
		}
	}

	mapping := make([]int, utf16Len+1)
	byteOffset := 0
	utf16Idx := 0
	for _, r := range runes {
		mapping[utf16Idx] = byteOffset
		byteOffset += len(string(r))
		if r >= 0x10000 {
			utf16Idx++
			mapping[utf16Idx] = byteOffset
		}
		utf16Idx++
	}
	mapping[utf16Idx] = byteOffset

	return mapping
}

func entityOffset(e tg.MessageEntityClass) int {
	switch ent := e.(type) {
	case *tg.MessageEntityUnknown:
		return ent.Offset
	case *tg.MessageEntityMention:
		return ent.Offset
	case *tg.MessageEntityHashtag:
		return ent.Offset
	case *tg.MessageEntityBotCommand:
		return ent.Offset
	case *tg.MessageEntityURL:
		return ent.Offset
	case *tg.MessageEntityEmail:
		return ent.Offset
	case *tg.MessageEntityBold:
		return ent.Offset
	case *tg.MessageEntityItalic:
		return ent.Offset
	case *tg.MessageEntityCode:
		return ent.Offset
	case *tg.MessageEntityPre:
		return ent.Offset
	case *tg.MessageEntityTextURL:
		return ent.Offset
	case *tg.MessageEntityMentionName:
		return ent.Offset
	case *tg.MessageEntityPhone:
		return ent.Offset
	case *tg.MessageEntityCashtag:
		return ent.Offset
	case *tg.MessageEntityUnderline:
		return ent.Offset
	case *tg.MessageEntityStrike:
		return ent.Offset
	case *tg.MessageEntityBlockquote:
		return ent.Offset
	case *tg.MessageEntitySpoiler:
		return ent.Offset
	case *tg.MessageEntityCustomEmoji:
		return ent.Offset
	default:
		return 0
	}
}

func entityLength(e tg.MessageEntityClass) int {
	switch ent := e.(type) {
	case *tg.MessageEntityUnknown:
		return ent.Length
	case *tg.MessageEntityMention:
		return ent.Length
	case *tg.MessageEntityHashtag:
		return ent.Length
	case *tg.MessageEntityBotCommand:
		return ent.Length
	case *tg.MessageEntityURL:
		return ent.Length
	case *tg.MessageEntityEmail:
		return ent.Length
	case *tg.MessageEntityBold:
		return ent.Length
	case *tg.MessageEntityItalic:
		return ent.Length
	case *tg.MessageEntityCode:
		return ent.Length
	case *tg.MessageEntityPre:
		return ent.Length
	case *tg.MessageEntityTextURL:
		return ent.Length
	case *tg.MessageEntityMentionName:
		return ent.Length
	case *tg.MessageEntityPhone:
		return ent.Length
	case *tg.MessageEntityCashtag:
		return ent.Length
	case *tg.MessageEntityUnderline:
		return ent.Length
	case *tg.MessageEntityStrike:
		return ent.Length
	case *tg.MessageEntityBlockquote:
		return ent.Length
	case *tg.MessageEntitySpoiler:
		return ent.Length
	case *tg.MessageEntityCustomEmoji:
		return ent.Length
	default:
		return 0
	}
}
