package slack

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/SeniorPomidorro/suptech-go-kit/pkg/transport"
)

func TestOpenView(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/views.open" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		payloadBytes, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read body: %v", err)
		}
		var payload map[string]any
		if err := json.Unmarshal(payloadBytes, &payload); err != nil {
			t.Fatalf("decode payload: %v", err)
		}
		if payload["trigger_id"] != "1337.42.abcd" {
			t.Fatalf("unexpected trigger_id: %+v", payload["trigger_id"])
		}
		view, ok := payload["view"].(map[string]any)
		if !ok {
			t.Fatalf("view payload missing")
		}
		if view["type"] != "modal" {
			t.Fatalf("unexpected view type: %v", view["type"])
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true,"view":{"id":"V1","type":"modal","callback_id":"deploy_modal","hash":"h1"}}`))
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

	result, err := client.Views().OpenView(context.Background(), "1337.42.abcd", &ModalViewRequest{
		Type:       "modal",
		CallbackID: "deploy_modal",
	})
	if err != nil {
		t.Fatalf("OpenView failed: %v", err)
	}
	if result.View.ID != "V1" {
		t.Fatalf("unexpected view id: %q", result.View.ID)
	}
}

func TestUpdateView(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/views.update" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		payloadBytes, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read body: %v", err)
		}
		var payload map[string]any
		if err := json.Unmarshal(payloadBytes, &payload); err != nil {
			t.Fatalf("decode payload: %v", err)
		}
		if payload["view_id"] != "V2" {
			t.Fatalf("unexpected view_id: %v", payload["view_id"])
		}
		if payload["hash"] != "h2" {
			t.Fatalf("unexpected hash: %v", payload["hash"])
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true,"view":{"id":"V2","type":"modal","hash":"h3"}}`))
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

	result, err := client.Views().UpdateView(context.Background(), ModalViewRequest{
		Type: "modal",
	}, "", "h2", "V2")
	if err != nil {
		t.Fatalf("UpdateView failed: %v", err)
	}
	if result.View.ID != "V2" {
		t.Fatalf("unexpected view id: %q", result.View.ID)
	}
}

func TestUpdateViewRequiresViewIDOrExternalID(t *testing.T) {
	t.Parallel()

	client, err := NewClient(WithToken("xoxb-test"), WithTransport(transport.New()))
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	_, err = client.Views().UpdateView(context.Background(), ModalViewRequest{Type: "modal"}, "", "", "")
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestCreateAndShareCanvas(t *testing.T) {
	t.Parallel()

	createCalled := false
	shareCalled := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/canvases.create":
			createCalled = true
			payloadBytes, err := io.ReadAll(r.Body)
			if err != nil {
				t.Fatalf("read body: %v", err)
			}
			var payload map[string]any
			if err := json.Unmarshal(payloadBytes, &payload); err != nil {
				t.Fatalf("decode payload: %v", err)
			}
			if payload["title"] != "Runbook" {
				t.Fatalf("unexpected title: %v", payload["title"])
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"ok":true,"canvas":{"id":"CANVAS1","title":"Runbook"}}`))
		case "/canvases.access.set":
			shareCalled = true
			payloadBytes, err := io.ReadAll(r.Body)
			if err != nil {
				t.Fatalf("read body: %v", err)
			}
			var payload map[string]any
			if err := json.Unmarshal(payloadBytes, &payload); err != nil {
				t.Fatalf("decode payload: %v", err)
			}
			if payload["canvas_id"] != "CANVAS1" {
				t.Fatalf("unexpected canvas id: %v", payload["canvas_id"])
			}
			_, _ = w.Write([]byte(`{"ok":true}`))
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
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

	canvas, err := client.Canvas().CreateCanvas(context.Background(), "Runbook", "# Incident")
	if err != nil {
		t.Fatalf("CreateCanvas failed: %v", err)
	}
	if canvas.ID != "CANVAS1" {
		t.Fatalf("unexpected canvas id: %q", canvas.ID)
	}

	if err := client.Canvas().ShareCanvas(context.Background(), "CANVAS1", "C123"); err != nil {
		t.Fatalf("ShareCanvas failed: %v", err)
	}

	if !createCalled || !shareCalled {
		t.Fatalf("expected create/share to be called, got create=%v share=%v", createCalled, shareCalled)
	}
}

func TestOpenViewReturnsSlackError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":false,"error":"invalid_trigger"}`))
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

	_, err = client.Views().OpenView(context.Background(), "bad", &ModalViewRequest{Type: "modal"})
	if err == nil {
		t.Fatalf("expected error")
	}
	var slackErr *Error
	if !errors.As(err, &slackErr) {
		t.Fatalf("expected slack.Error, got %T", err)
	}
	if slackErr.Code != "invalid_trigger" {
		t.Fatalf("unexpected error code: %q", slackErr.Code)
	}
}

func TestShareCanvasValidation(t *testing.T) {
	t.Parallel()

	client, err := NewClient(WithToken("xoxb-test"), WithTransport(transport.New()))
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	if err := client.Canvas().ShareCanvas(context.Background(), "", "C123"); err == nil {
		t.Fatalf("expected error for empty canvas ID")
	}
	if err := client.Canvas().ShareCanvas(context.Background(), "CANVAS", ""); err == nil {
		t.Fatalf("expected error for empty channel ID")
	}
}
