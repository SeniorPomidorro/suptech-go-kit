package atlassian

import (
	"encoding/json"
	"strings"
	"testing"
)

// helper to unmarshal ADF and return the doc-level node.
func mustParseADF(t *testing.T, raw json.RawMessage) adfNode {
	t.Helper()
	var doc adfNode
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatalf("unmarshal ADF: %v", err)
	}
	if doc.Type != adfTypeDoc || doc.Version != 1 {
		t.Fatalf("unexpected doc header: type=%q version=%d", doc.Type, doc.Version)
	}
	return doc
}

func hasMarkType(marks []adfMark, markType string) bool {
	for _, m := range marks {
		if m.Type == markType {
			return true
		}
	}
	return false
}

// collectText concatenates all text from nodes (ignoring marks).
func collectText(nodes []adfNode) string {
	var b strings.Builder
	for _, n := range nodes {
		b.WriteString(n.Text)
	}
	return b.String()
}

// --- parseInline: happy path ---

func TestParseInline_Bold(t *testing.T) {
	t.Parallel()
	nodes := parseInline("hello *world*")
	if len(nodes) != 2 {
		t.Fatalf("expected 2 nodes, got %d", len(nodes))
	}
	if nodes[0].Text != "hello " || len(nodes[0].Marks) != 0 {
		t.Fatalf("node 0: want plain 'hello ', got %q marks=%v", nodes[0].Text, nodes[0].Marks)
	}
	if nodes[1].Text != "world" || len(nodes[1].Marks) != 1 || nodes[1].Marks[0].Type != adfMarkStrong {
		t.Fatalf("node 1: want bold 'world', got %q marks=%v", nodes[1].Text, nodes[1].Marks)
	}
}

func TestParseInline_Italic(t *testing.T) {
	t.Parallel()
	nodes := parseInline("_italic_ text")
	if len(nodes) != 2 {
		t.Fatalf("expected 2 nodes, got %d", len(nodes))
	}
	if nodes[0].Text != "italic" || len(nodes[0].Marks) != 1 || nodes[0].Marks[0].Type != adfMarkEm {
		t.Fatalf("node 0: want italic 'italic', got %q marks=%v", nodes[0].Text, nodes[0].Marks)
	}
	if nodes[1].Text != " text" {
		t.Fatalf("node 1: want ' text', got %q", nodes[1].Text)
	}
}

func TestParseInline_Underline(t *testing.T) {
	t.Parallel()
	nodes := parseInline("+underlined+")
	if len(nodes) != 1 {
		t.Fatalf("expected 1 node, got %d", len(nodes))
	}
	if nodes[0].Text != "underlined" || len(nodes[0].Marks) != 1 || nodes[0].Marks[0].Type != adfMarkUnderline {
		t.Fatalf("want underline 'underlined', got %q marks=%v", nodes[0].Text, nodes[0].Marks)
	}
}

func TestParseInline_Code(t *testing.T) {
	t.Parallel()
	nodes := parseInline("run `go test` now")
	if len(nodes) != 3 {
		t.Fatalf("expected 3 nodes, got %d", len(nodes))
	}
	if nodes[0].Text != "run " {
		t.Fatalf("node 0: want 'run ', got %q", nodes[0].Text)
	}
	if nodes[1].Text != "go test" || len(nodes[1].Marks) != 1 || nodes[1].Marks[0].Type != adfMarkCode {
		t.Fatalf("node 1: want code 'go test', got %q marks=%v", nodes[1].Text, nodes[1].Marks)
	}
	if nodes[2].Text != " now" {
		t.Fatalf("node 2: want ' now', got %q", nodes[2].Text)
	}
}

func TestParseInline_PlainText(t *testing.T) {
	t.Parallel()
	nodes := parseInline("no formatting here")
	if len(nodes) != 1 {
		t.Fatalf("expected 1 node, got %d", len(nodes))
	}
	if nodes[0].Text != "no formatting here" || len(nodes[0].Marks) != 0 {
		t.Fatalf("unexpected: %+v", nodes[0])
	}
}

func TestParseInline_Nested_BoldItalic(t *testing.T) {
	t.Parallel()
	nodes := parseInline("*bold _and italic_*")
	if len(nodes) != 2 {
		t.Fatalf("expected 2 nodes, got %d: %+v", len(nodes), nodes)
	}
	if nodes[0].Text != "bold " || !hasMarkType(nodes[0].Marks, adfMarkStrong) {
		t.Fatalf("node 0: want strong 'bold ', got %q marks=%v", nodes[0].Text, nodes[0].Marks)
	}
	if nodes[1].Text != "and italic" || !hasMarkType(nodes[1].Marks, adfMarkStrong) || !hasMarkType(nodes[1].Marks, adfMarkEm) {
		t.Fatalf("node 1: want strong+em 'and italic', got %q marks=%v", nodes[1].Text, nodes[1].Marks)
	}
}

