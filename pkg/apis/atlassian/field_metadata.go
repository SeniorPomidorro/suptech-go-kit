package atlassian

import (
	"encoding/json"
	"fmt"
)

// FieldKind classifies a Jira field's data shape into a small set of
// categories useful for building UI form controls and round-tripping values.
//
// Source mapping comes from FieldMetadataSchema.Type (and Items for arrays).
// FieldMetadataSchema.System is consulted only to upgrade plain "string" to
// LongText for description/environment, which Jira renders multiline.
type FieldKind int

const (
	// FieldKindUnknown means the field type wasn't recognized — callers should
	// fall back to displaying a link to the issue in Jira so the user can edit
	// the value there.
	FieldKindUnknown FieldKind = iota
	// FieldKindString is a single-line text field.
	FieldKindString
	// FieldKindLongText is a multi-line text field (description, environment).
	FieldKindLongText
	// FieldKindNumber is a numeric field. Jira accepts both numbers and numeric
	// strings on the wire; UIs typically render this as a single-line input.
	FieldKindNumber
	// FieldKindDate is a calendar date with no time component (YYYY-MM-DD).
	FieldKindDate
	// FieldKindDateTime is a date+time field. Most UIs render it as a date
	// picker with the time defaulting to midnight in the issue's timezone.
	FieldKindDateTime
	// FieldKindOption is a single-pick value chosen from FieldMetadata.AllowedValues
	// (option, priority, resolution, version, component, issuetype).
	FieldKindOption
	// FieldKindMultiOption is a multi-pick value chosen from
	// FieldMetadata.AllowedValues (array of option/version/component).
	FieldKindMultiOption
	// FieldKindUser is a single-user picker that round-trips a Jira accountId.
	FieldKindUser
	// FieldKindMultiUser is a multi-user picker (array of accountIds).
	FieldKindMultiUser
	// FieldKindArrayOfStrings is a free-form array of strings (labels-style).
	FieldKindArrayOfStrings
)

// Kind classifies the schema for UI mapping. Returns FieldKindUnknown for
// types we don't have a canonical UI shape for; callers typically render a
// fallback link to Jira in that case.
func (s FieldMetadataSchema) Kind() FieldKind {
	switch s.Type {
	case "string":
		// description/environment are multi-line text; everything else is single-line.
		if s.System == "description" || s.System == "environment" {
			return FieldKindLongText
		}
		return FieldKindString
	case "number":
		return FieldKindNumber
	case "date":
		return FieldKindDate
	case "datetime":
		return FieldKindDateTime
	case "option", "priority", "resolution", "issuetype", "version", "component":
		return FieldKindOption
	case "user":
		return FieldKindUser
	case "array":
		switch s.Items {
		case "option", "version", "component":
			return FieldKindMultiOption
		case "user":
			return FieldKindMultiUser
		case "string":
			return FieldKindArrayOfStrings
		}
	}
	return FieldKindUnknown
}

// UIIdentifier returns the most stable single string that identifies this
// allowed value, suitable for round-tripping through a UI (e.g. as the
// `value` of an HTML/Slack option). Prefers ID over Value over Name; falls
// back to AccountID/GroupID for user/group-shaped values.
//
// The priority order differs from JiraPayload by design: a UI needs *one*
// identifier it can put on every option uniformly, and ID/Value/Name are
// the keys most option-like fields populate, so checking them first lets us
// produce a non-empty result for the common case without ever touching the
// user/group branches. JiraPayload, in contrast, must produce the exact
// shape Jira expects per field type, hence its different priority.
//
// Returns "" if the value carries none of the recognized identifiers.
func (av AllowedValue) UIIdentifier() string {
	switch {
	case av.ID != "":
		return av.ID
	case av.Value != "":
		return av.Value
	case av.Name != "":
		return av.Name
	case av.AccountID != "":
		return av.AccountID
	case av.GroupID != "":
		return av.GroupID
	}
	return ""
}

// JiraPayload returns the canonical shape Jira accepts when sending this
// allowed value back as a field value:
//
//   - user fields           → {"accountId": ...}
//   - group fields          → {"groupId":  ...}
//   - everything else (option/priority/resolution/version/component) →
//     {"id": ...}, falling back to {"value": ...} or {"name": ...} when the
//     option only carries those.
//
// AccountID/GroupID come first because user and group fields *only* accept
// their dedicated keys — sending {"id": "..."} for a user field is rejected
// by Jira. The other types accept `id`/`value`/`name` interchangeably, so
// we try ID first to be stable across renames.
//
// Returns nil if the value carries none of the recognized identifiers.
func (av AllowedValue) JiraPayload() map[string]any {
	switch {
	case av.AccountID != "":
		return map[string]any{"accountId": av.AccountID}
	case av.ID != "":
		return map[string]any{"id": av.ID}
	case av.Value != "":
		return map[string]any{"value": av.Value}
	case av.Name != "":
		return map[string]any{"name": av.Name}
	case av.GroupID != "":
		return map[string]any{"groupId": av.GroupID}
	}
	return nil
}

