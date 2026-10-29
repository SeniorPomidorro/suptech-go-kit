package atlassian

// Team types and states of the Atlassian Teams public API.
const (
	TeamTypeOpen         = "OPEN"
	TeamTypeMemberInvite = "MEMBER_INVITE"
	TeamTypeExternal     = "EXTERNAL"
	TeamStateActive      = "ACTIVE"
	// TeamSourceGroup is an external reference to an Atlassian group, by group ID.
	TeamSourceGroup = "ATLASSIAN_GROUP"
)

// OrgTeam is an Atlassian team of the organisation, the one behind the Jira Team field.
type OrgTeam struct {
	TeamID            string                 `json:"teamId"`
	OrganizationID    string                 `json:"organizationId,omitempty"`
	DisplayName       string                 `json:"displayName"`
	Description       string                 `json:"description,omitempty"`
	State             string                 `json:"state,omitempty"`
	TeamType          string                 `json:"teamType,omitempty"`
	ExternalReference *TeamExternalReference `json:"externalReference,omitempty"`
}

// LinkedTo reports whether the team syncs its members from the group.
func (t OrgTeam) LinkedTo(groupID string) bool {
	return groupID != "" && t.ExternalReference != nil && t.ExternalReference.Source == TeamSourceGroup && t.ExternalReference.ID == groupID
}

// TeamExternalReference is what a team syncs its members from.
type TeamExternalReference struct {
	Source string `json:"source"`
	ID     string `json:"id"`
}

// CreateTeamRequest is the body of a team creation.
type CreateTeamRequest struct {
	DisplayName string `json:"displayName"`
	Description string `json:"description"`
	TeamType    string `json:"teamType"`
	SiteID      string `json:"siteId,omitempty"`
}
