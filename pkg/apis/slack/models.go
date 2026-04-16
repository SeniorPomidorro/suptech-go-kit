package slack

import "time"

// ResponseMetadata is Slack cursor pagination metadata.
type ResponseMetadata struct {
	NextCursor string `json:"next_cursor"`
}

// UserGroup is a minimal Slack user group DTO.
type UserGroup struct {
	ID          string `json:"id"`
	TeamID      string `json:"team_id,omitempty"`
	Name        string `json:"name,omitempty"`
	Handle      string `json:"handle,omitempty"`
	Description string `json:"description,omitempty"`
}

// ListUserGroupUsersRequest contains parameters for usergroups.users.list.
type ListUserGroupUsersRequest struct {
	UserGroup       string `json:"usergroup"`
	IncludeDisabled bool   `json:"include_disabled,omitempty"`
	TeamID          string `json:"team_id,omitempty"`
}

// UpdateUserGroupUsersRequest contains parameters for usergroups.users.update.
type UpdateUserGroupUsersRequest struct {
	UserGroup          string   `json:"usergroup"`
	Users              []string `json:"users"`
	IncludeCount       bool     `json:"include_count,omitempty"`
	TeamID             string   `json:"team_id,omitempty"`
	AdditionalChannels []string `json:"additional_channels,omitempty"`
	IsShared           bool     `json:"is_shared,omitempty"`
}

// Conversation is a minimal Slack conversation DTO.
type Conversation struct {
	ID            string `json:"id"`
	Name          string `json:"name,omitempty"`
	IsChannel     bool   `json:"is_channel,omitempty"`
	IsGroup       bool   `json:"is_group,omitempty"`
	IsPrivate     bool   `json:"is_private,omitempty"`
	IsArchived    bool   `json:"is_archived,omitempty"`
	NumMembers    int    `json:"num_members,omitempty"`
	Creator       string `json:"creator,omitempty"`
	IsGeneral     bool   `json:"is_general,omitempty"`
	IsShared      bool   `json:"is_shared,omitempty"`
	IsExtShared   bool   `json:"is_ext_shared,omitempty"`
	IsOrgShared   bool   `json:"is_org_shared,omitempty"`
	ContextTeamID string `json:"context_team_id,omitempty"`
}

// User is Slack user DTO.
type User struct {
	ID                     string      `json:"id"`
	Name                   string      `json:"name,omitempty"`
	IsBot                  bool        `json:"is_bot,omitempty"`
	Updated                int64       `json:"updated,omitempty"`
	IsAppUser              bool        `json:"is_app_user,omitempty"`
	TeamID                 string      `json:"team_id,omitempty"`
	Deleted                bool        `json:"deleted,omitempty"`
	Color                  string      `json:"color,omitempty"`
	IsEmailConfirmed       bool        `json:"is_email_confirmed,omitempty"`
	RealName               string      `json:"real_name,omitempty"`
	TZ                     string      `json:"tz,omitempty"`
	TZLabel                string      `json:"tz_label,omitempty"`
	TZOffset               int         `json:"tz_offset,omitempty"`
	IsAdmin                bool        `json:"is_admin,omitempty"`
	IsOwner                bool        `json:"is_owner,omitempty"`
	IsPrimaryOwner         bool        `json:"is_primary_owner,omitempty"`
	IsRestricted           bool        `json:"is_restricted,omitempty"`
	IsUltraRestricted      bool        `json:"is_ultra_restricted,omitempty"`
	WhoCanShareContactCard string      `json:"who_can_share_contact_card,omitempty"`
	Profile                UserProfile `json:"profile,omitempty"`
}

// UserProfile is Slack user profile payload.
type UserProfile struct {
	RealName               string `json:"real_name,omitempty"`
	DisplayName            string `json:"display_name,omitempty"`
	AvatarHash             string `json:"avatar_hash,omitempty"`
	RealNameNormalized     string `json:"real_name_normalized,omitempty"`
	DisplayNameNormalized  string `json:"display_name_normalized,omitempty"`
	Image24                string `json:"image_24,omitempty"`
	Image32                string `json:"image_32,omitempty"`
	Image48                string `json:"image_48,omitempty"`
	Image72                string `json:"image_72,omitempty"`
	Image192               string `json:"image_192,omitempty"`
	Image512               string `json:"image_512,omitempty"`
	Image1024              string `json:"image_1024,omitempty"`
	ImageOriginal          string `json:"image_original,omitempty"`
	IsCustomImage          bool   `json:"is_custom_image,omitempty"`
	FirstName              string `json:"first_name,omitempty"`
	LastName               string `json:"last_name,omitempty"`
	Team                   string `json:"team,omitempty"`
	Email                  string `json:"email,omitempty"`
	Title                  string `json:"title,omitempty"`
	Phone                  string `json:"phone,omitempty"`
	Skype                  string `json:"skype,omitempty"`
	StatusText             string `json:"status_text,omitempty"`
	StatusTextCanonical    string `json:"status_text_canonical,omitempty"`
	StatusEmoji            string `json:"status_emoji,omitempty"`
	StatusEmojiDisplayInfo []any  `json:"status_emoji_display_info,omitempty"`
	StatusExpiration       int    `json:"status_expiration,omitempty"`
	BotID                  string `json:"bot_id,omitempty"`
	APIAppID               string `json:"api_app_id,omitempty"`
	AlwaysActive           bool   `json:"always_active,omitempty"`
}

// ListUsersRequest contains parameters for users.list.
type ListUsersRequest struct {
	Cursor        string `json:"cursor,omitempty"`
	IncludeLocale bool   `json:"include_locale,omitempty"`
	Limit         int    `json:"limit,omitempty"`
	TeamID        string
	FetchAll      bool
	// FetchAllThrottle defines pause between page requests when FetchAll=true.
	// If <=0, a safe default of 1 second is used.
	FetchAllThrottle time.Duration
}