func TestParseInline_NestedTriple(t *testing.T) {
	t.Parallel()
	nodes := parseInline("*bold _italic +all+_*")
	if len(nodes) != 3 {
		t.Fatalf("expected 3 nodes, got %d: %+v", len(nodes), nodes)
	}
	if nodes[0].Text != "bold " || !hasMarkType(nodes[0].Marks, adfMarkStrong) {
		t.Fatalf("node 0: want strong 'bold ', got %q marks=%v", nodes[0].Text, nodes[0].Marks)
	}
	if nodes[1].Text != "italic " || !hasMarkType(nodes[1].Marks, adfMarkStrong) || !hasMarkType(nodes[1].Marks, adfMarkEm) {
		t.Fatalf("node 1: want strong+em 'italic ', got %q marks=%v", nodes[1].Text, nodes[1].Marks)
	}
	n2 := nodes[2]
	if n2.Text != "all" ||
		!hasMarkType(n2.Marks, adfMarkStrong) ||
		!hasMarkType(n2.Marks, adfMarkEm) ||
		!hasMarkType(n2.Marks, adfMarkUnderline) {
		t.Fatalf("node 2: want strong+em+underline 'all', got %q marks=%v", n2.Text, n2.Marks)
	}
}

func TestParseInline_AdjacentFormats(t *testing.T) {
	t.Parallel()
	// *bold*_italic_ — two adjacent formatted segments.
	nodes := parseInline("*bold*_italic_")
	if len(nodes) != 2 {
		t.Fatalf("expected 2 nodes, got %d: %+v", len(nodes), nodes)
	}
	if nodes[0].Text != "bold" || !hasMarkType(nodes[0].Marks, adfMarkStrong) {
		t.Fatalf("node 0: want strong 'bold', got %q marks=%v", nodes[0].Text, nodes[0].Marks)
	}
	if nodes[1].Text != "italic" || !hasMarkType(nodes[1].Marks, adfMarkEm) {
		t.Fatalf("node 1: want em 'italic', got %q marks=%v", nodes[1].Text, nodes[1].Marks)
	}
}

func TestParseInline_EntireLineBold(t *testing.T) {
	t.Parallel()
	nodes := parseInline("*everything bold*")
	if len(nodes) != 1 {
		t.Fatalf("expected 1 node, got %d", len(nodes))
	}
	if nodes[0].Text != "everything bold" || !hasMarkType(nodes[0].Marks, adfMarkStrong) {
		t.Fatalf("want strong 'everything bold', got %q marks=%v", nodes[0].Text, nodes[0].Marks)
	}
}

func TestParseInline_MultipleCodeSpans(t *testing.T) {
	t.Parallel()
	nodes := parseInline("`a` and `b`")
	if len(nodes) != 3 {
		t.Fatalf("expected 3 nodes, got %d", len(nodes))
	}
	if nodes[0].Text != "a" || !hasMarkType(nodes[0].Marks, adfMarkCode) {
		t.Fatalf("node 0: want code 'a', got %q marks=%v", nodes[0].Text, nodes[0].Marks)
	}
	if nodes[1].Text != " and " {
		t.Fatalf("node 1: want ' and ', got %q", nodes[1].Text)
	}
	if nodes[2].Text != "b" || !hasMarkType(nodes[2].Marks, adfMarkCode) {
		t.Fatalf("node 2: want code 'b', got %q marks=%v", nodes[2].Text, nodes[2].Marks)
	}
}

func TestParseInline_EmptyCodeSpan(t *testing.T) {
	t.Parallel()
	// `` — two backticks with nothing between them.
	nodes := parseInline("before ``after")
	// Should produce code node with empty text.
	if len(nodes) != 3 {
		t.Fatalf("expected 3 nodes, got %d: %+v", len(nodes), nodes)
	}
	if nodes[1].Text != "" || !hasMarkType(nodes[1].Marks, adfMarkCode) {
		t.Fatalf("node 1: want empty code, got %q marks=%v", nodes[1].Text, nodes[1].Marks)
	}
}

// --- parseInline: unhappy path ---

func TestParseInline_EmptyString(t *testing.T) {
	t.Parallel()
	nodes := parseInline("")
	if len(nodes) != 0 {
		t.Fatalf("expected 0 nodes for empty string, got %d", len(nodes))
	}
}

func TestParseInline_UnclosedBold(t *testing.T) {
	t.Parallel()
	nodes := parseInline("hello *world")
	text := collectText(nodes)
	if text != "hello *world" {
		t.Fatalf("want 'hello *world', got %q", text)
	}
	for _, n := range nodes {
		if len(n.Marks) > 0 {
			t.Fatalf("unclosed bold produced marks: %v on %q", n.Marks, n.Text)
		}
	}
}

func TestParseInline_UnclosedItalic(t *testing.T) {
	t.Parallel()
	nodes := parseInline("_no close")
	text := collectText(nodes)
	if text != "_no close" {
		t.Fatalf("want '_no close', got %q", text)
	}
	for _, n := range nodes {
		if len(n.Marks) > 0 {
			t.Fatalf("unclosed italic produced marks: %v", n.Marks)
		}
	}
}

func TestParseInline_UnclosedUnderline(t *testing.T) {
	t.Parallel()
	nodes := parseInline("+no close")
	text := collectText(nodes)
	if text != "+no close" {
		t.Fatalf("want '+no close', got %q", text)
	}
	for _, n := range nodes {
		if len(n.Marks) > 0 {
			t.Fatalf("unclosed underline produced marks: %v", n.Marks)
		}
	}
}

func TestParseInline_UnclosedBacktick(t *testing.T) {
	t.Parallel()
	nodes := parseInline("hello `world")
	text := collectText(nodes)
	if text != "hello `world" {
		t.Fatalf("want 'hello `world', got %q", text)
	}
	for _, n := range nodes {
		if len(n.Marks) > 0 {
			t.Fatalf("unclosed backtick produced marks: %v", n.Marks)
		}
	}
}

