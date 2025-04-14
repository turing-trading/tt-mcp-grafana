//go:build cloud
// +build cloud

// This file contains cloud integration tests that run against a dedicated test instance
// at mcptests.grafana-dev.net. This instance is configured with a minimal setup on the OnCall side:
//   - One team
//   - Two schedules (only one has a team assigned)
//   - One shift in the schedule with a team assigned
//   - One user
// These tests expect this configuration to exist and will skip if the required
// environment variables (GRAFANA_URL, GRAFANA_API_KEY) are not set.

package tools

import (
	"context"
	"os"
	"testing"

	mcpgrafana "github.com/grafana/mcp-grafana"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createOnCallCloudTestContext(t *testing.T) context.Context {
	grafanaURL := os.Getenv("GRAFANA_URL")
	if grafanaURL == "" {
		t.Skip("GRAFANA_URL environment variable not set, skipping cloud OnCall integration tests")
	}

	grafanaApiKey := os.Getenv("GRAFANA_API_KEY")
	if grafanaApiKey == "" {
		t.Skip("GRAFANA_API_KEY environment variable not set, skipping cloud OnCall integration tests")
	}

	ctx := context.Background()
	ctx = mcpgrafana.WithGrafanaURL(ctx, grafanaURL)
	ctx = mcpgrafana.WithGrafanaAPIKey(ctx, grafanaApiKey)

	return ctx
}

func TestCloudOnCallSchedules(t *testing.T) {
	ctx := createOnCallCloudTestContext(t)

	// Test listing all schedules
	t.Run("list all schedules", func(t *testing.T) {
		result, err := listOnCallSchedulesHandler(ctx, ListOnCallSchedulesParams{})
		require.NoError(t, err, "Should not error when listing schedules")
		assert.NotNil(t, result, "Result should not be nil")
	})

	// Test pagination
	t.Run("list schedules with pagination", func(t *testing.T) {
		// Get first page
		page1, err := listOnCallSchedulesHandler(ctx, ListOnCallSchedulesParams{Page: 1})
		require.NoError(t, err, "Should not error when listing schedules page 1")
		assert.NotNil(t, page1, "Page 1 should not be nil")

		// Get second page
		page2, err := listOnCallSchedulesHandler(ctx, ListOnCallSchedulesParams{Page: 2})
		require.NoError(t, err, "Should not error when listing schedules page 2")
		assert.NotNil(t, page2, "Page 2 should not be nil")
	})

	// Get a team ID from an existing schedule to test filtering
	schedules, err := listOnCallSchedulesHandler(ctx, ListOnCallSchedulesParams{})
	require.NoError(t, err, "Should not error when listing schedules")

	if len(schedules) > 0 && schedules[0].TeamID != "" {
		teamID := schedules[0].TeamID

		// Test filtering by team ID
		t.Run("list schedules by team ID", func(t *testing.T) {
			result, err := listOnCallSchedulesHandler(ctx, ListOnCallSchedulesParams{
				TeamID: teamID,
			})
			require.NoError(t, err, "Should not error when listing schedules by team")
			assert.NotEmpty(t, result, "Should return at least one schedule")
			for _, schedule := range result {
				assert.Equal(t, teamID, schedule.TeamID, "All schedules should belong to the specified team")
			}
		})
	}

	// Test getting a specific schedule
	if len(schedules) > 0 {
		scheduleID := schedules[0].ID
		t.Run("get specific schedule", func(t *testing.T) {
			result, err := listOnCallSchedulesHandler(ctx, ListOnCallSchedulesParams{
				ScheduleID: scheduleID,
			})
			require.NoError(t, err, "Should not error when getting specific schedule")
			assert.Len(t, result, 1, "Should return exactly one schedule")
			assert.Equal(t, scheduleID, result[0].ID, "Should return the correct schedule")

			// Verify all summary fields are present
			schedule := result[0]
			assert.NotEmpty(t, schedule.Name, "Schedule should have a name")
			assert.NotEmpty(t, schedule.Timezone, "Schedule should have a timezone")
			assert.NotNil(t, schedule.Shifts, "Schedule should have a shifts field")
		})
	}
}

func TestCloudOnCallShift(t *testing.T) {
	ctx := createOnCallCloudTestContext(t)

	// First get a schedule to find a valid shift
	schedules, err := listOnCallSchedulesHandler(ctx, ListOnCallSchedulesParams{})
	require.NoError(t, err, "Should not error when listing schedules")
	require.NotEmpty(t, schedules, "Should have at least one schedule to test with")
	require.NotEmpty(t, schedules[0].Shifts, "Schedule should have at least one shift")

	shifts := schedules[0].Shifts
	shiftID := shifts[0]

	// Test getting shift details with valid ID
	t.Run("get shift details", func(t *testing.T) {
		result, err := getOnCallShiftHandler(ctx, GetOnCallShiftParams{
			ShiftID: shiftID,
		})
		require.NoError(t, err, "Should not error when getting shift details")
		assert.NotNil(t, result, "Result should not be nil")
		assert.Equal(t, shiftID, result.ID, "Should return the correct shift")
	})

	t.Run("get shift with invalid ID", func(t *testing.T) {
		_, err := getOnCallShiftHandler(ctx, GetOnCallShiftParams{
			ShiftID: "invalid-shift-id",
		})
		assert.Error(t, err, "Should error when getting shift with invalid ID")
	})
}

func TestCloudGetCurrentOnCallUsers(t *testing.T) {
	ctx := createOnCallCloudTestContext(t)

	// First get a schedule to use for testing
	schedules, err := listOnCallSchedulesHandler(ctx, ListOnCallSchedulesParams{})
	require.NoError(t, err, "Should not error when listing schedules")
	require.NotEmpty(t, schedules, "Should have at least one schedule to test with")

	scheduleID := schedules[0].ID

	// Test getting current on-call users
	t.Run("get current on-call users", func(t *testing.T) {
		result, err := getCurrentOnCallUsersHandler(ctx, GetCurrentOnCallUsersParams{
			ScheduleID: scheduleID,
		})
		require.NoError(t, err, "Should not error when getting current on-call users")
		assert.NotNil(t, result, "Result should not be nil")
		assert.Equal(t, scheduleID, result.ScheduleID, "Should return the correct schedule")
		assert.NotEmpty(t, result.ScheduleName, "Schedule should have a name")
		assert.NotNil(t, result.Users, "Users field should be present")

		// Assert that Users is of type []*aapi.User
		if len(result.Users) > 0 {
			user := result.Users[0]
			assert.NotEmpty(t, user.ID, "User should have an ID")
			assert.NotEmpty(t, user.Username, "User should have a username")
		}
	})

	t.Run("get current on-call users with invalid schedule ID", func(t *testing.T) {
		_, err := getCurrentOnCallUsersHandler(ctx, GetCurrentOnCallUsersParams{
			ScheduleID: "invalid-schedule-id",
		})
		assert.Error(t, err, "Should error when getting current on-call users with invalid schedule ID")
	})
}

func TestCloudOnCallTeams(t *testing.T) {
	ctx := createOnCallCloudTestContext(t)

	t.Run("list teams", func(t *testing.T) {
		result, err := listOnCallTeamsHandler(ctx, ListOnCallTeamsParams{})
		require.NoError(t, err, "Should not error when listing teams")
		assert.NotNil(t, result, "Result should not be nil")

		if len(result) > 0 {
			team := result[0]
			assert.NotEmpty(t, team.ID, "Team should have an ID")
			assert.NotEmpty(t, team.Name, "Team should have a name")
		}
	})

	// Test pagination
	t.Run("list teams with pagination", func(t *testing.T) {
		// Get first page
		page1, err := listOnCallTeamsHandler(ctx, ListOnCallTeamsParams{Page: 1})
		require.NoError(t, err, "Should not error when listing teams page 1")
		assert.NotNil(t, page1, "Page 1 should not be nil")

		// Get second page
		page2, err := listOnCallTeamsHandler(ctx, ListOnCallTeamsParams{Page: 2})
		require.NoError(t, err, "Should not error when listing teams page 2")
		assert.NotNil(t, page2, "Page 2 should not be nil")
	})
}

func TestCloudOnCallUsers(t *testing.T) {
	ctx := createOnCallCloudTestContext(t)

	t.Run("list all users", func(t *testing.T) {
		result, err := listOnCallUsersHandler(ctx, ListOnCallUsersParams{})
		require.NoError(t, err, "Should not error when listing users")
		assert.NotNil(t, result, "Result should not be nil")

		if len(result) > 0 {
			user := result[0]
			assert.NotEmpty(t, user.ID, "User should have an ID")
			assert.NotEmpty(t, user.Username, "User should have a username")
		}
	})

	// Test pagination
	t.Run("list users with pagination", func(t *testing.T) {
		// Get first page
		page1, err := listOnCallUsersHandler(ctx, ListOnCallUsersParams{Page: 1})
		require.NoError(t, err, "Should not error when listing users page 1")
		assert.NotNil(t, page1, "Page 1 should not be nil")

		// Get second page
		page2, err := listOnCallUsersHandler(ctx, ListOnCallUsersParams{Page: 2})
		require.NoError(t, err, "Should not error when listing users page 2")
		assert.NotNil(t, page2, "Page 2 should not be nil")
	})

	// Get a user ID and username from the list to test filtering
	users, err := listOnCallUsersHandler(ctx, ListOnCallUsersParams{})
	require.NoError(t, err, "Should not error when listing users")
	require.NotEmpty(t, users, "Should have at least one user to test with")

	userID := users[0].ID
	username := users[0].Username

	t.Run("get user by ID", func(t *testing.T) {
		result, err := listOnCallUsersHandler(ctx, ListOnCallUsersParams{
			UserID: userID,
		})
		require.NoError(t, err, "Should not error when getting user by ID")
		assert.NotNil(t, result, "Result should not be nil")
		assert.Len(t, result, 1, "Should return exactly one user")
		assert.Equal(t, userID, result[0].ID, "Should return the correct user")
		assert.NotEmpty(t, result[0].Username, "User should have a username")
	})

	t.Run("get user by username", func(t *testing.T) {
		result, err := listOnCallUsersHandler(ctx, ListOnCallUsersParams{
			Username: username,
		})
		require.NoError(t, err, "Should not error when getting user by username")
		assert.NotNil(t, result, "Result should not be nil")
		assert.Len(t, result, 1, "Should return exactly one user")
		assert.Equal(t, username, result[0].Username, "Should return the correct user")
		assert.NotEmpty(t, result[0].ID, "User should have an ID")
	})

	t.Run("get user with invalid ID", func(t *testing.T) {
		_, err := listOnCallUsersHandler(ctx, ListOnCallUsersParams{
			UserID: "invalid-user-id",
		})
		assert.Error(t, err, "Should error when getting user with invalid ID")
	})

	t.Run("get user with invalid username", func(t *testing.T) {
		result, err := listOnCallUsersHandler(ctx, ListOnCallUsersParams{
			Username: "invalid-username",
		})
		require.NoError(t, err, "Should not error when getting user with invalid username")
		assert.Empty(t, result, "Should return empty result set for invalid username")
	})
}

// Add tests for alert groups and alerts
func TestCloudOnCallAlertGroups(t *testing.T) {
	ctx := createOnCallCloudTestContext(t)

	t.Run("list alert groups", func(t *testing.T) {
		result, err := listOnCallAlertGroupsHandler(ctx, ListOnCallAlertGroupsParams{})
		require.NoError(t, err, "Should not error when listing alert groups")
		assert.NotNil(t, result, "Result should not be nil")

		// Note: The test instance may not have any alert groups
		// So we don't assert on the length, just that the call succeeds
	})

	// Test pagination
	t.Run("list alert groups with pagination", func(t *testing.T) {
		// Get first page
		page1, err := listOnCallAlertGroupsHandler(ctx, ListOnCallAlertGroupsParams{Page: 1})
		require.NoError(t, err, "Should not error when listing alert groups page 1")
		assert.NotNil(t, page1, "Page 1 should not be nil")

		// Get second page
		page2, err := listOnCallAlertGroupsHandler(ctx, ListOnCallAlertGroupsParams{Page: 2})
		require.NoError(t, err, "Should not error when listing alert groups page 2")
		assert.NotNil(t, page2, "Page 2 should not be nil")
	})
}

func TestCloudOnCallAlerts(t *testing.T) {
	ctx := createOnCallCloudTestContext(t)

	// First try to get an alert group to test with
	alertGroups, err := listOnCallAlertGroupsHandler(ctx, ListOnCallAlertGroupsParams{})
	require.NoError(t, err, "Should not error when listing alert groups")

	// This is a conditional test as the test instance may not have any alert groups
	if len(alertGroups) > 0 {
		alertGroupID := alertGroups[0].ID

		t.Run("get alerts for alert group", func(t *testing.T) {
			result, err := getOnCallAlertsHandler(ctx, GetOnCallAlertsParams{
				AlertGroupID: alertGroupID,
			})
			require.NoError(t, err, "Should not error when getting alerts for alert group")
			assert.NotNil(t, result, "Result should not be nil")
		})
	} else {
		t.Skip("No alert groups available to test alert retrieval")
	}

	t.Run("get alerts with invalid alert group ID", func(t *testing.T) {
		_, err := getOnCallAlertsHandler(ctx, GetOnCallAlertsParams{
			AlertGroupID: "invalid-alert-group-id",
		})
		assert.Error(t, err, "Should error when getting alerts with invalid alert group ID")
	})
}
