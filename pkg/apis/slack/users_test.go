package slack

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/SeniorPomidorro/suptech-go-kit/pkg/transport"
)

func TestListUsersSinglePageWithParams(t *testing.T) {
	t.Parallel()

	requests := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/users.list" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodGet {
			t.Fatalf("unexpected method: %s", r.Method)
		}

		q := r.URL.Query()
		if q.Get("cursor") != "cursor-1" {
			t.Fatalf("unexpected cursor: %q", q.Get("cursor"))
		}
		if q.Get("include_locale") != "true" {
			t.Fatalf("unexpected include_locale: %q", q.Get("include_locale"))
		}
		if q.Get("limit") != "2" {
			t.Fatalf("unexpected limit: %q", q.Get("limit"))
		}
		if q.Get("team_id") != "T-explicit" {
			t.Fatalf("unexpected team_id: %q", q.Get("team_id"))
		}

		requests++
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true,"members":[{"id":"U1","name":"alice"}],"response_metadata":{"next_cursor":"cursor-2"}}`))
	}))
	defer srv.Close()

	client, err := NewClient(
		WithBaseURL(srv.URL),
		WithToken("xoxb-test"),
		WithTeamID("T-from-client"),
		WithTransport(transport.New()),
	)
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	resp, err := client.Users().ListUsers(context.Background(), &ListUsersRequest{
		Cursor:        "cursor-1",
		IncludeLocale: true,
		Limit:         2,
		TeamID:        "T-explicit",
	})
	if err != nil {
		t.Fatalf("ListUsers failed: %v", err)
	}
	if requests != 1 {
		t.Fatalf("expected 1 request, got %d", requests)
	}
	if len(resp.Members) != 1 || resp.Members[0].ID != "U1" {
		t.Fatalf("unexpected members: %+v", resp.Members)
	}
	if resp.ResponseMetadata.NextCursor != "cursor-2" {
		t.Fatalf("unexpected next cursor: %q", resp.ResponseMetadata.NextCursor)
	}
}

func TestListUsersFetchAll(t *testing.T) {
	t.Parallel()

	requests := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/users.list" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		q := r.URL.Query()
		if q.Get("team_id") != "T123" {
			t.Fatalf("unexpected team_id: %q", q.Get("team_id"))
		}
		if q.Get("limit") != "2" {
			t.Fatalf("unexpected limit: %q", q.Get("limit"))
		}

		requests++
		w.Header().Set("Content-Type", "application/json")
		if requests == 1 {
			if q.Get("cursor") != "" {
				t.Fatalf("expected first request without cursor, got %q", q.Get("cursor"))
			}
			_, _ = w.Write([]byte(`{"ok":true,"members":[{"id":"U1"},{"id":"U2"}],"response_metadata":{"next_cursor":"cursor-1"}}`))
			return
		}

		if q.Get("cursor") != "cursor-1" {
			t.Fatalf("unexpected second request cursor: %q", q.Get("cursor"))
		}
		_, _ = w.Write([]byte(`{"ok":true,"members":[{"id":"U3"}],"response_metadata":{"next_cursor":""}}`))
	}))
	defer srv.Close()

	client, err := NewClient(
		WithBaseURL(srv.URL),
		WithToken("xoxb-test"),
		WithTeamID("T123"),
		WithTransport(transport.New()),
	)
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	resp, err := client.Users().ListUsers(context.Background(), &ListUsersRequest{
		Limit:            2,
		FetchAll:         true,
		FetchAllThrottle: time.Millisecond,
	})
	if err != nil {
		t.Fatalf("ListUsers failed: %v", err)
	}
	if requests != 2 {
		t.Fatalf("expected 2 requests, got %d", requests)
	}
	if len(resp.Members) != 3 {
		t.Fatalf("expected 3 users, got %d", len(resp.Members))
	}
	if resp.Members[2].ID != "U3" {
		t.Fatalf("unexpected last user: %+v", resp.Members[2])
	}
	if resp.ResponseMetadata.NextCursor != "" {
		t.Fatalf("expected empty next_cursor, got %q", resp.ResponseMetadata.NextCursor)
	}
}

func TestListUsersFetchAllThrottle(t *testing.T) {
	t.Parallel()

	requests := 0
	var firstRequestAt time.Time
	var secondRequestAt time.Time

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		now := time.Now()
		if requests == 1 {
			firstRequestAt = now
			_, _ = w.Write([]byte(`{"ok":true,"members":[{"id":"U1"}],"response_metadata":{"next_cursor":"cursor-1"}}`))
			return
		}

		secondRequestAt = now
		_, _ = w.Write([]byte(`{"ok":true,"members":[{"id":"U2"}],"response_metadata":{"next_cursor":""}}`))
	}))
	defer srv.Close()

	client, err := NewClient(
		WithBaseURL(srv.URL),
		WithToken("xoxb-test"),
		WithTransport(transport.New()),
	)
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	const throttle = 40 * time.Millisecond
	_, err = client.Users().ListUsers(context.Background(), &ListUsersRequest{
		FetchAll:         true,
		FetchAllThrottle: throttle,
	})
	if err != nil {
		t.Fatalf("ListUsers failed: %v", err)
	}
	if requests != 2 {
		t.Fatalf("expected 2 requests, got %d", requests)
	}

	delay := secondRequestAt.Sub(firstRequestAt)
	if delay < throttle {
		t.Fatalf("expected delay >= %s, got %s", throttle, delay)
	}
}

func TestListUsersFetchAllReturnsErrorOnRepeatedCursor(t *testing.T) {
	t.Parallel()

	requests := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		if r.URL.Query().Get("cursor") != "cursor-1" {
			t.Fatalf("unexpected cursor: %q", r.URL.Query().Get("cursor"))
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true,"members":[{"id":"U1"}],"response_metadata":{"next_cursor":"cursor-1"}}`))
	}))
	defer srv.Close()

	client, err := NewClient(
		WithBaseURL(srv.URL),
		WithToken("xoxb-test"),
		WithTransport(transport.New()),
	)
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	_, err = client.Users().ListUsers(context.Background(), &ListUsersRequest{
		Cursor:   "cursor-1",
		FetchAll: true,
	})
	if err == nil {
		t.Fatalf("expected error for repeated cursor")
	}
	if !strings.Contains(err.Error(), "repeated cursor") {
		t.Fatalf("expected repeated cursor error, got %v", err)
	}
	if requests != 1 {
		t.Fatalf("expected 1 request, got %d", requests)
	}
}