func TestParseInline_MultipleUnclosed(t *testing.T) {
	t.Parallel()
	// Two different unclosed delimiters.
	nodes := parseInline("*bold _italic")
	text := collectText(nodes)
	if text != "*bold _italic" {
		t.Fatalf("want '*bold _italic', got %q", text)
	}
	for _, n := range nodes {
		if len(n.Marks) > 0 {
			t.Fatalf("multiple unclosed produced marks: %v on %q", n.Marks, n.Text)
		}
	}
}

func TestParseInline_EmptyBold(t *testing.T) {
	t.Parallel()
	// ** — empty bold markers.
	nodes := parseInline("before ** after")
	// The two * cancel each other out producing no text between them.
	text := collectText(nodes)
	if text != "before  after" {
		t.Fatalf("want 'before  after', got %q", text)
	}
}

func TestParseInline_OnlyDelimiter(t *testing.T) {
	t.Parallel()
	nodes := parseInline("*")
	text := collectText(nodes)
	if text != "*" {
		t.Fatalf("want '*', got %q", text)
	}
}

func TestParseInline_OnlyBacktick(t *testing.T) {
	t.Parallel()
	nodes := parseInline("`")
	text := collectText(nodes)
	if text != "`" {
		t.Fatalf("want '`', got %q", text)
	}
}

func TestParseInline_DelimitersInsideCode(t *testing.T) {
	t.Parallel()
	// Backtick code should not parse inner delimiters.
	nodes := parseInline("`*not bold*`")
	if len(nodes) != 1 {
		t.Fatalf("expected 1 node, got %d: %+v", len(nodes), nodes)
	}
	if nodes[0].Text != "*not bold*" || !hasMarkType(nodes[0].Marks, adfMarkCode) {
		t.Fatalf("want code '*not bold*', got %q marks=%v", nodes[0].Text, nodes[0].Marks)
	}
}

func TestParseInline_PartialNesting_ClosedInWrongOrder(t *testing.T) {
	t.Parallel()
	// *bold _italic* end_ — when * closes, it consumes the inner _ from the stack too.
	// The second _ has no matching opener, so it becomes a literal character.
	nodes := parseInline("*bold _italic* end_")
	text := collectText(nodes)
	// "bold " [strong], "italic" [strong] from the *...* group,
	// then " end" (plain) and "_" (literal unclosed delimiter).
	if text != "bold italic end_" {
		t.Fatalf("text not preserved: want 'bold italic end_', got %q", text)
	}
	// Verify the first two nodes got the strong mark.
	if !hasMarkType(nodes[0].Marks, adfMarkStrong) || !hasMarkType(nodes[1].Marks, adfMarkStrong) {
		t.Fatalf("first two nodes should be strong: %+v, %+v", nodes[0].Marks, nodes[1].Marks)
	}
}

// --- TextToADF: happy path ---

func TestTextToADF_PlainSingleLine(t *testing.T) {
	t.Parallel()
	doc := mustParseADF(t, TextToADF("hello"))
	if len(doc.Content) != 1 || doc.Content[0].Type != adfTypeParagraph {
		t.Fatalf("expected 1 paragraph, got %+v", doc.Content)
	}
	if doc.Content[0].Content[0].Text != "hello" {
		t.Fatalf("text: %q", doc.Content[0].Content[0].Text)
	}
}

func TestTextToADF_PlainMultiLine(t *testing.T) {
	t.Parallel()
	doc := mustParseADF(t, TextToADF("first\nsecond\nthird"))
	if len(doc.Content) != 3 {
		t.Fatalf("expected 3 paragraphs, got %d", len(doc.Content))
	}
	for i, want := range []string{"first", "second", "third"} {
		if doc.Content[i].Content[0].Text != want {
			t.Fatalf("paragraph %d: want %q, got %q", i, want, doc.Content[i].Content[0].Text)
		}
	}
}

func TestTextToADF_EmptyLineSpacing(t *testing.T) {
	t.Parallel()
	doc := mustParseADF(t, TextToADF("before\n\nafter"))
	if len(doc.Content) != 3 {
		t.Fatalf("expected 3 paragraphs (with empty), got %d", len(doc.Content))
	}
	// Middle paragraph should have no content (empty paragraph).
	if len(doc.Content[1].Content) != 0 {
		t.Fatalf("middle paragraph should be empty, got %+v", doc.Content[1].Content)
	}
}

func TestTextToADF_FormattedMultiLine(t *testing.T) {
	t.Parallel()
	doc := mustParseADF(t, TextToADF("*bold line*\n_italic line_"))
	if len(doc.Content) != 2 {
		t.Fatalf("expected 2 paragraphs, got %d", len(doc.Content))
	}
	// First paragraph should have bold text.
	if !hasMarkType(doc.Content[0].Content[0].Marks, adfMarkStrong) {
		t.Fatalf("paragraph 0: expected strong mark")
	}
	// Second paragraph should have italic text.
	if !hasMarkType(doc.Content[1].Content[0].Marks, adfMarkEm) {
		t.Fatalf("paragraph 1: expected em mark")
	}
}

