# suptech-go-kit

`suptech-go-kit` is a Go toolkit of API clients for common engineering integrations:

- Atlassian (Jira Issues/Users, Jira Assets, Jira Operations)
- Slack (MVP + Iteration 2: Views, Canvas, Socket Mode)
- GitLab (raw file download)
- Shared HTTP transport with retry/timeout/error handling

The goal is to avoid rewriting the same integration plumbing for each API: auth, retries, pagination, error handling, and safe `context.Context` usage.

## What This Package Provides

- One shared `transport` layer for all clients
- Timeouts, retries for `429/5xx`, and `Retry-After` support
- Normalized HTTP errors via `transport.APIError`
- Slack `ok=false` responses mapped to `slack.Error`
- Cursor pagination in Slack `conversations.list`
- AQL search with `FetchAll` in Jira Assets
- Slack Socket Mode loop with automatic envelope ACK

## Installation

```bash
go get github.com/SeniorPomidorro/suptech-go-kit@latest
```

## Quick Start

### 1) Shared transport

```go
import (
	"time"

	"github.com/SeniorPomidorro/suptech-go-kit/pkg/transport"
)

tr := transport.New(
	transport.WithTimeout(10*time.Second),
	transport.WithRetry(transport.RetryConfig{MaxAttempts: 3}),
)
```

### 2) Jira/Atlassian client

```go
import (
	"context"

	"github.com/SeniorPomidorro/suptech-go-kit/pkg/apis/atlassian"
)

ctx := context.Background()

jira, err := atlassian.NewClient(
	atlassian.WithBaseURL("https://your-domain.atlassian.net"),
	atlassian.WithAuth(atlassian.Auth{
		Mode:  atlassian.AuthBasicEmailToken, // or AuthBearerToken / AuthBasicToken
		Email: "a@b.com",
		Token: "xxxx",
	}),
	atlassian.WithAssetsCloudID("cloud-id"),
	atlassian.WithAssetsWorkspaceID("workspace-id"),
	atlassian.WithOpsCloudID("cloud-id"),
	atlassian.WithTransport(tr),
)
if err != nil {
	panic(err)
}

issue, err := jira.Issues().GetIssue(ctx, "ABC-123")
if err != nil {
	panic(err)
}
_ = issue
```

### 3) Slack client

```go
import (
	"context"

	"github.com/SeniorPomidorro/suptech-go-kit/pkg/apis/slack"
)

ctx := context.Background()

sl, err := slack.NewClient(
	slack.WithToken("xoxb-..."),
	slack.WithTeamID("T123"), // optional
	slack.WithTransport(tr),
)
if err != nil {
	panic(err)
}

msg, err := sl.Messages().PostMessage(ctx, "C123", "hello")
if err != nil {
	panic(err)
}
_ = msg
```

### 4) GitLab client

```go
import (
	"context"

	"github.com/SeniorPomidorro/suptech-go-kit/pkg/apis/gitlab"
)

ctx := context.Background()

gl := gitlab.NewClient(
	gitlab.WithToken("glpat-..."),
	gitlab.WithTransport(tr),
)

raw, err := gl.DownloadRawFileByURL(ctx, "https://gitlab.com/group/project/-/raw/main/README.md")
if err != nil {
	panic(err)
}
_ = raw
```

## Package API Overview

### `pkg/transport`

- `transport.New(opts...)`
- `Do(req)`
- `DoJSON(req, out)`
- Options: `WithTimeout`, `WithRetry`, `WithLogger`, `WithBaseHeaders`

### `pkg/apis/atlassian`

- Issues: `GetIssue`, `SaveStoryPoints`, `FindIssues` (`FetchAll`), `ManageTags`, `CreateComment`, `AddAttachment`
- Users: `FindUsers`
- Assets: `SearchObjectsAQL` (`FetchAll`), `CreateObject`, `DeleteObject`, `UpdateObject`, `GetObject`
- Operations: `CreateAlert`, `GetAlert`, `ListAlerts`, `EnableOpsForTeam`, `ListTeams`, `GetUserNotificationSettings`, `ListSchedules`, `GetSchedule`, `ListOnCallResponders`

### `pkg/apis/slack`

- User groups: `CreateUserGroup`, `ListUserGroups`
- Conversations: `GetConversationList`, `CreateConversation`, `GetChannelByID`, `InviteUsersToChannel`
- Messages: `PostMessage`, `PostEphemeralMessage`, `UpdateMessage`
- Users: `GetUserByID`, `GetUsersByID`, `GetUsersByGroupID`, `GetUserByEmail`

- Views: `OpenView`, `UpdateView`
- Canvas: `CreateCanvas`, `ShareCanvas`
- Socket Mode runtime: `Run`, `RunWithHandler`

### `pkg/apis/gitlab`

- `DownloadRawFileByURL`

## Error Handling

HTTP-level API errors:

```go
import (
	"errors"
	"fmt"

	"github.com/SeniorPomidorro/suptech-go-kit/pkg/transport"
)

if err != nil {
	var apiErr *transport.APIError
	if errors.As(err, &apiErr) {
		fmt.Println(apiErr.StatusCode, apiErr.Body)
	}
}
```

Slack `ok=false` errors:

```go
import (
	"errors"
	"fmt"

	"github.com/SeniorPomidorro/suptech-go-kit/pkg/apis/slack"
)

if err != nil {
	var slackErr *slack.Error
	if errors.As(err, &slackErr) {
		fmt.Println(slackErr.Code) // e.g. channel_not_found
	}
}
```

## Socket Mode Example

```go
import (
	"context"
	"errors"

	"github.com/SeniorPomidorro/suptech-go-kit/pkg/apis/slack"
)

ctx := context.Background()

sm := slack.NewSocketModeClient(
	slack.WithAppLevelToken("xapp-..."),
	slack.WithSocketModeTransport(tr),
)

handler := slack.SocketModeHandlerFunc(func(ctx context.Context, event slack.SocketModeEvent) (*slack.SocketModeResponse, error) {
	return &slack.SocketModeResponse{
		Payload: map[string]any{"text": "processed"},
	}, nil
})

if err := sm.RunWithHandler(ctx, handler); err != nil && !errors.Is(err, context.Canceled) {
	panic(err)
}
```

## Principles

- Every public method accepts `context.Context`
- Secrets must never be logged
- API design and examples are production integration-oriented
