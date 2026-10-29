package atlassian

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

const teamsPageSize = "300"

// TeamsService provides Atlassian Teams public API methods, served through the site gateway.
type TeamsService struct {
	client *Client
}

// ListTeams returns every team of the organisation on the site, following the cursor until it is empty.
func (s *TeamsService) ListTeams(ctx context.Context, orgID, siteID string) ([]OrgTeam, error) {
	path, err := teamsPath(orgID, "")
	if err != nil {
		return nil, err
	}
	var all []OrgTeam
	seen := map[string]bool{}
	for cursor := ""; ; {
		query := url.Values{"size": {teamsPageSize}}
		if siteID != "" {
			query.Set("siteId", siteID)
		}
		if cursor != "" {
			query.Set("cursor", cursor)
		}
		req, err := s.client.newRequest(ctx, http.MethodGet, path, query, nil)
		if err != nil {
			return nil, err
		}
		var page struct {
			Entities []OrgTeam `json:"entities"`
			Cursor   string    `json:"cursor"`
		}
		if err := s.client.transport.DoJSON(req, &page); err != nil {
			return nil, err
		}
		all = append(all, page.Entities...)
		if page.Cursor == "" {
			return all, nil
		}
		if seen[page.Cursor] {
			return nil, fmt.Errorf("atlassian: teams cursor %q repeated", page.Cursor)
		}
		seen[page.Cursor] = true
		cursor = page.Cursor
	}
}

// CreateTeam creates a team; link it to a group with LinkTeam afterwards to keep its own name.
func (s *TeamsService) CreateTeam(ctx context.Context, orgID string, body *CreateTeamRequest) (*OrgTeam, error) {
	if body == nil || strings.TrimSpace(body.DisplayName) == "" {
		return nil, errors.New("atlassian: team display name is required")
	}
	if body.TeamType == "" {
		return nil, errors.New("atlassian: team type is required")
	}
	path, err := teamsPath(orgID, "")
	if err != nil {
		return nil, err
	}
	req, err := s.client.newRequest(ctx, http.MethodPost, path, nil, body)
	if err != nil {
		return nil, err
	}
	// not replayable, so never retried: a create the gateway answered with 5xx may have made the team already
	req.GetBody = nil
	var team OrgTeam
	if err := s.client.transport.DoJSON(req, &team); err != nil {
		return nil, err
	}
	return &team, nil
}

// LinkTeam links an existing team to an external reference: from then on the team's members are synced from it.
func (s *TeamsService) LinkTeam(ctx context.Context, orgID, teamID, siteID string, ref TeamExternalReference) error {
	if strings.TrimSpace(teamID) == "" {
		return errors.New("atlassian: team ID is required")
	}
	if ref.Source == "" || strings.TrimSpace(ref.ID) == "" {
		return errors.New("atlassian: external reference source and ID are required")
	}
	path, err := teamPath(orgID, teamID, "/external/link")
	if err != nil {
		return err
	}
	body := struct {
		ExternalReference TeamExternalReference `json:"externalReference"`
		SiteID            string                `json:"siteId,omitempty"`
	}{ref, siteID}
	req, err := s.client.newRequest(ctx, http.MethodPost, path, nil, body)
	if err != nil {
		return err
	}
	return s.client.doNoResponseBody(req)
}

// DeleteTeam deletes a team.
func (s *TeamsService) DeleteTeam(ctx context.Context, orgID, teamID, siteID string) error {
	if strings.TrimSpace(teamID) == "" {
		return errors.New("atlassian: team ID is required")
	}
	path, err := teamPath(orgID, teamID, "")
	if err != nil {
		return err
	}
	var query url.Values
	if siteID != "" {
		query = url.Values{"siteId": {siteID}}
	}
	req, err := s.client.newRequest(ctx, http.MethodDelete, path, query, nil)
	if err != nil {
		return err
	}
	return s.client.doNoResponseBody(req)
}

func teamsPath(orgID, suffix string) (string, error) {
	if strings.TrimSpace(orgID) == "" {
		return "", errors.New("atlassian: organisation ID is required")
	}
	if !pathSegment(orgID) {
		return "", fmt.Errorf("atlassian: organisation ID %q is not a single path segment", orgID)
	}
	// unescaped: the request URL escapes its path itself
	return "/gateway/api/public/teams/v1/org/" + orgID + "/teams" + suffix, nil
}

func teamPath(orgID, teamID, suffix string) (string, error) {
	if !pathSegment(teamID) {
		return "", fmt.Errorf("atlassian: team ID %q is not a single path segment", teamID)
	}
	return teamsPath(orgID, "/"+teamID+suffix)
}

// pathSegment rejects ids that would leave their segment: a slash, or a dot segment the URL resolution collapses.
func pathSegment(id string) bool {
	return !strings.ContainsAny(id, "/\\") && id != "." && id != ".."
}