func TestTextToADF_CodeBlock(t *testing.T) {
	t.Parallel()
	input := "before\n```\nfmt.Println(\"hi\")\n```\nafter"
	doc := mustParseADF(t, TextToADF(input))

	if len(doc.Content) != 3 {
		t.Fatalf("expected 3 blocks, got %d", len(doc.Content))
	}
	if doc.Content[0].Type != adfTypeParagraph {
		t.Fatalf("block 0: want paragraph, got %q", doc.Content[0].Type)
	}
	if doc.Content[1].Type != adfTypeCodeBlock {
		t.Fatalf("block 1: want codeBlock, got %q", doc.Content[1].Type)
	}
	if doc.Content[1].Content[0].Text != `fmt.Println("hi")` {
		t.Fatalf("code block text: %q", doc.Content[1].Content[0].Text)
	}
	if doc.Content[2].Type != adfTypeParagraph {
		t.Fatalf("block 2: want paragraph, got %q", doc.Content[2].Type)
	}
}

func TestTextToADF_CodeBlockWithLang(t *testing.T) {
	t.Parallel()
	doc := mustParseADF(t, TextToADF("```go\npackage main\n```"))
	if len(doc.Content) != 1 {
		t.Fatalf("expected 1 block, got %d", len(doc.Content))
	}
	if doc.Content[0].Type != adfTypeCodeBlock {
		t.Fatalf("want codeBlock, got %q", doc.Content[0].Type)
	}
	if doc.Content[0].Content[0].Text != "package main" {
		t.Fatalf("code text: %q", doc.Content[0].Content[0].Text)
	}
}

func TestTextToADF_MultipleCodeBlocks(t *testing.T) {
	t.Parallel()
	input := "```\nblock1\n```\nmiddle\n```\nblock2\n```"
	doc := mustParseADF(t, TextToADF(input))
	if len(doc.Content) != 3 {
		t.Fatalf("expected 3 blocks, got %d", len(doc.Content))
	}
	if doc.Content[0].Type != adfTypeCodeBlock || doc.Content[0].Content[0].Text != "block1" {
		t.Fatalf("block 0: want codeBlock 'block1', got type=%q text=%q", doc.Content[0].Type, doc.Content[0].Content[0].Text)
	}
	if doc.Content[1].Type != adfTypeParagraph {
		t.Fatalf("block 1: want paragraph, got %q", doc.Content[1].Type)
	}
	if doc.Content[2].Type != adfTypeCodeBlock || doc.Content[2].Content[0].Text != "block2" {
		t.Fatalf("block 2: want codeBlock 'block2', got type=%q text=%q", doc.Content[2].Type, doc.Content[2].Content[0].Text)
	}
}

func TestTextToADF_CodeBlockMultiLine(t *testing.T) {
	t.Parallel()
	input := "```\nline1\nline2\nline3\n```"
	doc := mustParseADF(t, TextToADF(input))
	if len(doc.Content) != 1 || doc.Content[0].Type != adfTypeCodeBlock {
		t.Fatalf("expected 1 codeBlock, got %+v", doc.Content)
	}
	if doc.Content[0].Content[0].Text != "line1\nline2\nline3" {
		t.Fatalf("code text: %q", doc.Content[0].Content[0].Text)
	}
}

// --- TextToADF: unhappy path ---

func TestTextToADF_EmptyString(t *testing.T) {
	t.Parallel()
	doc := mustParseADF(t, TextToADF(""))
	if len(doc.Content) != 1 {
		t.Fatalf("expected 1 empty paragraph, got %d blocks", len(doc.Content))
	}
	if doc.Content[0].Type != adfTypeParagraph || len(doc.Content[0].Content) != 0 {
		t.Fatalf("expected empty paragraph, got %+v", doc.Content[0])
	}
}

func TestTextToADF_CodeBlockUnclosed(t *testing.T) {
	t.Parallel()
	doc := mustParseADF(t, TextToADF("```\nsome code"))
	if len(doc.Content) != 1 || doc.Content[0].Type != adfTypeCodeBlock {
		t.Fatalf("expected 1 codeBlock, got %+v", doc.Content)
	}
	if doc.Content[0].Content[0].Text != "some code" {
		t.Fatalf("code text: %q", doc.Content[0].Content[0].Text)
	}
}

func TestTextToADF_CodeBlockEmpty(t *testing.T) {
	t.Parallel()
	doc := mustParseADF(t, TextToADF("```\n```"))
	if len(doc.Content) != 1 || doc.Content[0].Type != adfTypeCodeBlock {
		t.Fatalf("expected 1 codeBlock, got %+v", doc.Content)
	}
}

func TestTextToADF_OnlyFenceMarkers(t *testing.T) {
	t.Parallel()
	// Just ``` with nothing else — empty code block.
	doc := mustParseADF(t, TextToADF("```"))
	if len(doc.Content) != 1 || doc.Content[0].Type != adfTypeCodeBlock {
		t.Fatalf("expected 1 codeBlock, got %+v", doc.Content)
	}
}

func TestTextToADF_FenceMidSentence(t *testing.T) {
	t.Parallel()
	// Triple backticks in the middle of a sentence should NOT be treated as a fence.
	input := "use ``` to open a code fence"
	doc := mustParseADF(t, TextToADF(input))
	if len(doc.Content) != 1 || doc.Content[0].Type != adfTypeParagraph {
		t.Fatalf("expected 1 paragraph, got %d blocks: %+v", len(doc.Content), doc.Content)
	}
	text := ADFToText(TextToADF(input))
	if text != input {
		t.Fatalf("text corrupted:\ninput:  %q\noutput: %q", input, text)
	}
}

