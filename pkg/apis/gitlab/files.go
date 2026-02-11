package gitlab

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/SeniorPomidorro/suptech-go-kit/pkg/transport"
)

// DownloadRawFileByURL downloads raw file bytes from GitLab by absolute URL.
func (c *Client) DownloadRawFileByURL(ctx context.Context, rawURL string) ([]byte, error) {
	if strings.TrimSpace(rawURL) == "" {
		return nil, errors.New("gitlab: raw URL is required")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, fmt.Errorf("gitlab: create request: %w", err)
	}
	if strings.TrimSpace(c.token) != "" {
		req.Header.Set("PRIVATE-TOKEN", c.token)
	}

	resp, err := c.transport.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, transport.NewAPIError(resp, 0)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("gitlab: read response body: %w", err)
	}
	return data, nil
}
