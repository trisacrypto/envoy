package directory_test

import (
	"self-hosted-node/pkg/bufconn"
	"self-hosted-node/pkg/config"
	"self-hosted-node/pkg/directory"
	"self-hosted-node/pkg/trisa/gds"
	mockgds "self-hosted-node/pkg/trisa/gds/mock"

	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
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

func TestSync(t *testing.T) {
	bufnet := bufconn.New()
	defer bufnet.Close()

	conf := config.TRISAConfig{
		Directory: config.DirectoryConfig{
			Insecure:        true,
			Endpoint:        "bufnet",
			MembersEndpoint: "bufnet",
		},
	}

	gds := gds.New(conf)
	err := gds.Connect(grpc.WithContextDialer(bufnet.Dialer), grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err, "unable to connect to mock GDS via directory client")

	mock := mockgds.New(bufnet)

	t.Run("Counterparty", func(t *testing.T) {
		err := mock.UseFixture(mockgds.DetailRPC, "testdata/detail.pb.json")
		require.NoError(t, err, "could not load detail.pb.json test fixture")
	})
}