func TestTextToADF_BlankLineAfterCodeBlock(t *testing.T) {
	t.Parallel()
	// Blank line between code block and next paragraph must be preserved.
	input := "before\n```\ncode\n```\n\nafter"
	doc := mustParseADF(t, TextToADF(input))
	// Expect: paragraph("before"), codeBlock, paragraph(""), paragraph("after")
	if len(doc.Content) != 4 {
		t.Fatalf("expected 4 blocks, got %d", len(doc.Content))
	}
	if doc.Content[0].Type != adfTypeParagraph {
		t.Fatalf("block 0: want paragraph, got %q", doc.Content[0].Type)
	}
	if doc.Content[1].Type != adfTypeCodeBlock {
		t.Fatalf("block 1: want codeBlock, got %q", doc.Content[1].Type)
	}
	// Block 2: empty paragraph (the blank line).
	if doc.Content[2].Type != adfTypeParagraph || len(doc.Content[2].Content) != 0 {
		t.Fatalf("block 2: want empty paragraph, got %+v", doc.Content[2])
	}
	if doc.Content[3].Type != adfTypeParagraph {
		t.Fatalf("block 3: want paragraph, got %q", doc.Content[3].Type)
	}
}

func TestTextToADF_LangTagNoNewline(t *testing.T) {
	t.Parallel()
	// "```go" at EOF with no newline — should produce empty code block, not "go" as code content.
	doc := mustParseADF(t, TextToADF("```go"))
	if len(doc.Content) != 1 || doc.Content[0].Type != adfTypeCodeBlock {
		t.Fatalf("expected 1 codeBlock, got %+v", doc.Content)
	}
	if len(doc.Content[0].Content) > 0 && doc.Content[0].Content[0].Text != "" {
		t.Fatalf("language tag leaked as code content: %q", doc.Content[0].Content[0].Text)
	}
}

func TestTextToADF_UnclosedFormattingPreservesText(t *testing.T) {
	t.Parallel()
	input := "hello *unclosed bold"
	mustParseADF(t, TextToADF(input)) // validate it's valid ADF
	// The text must be fully preserved even if formatting is broken.
	text := ADFToText(TextToADF(input))
	if text != input {
		t.Fatalf("unclosed formatting lost text:\ninput:  %q\noutput: %q", input, text)
	}
}

func TestTextToADF_BacktickInsideFormatting(t *testing.T) {
	t.Parallel()
	// Inline code inside bold context — code takes precedence.
	input := "*bold `code` bold*"
	doc := mustParseADF(t, TextToADF(input))
	para := doc.Content[0]
	// Should have 3 nodes: "bold " [strong], "code" [code, strong], " bold" [strong]
	if len(para.Content) != 3 {
		t.Fatalf("expected 3 inline nodes, got %d: %+v", len(para.Content), para.Content)
	}
	if !hasMarkType(para.Content[1].Marks, adfMarkCode) {
		t.Fatalf("code span should have code mark: %+v", para.Content[1])
	}
}

// --- ADFToText: happy path ---

func TestADFToText_SimpleParagraphs(t *testing.T) {
	t.Parallel()
	adf := `{"type":"doc","version":1,"content":[{"type":"paragraph","content":[{"type":"text","text":"hello"}]},{"type":"paragraph","content":[{"type":"text","text":"world"}]}]}`
	got := ADFToText(json.RawMessage(adf))
	if got != "hello\nworld" {
		t.Fatalf("want %q, got %q", "hello\nworld", got)
	}
}

func TestADFToText_BoldMark(t *testing.T) {
	t.Parallel()
	adf := `{"type":"doc","version":1,"content":[{"type":"paragraph","content":[{"type":"text","text":"normal "},{"type":"text","text":"bold","marks":[{"type":"strong"}]},{"type":"text","text":" end"}]}]}`
	got := ADFToText(json.RawMessage(adf))
	if got != "normal *bold* end" {
		t.Fatalf("want %q, got %q", "normal *bold* end", got)
	}
}

func TestADFToText_ItalicMark(t *testing.T) {
	t.Parallel()
	adf := `{"type":"doc","version":1,"content":[{"type":"paragraph","content":[{"type":"text","text":"word","marks":[{"type":"em"}]}]}]}`
	got := ADFToText(json.RawMessage(adf))
	if got != "_word_" {
		t.Fatalf("want %q, got %q", "_word_", got)
	}
}

func TestADFToText_UnderlineMark(t *testing.T) {
	t.Parallel()
	adf := `{"type":"doc","version":1,"content":[{"type":"paragraph","content":[{"type":"text","text":"word","marks":[{"type":"underline"}]}]}]}`
	got := ADFToText(json.RawMessage(adf))
	if got != "+word+" {
		t.Fatalf("want %q, got %q", "+word+", got)
	}
}

func TestADFToText_CodeMark(t *testing.T) {
	t.Parallel()
	adf := `{"type":"doc","version":1,"content":[{"type":"paragraph","content":[{"type":"text","text":"cmd","marks":[{"type":"code"}]}]}]}`
	got := ADFToText(json.RawMessage(adf))
	if got != "`cmd`" {
		t.Fatalf("want %q, got %q", "`cmd`", got)
	}
}

func TestADFToText_MultipleMarks(t *testing.T) {
	t.Parallel()
	adf := `{"type":"doc","version":1,"content":[{"type":"paragraph","content":[{"type":"text","text":"both","marks":[{"type":"strong"},{"type":"em"}]}]}]}`
	got := ADFToText(json.RawMessage(adf))
	if got != "*_both_*" {
		t.Fatalf("want %q, got %q", "*_both_*", got)
	}
}

