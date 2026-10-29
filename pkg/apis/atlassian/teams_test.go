package atlassian

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/SeniorPomidorro/suptech-go-kit/pkg/transport"
)

const teamsBase = "/gateway/api/public/teams/v1/org/org-1/teams"

func newTeamsClient(t *testing.T, h http.HandlerFunc) *Client {
	t.Helper()
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	client, err := NewClient(WithBaseURL(srv.URL), WithAuth(Auth{Mode: AuthBasicToken, Token: "tok"}), WithTransport(transport.New()))
	if err != nil {
		t.Fatalf("new client: %v", err)
	}
	return client
}

func TestListTeams(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32
	client := newTeamsClient(t, func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.URL.Path != teamsBase || r.URL.Query().Get("siteId") != "site-1" || r.URL.Query().Get("size") != "300" {
			t.Fatalf("unexpected request: %s", r.URL)
		}
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Query().Get("cursor") == "" {
			_, _ = w.Write([]byte(`{"entities":[{"teamId":"t1","displayName":"Accounts","state":"ACTIVE","teamType":"EXTERNAL","externalReference":{"source":"ATLASSIAN_GROUP","id":"g1"}}],"cursor":"next"}`))
			return
		}
		if r.URL.Query().Get("cursor") != "next" {
			t.Fatalf("unexpected cursor: %q", r.URL.Query().Get("cursor"))
		}
		_, _ = w.Write([]byte(`{"entities":[{"teamId":"t2","displayName":"Loans","state":"ACTIVE","teamType":"OPEN","externalReference":null}],"cursor":null}`))
	})

	teams, err := client.Teams().ListTeams(context.Background(), "org-1", "site-1")
	if err != nil {
		t.Fatalf("ListTeams: %v", err)
	}
	if calls.Load() != 2 || len(teams) != 2 {
		t.Fatalf("expected 2 teams in 2 calls, got %d in %d", len(teams), calls.Load())
	}
	if !teams[0].LinkedTo("g1") || teams[0].LinkedTo("g2") || teams[1].LinkedTo("g1") || teams[1].LinkedTo("") {
		t.Fatalf("unexpected links: %+v", teams)
	}
}

func TestListTeamsPaging(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		pages   map[string]string
		want    int
		wantErr bool
	}{
		{"an_empty_page_with_a_cursor_goes_on", map[string]string{
			"":   `{"entities":[],"cursor":"c1"}`,
			"c1": `{"entities":[{"teamId":"t1"}]}`,
		}, 1, false},
		{"a_repeated_cursor_is_an_error", map[string]string{
			"":   `{"entities":[{"teamId":"t1"}],"cursor":"c1"}`,
			"c1": `{"entities":[{"teamId":"t2"}],"cursor":"c1"}`,
		}, 0, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			client := newTeamsClient(t, func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(c.pages[r.URL.Query().Get("cursor")]))
			})

			teams, err := client.Teams().ListTeams(context.Background(), "org-1", "")
			if (err != nil) != c.wantErr || len(teams) != c.want {
				t.Fatalf("got %d teams, err %v", len(teams), err)
			}
		})
	}
}

