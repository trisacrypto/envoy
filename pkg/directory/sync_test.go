package directory_test

import (
	"self-hosted-node/pkg/config"
	"self-hosted-node/pkg/directory"

	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestStartStop(t *testing.T) {
	t.Run("Enabled", func(t *testing.T) {
		sync, err := directory.New(config.DirectorySyncConfig{Enabled: true, Interval: 2 * time.Hour}, nil, nil, nil)
		require.NoError(t, err, "could not create directory")

		require.ErrorIs(t, sync.Stop(), directory.ErrSyncNotRunning)

		err = sync.Run()
		require.NoError(t, err, "expected to be able to run the directory sync")
		require.ErrorIs(t, sync.Run(), directory.ErrSyncAlreadyRunning)

		err = sync.Stop()
		require.NoError(t, err, "should be able to shutdown a running sync")
		require.ErrorIs(t, sync.Stop(), directory.ErrSyncNotRunning)
	})

	t.Run("Disabled", func(t *testing.T) {
		sync, err := directory.New(config.DirectorySyncConfig{Enabled: false}, nil, nil, nil)
		require.NoError(t, err, "could not create directory")
		require.ErrorIs(t, sync.Stop(), directory.ErrSyncNotRunning)

		// Multiple calls to Run and Stop should do nothing
		for i := 0; i < 4; i++ {
			require.NoError(t, sync.Run(), "expected no error trying to start a disabled dss")
			require.ErrorIs(t, sync.Stop(), directory.ErrSyncNotRunning)
		}
	})
}
