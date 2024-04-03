package directory_test

import (
	"self-hosted-node/pkg/bufconn"
	"self-hosted-node/pkg/config"
	"self-hosted-node/pkg/directory"
	"self-hosted-node/pkg/store/models"
	"self-hosted-node/pkg/trisa/gds"
	mockgds "self-hosted-node/pkg/trisa/gds/mock"
	"self-hosted-node/pkg/trisa/network"
	"self-hosted-node/pkg/ulids"

	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestStartStop(t *testing.T) {
	t.Run("Enabled", func(t *testing.T) {
		sync, err := directory.New(config.DirectorySyncConfig{Enabled: true, Interval: 48 * time.Hour}, nil, nil, nil)
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
		require.NoError(t, sync.Stop(), "calling stop should do nothing when not enabled")

		// Multiple calls to Run and Stop should do nothing
		for i := 0; i < 4; i++ {
			require.NoError(t, sync.Run(), "expected no error trying to start a disabled dss")
			require.NoError(t, sync.Stop(), "expected no error trying to stop a disabled dss")
		}
	})
}

func TestSync(t *testing.T) {
	bufnet := bufconn.New()
	defer bufnet.Close()

	conf := config.Config{
		Node: config.TRISAConfig{
			Pool:                "testdata/trisatest.dev.pem",
			Certs:               "testdata/client.trisatest.dev.pem",
			KeyExchangeCacheTTL: 24 * time.Hour,
			Directory: config.DirectoryConfig{
				Insecure:        true,
				Endpoint:        "bufnet",
				MembersEndpoint: "bufnet",
			},
		},
		DirectorySync: config.DirectorySyncConfig{
			Enabled:  true,
			Interval: 48 * time.Hour,
		},
	}

	network, err := network.NewMocked(&conf.Node)
	require.NoError(t, err, "could not create mocked network")

	dir, err := network.Directory()
	require.NoError(t, err, "could not fetch mocked directory service")

	mock := dir.(*gds.MockGDS).GetMock()

	sync, err := directory.New(conf.DirectorySync, network, nil, nil)
	require.NoError(t, err, "could not create sync service")

	t.Run("Counterparty", func(t *testing.T) {
		err := mock.UseFixture(mockgds.DetailRPC, "testdata/detail.pb.json")
		require.NoError(t, err, "could not load detail.pb.json test fixture")

		vasp, err := sync.Counterparty("b5b20dc2-dc0c-4acf-9861-5b73f1ccc170")
		require.NoError(t, err, "expected no error fetching counterparty")

		require.True(t, ulids.IsZero(vasp.ID))
		require.Equal(t, models.SourceDirectorySync, vasp.Source)
		require.Equal(t, "b5b20dc2-dc0c-4acf-9861-5b73f1ccc170", vasp.DirectoryID.String)
		require.Equal(t, "trisatest.dev", vasp.RegisteredDirectory.String)
		require.Equal(t, models.ProtocolTRISA, vasp.Protocol)
		require.Equal(t, "alice.vaspbot.net", vasp.CommonName)
		require.Equal(t, "alice.vaspbot.net:443", vasp.Endpoint)
		require.Equal(t, "AliceVASP", vasp.Name)
		require.Equal(t, "https://alicevasp.com", vasp.Website.String)
		require.Equal(t, "US", vasp.Country)
		require.Equal(t, "BUSINESS_ENTITY", vasp.BusinessCategory)
		require.Len(t, vasp.VASPCategories, 2)
		require.Equal(t, time.Date(2024, time.April, 2, 21, 53, 21, 0, time.UTC), vasp.VerifiedOn.Time)
		require.NotNil(t, vasp.IVMSRecord)
		require.Zero(t, vasp.Created)
		require.Zero(t, vasp.Modified)
	})
}