func TestADFToText_CodeBlock(t *testing.T) {
	t.Parallel()
	adf := `{"type":"doc","version":1,"content":[{"type":"codeBlock","content":[{"type":"text","text":"x := 1"}]}]}`
	got := ADFToText(json.RawMessage(adf))
	want := "```\nx := 1\n```"
	if got != want {
		t.Fatalf("want %q, got %q", want, got)
	}
}

func TestADFToText_CodeBlockMultiLine(t *testing.T) {
	t.Parallel()
	adf := `{"type":"doc","version":1,"content":[{"type":"codeBlock","content":[{"type":"text","text":"a\nb\nc"}]}]}`
	got := ADFToText(json.RawMessage(adf))
	want := "```\na\nb\nc\n```"
	if got != want {
		t.Fatalf("want %q, got %q", want, got)
	}
}

func TestADFToText_MixedBlocks(t *testing.T) {
	t.Parallel()
	adf := `{"type":"doc","version":1,"content":[{"type":"paragraph","content":[{"type":"text","text":"intro"}]},{"type":"codeBlock","content":[{"type":"text","text":"code"}]},{"type":"paragraph","content":[{"type":"text","text":"outro"}]}]}`
	got := ADFToText(json.RawMessage(adf))
	want := "intro\n```\ncode\n```\noutro"
	if got != want {
		t.Fatalf("want %q, got %q", want, got)
	}
}

func TestADFToText_EmptyParagraph(t *testing.T) {
	t.Parallel()
	adf := `{"type":"doc","version":1,"content":[{"type":"paragraph","content":[{"type":"text","text":"before"}]},{"type":"paragraph"},{"type":"paragraph","content":[{"type":"text","text":"after"}]}]}`
	got := ADFToText(json.RawMessage(adf))
	if got != "before\n\nafter" {
		t.Fatalf("want %q, got %q", "before\n\nafter", got)
	}
}

// --- ADFToText: unhappy path ---

func TestADFToText_EmptyInput(t *testing.T) {
	t.Parallel()
	got := ADFToText(json.RawMessage(""))
	if got != "" {
		t.Fatalf("want empty, got %q", got)
	}
}

func TestADFToText_NilInput(t *testing.T) {
	t.Parallel()
	got := ADFToText(nil)
	if got != "" {
		t.Fatalf("want empty, got %q", got)
	}
}

func TestADFToText_InvalidJSON(t *testing.T) {
	t.Parallel()
	got := ADFToText(json.RawMessage("not json"))
	if got != "" {
		t.Fatalf("want empty, got %q", got)
	}
}

func TestADFToText_EmptyDoc(t *testing.T) {
	t.Parallel()
	got := ADFToText(json.RawMessage(`{"type":"doc","version":1,"content":[]}`))
	if got != "" {
		t.Fatalf("want empty, got %q", got)
	}
}

func TestADFToText_UnknownMarkType(t *testing.T) {
	t.Parallel()
	// Unknown marks should be silently ignored (no delimiters added).
	adf := `{"type":"doc","version":1,"content":[{"type":"paragraph","content":[{"type":"text","text":"linked","marks":[{"type":"link"}]}]}]}`
	got := ADFToText(json.RawMessage(adf))
	if got != "linked" {
		t.Fatalf("want %q, got %q", "linked", got)
	}
}

func TestADFToText_UnknownNodeType(t *testing.T) {
	t.Parallel()
	// Unknown top-level node type — should extract text via default fallback.
	adf := `{"type":"doc","version":1,"content":[{"type":"mediaGroup","content":[{"type":"text","text":"fallback text"}]}]}`
	got := ADFToText(json.RawMessage(adf))
	if got != "fallback text" {
		t.Fatalf("want %q, got %q", "fallback text", got)
	}
}

func TestADFToText_NoContent(t *testing.T) {
	t.Parallel()
	// Doc with a paragraph that has no content field at all.
	adf := `{"type":"doc","version":1,"content":[{"type":"paragraph"}]}`
	got := ADFToText(json.RawMessage(adf))
	if got != "" {
		t.Fatalf("want empty, got %q", got)
	}
}

func TestADFToText_InlineCardWithContent(t *testing.T) {
	t.Parallel()
	// A paragraph with a non-text node (e.g. inlineCard) that has child text content.
	// renderInlineNodes should recurse into it and extract the text.
	adf := `{"type":"doc","version":1,"content":[{"type":"paragraph","content":[{"type":"text","text":"see "},{"type":"inlineCard","content":[{"type":"text","text":"PROJ-1"}]}]}]}`
	got := ADFToText(json.RawMessage(adf))
	if got != "see PROJ-1" {
		t.Fatalf("want %q, got %q", "see PROJ-1", got)
	}
}

func TestADFToText_MixedKnownAndUnknownMarks(t *testing.T) {
	t.Parallel()
	// Text with both known (strong) and unknown (link) marks.
	adf := `{"type":"doc","version":1,"content":[{"type":"paragraph","content":[{"type":"text","text":"click","marks":[{"type":"strong"},{"type":"link"}]}]}]}`
	got := ADFToText(json.RawMessage(adf))
	if got != "*click*" {
		t.Fatalf("want %q, got %q", "*click*", got)
	}
}

// --- ADFToText: plain string fallback ---

