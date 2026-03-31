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
		_, _ = w.Write([]byte(`{"result":"Request will be processed","requestId":"req-1","took":0.3}`))
	}))
	defer srv.Close()

	client, err := NewClient(
		WithBaseURL(srv.URL),
		WithCloudBaseURL(srv.URL),
		WithOpsCloudID("cloud-1"),
		WithTransport(transport.New()),
	)
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	resp, err := client.Operations().CreateAlert(context.Background(), map[string]any{"message": "Disk full"})
	if err != nil {
		t.Fatalf("CreateAlert failed: %v", err)
	}
	if resp.RequestID != "req-1" {
		t.Fatalf("unexpected request id: %q", resp.RequestID)
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
		if q.Get("size") != "10" {
			t.Fatalf("unexpected size: %q", q.Get("size"))
		}
		if q.Get("offset") != "20" {
			t.Fatalf("unexpected offset: %q", q.Get("offset"))
		}
		if q.Get("order") != "desc" {
			t.Fatalf("unexpected order: %q", q.Get("order"))
		}
		if q.Get("sort") != "createdAt" {
			t.Fatalf("unexpected sort: %q", q.Get("sort"))
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"count":1,"values":[{"id":"al-2","status":"open"}]}`))
	}))
	defer srv.Close()

	client, err := NewClient(
		WithBaseURL(srv.URL),
		WithCloudBaseURL(srv.URL),
		WithOpsCloudID("cloud-1"),
		WithTransport(transport.New()),
	)
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	result, err := client.Operations().ListAlerts(context.Background(), &ListAlertsOptions{
		Query:  "status:open",
		Size:   10,
		Offset: 20,
		Order:  "desc",
		Sort:   "createdAt",
	})
	if err != nil {
		t.Fatalf("ListAlerts failed: %v", err)
	}
	if len(result.Values) != 1 {
		t.Fatalf("expected one alert, got %d", len(result.Values))
	}
	if result.Count != 1 {
		t.Fatalf("expected count=1, got %d", result.Count)
	}
}

func TestOperationsEnableTeamAndSchedules(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/jsm/ops/api/cloud-1/v1/teams/team-1/enable-ops":
			if r.Method != http.MethodPost {
				t.Fatalf("unexpected method: %s", r.Method)
			}
			w.WriteHeader(http.StatusNoContent)
		case "/jsm/ops/api/cloud-1/v1/schedules":
			if r.URL.Query().Get("query") != "Primary" {
				t.Fatalf("unexpected query: %q", r.URL.Query().Get("query"))
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"values":[{"id":"sch-1","name":"Primary","teamId":"team-1"}]}`))
		case "/jsm/ops/api/cloud-1/v1/schedules/sch-1/on-calls":
			if r.URL.Query().Get("date") != "2026-01-01T00:00:00Z" {
				t.Fatalf("unexpected date: %q", r.URL.Query().Get("date"))
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"onCallParticipants":[{"id":"u-1","type":"user"}]}`))
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer srv.Close()

	client, err := NewClient(
		WithBaseURL(srv.URL),
		WithCloudBaseURL(srv.URL),
		WithAssetsCloudID("cloud-1"),
		WithTransport(transport.New()),
	)
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	if err := client.Operations().EnableOpsForTeam(context.Background(), "team-1"); err != nil {
		t.Fatalf("EnableOpsForTeam failed: %v", err)
	}

	schedules, err := client.Operations().ListSchedules(context.Background(), &ListSchedulesOptions{Query: "Primary"})
	if err != nil {
		t.Fatalf("ListSchedules failed: %v", err)
	}
	if len(schedules.Values) != 1 || schedules.Values[0].ID != "sch-1" {
		t.Fatalf("unexpected schedules response: %+v", schedules.Values)
	}

	onCalls, err := client.Operations().ListOnCalls(context.Background(), "sch-1", &ListOnCallOptions{
		Date: "2026-01-01T00:00:00Z",
	})
	if err != nil {
		t.Fatalf("ListOnCalls failed: %v", err)
	}
	if len(onCalls.OnCallParticipants) != 1 || onCalls.OnCallParticipants[0].ID != "u-1" {
		t.Fatalf("unexpected on-call response: %+v", onCalls)
	}
}

func TestOperationsGetMethods(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/jsm/ops/api/cloud-2/v1/alerts/al-100":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"id":"al-100","status":"open","message":"CPU high"}`))
		case "/jsm/ops/api/cloud-2/v1/teams":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"platformTeams":[{"teamId":"team-2","teamName":"Ops"}]}`))
		case "/jsm/ops/api/cloud-2/v1/notification-rules":
			if r.URL.Query().Get("size") != "10" {
				t.Fatalf("unexpected size: %q", r.URL.Query().Get("size"))
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"values":[{"id":"rule-1","name":"Default","actionType":"create-alert","enabled":true}]}`))
		case "/jsm/ops/api/cloud-2/v1/schedules/sch-2":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"id":"sch-2","name":"Night","timezone":"UTC","enabled":true}`))
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer srv.Close()

	client, err := NewClient(
		WithBaseURL(srv.URL),
		WithCloudBaseURL(srv.URL),
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
	if alert.ID != "al-100" || alert.Message != "CPU high" {
		t.Fatalf("unexpected alert: %+v", alert)
	}

	teams, err := client.Operations().ListTeams(context.Background())
	if err != nil {
		t.Fatalf("ListTeams failed: %v", err)
	}
	if len(teams.PlatformTeams) != 1 || teams.PlatformTeams[0].TeamID != "team-2" {
		t.Fatalf("unexpected teams response: %+v", teams.PlatformTeams)
	}

	rules, err := client.Operations().ListNotificationRules(context.Background(), &ListNotificationRulesOptions{Size: 10})
	if err != nil {
		t.Fatalf("ListNotificationRules failed: %v", err)
	}
	if len(rules.Values) != 1 || rules.Values[0].ID != "rule-1" {
		t.Fatalf("unexpected notification rules: %+v", rules.Values)
	}

	schedule, err := client.Operations().GetSchedule(context.Background(), "sch-2")
	if err != nil {
		t.Fatalf("GetSchedule failed: %v", err)
	}
	if schedule.ID != "sch-2" || schedule.Timezone != "UTC" {
		t.Fatalf("unexpected schedule: %+v", schedule)
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

	_, err = client.Operations().ListTeams(context.Background())
	if err == nil || !strings.Contains(err.Error(), "operations cloud ID is required") {
		t.Fatalf("expected missing cloud id error, got: %v", err)
	}
}
