package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	aapi "github.com/grafana/amixr-api-go-client"
	mcpgrafana "github.com/grafana/mcp-grafana"
	"github.com/mark3labs/mcp-go/server"
)

type ListOnCallSchedulesParams struct {
	TeamID     string `json:"teamId,omitempty" jsonschema:"description=The ID of the team to list schedules for"`
	ScheduleID string `json:"scheduleId,omitempty" jsonschema:"description=The ID of the schedule to get details for. If provided, returns only that schedule's details"`
	Page       int    `json:"page,omitempty" jsonschema:"description=The page number to return (1-based)"`
}

// ScheduleSummary represents a simplified view of an OnCall schedule
type ScheduleSummary struct {
	ID       string   `json:"id" jsonschema:"description=The unique identifier of the schedule"`
	Name     string   `json:"name" jsonschema:"description=The name of the schedule"`
	TeamID   string   `json:"teamId" jsonschema:"description=The ID of the team this schedule belongs to"`
	Timezone string   `json:"timezone" jsonschema:"description=The timezone for this schedule"`
	Shifts   []string `json:"shifts" jsonschema:"description=List of shift IDs in this schedule"`
}

func listOnCallSchedulesHandler(ctx context.Context, args ListOnCallSchedulesParams) ([]*ScheduleSummary, error) {
	return fetchOnCallSchedules(ctx, args)
}

var ListOnCallSchedules = mcpgrafana.MustTool(
	"list_oncall_schedules",
	"List OnCall schedules. A schedule is a calendar-based system defining when team members are on-call. Optionally provide a scheduleId to get details for a specific schedule",
	listOnCallSchedulesHandler,
)

type GetOnCallShiftParams struct {
	ShiftID string `json:"shiftId" jsonschema:"required,description=The ID of the shift to get details for"`
}

func getOnCallShiftHandler(ctx context.Context, args GetOnCallShiftParams) (*aapi.OnCallShift, error) {
	return fetchOnCallShift(ctx, args)
}

var GetOnCallShift = mcpgrafana.MustTool(
	"get_oncall_shift",
	"Get details for a specific OnCall shift. A shift represents a designated time period within a rotation when a team or individual is actively on-call",
	getOnCallShiftHandler,
)

// CurrentOnCallUsers represents the currently on-call users for a schedule
type CurrentOnCallUsers struct {
	ScheduleID   string       `json:"scheduleId" jsonschema:"description=The ID of the schedule"`
	ScheduleName string       `json:"scheduleName" jsonschema:"description=The name of the schedule"`
	Users        []*aapi.User `json:"users" jsonschema:"description=List of users currently on call"`
}

type GetCurrentOnCallUsersParams struct {
	ScheduleID string `json:"scheduleId" jsonschema:"required,description=The ID of the schedule to get current on-call users for"`
}

func getCurrentOnCallUsersHandler(ctx context.Context, args GetCurrentOnCallUsersParams) (*CurrentOnCallUsers, error) {
	return fetchCurrentOnCallUsers(ctx, args)
}

var GetCurrentOnCallUsers = mcpgrafana.MustTool(
	"get_current_oncall_users",
	"Get users currently on-call for a specific schedule. A schedule is a calendar-based system defining when team members are on-call. This tool will return info about all users currently on-call for the schedule, regardless of team.",
	getCurrentOnCallUsersHandler,
)

type ListOnCallTeamsParams struct {
	Page int `json:"page,omitempty" jsonschema:"description=The page number to return"`
}

func listOnCallTeamsHandler(ctx context.Context, args ListOnCallTeamsParams) ([]*aapi.Team, error) {
	return fetchOnCallTeams(ctx, args)
}

var ListOnCallTeams = mcpgrafana.MustTool(
	"list_oncall_teams",
	"List teams from Grafana OnCall",
	listOnCallTeamsHandler,
)

