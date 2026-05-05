package blockkit

import (
	"reflect"
	"strings"
	"testing"
	"unicode/utf8"
)

func TestInput(t *testing.T) {
	t.Parallel()

	el := PlainTextInput("act", "", false)
	got := Input("blk", "Label", el)

	if got["type"] != "input" {
		t.Fatalf("type: %v", got["type"])
	}
	if got["block_id"] != "blk" {
		t.Fatalf("block_id: %v", got["block_id"])
	}
	if got["optional"] != false {
		t.Fatalf("optional: %v", got["optional"])
	}
	label, _ := got["label"].(map[string]any)
	if label == nil || label["type"] != "plain_text" || label["text"] != "Label" {
		t.Fatalf("label: %+v", label)
	}
	if !reflect.DeepEqual(got["element"], el) {
		t.Fatalf("element: %+v", got["element"])
	}
}

func TestInputTruncatesLongLabels(t *testing.T) {
	t.Parallel()

	long := strings.Repeat("x", LabelLimit+10)
	got := Input("blk", long, PlainTextInput("a", "", false))
	label := got["label"].(map[string]any)
	if got := label["text"].(string); utf8.RuneCountInString(got) > LabelLimit {
		t.Fatalf("label not truncated: %d runes", utf8.RuneCountInString(got))
	}
}

func TestPlainTextInput(t *testing.T) {
	t.Parallel()

	bare := PlainTextInput("a", "", false)
	if bare["type"] != "plain_text_input" || bare["action_id"] != "a" {
		t.Fatalf("base: %+v", bare)
	}
	if _, ok := bare["initial_value"]; ok {
		t.Fatalf("empty initial should not be emitted")
	}
	if _, ok := bare["multiline"]; ok {
		t.Fatalf("multiline=false should not be emitted")
	}

	full := PlainTextInput("a", "hello", true)
	if full["initial_value"] != "hello" || full["multiline"] != true {
		t.Fatalf("full: %+v", full)
	}
}

func TestDatepicker(t *testing.T) {
	t.Parallel()

	bare := Datepicker("a", "")
	if _, ok := bare["initial_date"]; ok {
		t.Fatalf("empty initial_date should not be emitted")
	}
	full := Datepicker("a", "2026-05-04")
	if full["initial_date"] != "2026-05-04" {
		t.Fatalf("initial_date: %v", full["initial_date"])
	}
}

func TestDatetimepicker(t *testing.T) {
	t.Parallel()

	bare := Datetimepicker("a", 0)
	if bare["type"] != "datetimepicker" || bare["action_id"] != "a" {
		t.Fatalf("base: %+v", bare)
	}
	if _, ok := bare["initial_date_time"]; ok {
		t.Fatalf("zero initialUnix should not be emitted")
	}

	full := Datetimepicker("a", 1714838400)
	if full["initial_date_time"] != int64(1714838400) {
		t.Fatalf("initial_date_time: %v (%T)", full["initial_date_time"], full["initial_date_time"])
	}
}

func TestStaticSelectInitialOption(t *testing.T) {
	t.Parallel()

	opts := []map[string]any{
		Option("Highest", "1"),
		Option("Medium", "3"),
	}
	el := StaticSelect("a", "Pick", opts, "3")
	got, ok := el["initial_option"].(map[string]any)
	if !ok || got["value"] != "3" {
		t.Fatalf("initial_option: %+v", el["initial_option"])
	}
	// Mismatched initialValue should leave initial_option unset rather than picking
	// an arbitrary option — that would silently submit the wrong value.
	missing := StaticSelect("a", "Pick", opts, "999")
	if _, ok := missing["initial_option"]; ok {
		t.Fatalf("initial_option should be absent when value not in options")
	}
}

