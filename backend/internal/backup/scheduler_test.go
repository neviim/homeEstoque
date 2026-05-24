package backup_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/neviim/homeestoque/backend/internal/backup"
	"github.com/neviim/homeestoque/backend/internal/testutil"
)

func TestUpdateSchedule_PersistsAndValidates(t *testing.T) {
	env := testutil.NewBackupEnv(t)
	ctx := context.Background()

	wd := 3 // quarta
	want := backup.Schedule{
		Enabled:        true,
		Frequency:      "weekly",
		Weekday:        &wd,
		TimeOfDay:      "04:30",
		RetentionCount: 5,
	}
	got, err := env.Manager.UpdateSchedule(ctx, want)
	require.NoError(t, err)
	assert.True(t, got.Enabled)
	assert.Equal(t, "weekly", got.Frequency)
	require.NotNil(t, got.Weekday)
	assert.Equal(t, 3, *got.Weekday)
	assert.Equal(t, "04:30", got.TimeOfDay)
	assert.Equal(t, 5, got.RetentionCount)
}

func TestUpdateSchedule_RejectsInvalidTime(t *testing.T) {
	env := testutil.NewBackupEnv(t)
	ctx := context.Background()

	bad := backup.Schedule{Enabled: true, Frequency: "daily", TimeOfDay: "99:99", RetentionCount: 3}
	_, err := env.Manager.UpdateSchedule(ctx, bad)
	require.Error(t, err)
}

func TestUpdateSchedule_RejectsWeeklyWithoutWeekday(t *testing.T) {
	env := testutil.NewBackupEnv(t)
	ctx := context.Background()

	bad := backup.Schedule{Enabled: true, Frequency: "weekly", TimeOfDay: "03:00", RetentionCount: 3}
	_, err := env.Manager.UpdateSchedule(ctx, bad)
	require.Error(t, err)
}

func TestUpdateSchedule_ClampsRetention(t *testing.T) {
	env := testutil.NewBackupEnv(t)
	ctx := context.Background()

	got, err := env.Manager.UpdateSchedule(ctx, backup.Schedule{
		Enabled: false, Frequency: "daily", TimeOfDay: "03:00", RetentionCount: 999,
	})
	require.NoError(t, err)
	assert.Equal(t, 100, got.RetentionCount, "retention é clamped em 100")

	got, err = env.Manager.UpdateSchedule(ctx, backup.Schedule{
		Enabled: false, Frequency: "daily", TimeOfDay: "03:00", RetentionCount: 0,
	})
	require.NoError(t, err)
	assert.Equal(t, 1, got.RetentionCount, "retention mínimo é 1")
}

func TestSchedulerStart_DisabledByDefault(t *testing.T) {
	env := testutil.NewBackupEnv(t)
	env.Manager.StartScheduler(context.Background())
	defer env.Manager.StopScheduler()

	// Schedule default: enabled=false. Nada deve ter rodado.
	list, err := env.Manager.List(context.Background())
	require.NoError(t, err)
	assert.Empty(t, list)
}