func TestADFToText_PlainStringFallback(t *testing.T) {
	t.Parallel()
	adf := `"just a plain string from API v2"`
	got := ADFToText(json.RawMessage(adf))
	if got != "just a plain string from API v2" {
		t.Fatalf("want %q, got %q", "just a plain string from API v2", got)
	}
}

func TestADFToText_PlainStringEmpty(t *testing.T) {
	t.Parallel()
	adf := `""`
	got := ADFToText(json.RawMessage(adf))
	if got != "" {
		t.Fatalf("want %q, got %q", "", got)
	}
}

func TestADFToText_PlainStringWithSpecialChars(t *testing.T) {
	t.Parallel()
	adf := `"line1\nline2\ttab"`
	got := ADFToText(json.RawMessage(adf))
	if got != "line1\nline2\ttab" {
		t.Fatalf("want %q, got %q", "line1\nline2\ttab", got)
	}
}

// --- ADFToText: heading ---

func TestADFToText_Heading(t *testing.T) {
	t.Parallel()
	adf := `{"type":"doc","version":1,"content":[{"type":"heading","content":[{"type":"text","text":"Title"}]},{"type":"paragraph","content":[{"type":"text","text":"body"}]}]}`
	got := ADFToText(json.RawMessage(adf))
	if got != "Title\nbody" {
		t.Fatalf("want %q, got %q", "Title\nbody", got)
	}
}

func TestADFToText_HeadingWithMarks(t *testing.T) {
	t.Parallel()
	adf := `{"type":"doc","version":1,"content":[{"type":"heading","content":[{"type":"text","text":"bold title","marks":[{"type":"strong"}]}]}]}`
	got := ADFToText(json.RawMessage(adf))
	if got != "*bold title*" {
		t.Fatalf("want %q, got %q", "*bold title*", got)
	}
}

// --- ADFToText: hardBreak ---

func TestADFToText_HardBreak(t *testing.T) {
	t.Parallel()
	adf := `{"type":"doc","version":1,"content":[{"type":"paragraph","content":[{"type":"text","text":"line1"},{"type":"hardBreak"},{"type":"text","text":"line2"}]}]}`
	got := ADFToText(json.RawMessage(adf))
	if got != "line1\nline2" {
		t.Fatalf("want %q, got %q", "line1\nline2", got)
	}
}

func TestADFToText_MultipleHardBreaks(t *testing.T) {
	t.Parallel()
	adf := `{"type":"doc","version":1,"content":[{"type":"paragraph","content":[{"type":"text","text":"a"},{"type":"hardBreak"},{"type":"hardBreak"},{"type":"text","text":"b"}]}]}`
	got := ADFToText(json.RawMessage(adf))
	if got != "a\n\nb" {
		t.Fatalf("want %q, got %q", "a\n\nb", got)
	}
}

// --- ADFToText: bullet list ---

func TestADFToText_BulletList(t *testing.T) {
	t.Parallel()
	adf := `{"type":"doc","version":1,"content":[{"type":"bulletList","content":[{"type":"listItem","content":[{"type":"paragraph","content":[{"type":"text","text":"item 1"}]}]},{"type":"listItem","content":[{"type":"paragraph","content":[{"type":"text","text":"item 2"}]}]}]}]}`
	got := ADFToText(json.RawMessage(adf))
	want := "- item 1\n- item 2"
	if got != want {
		t.Fatalf("want %q, got %q", want, got)
	}
}

func TestADFToText_BulletListNested(t *testing.T) {
	t.Parallel()
	adf := `{"type":"doc","version":1,"content":[{"type":"bulletList","content":[{"type":"listItem","content":[{"type":"paragraph","content":[{"type":"text","text":"parent"}]},{"type":"bulletList","content":[{"type":"listItem","content":[{"type":"paragraph","content":[{"type":"text","text":"child"}]}]}]}]}]}]}`
	got := ADFToText(json.RawMessage(adf))
	want := "- parent\n  - child"
	if got != want {
		t.Fatalf("want %q, got %q", want, got)
	}
}

func TestADFToText_BulletListWithMarks(t *testing.T) {
	t.Parallel()
	adf := `{"type":"doc","version":1,"content":[{"type":"bulletList","content":[{"type":"listItem","content":[{"type":"paragraph","content":[{"type":"text","text":"bold item","marks":[{"type":"strong"}]}]}]}]}]}`
	got := ADFToText(json.RawMessage(adf))
	want := "- *bold item*"
	if got != want {
		t.Fatalf("want %q, got %q", want, got)
	}
}

// --- ADFToText: ordered list ---

func TestADFToText_OrderedList(t *testing.T) {
	t.Parallel()
	adf := `{"type":"doc","version":1,"content":[{"type":"orderedList","content":[{"type":"listItem","content":[{"type":"paragraph","content":[{"type":"text","text":"first"}]}]},{"type":"listItem","content":[{"type":"paragraph","content":[{"type":"text","text":"second"}]}]}]}]}`
	got := ADFToText(json.RawMessage(adf))
	want := "- first\n- second"
	if got != want {
		t.Fatalf("want %q, got %q", want, got)
	}
}

// --- ADFToText: mixed blocks with lists ---

func TestADFToText_MixedParagraphAndList(t *testing.T) {
	t.Parallel()
	adf := `{"type":"doc","version":1,"content":[{"type":"paragraph","content":[{"type":"text","text":"intro"}]},{"type":"bulletList","content":[{"type":"listItem","content":[{"type":"paragraph","content":[{"type":"text","text":"item"}]}]}]},{"type":"paragraph","content":[{"type":"text","text":"outro"}]}]}`
	got := ADFToText(json.RawMessage(adf))
	want := "intro\n- item\noutro"
	if got != want {
		t.Fatalf("want %q, got %q", want, got)
	}
}