func TestMultiStaticSelectInitialOptions(t *testing.T) {
	t.Parallel()

	opts := []map[string]any{
		Option("billing", "10"),
		Option("checkout", "11"),
		Option("auth", "12"),
	}
	el := MultiStaticSelect("a", "Pick", opts, []string{"10", "12", "999"})
	picked, ok := el["initial_options"].([]map[string]any)
	if !ok || len(picked) != 2 {
		t.Fatalf("expected 2 initial_options (skipping unknown), got %+v", el["initial_options"])
	}
	if picked[0]["value"] != "10" || picked[1]["value"] != "12" {
		t.Fatalf("initial_options preserve order: %+v", picked)
	}

	// No initial values → initial_options must be absent.
	bare := MultiStaticSelect("a", "Pick", opts, nil)
	if _, ok := bare["initial_options"]; ok {
		t.Fatalf("initial_options should be absent when none match")
	}
}

func TestUsersSelectAndMultiUsersSelect(t *testing.T) {
	t.Parallel()

	bare := UsersSelect("a", "")
	if _, ok := bare["initial_user"]; ok {
		t.Fatalf("empty initial_user should not be emitted")
	}
	full := UsersSelect("a", "U123")
	if full["initial_user"] != "U123" {
		t.Fatalf("initial_user: %v", full["initial_user"])
	}

	bareMulti := MultiUsersSelect("a", nil)
	if _, ok := bareMulti["initial_users"]; ok {
		t.Fatalf("nil initial_users should not be emitted")
	}
	multi := MultiUsersSelect("a", []string{"U1", "U2"})
	users, _ := multi["initial_users"].([]string)
	if !reflect.DeepEqual(users, []string{"U1", "U2"}) {
		t.Fatalf("initial_users: %+v", users)
	}
}

func TestOptionTruncates(t *testing.T) {
	t.Parallel()

	long := strings.Repeat("y", OptionTextLimit+5)
	opt := Option(long, long)
	text := opt["text"].(map[string]any)["text"].(string)
	val := opt["value"].(string)
	if utf8.RuneCountInString(text) > OptionTextLimit {
		t.Fatalf("text not truncated: %d runes", utf8.RuneCountInString(text))
	}
	if utf8.RuneCountInString(val) > OptionValueLimit {
		t.Fatalf("value not truncated: %d runes", utf8.RuneCountInString(val))
	}
}

func TestSection(t *testing.T) {
	t.Parallel()

	got := Section("hello *world*")
	if got["type"] != "section" {
		t.Fatalf("type: %v", got["type"])
	}
	text := got["text"].(map[string]any)
	if text["type"] != "mrkdwn" || text["text"] != "hello *world*" {
		t.Fatalf("text: %+v", text)
	}
}

func TestTruncateText(t *testing.T) {
	t.Parallel()

	if got := TruncateText("hello", 10); got != "hello" {
		t.Fatalf("under-limit: %q", got)
	}
	if got := TruncateText("hello", 5); got != "hello" {
		t.Fatalf("at-limit: %q", got)
	}
	got := TruncateText("hello world", 8)
	if got != "hello w…" {
		t.Fatalf("truncated: %q", got)
	}
	if got := TruncateText("xy", 1); got != "x" {
		t.Fatalf("max=1: %q", got)
	}
	if got := TruncateText("anything", 0); got != "" {
		t.Fatalf("max=0: %q", got)
	}
}

func TestTruncateTextHandlesMultibyteRunes(t *testing.T) {
	t.Parallel()

	// Cyrillic — each rune is 2 bytes. Naive byte-slicing at max-1=9 would
	// land mid-rune and produce invalid UTF-8.
	got := TruncateText("Длинное название поля", 10)
	if utf8.RuneCountInString(got) != 10 {
		t.Fatalf("rune count: got %d (%q)", utf8.RuneCountInString(got), got)
	}
	if !utf8.ValidString(got) {
		t.Fatalf("invalid UTF-8 produced: %q", got)
	}
	if got != "Длинное н…" {
		t.Fatalf("unexpected truncation: %q", got)
	}

	// Emoji — 4 bytes per rune. Same trap.
	emoji := TruncateText("🚀🚀🚀🚀🚀", 3)
	if utf8.RuneCountInString(emoji) != 3 {
		t.Fatalf("emoji rune count: got %d (%q)", utf8.RuneCountInString(emoji), emoji)
	}
	if !utf8.ValidString(emoji) {
		t.Fatalf("invalid UTF-8 in emoji: %q", emoji)
	}
}
