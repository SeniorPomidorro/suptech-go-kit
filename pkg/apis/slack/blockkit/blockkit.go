// Package blockkit builds Slack Block Kit elements as map[string]any values
// suitable for slack.PostMessageRequest.Blocks ([]any) and
// slack.ModalViewRequest.Blocks ([]map[string]any) without pulling in any
// third-party SDK.
//
// Each function returns a fresh map per call — callers can mutate the result
// (e.g. add an "optional" or "hint" field) without affecting other blocks.
//
// Slack imposes 75-char limits on most user-facing text fields (option text,
// option value, placeholder, label) and 24 chars on view titles. Builders in
// this package call TruncateText defensively so an oversized input doesn't
// fail the API call; titles are not truncated automatically since callers
// usually want explicit control over them.
package blockkit

// Slack API text-length limits relevant to Block Kit. Exposed as constants
// so callers can use the same numbers when truncating their own strings.
const (
	OptionTextLimit  = 75
	OptionValueLimit = 75
	PlaceholderLimit = 75
	LabelLimit       = 75
)

// plainText builds a plain_text composition object — the shape Slack uses
// for labels, placeholders, button captions and option text. text is
// truncated to limit runes; passing 0 disables truncation (callers that
// know the field has no Slack-side limit).
func plainText(text string, limit int) map[string]any {
	if limit > 0 {
		text = TruncateText(text, limit)
	}
	return map[string]any{
		"type": "plain_text",
		"text": text,
	}
}

// Input wraps an interactive element in an `input` block. block_id is what
// shows up under view.state.values when the modal is submitted; pair it with
// the action_id used inside element to look up the user's choice.
func Input(blockID, label string, element map[string]any) map[string]any {
	return map[string]any{
		"type":     "input",
		"block_id": blockID,
		"label":    plainText(label, LabelLimit),
		"element":  element,
		"optional": false,
	}
}

// PlainTextInput builds a plain_text_input element. Empty initial leaves the
// input blank; multiline=true makes it a textarea.
func PlainTextInput(actionID, initial string, multiline bool) map[string]any {
	el := map[string]any{
		"type":      "plain_text_input",
		"action_id": actionID,
	}
	if initial != "" {
		el["initial_value"] = initial
	}
	if multiline {
		el["multiline"] = true
	}
	return el
}

// Datepicker builds a datepicker element. initialDate is YYYY-MM-DD; pass ""
// to leave it unset.
func Datepicker(actionID, initialDate string) map[string]any {
	el := map[string]any{
		"type":      "datepicker",
		"action_id": actionID,
	}
	if initialDate != "" {
		el["initial_date"] = initialDate
	}
	return el
}

// Datetimepicker builds a datetimepicker element — a single combined date+time
// control. initialUnix is the preset moment in Unix seconds; pass 0 to leave
// it unset. The user's selection arrives back on view.state.values via
// SelectedDateTime (Unix seconds), regardless of the user's timezone.
func Datetimepicker(actionID string, initialUnix int64) map[string]any {
	el := map[string]any{
		"type":      "datetimepicker",
		"action_id": actionID,
	}
	if initialUnix > 0 {
		el["initial_date_time"] = initialUnix
	}
	return el
}

// StaticSelect builds a static_select element. options should come from
// Option(); initialValue selects the option whose value matches.
func StaticSelect(actionID, placeholder string, options []map[string]any, initialValue string) map[string]any {
	el := map[string]any{
		"type":        "static_select",
		"action_id":   actionID,
		"placeholder": plainText(placeholder, PlaceholderLimit),
		"options":     options,
	}
	if initialValue != "" {
		for _, opt := range options {
			if opt["value"] == initialValue {
				el["initial_option"] = opt
				break
			}
		}
	}
	return el
}

// MultiStaticSelect builds a multi_static_select element. initialValues are
// matched against options by value; values without a matching option are
// silently skipped.
func MultiStaticSelect(actionID, placeholder string, options []map[string]any, initialValues []string) map[string]any {
	el := map[string]any{
		"type":        "multi_static_select",
		"action_id":   actionID,
		"placeholder": plainText(placeholder, PlaceholderLimit),
		"options":     options,
	}
	if len(initialValues) > 0 {
		picked := make([]map[string]any, 0, len(initialValues))
		for _, v := range initialValues {
			for _, opt := range options {
				if opt["value"] == v {
					picked = append(picked, opt)
					break
				}
			}
		}
		if len(picked) > 0 {
			el["initial_options"] = picked
		}
	}
	return el
}

// UsersSelect builds a users_select element. initialUser is a Slack user ID
// (e.g. "U123ABC"); pass "" to leave it unset.
func UsersSelect(actionID, initialUser string) map[string]any {
	el := map[string]any{
		"type":      "users_select",
		"action_id": actionID,
	}
	if initialUser != "" {
		el["initial_user"] = initialUser
	}
	return el
}

// MultiUsersSelect builds a multi_users_select element.
func MultiUsersSelect(actionID string, initialUsers []string) map[string]any {
	el := map[string]any{
		"type":      "multi_users_select",
		"action_id": actionID,
	}
	if len(initialUsers) > 0 {
		el["initial_users"] = initialUsers
	}
	return el
}

// Option builds one option object for static_select / multi_static_select.
// Both text and value are truncated to Slack's 75-char limit.
func Option(text, value string) map[string]any {
	return map[string]any{
		"text":  plainText(text, OptionTextLimit),
		"value": TruncateText(value, OptionValueLimit),
	}
}

// Section builds a `section` block with a single mrkdwn text field. Useful
// as a fallback when no interactive element fits, or as a header line in a
// modal. mrkdwn supports Slack's mrkdwn formatting (*bold*, `code`, links).
func Section(mrkdwn string) map[string]any {
	return map[string]any{
		"type": "section",
		"text": map[string]any{
			"type": "mrkdwn",
			"text": mrkdwn,
		},
	}
}

// TruncateText shortens s to at most max runes (Unicode code points), the
// unit Slack itself uses for length limits. Strings already at or below the
// limit are returned unchanged; longer strings get max-1 leading runes plus
// a one-rune ellipsis. Used internally by the builders and exposed for
// callers that need to trim other strings to Slack-side limits.
//
// Operating in runes rather than bytes matters for non-ASCII content
// (Cyrillic, CJK, emoji): a byte-slice cut at max-1 could split a
// multi-byte rune and produce invalid UTF-8.
func TruncateText(s string, max int) string {
	if max <= 0 {
		return ""
	}
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	if max == 1 {
		return string(r[:1])
	}
	return string(r[:max-1]) + "…"
}
