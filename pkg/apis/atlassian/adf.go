package atlassian

import (
	"encoding/json"
	"strings"
)

// ADF node types.
const (
	adfTypeDoc         = "doc"
	adfTypeParagraph   = "paragraph"
	adfTypeText        = "text"
	adfTypeCodeBlock   = "codeBlock"
	adfTypeHeading     = "heading"
	adfTypeBulletList  = "bulletList"
	adfTypeOrderedList = "orderedList"
	adfTypeListItem    = "listItem"
	adfTypeHardBreak   = "hardBreak"
)

// ADF mark types.
const (
	adfMarkStrong    = "strong"
	adfMarkEm        = "em"
	adfMarkUnderline = "underline"
	adfMarkCode      = "code"
)

type adfMark struct {
	Type string `json:"type"`
}

type adfNode struct {
	Type    string    `json:"type"`
	Version int       `json:"version,omitempty"`
	Text    string    `json:"text,omitempty"`
	Marks   []adfMark `json:"marks,omitempty"`
	Content []adfNode `json:"content,omitempty"`
}

// Inline marker definitions: delimiter → ADF mark type.
// Order matters: longer delimiters are not needed here since all are single-char,
// but we process them by scanning left-to-right with a stack.
var inlineMarkers = []struct {
	delim    byte
	markType string
}{
	{'*', adfMarkStrong},
	{'_', adfMarkEm},
	{'+', adfMarkUnderline},
	{'`', adfMarkCode},
}

// TextToADF converts text with inline formatting into Atlassian Document Format.
//
// Supported markup:
//
//	*bold*        → strong
//	_italic_      → em
//	+underline+   → underline
//	`code`        → code
//	```           → codeBlock (multi-line)
//
// Nesting is supported: *bold _and italic_* produces text with both marks.
func TextToADF(text string) json.RawMessage {
	blocks := splitCodeBlocks(text)

	doc := adfNode{
		Type:    adfTypeDoc,
		Version: 1,
		Content: blocks,
	}

	data, _ := json.Marshal(doc)
	return data
}

// splitCodeBlocks splits text into paragraphs and code blocks (``` delimited).
// Fences are only recognised at the beginning of a line (or the beginning of the string).
func splitCodeBlocks(text string) []adfNode {
	var nodes []adfNode
	const fence = "```"

	for {
		start := indexFence(text, fence)
		if start == -1 {
			// Produce paragraphs when there is remaining text, or when no nodes
			// have been emitted yet (preserves empty-string → single empty paragraph).
			if text != "" || len(nodes) == 0 {
				nodes = append(nodes, textToParagraphs(text)...)
			}
			break
		}

		// Everything before the fence → paragraphs.
		before := text[:start]
		if before != "" {
			before = strings.TrimRight(before, "\n")
			nodes = append(nodes, textToParagraphs(before)...)
		}

		rest := text[start+len(fence):]
		// Skip optional language tag on the same line as opening fence.
		if nl := strings.IndexByte(rest, '\n'); nl != -1 {
			rest = rest[nl+1:]
		} else {
			// No newline after the opening fence (e.g. "```go" at EOF) —
			// the entire opening line is a language tag with no code body.
			rest = ""
		}

		end := indexFence(rest, fence)
		if end == -1 {
			// No closing fence — treat the rest as a code block.
			code := strings.TrimRight(rest, "\n")
			nodes = append(nodes, makeCodeBlock(code))
			break
		}

		code := rest[:end]
		code = strings.TrimRight(code, "\n")
		nodes = append(nodes, makeCodeBlock(code))

		// Continue after the closing fence.
		after := rest[end+len(fence):]
		// Strip only the single newline that immediately follows the closing fence.
		after = strings.TrimPrefix(after, "\n")
		text = after
	}
	return nodes
}

// indexFence returns the index of the first occurrence of fence that appears at
// the start of a line (i.e. at position 0 or right after '\n'). Returns -1 if
// no line-start fence is found.
func indexFence(s, fence string) int {
	off := 0
	for {
		i := strings.Index(s[off:], fence)
		if i == -1 {
			return -1
		}
		abs := off + i
		if abs == 0 || s[abs-1] == '\n' {
			return abs
		}
		off = abs + 1
	}
}

func makeCodeBlock(code string) adfNode {
	return adfNode{
		Type:    adfTypeCodeBlock,
		Content: []adfNode{{Type: adfTypeText, Text: code}},
	}
}

// textToParagraphs splits plain text lines into paragraph nodes with inline formatting.
func textToParagraphs(text string) []adfNode {
	lines := strings.Split(text, "\n")
	paragraphs := make([]adfNode, 0, len(lines))

	for _, line := range lines {
		p := adfNode{Type: adfTypeParagraph}
		if line != "" {
			p.Content = parseInline(line)
		}
		paragraphs = append(paragraphs, p)
	}
	return paragraphs
}

