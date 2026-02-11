package gitlab

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/SeniorPomidorro/suptech-go-kit/pkg/transport"
)

func TestDownloadRawFileByURL(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("unexpected method: %s", r.Method)
		}
		if got := r.Header.Get("PRIVATE-TOKEN"); got != "token-123" {
			t.Fatalf("unexpected PRIVATE-TOKEN: %q", got)
		}
		_, _ = w.Write([]byte("raw-content"))
	}))
	defer srv.Close()

	client := NewClient(
		WithToken("token-123"),
		WithTransport(transport.New()),
	)

	data, err := client.DownloadRawFileByURL(context.Background(), srv.URL+"/my/file.txt")
	if err != nil {
		t.Fatalf("DownloadRawFileByURL failed: %v", err)
	}
	if string(data) != "raw-content" {
		t.Fatalf("unexpected body: %q", string(data))
	}
}

func TestDownloadRawFileByURLWithoutToken(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("PRIVATE-TOKEN"); got != "" {
			t.Fatalf("expected empty PRIVATE-TOKEN, got %q", got)
		}
		_, _ = w.Write([]byte("public-raw-content"))
	}))
	defer srv.Close()

	client := NewClient(WithTransport(transport.New()))
	data, err := client.DownloadRawFileByURL(context.Background(), srv.URL+"/public/file.txt")
	if err != nil {
		t.Fatalf("DownloadRawFileByURL failed: %v", err)
	}
	if string(data) != "public-raw-content" {
		t.Fatalf("unexpected body: %q", string(data))
	}
}

func TestDownloadRawFileByURLReturnsAPIError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte("file not found"))
	}))
	defer srv.Close()

	client := NewClient(WithToken("token-123"), WithTransport(transport.New()))
	_, err := client.DownloadRawFileByURL(context.Background(), srv.URL+"/missing/file.txt")
	if err == nil {
		t.Fatalf("expected error")
	}

	var apiErr *transport.APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected transport.APIError, got %T", err)
	}
	if apiErr.StatusCode != http.StatusNotFound {
		t.Fatalf("unexpected status code: %d", apiErr.StatusCode)
	}
}