// FindAllowedValue returns the first AllowedValue whose ID, Value, Name, or
// AccountID matches identifier. Useful for translating a UI-side identifier
// back into the structured value Jira expects.
func (m FieldMetadata) FindAllowedValue(identifier string) (AllowedValue, bool) {
	if identifier == "" {
		return AllowedValue{}, false
	}
	for _, av := range m.AllowedValues {
		if av.ID == identifier || av.Value == identifier || av.Name == identifier || av.AccountID == identifier {
			return av, true
		}
	}
	return AllowedValue{}, false
}

// MatchAllowedValue resolves a raw issue.fields[key] value (single-option) to
// the matching AllowedValue. raw is typically map[string]any decoded from
// JSON — Jira returns options as objects with id+value or id+name keys.
//
// Returns ok=false when raw is nil, not a JSON object, or doesn't match any
// of the configured allowed values.
func (m FieldMetadata) MatchAllowedValue(raw any) (AllowedValue, bool) {
	if raw == nil {
		return AllowedValue{}, false
	}
	obj, ok := raw.(map[string]any)
	if !ok {
		return AllowedValue{}, false
	}
	id, _ := obj["id"].(string)
	value, _ := obj["value"].(string)
	name, _ := obj["name"].(string)
	accountID, _ := obj["accountId"].(string)

	for _, av := range m.AllowedValues {
		if id != "" && av.ID == id {
			return av, true
		}
		if value != "" && av.Value == value {
			return av, true
		}
		if name != "" && av.Name == name {
			return av, true
		}
		if accountID != "" && av.AccountID == accountID {
			return av, true
		}
	}
	return AllowedValue{}, false
}

// MatchAllowedValues resolves a raw multi-option value (an array of objects
// from issue.fields[key]) to the matching AllowedValues. Items that don't
// resolve are skipped silently — most UIs only need the matched subset for
// pre-selection.
func (m FieldMetadata) MatchAllowedValues(raw any) []AllowedValue {
	arr, ok := raw.([]any)
	if !ok {
		return nil
	}
	out := make([]AllowedValue, 0, len(arr))
	for _, item := range arr {
		if av, ok := m.MatchAllowedValue(item); ok {
			out = append(out, av)
		}
	}
	return out
}

// ParseStringValue best-effort renders a primitive Jira field value
// (issue.fields[key]) to its string form for re-editing in a UI input.
// Objects and arrays fall back to a JSON encoding so the user can at least
// see and edit them as text.
func ParseStringValue(raw any) string {
	if raw == nil {
		return ""
	}
	switch x := raw.(type) {
	case string:
		return x
	case json.Number:
		return x.String()
	case float64:
		// Render integers without a decimal point — Jira often returns ints
		// as float64 after JSON decoding.
		if x == float64(int64(x)) {
			return fmt.Sprintf("%d", int64(x))
		}
		return fmt.Sprintf("%v", x)
	case bool:
		return fmt.Sprintf("%t", x)
	default:
		b, err := json.Marshal(x)
		if err != nil {
			return ""
		}
		return string(b)
	}
}

// ParseDateValue extracts the YYYY-MM-DD prefix from a Jira date or datetime
// value as encoded in issue.fields. Date fields arrive as already-formatted
// YYYY-MM-DD strings; datetime fields as e.g. "2026-05-04T11:00:00.000+0400".
// Returns "" when raw is empty or doesn't start with a date-shaped prefix.
func ParseDateValue(raw any) string {
	s := ParseStringValue(raw)
	// YYYY-MM-DD is exactly 10 chars with dashes at positions 4 and 7.
	if len(s) >= 10 && s[4] == '-' && s[7] == '-' {
		return s[:10]
	}
	return ""
}

// ParseStringArray decodes an array-of-strings issue.fields value (such as
// the labels system field) into a Go slice. Non-string elements are skipped.
func ParseStringArray(raw any) []string {
	arr, ok := raw.([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(arr))
	for _, item := range arr {
		if s, ok := item.(string); ok {
			out = append(out, s)
		}
	}
	return out
}
