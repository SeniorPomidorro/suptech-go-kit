package atlassian

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/SeniorPomidorro/suptech-go-kit/pkg/transport"
)

func TestOperationsCreateAlert(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("unexpected method: %s", r.Method)
		}
		if r.URL.Path != "/jsm/ops/api/cloud-1/v1/alerts" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"al-1","message":"Disk full"}`))
	}))
	defer srv.Close()

	client, err := NewClient(
		WithBaseURL(srv.URL),
		WithOpsCloudID("cloud-1"),
		WithTransport(transport.New()),
	)
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	alert, err := client.Operations().CreateAlert(context.Background(), map[string]any{"message": "Disk full"})
	if err != nil {
		t.Fatalf("CreateAlert failed: %v", err)
	}
	if alert.ID != "al-1" {
		t.Fatalf("unexpected alert id: %q", alert.ID)
	}
}

func TestOperationsListAlertsQuery(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/jsm/ops/api/cloud-1/v1/alerts" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		q := r.URL.Query()
		if q.Get("query") != "status:open" {
			t.Fatalf("unexpected query filter: %q", q.Get("query"))
		}
		if q.Get("limit") != "10" {
			t.Fatalf("unexpected limit: %q", q.Get("limit"))
		}
		if q.Get("offset") != "20" {
			t.Fatalf("unexpected offset: %q", q.Get("offset"))
		}
		if q.Get("order") != "createdAt:desc" {
			t.Fatalf("unexpected order: %q", q.Get("order"))
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"total":1,"values":[{"id":"al-2","status":"open"}]}`))
	}))
	defer srv.Close()

	client, err := NewClient(
		WithBaseURL(srv.URL),
		WithOpsCloudID("cloud-1"),
		WithTransport(transport.New()),
	)
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	result, err := client.Operations().ListAlerts(context.Background(), ListAlertsOptions{
		Query:  "status:open",
		Limit:  10,
		Offset: 20,
		Order:  "createdAt:desc",
	})
	if err != nil {
		t.Fatalf("ListAlerts failed: %v", err)
	}
	if len(result.Values) != 1 {
		t.Fatalf("expected one alert, got %d", len(result.Values))
	}
}

func TestOperationsEnableTeamAndSchedules(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/jsm/ops/api/cloud-1/v1/teams/team-1/enable":
			if r.Method != http.MethodPost {
				t.Fatalf("unexpected method: %s", r.Method)
			}
			w.WriteHeader(http.StatusNoContent)
		case "/jsm/ops/api/cloud-1/v1/schedules":
			if r.URL.Query().Get("teamId") != "team-1" {
				t.Fatalf("unexpected teamId: %q", r.URL.Query().Get("teamId"))
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"values":[{"id":"sch-1","name":"Primary"}]}`))
		case "/jsm/ops/api/cloud-1/v1/schedules/sch-1/on-call/responders":
			if r.URL.Query().Get("from") != "2026-01-01T00:00:00Z" {
				t.Fatalf("unexpected from: %q", r.URL.Query().Get("from"))
			}
			if r.URL.Query().Get("to") != "2026-01-01T12:00:00Z" {
				t.Fatalf("unexpected to: %q", r.URL.Query().Get("to"))
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"responders":[{"id":"u-1","name":"Alice"}]}`))
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer srv.Close()

	client, err := NewClient(
		WithBaseURL(srv.URL),
		WithAssetsCloudID("cloud-1"), // verifies fallback for operations cloud ID
		WithTransport(transport.New()),
	)
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	if err := client.Operations().EnableOpsForTeam(context.Background(), "team-1"); err != nil {
		t.Fatalf("EnableOpsForTeam failed: %v", err)
	}

	schedules, err := client.Operations().ListSchedules(context.Background(), ListSchedulesOptions{TeamID: "team-1"})
	if err != nil {
		t.Fatalf("ListSchedules failed: %v", err)
	}
	if len(schedules.Values) != 1 || schedules.Values[0].ID != "sch-1" {
		t.Fatalf("unexpected schedules response: %+v", schedules.Values)
	}

	responders, err := client.Operations().ListOnCallResponders(context.Background(), "sch-1", ListOnCallRespondersOptions{
		From: "2026-01-01T00:00:00Z",
		To:   "2026-01-01T12:00:00Z",
	})
	if err != nil {
		t.Fatalf("ListOnCallResponders failed: %v", err)
	}
	if len(responders) != 1 || responders[0].ID != "u-1" {
		t.Fatalf("unexpected responders response: %+v", responders)
	}
}

func TestOperationsGetMethods(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/jsm/ops/api/cloud-2/v1/alerts/al-100":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"id":"al-100","status":"open"}`))
		case "/jsm/ops/api/cloud-2/v1/teams":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"teams":[{"id":"team-2","name":"Ops"}]}`))
		case "/jsm/ops/api/cloud-2/v1/users/user-2/notification-settings":
			if r.URL.Query().Get("teamId") != "team-2" {
				t.Fatalf("unexpected teamId: %q", r.URL.Query().Get("teamId"))
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"userId":"user-2","channels":["email"]}`))
		case "/jsm/ops/api/cloud-2/v1/schedules/sch-2":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"id":"sch-2","name":"Night"}`))
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer srv.Close()

	client, err := NewClient(
		WithBaseURL(srv.URL),
		WithOpsCloudID("cloud-2"),
		WithTransport(transport.New()),
	)
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	alert, err := client.Operations().GetAlert(context.Background(), "al-100")
	if err != nil {
		t.Fatalf("GetAlert failed: %v", err)
	}
	if alert.ID != "al-100" {
		t.Fatalf("unexpected alert id: %q", alert.ID)
	}

	teams, err := client.Operations().ListTeams(context.Background(), ListTeamsOptions{})
	if err != nil {
		t.Fatalf("ListTeams failed: %v", err)
	}
	if len(teams.Values) != 1 || teams.Values[0].ID != "team-2" {
		t.Fatalf("unexpected teams response: %+v", teams.Values)
	}

	settings, err := client.Operations().GetUserNotificationSettings(context.Background(), "user-2", GetUserNotificationSettingsOptions{
		TeamID: "team-2",
	})
	if err != nil {
		t.Fatalf("GetUserNotificationSettings failed: %v", err)
	}
	if settings.UserID != "user-2" {
		t.Fatalf("unexpected settings user id: %q", settings.UserID)
	}

	schedule, err := client.Operations().GetSchedule(context.Background(), "sch-2")
	if err != nil {
		t.Fatalf("GetSchedule failed: %v", err)
	}
	if schedule.ID != "sch-2" {
		t.Fatalf("unexpected schedule id: %q", schedule.ID)
	}
}

func TestOperationsRequireCloudID(t *testing.T) {
	t.Parallel()

	client, err := NewClient(
		WithBaseURL("https://api.atlassian.com"),
		WithTransport(transport.New()),
	)
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	_, err = client.Operations().ListTeams(context.Background(), ListTeamsOptions{})
	if err == nil || !strings.Contains(err.Error(), "operations cloud ID is required") {
		t.Fatalf("expected missing cloud id error, got: %v", err)
	}
}