func TestCreateTeam(t *testing.T) {
	t.Parallel()

	client := newTeamsClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != teamsBase {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL)
		}
		if got := r.Header.Get("Authorization"); got != "Basic tok" {
			t.Fatalf("unexpected auth: %q", got)
		}
		var body map[string]string
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		want := map[string]string{"displayName": "Loans", "description": "", "teamType": "OPEN", "siteId": "site-1"}
		if len(body) != len(want) {
			t.Fatalf("unexpected body: %v", body)
		}
		for k, v := range want {
			if body[k] != v {
				t.Fatalf("unexpected body: %v", body)
			}
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"teamId":"t9","displayName":"Loans","teamType":"OPEN","state":"ACTIVE"}`))
	})

	team, err := client.Teams().CreateTeam(context.Background(), "org-1", &CreateTeamRequest{DisplayName: "Loans", TeamType: TeamTypeOpen, SiteID: "site-1"})
	if err != nil {
		t.Fatalf("CreateTeam: %v", err)
	}
	if team.TeamID != "t9" {
		t.Fatalf("unexpected team: %+v", team)
	}
}

func TestCreateTeamIsNeverRetried(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	t.Cleanup(srv.Close)
	client, err := NewClient(WithBaseURL(srv.URL), WithTransport(transport.New(transport.WithRetry(transport.RetryConfig{MaxAttempts: 3}))))
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	_, err = client.Teams().CreateTeam(context.Background(), "org-1", &CreateTeamRequest{DisplayName: "Loans", TeamType: TeamTypeOpen})
	var apiErr *transport.APIError
	if !errors.As(err, &apiErr) || apiErr.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("expected a 503 api error, got %v", err)
	}
	if calls.Load() != 1 {
		t.Fatalf("expected one attempt, got %d", calls.Load())
	}
}

func TestLinkTeam(t *testing.T) {
	t.Parallel()

	client := newTeamsClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != teamsBase+"/t9/external/link" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL)
		}
		var body struct {
			ExternalReference TeamExternalReference `json:"externalReference"`
			SiteID            string                `json:"siteId"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if body.ExternalReference != (TeamExternalReference{Source: TeamSourceGroup, ID: "g1"}) || body.SiteID != "site-1" {
			t.Fatalf("unexpected body: %+v", body)
		}
		w.WriteHeader(http.StatusNoContent)
	})

	if err := client.Teams().LinkTeam(context.Background(), "org-1", "t9", "site-1", TeamExternalReference{Source: TeamSourceGroup, ID: "g1"}); err != nil {
		t.Fatalf("LinkTeam: %v", err)
	}
}

func TestDeleteTeam(t *testing.T) {
	t.Parallel()

	client := newTeamsClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete || r.URL.Query().Get("siteId") != "site-1" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL)
		}
		switch r.URL.Path {
		case teamsBase + "/t9":
			w.WriteHeader(http.StatusNoContent)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	})
	teams := client.Teams()

	if err := teams.DeleteTeam(context.Background(), "org-1", "t9", "site-1"); err != nil {
		t.Fatalf("DeleteTeam: %v", err)
	}
	err := teams.DeleteTeam(context.Background(), "org-1", "gone", "site-1")
	var apiErr *transport.APIError
	if !errors.As(err, &apiErr) || apiErr.StatusCode != http.StatusNotFound {
		t.Fatalf("expected a 404 api error, got %v", err)
	}
}

func TestTeamsValidation(t *testing.T) {
	t.Parallel()

	client := newTeamsClient(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("unexpected request: %s", r.URL)
	})
	teams := client.Teams()
	ctx := context.Background()

	if _, err := teams.ListTeams(ctx, "", "site-1"); err == nil {
		t.Fatal("expected an error without an organisation")
	}
	if _, err := teams.CreateTeam(ctx, "org-1", &CreateTeamRequest{TeamType: TeamTypeOpen}); err == nil {
		t.Fatal("expected an error without a display name")
	}
	if _, err := teams.CreateTeam(ctx, "org-1", &CreateTeamRequest{DisplayName: "Loans"}); err == nil {
		t.Fatal("expected an error without a team type")
	}
	if err := teams.LinkTeam(ctx, "org-1", "t9", "", TeamExternalReference{Source: TeamSourceGroup}); err == nil {
		t.Fatal("expected an error without a reference id")
	}
	if err := teams.DeleteTeam(ctx, "org-1", " ", ""); err == nil {
		t.Fatal("expected an error without a team id")
	}
	for _, id := range []string{"..", ".", "../../other", "t/1", `t\1`} {
		if err := teams.DeleteTeam(ctx, "org-1", id, ""); err == nil {
			t.Fatalf("expected an error for team id %q", id)
		}
		if err := teams.LinkTeam(ctx, "org-1", id, "", TeamExternalReference{Source: TeamSourceGroup, ID: "g1"}); err == nil {
			t.Fatalf("expected an error for team id %q", id)
		}
	}
	if _, err := teams.ListTeams(ctx, "../org", ""); err == nil {
		t.Fatal("expected an error for an organisation id with a slash")
	}
}