// ListUsersResponse is the response from users.list.
type ListUsersResponse struct {
	Members          []User           `json:"members"`
	CacheTS          int64            `json:"cache_ts,omitempty"`
	ResponseMetadata ResponseMetadata `json:"response_metadata"`
}

// Message is a minimal Slack message DTO.
type Message struct {
	Type       string `json:"type,omitempty"`
	SubType    string `json:"subtype,omitempty"`
	User       string `json:"user,omitempty"`
	Text       string `json:"text,omitempty"`
	TS         string `json:"ts,omitempty"`
	ThreadTS   string `json:"thread_ts,omitempty"`
	ReplyCount int    `json:"reply_count,omitempty"`
}

// GetHistoryRequest contains parameters for conversations.history.
type GetHistoryRequest struct {
	Channel            string `json:"channel"`
	Cursor             string `json:"cursor,omitempty"`
	IncludeAllMetadata bool   `json:"include_all_metadata,omitempty"`
	Inclusive          bool   `json:"inclusive,omitempty"`
	Latest             string `json:"latest,omitempty"`
	Oldest             string `json:"oldest,omitempty"`
	Limit              int    `json:"limit,omitempty"`
}

// GetRepliesRequest contains parameters for conversations.replies.
type GetRepliesRequest struct {
	Channel            string `json:"channel"`
	TS                 string `json:"ts"`
	Cursor             string `json:"cursor,omitempty"`
	IncludeAllMetadata bool   `json:"include_all_metadata,omitempty"`
	Inclusive          bool   `json:"inclusive,omitempty"`
	Latest             string `json:"latest,omitempty"`
	Oldest             string `json:"oldest,omitempty"`
	Limit              int    `json:"limit,omitempty"`
}

// HistoryResponse is the response from conversations.history and conversations.replies.
type HistoryResponse struct {
	Messages         []Message        `json:"messages"`
	HasMore          bool             `json:"has_more"`
	PinCount         int              `json:"pin_count,omitempty"`
	ResponseMetadata ResponseMetadata `json:"response_metadata"`
}

// PostedMessage contains main fields returned by chat post/update methods.
type PostedMessage struct {
	Channel string  `json:"channel,omitempty"`
	TS      string  `json:"ts,omitempty"`
	Message Message `json:"message,omitempty"`
}

// EphemeralPostResult contains fields returned by chat.postEphemeral.
type EphemeralPostResult struct {
	Channel   string `json:"channel,omitempty"`
	MessageTS string `json:"message_ts,omitempty"`
}

// PostMessageRequest is the payload for chat.postMessage.
// Blocks and Attachments accept any JSON-serializable structs
// (e.g. slack-go block types, maps, or custom structs).
type PostMessageRequest struct {
	Channel     string         `json:"channel"`
	Text        string         `json:"text,omitempty"`
	Blocks      []any          `json:"blocks,omitempty"`
	Attachments []any          `json:"attachments,omitempty"`
	ThreadTS    string         `json:"thread_ts,omitempty"`
	Metadata    map[string]any `json:"metadata,omitempty"`
	UnfurlLinks *bool          `json:"unfurl_links,omitempty"`
	UnfurlMedia *bool          `json:"unfurl_media,omitempty"`
}

// PostEphemeralRequest is the payload for chat.postEphemeral.
type PostEphemeralRequest struct {
	Channel     string `json:"channel"`
	User        string `json:"user"`
	Text        string `json:"text,omitempty"`
	Blocks      []any  `json:"blocks,omitempty"`
	Attachments []any  `json:"attachments,omitempty"`
	ThreadTS    string `json:"thread_ts,omitempty"`
}

// UpdateMessageRequest is the payload for chat.update.
type UpdateMessageRequest struct {
	Channel     string         `json:"channel"`
	TS          string         `json:"ts"`
	Text        string         `json:"text,omitempty"`
	Blocks      []any          `json:"blocks,omitempty"`
	Attachments []any          `json:"attachments,omitempty"`
	Metadata    map[string]any `json:"metadata,omitempty"`
}

// ViewText represents Slack plain_text/markdown text object.
type ViewText struct {
	Type  string `json:"type"`
	Text  string `json:"text"`
	Emoji bool   `json:"emoji,omitempty"`
}

// ModalViewRequest describes modal/home view payload for views.open/views.update.
type ModalViewRequest struct {
	Type            string           `json:"type,omitempty"`
	CallbackID      string           `json:"callback_id,omitempty"`
	Title           *ViewText        `json:"title,omitempty"`
	Close           *ViewText        `json:"close,omitempty"`
	Submit          *ViewText        `json:"submit,omitempty"`
	Blocks          []map[string]any `json:"blocks,omitempty"`
	PrivateMetadata string           `json:"private_metadata,omitempty"`
	ExternalID      string           `json:"external_id,omitempty"`
}

// View is a minimal Slack view response DTO.
type View struct {
	ID         string `json:"id,omitempty"`
	ExternalID string `json:"external_id,omitempty"`
	Type       string `json:"type,omitempty"`
	CallbackID string `json:"callback_id,omitempty"`
	Hash       string `json:"hash,omitempty"`
}

// OpenViewResult contains fields returned by views.open/views.update.
type OpenViewResult struct {
	View View `json:"view"`
}

// Canvas represents minimal Slack canvas DTO.
type Canvas struct {
	ID    string `json:"id,omitempty"`
	Title string `json:"title,omitempty"`
}