func TestListUsersDecodesMergedUserFields(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"ok": true,
			"members": [
				{
					"id": "U0ASZ2RQB6D",
					"name": "roman.politov",
					"is_bot": false,
					"updated": 1776191397,
					"is_app_user": false,
					"team_id": "TLZA75T1Q",
					"deleted": false,
					"color": "4ec0d6",
					"is_email_confirmed": true,
					"real_name": "Roman Politov",
					"tz": "Europe/Amsterdam",
					"tz_label": "Central European Summer Time",
					"tz_offset": 7200,
					"is_admin": false,
					"is_owner": false,
					"is_primary_owner": false,
					"is_restricted": false,
					"is_ultra_restricted": false,
					"who_can_share_contact_card": "EVERYONE",
					"profile": {
						"real_name": "Roman Politov",
						"display_name": "Roman Politov",
						"avatar_hash": "8f24c2d389cf",
						"real_name_normalized": "Roman Politov",
						"display_name_normalized": "Roman Politov",
						"image_24": "https://avatars.slack-edge.com/user_24.jpg",
						"image_32": "https://avatars.slack-edge.com/user_32.jpg",
						"image_48": "https://avatars.slack-edge.com/user_48.jpg",
						"image_72": "https://avatars.slack-edge.com/user_72.jpg",
						"image_192": "https://avatars.slack-edge.com/user_192.jpg",
						"image_512": "https://avatars.slack-edge.com/user_512.jpg",
						"image_1024": "https://avatars.slack-edge.com/user_1024.jpg",
						"image_original": "https://avatars.slack-edge.com/user_original.jpg",
						"is_custom_image": true,
						"first_name": "Roman",
						"last_name": "Politov",
						"team": "TLZA75T1Q",
						"email": "roman.politov@tabby.ai",
						"title": "Backend Engineer | Billing",
						"phone": "",
						"skype": "",
						"status_text": "",
						"status_text_canonical": "",
						"status_emoji": "",
						"status_emoji_display_info": [],
						"status_expiration": 0
					}
				},
				{
					"id": "U0ASZCH12DC",
					"name": "polisbot",
					"is_bot": true,
					"updated": 1776247625,
					"is_app_user": false,
					"team_id": "TLZA75T1Q",
					"deleted": false,
					"color": "8d4b84",
					"is_email_confirmed": false,
					"real_name": "PolisBot",
					"tz": "America/Los_Angeles",
					"tz_label": "Pacific Daylight Time",
					"tz_offset": -25200,
					"is_admin": false,
					"is_owner": false,
					"is_primary_owner": false,
					"is_restricted": false,
					"is_ultra_restricted": false,
					"who_can_share_contact_card": "EVERYONE",
					"profile": {
						"real_name": "PolisBot",
						"display_name": "",
						"avatar_hash": "7e3649bbf40e",
						"real_name_normalized": "PolisBot",
						"display_name_normalized": "",
						"image_24": "https://avatars.slack-edge.com/bot_24.png",
						"image_32": "https://avatars.slack-edge.com/bot_32.png",
						"image_48": "https://avatars.slack-edge.com/bot_48.png",
						"image_72": "https://avatars.slack-edge.com/bot_72.png",
						"image_192": "https://avatars.slack-edge.com/bot_192.png",
						"image_512": "https://avatars.slack-edge.com/bot_512.png",
						"image_1024": "https://avatars.slack-edge.com/bot_1024.png",
						"image_original": "https://avatars.slack-edge.com/bot_original.png",
						"is_custom_image": true,
						"first_name": "PolisBot",
						"last_name": "",
						"team": "TLZA75T1Q",
						"title": "",
						"phone": "",
						"skype": "",
						"status_text": "",
						"status_text_canonical": "",
						"status_emoji": "",
						"status_emoji_display_info": [],
						"status_expiration": 0,
						"bot_id": "B0ASV30UG2X",
						"api_app_id": "A0ARMEPSK43",
						"always_active": true
					}
				}
			],
			"response_metadata": {
				"next_cursor": ""
			}
		}`))
	}))
	defer srv.Close()

	client, err := NewClient(
		WithBaseURL(srv.URL),
		WithToken("xoxb-test"),
		WithTransport(transport.New()),
	)
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	resp, err := client.Users().ListUsers(context.Background(), nil)
	if err != nil {
		t.Fatalf("ListUsers failed: %v", err)
	}
	if len(resp.Members) != 2 {
		t.Fatalf("expected 2 users, got %d", len(resp.Members))
	}

	human := resp.Members[0]
	if human.RealName != "Roman Politov" {
		t.Fatalf("unexpected human real_name: %q", human.RealName)
	}
	if human.Profile.Email != "roman.politov@tabby.ai" {
		t.Fatalf("unexpected human email: %q", human.Profile.Email)
	}
	if human.Profile.ImageOriginal == "" {
		t.Fatalf("expected human image_original")
	}
	if human.WhoCanShareContactCard != "EVERYONE" {
		t.Fatalf("unexpected who_can_share_contact_card: %q", human.WhoCanShareContactCard)
	}

	bot := resp.Members[1]
	if !bot.IsBot {
		t.Fatalf("expected second user to be bot")
	}
	if bot.Profile.BotID != "B0ASV30UG2X" {
		t.Fatalf("unexpected bot_id: %q", bot.Profile.BotID)
	}
	if bot.Profile.APIAppID != "A0ARMEPSK43" {
		t.Fatalf("unexpected api_app_id: %q", bot.Profile.APIAppID)
	}
	if !bot.Profile.AlwaysActive {
		t.Fatalf("expected always_active=true")
	}
}

func TestListUsersSlackError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":false,"error":"missing_scope","needed":"users:read","provided":"users:read.email","warning":"missing_charset"}`))
	}))
	defer srv.Close()

	client, err := NewClient(
		WithBaseURL(srv.URL),
		WithToken("xoxb-test"),
		WithTransport(transport.New()),
	)
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	_, err = client.Users().ListUsers(context.Background(), nil)
	if err == nil {
		t.Fatalf("expected error")
	}
	slackErr, ok := err.(*Error)
	if !ok {
		t.Fatalf("expected slack.Error, got %T", err)
	}
	if slackErr.Code != "missing_scope" {
		t.Fatalf("unexpected error code: %q", slackErr.Code)
	}
	if slackErr.Needed != "users:read" {
		t.Fatalf("unexpected needed: %q", slackErr.Needed)
	}
	if slackErr.Provided != "users:read.email" {
		t.Fatalf("unexpected provided: %q", slackErr.Provided)
	}
	if slackErr.Warning != "missing_charset" {
		t.Fatalf("unexpected warning: %q", slackErr.Warning)
	}
}
