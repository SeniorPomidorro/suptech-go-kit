package atlassian

import (
	"encoding/json"
	"reflect"
	"testing"
	"time"
)

func TestFieldMetadataSchemaKind(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		schema FieldMetadataSchema
		want   FieldKind
	}{
		{"summary", FieldMetadataSchema{Type: "string", System: "summary"}, FieldKindString},
		{"description→long-text", FieldMetadataSchema{Type: "string", System: "description"}, FieldKindLongText},
		{"environment→long-text", FieldMetadataSchema{Type: "string", System: "environment"}, FieldKindLongText},
		{"custom-textarea→long-text", FieldMetadataSchema{
			Type:   "string",
			Custom: "com.atlassian.jira.plugin.system.customfieldtypes:textarea",
		}, FieldKindLongText},
		{"custom-textfield→string", FieldMetadataSchema{
			Type:   "string",
			Custom: "com.atlassian.jira.plugin.system.customfieldtypes:textfield",
		}, FieldKindString},
		{"number", FieldMetadataSchema{Type: "number"}, FieldKindNumber},
		{"date", FieldMetadataSchema{Type: "date"}, FieldKindDate},
		{"datetime", FieldMetadataSchema{Type: "datetime"}, FieldKindDateTime},
		{"option", FieldMetadataSchema{Type: "option"}, FieldKindOption},
		{"priority", FieldMetadataSchema{Type: "priority", System: "priority"}, FieldKindOption},
		{"resolution", FieldMetadataSchema{Type: "resolution", System: "resolution"}, FieldKindOption},
		{"version", FieldMetadataSchema{Type: "version"}, FieldKindOption},
		{"user", FieldMetadataSchema{Type: "user", System: "assignee"}, FieldKindUser},
		{"array<option>", FieldMetadataSchema{Type: "array", Items: "option"}, FieldKindMultiOption},
		{"array<version>", FieldMetadataSchema{Type: "array", Items: "version"}, FieldKindMultiOption},
		{"array<component>", FieldMetadataSchema{Type: "array", Items: "component"}, FieldKindMultiOption},
		{"array<resolution>", FieldMetadataSchema{Type: "array", Items: "resolution"}, FieldKindMultiOption},
		{"array<priority>", FieldMetadataSchema{Type: "array", Items: "priority"}, FieldKindMultiOption},
		{"array<issuetype>", FieldMetadataSchema{Type: "array", Items: "issuetype"}, FieldKindMultiOption},
		{"array<user>", FieldMetadataSchema{Type: "array", Items: "user"}, FieldKindMultiUser},
		{"array<string>", FieldMetadataSchema{Type: "array", Items: "string"}, FieldKindArrayOfStrings},
		{"array<unknown>", FieldMetadataSchema{Type: "array", Items: "issuelinks"}, FieldKindUnknown},
		{"unknown-type", FieldMetadataSchema{Type: "team"}, FieldKindUnknown},
		// Atlassian Cloud Assets — multi (the common case in modern Jira).
		// Detected by items=="cmdb-object-field" regardless of array shape.
		{"cmdb-multi-by-items", FieldMetadataSchema{
			Type:   "array",
			Items:  cmdbObjectFieldItemType,
			Custom: "com.atlassian.jira.plugins.cmdb:cmdb-object-cftype",
		}, FieldKindMultiAsset},
		// Same field type, detected purely from Custom prefix when Items is
		// missing — paranoia path for older API responses.
		{"cmdb-multi-by-custom", FieldMetadataSchema{
			Type:   "array",
			Custom: "com.atlassian.jira.plugins.cmdb:cmdb-object-cftype",
		}, FieldKindMultiAsset},
		// Single-asset variant — same custom prefix, scalar type.
		{"cmdb-single", FieldMetadataSchema{
			Type:   "string",
			Custom: "com.atlassian.jira.plugins.cmdb:cmdb-object-cftype",
		}, FieldKindAsset},
		// Legacy RIADA Insight plugin URI prefix — detected the same way.
		{"insight-legacy", FieldMetadataSchema{
			Type:   "array",
			Items:  cmdbObjectFieldItemType,
			Custom: "com.riadalabs.jira.plugins.insight:rlabs-insight-customfield-cmdb-cf",
		}, FieldKindMultiAsset},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := tc.schema.Kind(); got != tc.want {
				t.Fatalf("Kind() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestAllowedValueUIIdentifier(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		av   AllowedValue
		want string
	}{
		{"id-wins", AllowedValue{ID: "1", Value: "v", Name: "n"}, "1"},
		{"value-fallback", AllowedValue{Value: "billing", Name: "Billing"}, "billing"},
		{"name-fallback", AllowedValue{Name: "Highest"}, "Highest"},
		{"account-id", AllowedValue{AccountID: "abc"}, "abc"},
		{"group-id", AllowedValue{GroupID: "g-1"}, "g-1"},
		{"empty", AllowedValue{}, ""},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := tc.av.UIIdentifier(); got != tc.want {
				t.Fatalf("UIIdentifier() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestAllowedValueJiraPayload(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		av   AllowedValue
		want map[string]any
	}{
		{"prefers-account-id", AllowedValue{AccountID: "abc", ID: "1", Name: "n"}, map[string]any{"accountId": "abc"}},
		{"id-over-value", AllowedValue{ID: "10", Value: "billing", Name: "Billing"}, map[string]any{"id": "10"}},
		{"value-when-no-id", AllowedValue{Value: "billing", Name: "Billing"}, map[string]any{"value": "billing"}},
		{"name-only", AllowedValue{Name: "Highest"}, map[string]any{"name": "Highest"}},
		{"group-id", AllowedValue{GroupID: "g-1"}, map[string]any{"groupId": "g-1"}},
		{"empty", AllowedValue{}, nil},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := tc.av.JiraPayload(); !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("JiraPayload() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestFieldMetadataOptionPairs(t *testing.T) {
	t.Parallel()

	meta := FieldMetadata{
		AllowedValues: []AllowedValue{
			{ID: "1", Name: "Highest"},
			{ID: "10100", Value: "billing"}, // option without Name → Value as label
			{AccountID: "u1", DisplayName: "Alice"},
			{Name: "name-only"}, // Name is both label and UIIdentifier fallback
			{ID: "x"},           // missing label → skipped
			{},                  // empty → skipped
		},
	}

	got := meta.OptionPairs()
	want := []OptionPair{
		{Label: "Highest", Value: "1"},
		{Label: "billing", Value: "10100"},
		{Label: "Alice", Value: "u1"},
		{Label: "name-only", Value: "name-only"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("OptionPairs() = %+v, want %+v", got, want)
	}
}

func TestFieldMetadataFindAllowedValue(t *testing.T) {
	t.Parallel()

	meta := FieldMetadata{
		AllowedValues: []AllowedValue{
			{ID: "1", Name: "Highest"},
			{ID: "10100", Value: "billing"},
			{AccountID: "u1", DisplayName: "Alice"},
		},
	}

	if av, ok := meta.FindAllowedValue("1"); !ok || av.Name != "Highest" {
		t.Fatalf("by id: got (%+v, %v)", av, ok)
	}
	if av, ok := meta.FindAllowedValue("billing"); !ok || av.ID != "10100" {
		t.Fatalf("by value: got (%+v, %v)", av, ok)
	}
	if av, ok := meta.FindAllowedValue("Alice"); ok {
		// DisplayName isn't part of the search keys — should not match.
		t.Fatalf("display-name unexpectedly matched: %+v", av)
	}
	if av, ok := meta.FindAllowedValue("u1"); !ok || av.DisplayName != "Alice" {
		t.Fatalf("by accountId: got (%+v, %v)", av, ok)
	}
	if _, ok := meta.FindAllowedValue(""); ok {
		t.Fatalf("empty identifier should not match")
	}
	if _, ok := meta.FindAllowedValue("nope"); ok {
		t.Fatalf("unknown identifier should not match")
	}
}

func TestFieldMetadataMatchAllowedValue(t *testing.T) {
	t.Parallel()

	meta := FieldMetadata{
		AllowedValues: []AllowedValue{
			{ID: "1", Name: "Highest"},
			{ID: "10100", Value: "billing"},
			{AccountID: "u1", DisplayName: "Alice"},
			{AccountID: "u2", DisplayName: "Bob", EmailAddress: "bob@example.com"},
		},
	}

	// Priority: object with id+name.
	if av, ok := meta.MatchAllowedValue(map[string]any{"id": "1", "name": "Highest"}); !ok || av.ID != "1" {
		t.Fatalf("priority match: got (%+v, %v)", av, ok)
	}
	// Option: object with id+value.
	if av, ok := meta.MatchAllowedValue(map[string]any{"id": "10100", "value": "billing"}); !ok || av.Value != "billing" {
		t.Fatalf("option match: got (%+v, %v)", av, ok)
	}
	// Match by value when the id is missing.
	if av, ok := meta.MatchAllowedValue(map[string]any{"value": "billing"}); !ok || av.ID != "10100" {
		t.Fatalf("by value: got (%+v, %v)", av, ok)
	}
	// User-shaped raw: assignee/reporter come back from issue.fields as an
	// object keyed by accountId+displayName+emailAddress. Match must hit on
	// accountId — the most stable identifier across user renames.
	if av, ok := meta.MatchAllowedValue(map[string]any{"accountId": "u1", "displayName": "Alice"}); !ok || av.DisplayName != "Alice" {
		t.Fatalf("user match by accountId: got (%+v, %v)", av, ok)
	}
	// User-shaped raw with extra fields (emailAddress) should still match
	// only on accountId — emailAddress is informational, not a key.
	if av, ok := meta.MatchAllowedValue(map[string]any{"accountId": "u2", "displayName": "Bob", "emailAddress": "bob@example.com"}); !ok || av.AccountID != "u2" {
		t.Fatalf("user match with email: got (%+v, %v)", av, ok)
	}
	// User without accountId in the raw payload — nothing to match against.
	if _, ok := meta.MatchAllowedValue(map[string]any{"displayName": "Alice"}); ok {
		t.Fatalf("displayName alone should not match")
	}
	// Nil.
	if _, ok := meta.MatchAllowedValue(nil); ok {
		t.Fatalf("nil should not match")
	}
	// Wrong shape.
	if _, ok := meta.MatchAllowedValue("oops"); ok {
		t.Fatalf("string should not match")
	}
	// No allowed value matches.
	if _, ok := meta.MatchAllowedValue(map[string]any{"id": "999"}); ok {
		t.Fatalf("unknown id should not match")
	}
}

func TestFieldMetadataMatchAllowedValues(t *testing.T) {
	t.Parallel()

	meta := FieldMetadata{
		AllowedValues: []AllowedValue{
			{ID: "10100", Value: "billing"},
			{ID: "10101", Value: "checkout"},
		},
	}

	got := meta.MatchAllowedValues([]any{
		map[string]any{"id": "10100", "value": "billing"},
		map[string]any{"id": "99999"}, // unknown — skipped
		map[string]any{"value": "checkout"},
	})

	if len(got) != 2 || got[0].Value != "billing" || got[1].Value != "checkout" {
		t.Fatalf("unexpected matches: %+v", got)
	}

	if got := meta.MatchAllowedValues("not-an-array"); got != nil {
		t.Fatalf("non-array input should return nil, got %+v", got)
	}
}

func TestParseStringValue(t *testing.T) {
	t.Parallel()

	if got := ParseStringValue(nil); got != "" {
		t.Fatalf("nil → %q", got)
	}
	if got := ParseStringValue("hello"); got != "hello" {
		t.Fatalf("string: %q", got)
	}
	if got := ParseStringValue(float64(42)); got != "42" {
		t.Fatalf("int-as-float64: %q", got)
	}
	if got := ParseStringValue(float64(3.5)); got != "3.5" {
		t.Fatalf("real float: %q", got)
	}
	if got := ParseStringValue(true); got != "true" {
		t.Fatalf("bool: %q", got)
	}
	// json.Number is what callers get when they decode with
	// json.Decoder.UseNumber() — fairly common defensive setting for
	// preserving precision. Make sure we don't drop into the default
	// JSON-marshal branch.
	if got := ParseStringValue(json.Number("123")); got != "123" {
		t.Fatalf("json.Number int: %q", got)
	}
	if got := ParseStringValue(json.Number("3.14")); got != "3.14" {
		t.Fatalf("json.Number float: %q", got)
	}
	// Object falls back to JSON.
	if got := ParseStringValue(map[string]any{"k": "v"}); got != `{"k":"v"}` {
		t.Fatalf("map: %q", got)
	}
}

func TestParseDateValue(t *testing.T) {
	t.Parallel()

	cases := []struct {
		in   any
		want string
	}{
		{nil, ""},
		{"2026-05-04", "2026-05-04"},
		{"2026-05-04T11:00:00.000+0400", "2026-05-04"},
		{"not-a-date", ""},
		{"", ""},
	}
	for _, tc := range cases {
		if got := ParseDateValue(tc.in); got != tc.want {
			t.Fatalf("ParseDateValue(%v) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestParseAssetReferences(t *testing.T) {
	t.Parallel()

	// Modern shape — full envelope as returned by the issue endpoint.
	full := []any{
		map[string]any{
			"workspaceId": "ws-1",
			"id":          "ws-1:3212",
			"objectId":    "3212",
		},
		map[string]any{
			"workspaceId": "ws-1",
			"id":          "ws-1:4040",
			"objectId":    "4040",
		},
	}
	refs := ParseAssetReferences(full)
	if !reflect.DeepEqual(refs, []AssetReference{
		{WorkspaceID: "ws-1", ID: "ws-1:3212", ObjectID: "3212"},
		{WorkspaceID: "ws-1", ID: "ws-1:4040", ObjectID: "4040"},
	}) {
		t.Fatalf("full: %+v", refs)
	}

	// Slim shape — only id="<ws>:<obj>" present; objectId+workspaceId are
	// recovered from the composite.
	slim := []any{
		map[string]any{"id": "ws-2:9001"},
	}
	refs = ParseAssetReferences(slim)
	if !reflect.DeepEqual(refs, []AssetReference{
		{WorkspaceID: "ws-2", ID: "ws-2:9001", ObjectID: "9001"},
	}) {
		t.Fatalf("slim: %+v", refs)
	}

	// id without colon — treated as raw object id; workspaceId left empty.
	noColon := []any{
		map[string]any{"id": "12345"},
	}
	refs = ParseAssetReferences(noColon)
	if !reflect.DeepEqual(refs, []AssetReference{
		{ID: "12345", ObjectID: "12345"},
	}) {
		t.Fatalf("noColon: %+v", refs)
	}

	// Skip items that don't yield an objectId at all (no id, no objectId).
	junk := []any{
		map[string]any{"workspaceId": "ws-3"},
		map[string]any{"id": "3212", "objectId": "3212"},
	}
	refs = ParseAssetReferences(junk)
	if len(refs) != 1 || refs[0].ObjectID != "3212" {
		t.Fatalf("skip junk: %+v", refs)
	}

	// Non-array inputs return nil.
	if got := ParseAssetReferences(nil); got != nil {
		t.Fatalf("nil: %+v", got)
	}
	if got := ParseAssetReferences("oops"); got != nil {
		t.Fatalf("string: %+v", got)
	}
}

func TestParseAssetObjectIDs(t *testing.T) {
	t.Parallel()

	got := ParseAssetObjectIDs([]any{
		map[string]any{"workspaceId": "ws", "id": "ws:1", "objectId": "1"},
		map[string]any{"id": "ws:2"},
	})
	if !reflect.DeepEqual(got, []string{"1", "2"}) {
		t.Fatalf("ids: %+v", got)
	}

	if got := ParseAssetObjectIDs([]any{}); got != nil {
		t.Fatalf("empty array: %+v", got)
	}
}

func TestParseDatetimeUnix(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		in   any
		want int64
	}{
		{"empty", "", 0},
		{"nil", nil, 0},
		{"unparseable", "not a date", 0},
		{
			"jira-classic-with-millis-and-offset",
			"2026-05-04T11:00:00.000+0400",
			time.Date(2026, 5, 4, 11, 0, 0, 0, time.FixedZone("+0400", 4*3600)).Unix(),
		},
		{
			"rfc3339-utc",
			"2026-05-04T11:00:00Z",
			time.Date(2026, 5, 4, 11, 0, 0, 0, time.UTC).Unix(),
		},
		{
			"rfc3339-with-colon-offset",
			"2026-05-04T11:00:00+04:00",
			time.Date(2026, 5, 4, 11, 0, 0, 0, time.FixedZone("+04:00", 4*3600)).Unix(),
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := ParseDatetimeUnix(tc.in); got != tc.want {
				t.Fatalf("ParseDatetimeUnix(%v) = %d, want %d", tc.in, got, tc.want)
			}
		})
	}
}

func TestFormatDatetimeJira(t *testing.T) {
	t.Parallel()

	if got := FormatDatetimeJira(0, nil); got != "" {
		t.Fatalf("zero unix: %q", got)
	}
	if got := FormatDatetimeJira(-1, nil); got != "" {
		t.Fatalf("negative unix: %q", got)
	}

	utc, _ := time.LoadLocation("UTC")
	got := FormatDatetimeJira(time.Date(2026, 5, 4, 11, 0, 0, 0, utc).Unix(), utc)
	if got != "2026-05-04T11:00:00.000+0000" {
		t.Fatalf("UTC format: %q", got)
	}

	// nil loc must default to UTC, not time.Local — otherwise output drifts
	// across environments based on the host TZ.
	if got := FormatDatetimeJira(time.Date(2026, 5, 4, 11, 0, 0, 0, utc).Unix(), nil); got != "2026-05-04T11:00:00.000+0000" {
		t.Fatalf("nil loc should default to UTC, got %q", got)
	}

	// Round-trip the classic Jira shape: parse then format in the same zone
	// should yield bit-equivalent output.
	loc := time.FixedZone("+0400", 4*3600)
	unix := ParseDatetimeUnix("2026-05-04T11:00:00.000+0400")
	if unix == 0 {
		t.Fatalf("parse failed before round-trip")
	}
	if got := FormatDatetimeJira(unix, loc); got != "2026-05-04T11:00:00.000+0400" {
		t.Fatalf("round-trip: %q", got)
	}
}

func TestParseStringArray(t *testing.T) {
	t.Parallel()

	if got := ParseStringArray([]any{"a", "b", "c"}); !reflect.DeepEqual(got, []string{"a", "b", "c"}) {
		t.Fatalf("plain: %+v", got)
	}
	if got := ParseStringArray([]any{"a", 1, "b"}); !reflect.DeepEqual(got, []string{"a", "b"}) {
		t.Fatalf("mixed: %+v", got)
	}
	if got := ParseStringArray("oops"); got != nil {
		t.Fatalf("non-array: %+v", got)
	}
	if got := ParseStringArray(nil); got != nil {
		t.Fatalf("nil: %+v", got)
	}
}
