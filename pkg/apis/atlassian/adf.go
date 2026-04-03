package atlassian

import (
	"encoding/json"
	"strings"
)

// ADF node types.
const (
	adfTypeDoc       = "doc"
	adfTypeParagraph = "paragraph"
	adfTypeText      = "text"
)

type adfNode struct {
	Type    string    `json:"type"`
	Version int       `json:"version,omitempty"`
	Text    string    `json:"text,omitempty"`
	Content []adfNode `json:"content,omitempty"`
}

// TextToADF converts plain text into Atlassian Document Format.
// Each non-empty line becomes a separate paragraph node.
// Empty lines produce empty paragraphs (visual spacing).
func TextToADF(text string) json.RawMessage {
	lines := strings.Split(text, "\n")
	paragraphs := make([]adfNode, 0, len(lines))

	for _, line := range lines {
		p := adfNode{Type: adfTypeParagraph}
		if line != "" {
			p.Content = []adfNode{{Type: adfTypeText, Text: line}}
		}
		paragraphs = append(paragraphs, p)
	}

	doc := adfNode{
		Type:    adfTypeDoc,
		Version: 1,
		Content: paragraphs,
	}

	data, _ := json.Marshal(doc)
	return data
}

// ADFToText extracts plain text from an Atlassian Document Format document.
// Formatting marks (bold, links, etc.) are discarded; only text content is preserved.
// Returns empty string if the input is not valid ADF.
func ADFToText(adf json.RawMessage) string {
	if len(adf) == 0 {
		return ""
	}

	var doc adfNode
	if err := json.Unmarshal(adf, &doc); err != nil {
		return ""
	}

	var b strings.Builder
	extractText(&b, doc.Content, true)
	return strings.TrimRight(b.String(), "\n")
}

func extractText(b *strings.Builder, nodes []adfNode, topLevel bool) {
	for i, node := range nodes {
		if node.Text != "" {
			b.WriteString(node.Text)
		}
		if len(node.Content) > 0 {
			extractText(b, node.Content, false)
		}
		if topLevel && i < len(nodes)-1 {
			b.WriteByte('\n')
		}
	}
}