type ListOnCallUsersParams struct {
	UserID   string `json:"userId,omitempty" jsonschema:"description=The ID of the user to get details for. If provided, returns only that user's details"`
	Username string `json:"username,omitempty" jsonschema:"description=The username to filter users by. If provided, returns only the user matching this username"`
	Page     int    `json:"page,omitempty" jsonschema:"description=The page number to return"`
}

func listOnCallUsersHandler(ctx context.Context, args ListOnCallUsersParams) ([]*aapi.User, error) {
	return fetchOnCallUsers(ctx, args)
}

var ListOnCallUsers = mcpgrafana.MustTool(
	"list_oncall_users",
	"List users from Grafana OnCall. If user ID is provided, returns details for that specific user. If username is provided, returns the user matching that username",
	listOnCallUsersHandler,
)

type ListOnCallAlertGroupsParams struct {
	ID            string `json:"id,omitempty" jsonschema:"description=Exact match, alert group ID"`
	RouteID       string `json:"route_id,omitempty" jsonschema:"description=Exact match, route ID"`
	IntegrationID string `json:"integration_id,omitempty" jsonschema:"description=Exact match, integration ID"`
	State         string `json:"state,omitempty" jsonschema:"description=Possible values: new, acknowledged, resolved or silenced"`
	TeamID        string `json:"team_id,omitempty" jsonschema:"description=Exact match, team ID"`
	StartedAt     string `json:"started_at,omitempty" jsonschema:"description=Filter alert groups by start time in ISO 8601 format with start and end timestamps separated by underscore. Example: 2024-03-20T10:00:00_2024-03-21T10:00:00"`
	Labels        string `json:"labels,omitempty" jsonschema:"description=Filter alert groups by labels. Expected format: key1:value1,key2:value2"`
	TeamName      string `json:"team_name,omitempty" jsonschema:"description=Team name. If provided, returns only alert groups for this team. It may not be an exact match."`
	Name          string `json:"name,omitempty" jsonschema:"description=Filter alert groups by name"`
	Page          int    `json:"page,omitempty" jsonschema:"description=The page number to return (1-based)"`
}

func listOnCallAlertGroupsHandler(ctx context.Context, args ListOnCallAlertGroupsParams) ([]*aapi.AlertGroup, error) {
	return fetchOnCallAlertGroups(ctx, args)
}

var ListOnCallAlertGroups = mcpgrafana.MustTool(
	"list_oncall_alert_groups",
	"List alert groups from Grafana OnCall. Optionally filter by alert group ID, route ID, integration ID, state, team ID, labels, or name.",
	listOnCallAlertGroupsHandler,
)

type GetOnCallAlertsParams struct {
	AlertGroupID string `json:"alertGroupId" jsonschema:"required,description=The ID of the alert group to get alerts for"`
	Page         int    `json:"page,omitempty" jsonschema:"description=The page number to return (1-based)"`
}

func getOnCallAlertsHandler(ctx context.Context, args GetOnCallAlertsParams) ([]*aapi.Alert, error) {
	return fetchOnCallAlerts(ctx, args)
}

var GetOnCallAlerts = mcpgrafana.MustTool(
	"get_oncall_alerts",
	"Get alerts for a specific alert group in Grafana OnCall",
	getOnCallAlertsHandler,
)

func AddOnCallTools(mcp *server.MCPServer) {
	ListOnCallSchedules.Register(mcp)
	GetOnCallShift.Register(mcp)
	GetCurrentOnCallUsers.Register(mcp)
	ListOnCallTeams.Register(mcp)
	ListOnCallUsers.Register(mcp)
	ListOnCallAlertGroups.Register(mcp)
	GetOnCallAlerts.Register(mcp)
}