func TestADFToText_EmptyBulletList(t *testing.T) {
	t.Parallel()
	adf := `{"type":"doc","version":1,"content":[{"type":"bulletList","content":[]}]}`
	got := ADFToText(json.RawMessage(adf))
	if got != "" {
		t.Fatalf("want %q, got %q", "", got)
	}
}

// --- ADFToText: inline fallback for unknown node types ---

func TestADFToText_InlineMention(t *testing.T) {
	t.Parallel()
	// mention node has text field but is not type "text" — should still render.
	adf := `{"type":"doc","version":1,"content":[{"type":"paragraph","content":[{"type":"text","text":"hello "},{"type":"mention","text":"@john"}]}]}`
	got := ADFToText(json.RawMessage(adf))
	if got != "hello @john" {
		t.Fatalf("want %q, got %q", "hello @john", got)
	}
}

func TestADFToText_InlineEmoji(t *testing.T) {
	t.Parallel()
	adf := `{"type":"doc","version":1,"content":[{"type":"paragraph","content":[{"type":"emoji","text":":fire:"}]}]}`
	got := ADFToText(json.RawMessage(adf))
	if got != ":fire:" {
		t.Fatalf("want %q, got %q", ":fire:", got)
	}
}

func TestADFToText_InlineUnknownWithContent(t *testing.T) {
	t.Parallel()
	// Unknown inline node with nested text content.
	adf := `{"type":"doc","version":1,"content":[{"type":"paragraph","content":[{"type":"status","content":[{"type":"text","text":"IN PROGRESS"}]}]}]}`
	got := ADFToText(json.RawMessage(adf))
	if got != "IN PROGRESS" {
		t.Fatalf("want %q, got %q", "IN PROGRESS", got)
	}
}

func TestADFToText_InlineUnknownEmpty(t *testing.T) {
	t.Parallel()
	// Unknown inline node with no text and no content — should produce nothing.
	adf := `{"type":"doc","version":1,"content":[{"type":"paragraph","content":[{"type":"text","text":"before"},{"type":"date"},{"type":"text","text":"after"}]}]}`
	got := ADFToText(json.RawMessage(adf))
	if got != "beforeafter" {
		t.Fatalf("want %q, got %q", "beforeafter", got)
	}
}

// --- ADFToText: listItem with direct text children (no paragraph wrapper) ---

func TestADFToText_BulletListDirectText(t *testing.T) {
	t.Parallel()
	// listItem contains text nodes directly, without a paragraph wrapper.
	adf := `{"type":"doc","version":1,"content":[{"type":"bulletList","content":[{"type":"listItem","content":[{"type":"text","text":"item one"}]},{"type":"listItem","content":[{"type":"text","text":"item two"}]}]}]}`
	got := ADFToText(json.RawMessage(adf))
	want := "- item one\n- item two"
	if got != want {
		t.Fatalf("want %q, got %q", want, got)
	}
}

// --- Round-trip tests ---

func TestRoundTrip_Bold(t *testing.T) {
	t.Parallel()
	assertRoundTrip(t, "hello *world*")
}

func TestRoundTrip_Italic(t *testing.T) {
	t.Parallel()
	assertRoundTrip(t, "_italic_ text")
}

func TestRoundTrip_Underline(t *testing.T) {
	t.Parallel()
	assertRoundTrip(t, "some +underlined+ text")
}

func TestRoundTrip_InlineCode(t *testing.T) {
	t.Parallel()
	assertRoundTrip(t, "run `go test` now")
}

func TestRoundTrip_CodeBlock(t *testing.T) {
	t.Parallel()
	assertRoundTrip(t, "before\n```\ncode here\n```\nafter")
}

func TestRoundTrip_PlainText(t *testing.T) {
	t.Parallel()
	assertRoundTrip(t, "First line\nSecond line\nThird line")
}

func TestRoundTrip_Mixed(t *testing.T) {
	t.Parallel()
	assertRoundTrip(t, "*bold* and _italic_ and `code`")
}

func TestRoundTrip_AllFormats(t *testing.T) {
	t.Parallel()
	assertRoundTrip(t, "*bold* _italic_ +underline+ `code`")
}

func TestRoundTrip_MultiLineFormatted(t *testing.T) {
	t.Parallel()
	assertRoundTrip(t, "*bold line*\n_italic line_\n+underline line+")
}

func TestRoundTrip_CodeBlockWithSurrounding(t *testing.T) {
	t.Parallel()
	assertRoundTrip(t, "*bold intro*\n```\ncode\n```\n_italic outro_")
}

func TestRoundTrip_UnclosedDelimiter(t *testing.T) {
	t.Parallel()
	// Unclosed delimiters should survive round-trip as literal characters.
	assertRoundTrip(t, "hello *world")
}

func TestRoundTrip_PlainWithSpecialChars(t *testing.T) {
	t.Parallel()
	// Text that happens to have no matching pairs should pass through unchanged.
	assertRoundTrip(t, "price is 5+ dollars")
}

func assertRoundTrip(t *testing.T, input string) {
	t.Helper()
	got := ADFToText(TextToADF(input))
	if got != input {
		t.Fatalf("round-trip failed:\ninput:  %q\noutput: %q", input, got)
	}
}