// parseInline parses a single line for inline formatting markers with nesting support.
// It uses a stack-based approach: when a delimiter is found, we push it;
// when the same delimiter is found again, we pop and wrap the content with the mark.
func parseInline(text string) []adfNode {
	type stackEntry struct {
		delim    byte
		markType string
		pos      int // position in the result slice where this group starts
	}

	var (
		result []adfNode
		stack  []stackEntry
		buf    strings.Builder
	)

	flushBuf := func() {
		if buf.Len() > 0 {
			result = append(result, adfNode{Type: adfTypeText, Text: buf.String()})
			buf.Reset()
		}
	}

	i := 0
	for i < len(text) {
		ch := text[i]

		// Check if this character is an inline delimiter.
		var marker *struct {
			delim    byte
			markType string
		}
		for idx := range inlineMarkers {
			if inlineMarkers[idx].delim == ch {
				marker = &inlineMarkers[idx]
				break
			}
		}

		if marker == nil {
			buf.WriteByte(ch)
			i++
			continue
		}

		// For backtick inline code: no nesting, find closing backtick directly.
		if ch == '`' {
			end := strings.IndexByte(text[i+1:], '`')
			if end == -1 {
				buf.WriteByte(ch)
				i++
				continue
			}
			flushBuf()
			result = append(result, adfNode{
				Type:  adfTypeText,
				Text:  text[i+1 : i+1+end],
				Marks: []adfMark{{Type: adfMarkCode}},
			})
			i = i + 1 + end + 1
			continue
		}

		// Check if we already have this delimiter on the stack (closing).
		found := -1
		for si := len(stack) - 1; si >= 0; si-- {
			if stack[si].delim == ch {
				found = si
				break
			}
		}

		if found >= 0 {
			// Closing: flush buffer, then wrap everything from stack entry's pos onward.
			flushBuf()
			entry := stack[found]
			stack = stack[:found]

			// Collect the nodes that belong to this marked group.
			group := make([]adfNode, len(result)-entry.pos)
			copy(group, result[entry.pos:])
			result = result[:entry.pos]

			// Apply the mark to every text node in the group.
			for gi := range group {
				group[gi].Marks = append(group[gi].Marks, adfMark{Type: entry.markType})
			}
			result = append(result, group...)
		} else {
			// Opening: flush buffer and push onto stack.
			flushBuf()
			stack = append(stack, stackEntry{
				delim:    ch,
				markType: marker.markType,
				pos:      len(result),
			})
		}
		i++
	}

	// Flush remaining buffer.
	flushBuf()

	// Any unclosed delimiters on the stack: re-insert delimiter characters as literal text.
	// We process from innermost to outermost to maintain correct positions.
	if len(stack) > 0 {
		for si := len(stack) - 1; si >= 0; si-- {
			entry := stack[si]
			// Insert the delimiter character as a text node at the original position.
			delimNode := adfNode{Type: adfTypeText, Text: string(entry.delim)}
			expanded := make([]adfNode, 0, len(result)+1)
			expanded = append(expanded, result[:entry.pos]...)
			expanded = append(expanded, delimNode)
			expanded = append(expanded, result[entry.pos:]...)
			result = expanded
		}
	}

	return result
}

// ADFToText extracts text from an ADF document, restoring inline formatting markers.
//
// Supported marks are converted back:
//
//	strong    → *text*
//	em        → _text_
//	underline → +text+
//	code      → `text`
//
// Code blocks are rendered with ``` fences.
// Lists (bullet/ordered), headings, and hardBreak nodes are also supported.
//
// If the input is a plain JSON string (e.g. from Jira API v2), it is returned as-is.
// Returns empty string if the input is empty or not valid JSON.
func ADFToText(adf json.RawMessage) string {
	if len(adf) == 0 {
		return ""
	}

	// Plain string fallback (Jira API v2 returns text fields as strings).
	var plain string
	if err := json.Unmarshal(adf, &plain); err == nil {
		return plain
	}

	var doc adfNode
	if err := json.Unmarshal(adf, &doc); err != nil {
		return ""
	}

	var b strings.Builder
	renderBlocks(&b, doc.Content, 0)
	return strings.TrimSpace(b.String())
}

func renderBlocks(b *strings.Builder, nodes []adfNode, depth int) {
	for _, node := range nodes {
		switch node.Type {
		case adfTypeCodeBlock:
			b.WriteString("```\n")
			extractPlainText(b, node.Content)
			b.WriteString("\n```\n")
		case adfTypeParagraph, adfTypeHeading:
			renderInlineNodes(b, node.Content)
			b.WriteByte('\n')
		case adfTypeBulletList, adfTypeOrderedList:
			renderBlocks(b, node.Content, depth)
		case adfTypeListItem:
			b.WriteString(strings.Repeat("  ", depth))
			b.WriteString("- ")
			renderBlocks(b, node.Content, depth+1)
		case adfTypeHardBreak:
			b.WriteByte('\n')
		case adfTypeText:
			renderInlineNodes(b, []adfNode{node})
		default:
			renderBlocks(b, node.Content, depth)
		}
	}
}

var markToDelim = map[string]byte{
	adfMarkStrong:    '*',
	adfMarkEm:        '_',
	adfMarkUnderline: '+',
	adfMarkCode:      '`',
}

func renderInlineNodes(b *strings.Builder, nodes []adfNode) {
	for _, node := range nodes {
		if node.Type == adfTypeHardBreak {
			b.WriteByte('\n')
			continue
		}
		if node.Type == adfTypeText {
			// Determine which delimiters wrap this text node.
			var open, close strings.Builder
			for _, m := range node.Marks {
				if d, ok := markToDelim[m.Type]; ok {
					open.WriteByte(d)
					close.WriteByte(d)
				}
			}
			b.WriteString(open.String())
			b.WriteString(node.Text)
			// Close in reverse order.
			cl := close.String()
			for j := len(cl) - 1; j >= 0; j-- {
				b.WriteByte(cl[j])
			}
		}
		if len(node.Content) > 0 {
			renderInlineNodes(b, node.Content)
		}
	}
}

func extractPlainText(b *strings.Builder, nodes []adfNode) {
	for _, node := range nodes {
		if node.Text != "" {
			b.WriteString(node.Text)
		}
		if len(node.Content) > 0 {
			extractPlainText(b, node.Content)
		}
	}
}