// getOnCallURLFromSettings retrieves the OnCall API URL from the Grafana settings endpoint.
// It makes a GET request to <grafana-url>/api/plugins/grafana-irm-app/settings and extracts
// the OnCall URL from the jsonData.onCallApiUrl field in the response.
// Returns the OnCall URL if found, or an error if the URL cannot be retrieved.
func getOnCallURLFromSettings(ctx context.Context, grafanaURL, grafanaAPIKey string) (string, error) {
	settingsURL := fmt.Sprintf("%s/api/plugins/grafana-irm-app/settings", strings.TrimRight(grafanaURL, "/"))

	req, err := http.NewRequestWithContext(ctx, "GET", settingsURL, nil)
	if err != nil {
		return "", fmt.Errorf("creating settings request: %w", err)
	}

	if grafanaAPIKey != "" {
		req.Header.Set("Authorization", "Bearer "+grafanaAPIKey)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("fetching settings: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status code from settings API: %d", resp.StatusCode)
	}

	var settings struct {
		JSONData struct {
			OnCallAPIURL string `json:"onCallApiUrl"`
		} `json:"jsonData"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&settings); err != nil {
		return "", fmt.Errorf("decoding settings response: %w", err)
	}

	if settings.JSONData.OnCallAPIURL == "" {
		return "", fmt.Errorf("OnCall API URL is not set in settings")
	}

	return settings.JSONData.OnCallAPIURL, nil
}

func oncallClientFromContext(ctx context.Context) (*aapi.Client, error) {
	// Get the standard Grafana URL and API key
	grafanaURL, grafanaAPIKey := mcpgrafana.GrafanaURLFromContext(ctx), mcpgrafana.GrafanaAPIKeyFromContext(ctx)

	// Try to get OnCall URL from settings endpoint
	grafanaOnCallURL, err := getOnCallURLFromSettings(ctx, grafanaURL, grafanaAPIKey)
	if err != nil {
		return nil, fmt.Errorf("getting OnCall URL from settings: %w", err)
	}

	grafanaOnCallURL = strings.TrimRight(grafanaOnCallURL, "/")

	client, err := aapi.NewWithGrafanaURL(grafanaOnCallURL, grafanaAPIKey, grafanaURL)
	if err != nil {
		return nil, fmt.Errorf("creating OnCall client: %w", err)
	}

	return client, nil
}

// getUserServiceFromContext creates a new UserService using the OnCall client from the context
func getUserServiceFromContext(ctx context.Context) (*aapi.UserService, error) {
	client, err := oncallClientFromContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("getting OnCall client: %w", err)
	}

	return aapi.NewUserService(client), nil
}

// getScheduleServiceFromContext creates a new ScheduleService using the OnCall client from the context
func getScheduleServiceFromContext(ctx context.Context) (*aapi.ScheduleService, error) {
	client, err := oncallClientFromContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("getting OnCall client: %w", err)
	}

	return aapi.NewScheduleService(client), nil
}

// getTeamServiceFromContext creates a new TeamService using the OnCall client from the context
func getTeamServiceFromContext(ctx context.Context) (*aapi.TeamService, error) {
	client, err := oncallClientFromContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("getting OnCall client: %w", err)
	}

	return aapi.NewTeamService(client), nil
}

// getOnCallShiftServiceFromContext creates a new OnCallShiftService using the OnCall client from the context
func getOnCallShiftServiceFromContext(ctx context.Context) (*aapi.OnCallShiftService, error) {
	client, err := oncallClientFromContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("getting OnCall client: %w", err)
	}

	return aapi.NewOnCallShiftService(client), nil
}

// getAlertGroupServiceFromContext creates a new AlertGroupService using the OnCall client from the context
func getAlertGroupServiceFromContext(ctx context.Context) (*aapi.AlertGroupService, error) {
	client, err := oncallClientFromContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("getting OnCall client: %w", err)
	}

	return aapi.NewAlertGroupService(client), nil
}

// getAlertServiceFromContext creates a new AlertService using the OnCall client from the context
func getAlertServiceFromContext(ctx context.Context) (*aapi.AlertService, error) {
	client, err := oncallClientFromContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("getting OnCall client: %w", err)
	}

	return aapi.NewAlertService(client), nil
}

// --- API Call Implementation Functions ---

// fetchOnCallSchedules performs the API call to list or get OnCall schedules.
func fetchOnCallSchedules(ctx context.Context, args ListOnCallSchedulesParams) ([]*ScheduleSummary, error) {
	scheduleService, err := getScheduleServiceFromContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("getting OnCall schedule service: %w", err)
	}

	if args.ScheduleID != "" {
		schedule, _, err := scheduleService.GetSchedule(args.ScheduleID, &aapi.GetScheduleOptions{})
		if err != nil {
			return nil, fmt.Errorf("getting OnCall schedule %s: %w", args.ScheduleID, err)
		}
		summary := &ScheduleSummary{
			ID:       schedule.ID,
			Name:     schedule.Name,
			TeamID:   schedule.TeamId,
			Timezone: schedule.TimeZone,
		}
		if schedule.Shifts != nil {
			summary.Shifts = *schedule.Shifts
		}
		return []*ScheduleSummary{summary}, nil
	}

	listOptions := &aapi.ListScheduleOptions{}
	if args.Page > 0 {
		listOptions.Page = args.Page
	}
	if args.TeamID != "" {
		listOptions.TeamID = args.TeamID
	}

	response, _, err := scheduleService.ListSchedules(listOptions)
	if err != nil {
		return nil, fmt.Errorf("listing OnCall schedules: %w", err)
	}

	// Convert schedules to summaries
	summaries := make([]*ScheduleSummary, 0, len(response.Schedules))
	for _, schedule := range response.Schedules {
		summary := &ScheduleSummary{
			ID:       schedule.ID,
			Name:     schedule.Name,
			TeamID:   schedule.TeamId,
			Timezone: schedule.TimeZone,
		}
		if schedule.Shifts != nil {
			summary.Shifts = *schedule.Shifts
		}
		summaries = append(summaries, summary)
	}

	return summaries, nil
}

// fetchOnCallShift performs the API call to get details for a specific OnCall shift.
func fetchOnCallShift(ctx context.Context, args GetOnCallShiftParams) (*aapi.OnCallShift, error) {
	shiftService, err := getOnCallShiftServiceFromContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("getting OnCall shift service: %w", err)
	}

	shift, _, err := shiftService.GetOnCallShift(args.ShiftID, &aapi.GetOnCallShiftOptions{})
	if err != nil {
		return nil, fmt.Errorf("getting OnCall shift %s: %w", args.ShiftID, err)
	}

	return shift, nil
}

// fetchCurrentOnCallUsers performs the API calls to get the current on-call users for a schedule.
func fetchCurrentOnCallUsers(ctx context.Context, args GetCurrentOnCallUsersParams) (*CurrentOnCallUsers, error) {
	scheduleService, err := getScheduleServiceFromContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("getting OnCall schedule service: %w", err)
	}

	schedule, _, err := scheduleService.GetSchedule(args.ScheduleID, &aapi.GetScheduleOptions{})
	if err != nil {
		return nil, fmt.Errorf("getting schedule %s: %w", args.ScheduleID, err)
	}

	// Create the result with the schedule info
	result := &CurrentOnCallUsers{
		ScheduleID:   schedule.ID,
		ScheduleName: schedule.Name,
		Users:        make([]*aapi.User, 0, len(schedule.OnCallNow)),
	}

	// If there are no users on call, return early
	if len(schedule.OnCallNow) == 0 {
		return result, nil
	}

	// Get the user service to fetch user details
	userService, err := getUserServiceFromContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("getting OnCall user service: %w", err)
	}

	// Fetch details for each user currently on call
	for _, userID := range schedule.OnCallNow {
		user, _, err := userService.GetUser(userID, &aapi.GetUserOptions{})
		if err != nil {
			// Log the error but continue with other users
			fmt.Printf("Error fetching user %s: %v\n", userID, err)
			continue
		}
		result.Users = append(result.Users, user)
	}

	return result, nil
}

// fetchOnCallTeams performs the API call to list OnCall teams.
func fetchOnCallTeams(ctx context.Context, args ListOnCallTeamsParams) ([]*aapi.Team, error) {
	teamService, err := getTeamServiceFromContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("getting OnCall team service: %w", err)
	}

	listOptions := &aapi.ListTeamOptions{}
	if args.Page > 0 {
		listOptions.Page = args.Page
	}

	response, _, err := teamService.ListTeams(listOptions)
	if err != nil {
		return nil, fmt.Errorf("listing OnCall teams: %w", err)
	}

	return response.Teams, nil
}

// fetchOnCallUsers performs the API call to list or get OnCall users.
func fetchOnCallUsers(ctx context.Context, args ListOnCallUsersParams) ([]*aapi.User, error) {
	userService, err := getUserServiceFromContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("getting OnCall user service: %w", err)
	}

	if args.UserID != "" {
		user, _, err := userService.GetUser(args.UserID, &aapi.GetUserOptions{})
		if err != nil {
			return nil, fmt.Errorf("getting OnCall user %s: %w", args.UserID, err)
		}
		return []*aapi.User{user}, nil
	}

	// Otherwise, list all users
	listOptions := &aapi.ListUserOptions{}
	if args.Page > 0 {
		listOptions.Page = args.Page
	}
	if args.Username != "" {
		listOptions.Username = args.Username
	}

	response, _, err := userService.ListUsers(listOptions)
	if err != nil {
		return nil, fmt.Errorf("listing OnCall users: %w", err)
	}

	return response.Users, nil
}

// fetchOnCallAlertGroups performs the API call to list OnCall alert groups.
func fetchOnCallAlertGroups(ctx context.Context, args ListOnCallAlertGroupsParams) ([]*aapi.AlertGroup, error) {
	alertGroupService, err := getAlertGroupServiceFromContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("getting OnCall alert group service: %w", err)
	}

	listOptions := &aapi.ListAlertGroupOptions{}

	if args.Page > 0 {
		listOptions.Page = args.Page
	}
	if args.ID != "" {
		listOptions.AlertGroupID = args.ID
	}
	if args.RouteID != "" {
		listOptions.RouteID = args.RouteID
	}
	if args.IntegrationID != "" {
		listOptions.IntegrationID = args.IntegrationID
	}
	if args.State != "" {
		listOptions.State = args.State
	}
	if args.Name != "" {
		listOptions.Name = args.Name
	}
	if args.TeamID != "" {
		listOptions.TeamID = args.TeamID
	}
	if args.StartedAt != "" {
		listOptions.StartedAt = args.StartedAt
	}
	if args.Labels != "" {
		listOptions.Labels = strings.Split(args.Labels, ",")
	}

	response, _, err := alertGroupService.ListAlertGroups(listOptions)
	if err != nil {
		return nil, fmt.Errorf("listing OnCall alert groups: %w", err)
	}

	return response.AlertGroups, nil
}

// fetchOnCallAlerts performs the API call to get alerts for a specific alert group.
func fetchOnCallAlerts(ctx context.Context, args GetOnCallAlertsParams) ([]*aapi.Alert, error) {
	alertService, err := getAlertServiceFromContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("getting OnCall alert service: %w", err)
	}

	listOptions := &aapi.ListAlertOptions{
		AlertGroupID: args.AlertGroupID,
	}
	if args.Page > 0 {
		listOptions.Page = args.Page
	}

	response, _, err := alertService.ListAlerts(listOptions)
	if err != nil {
		return nil, fmt.Errorf("listing OnCall alerts for alert group %s: %w", args.AlertGroupID, err)
	}

	return response.Alerts, nil
}
