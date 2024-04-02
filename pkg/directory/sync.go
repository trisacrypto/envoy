package directory

import (
	"fmt"
	"sync"
	"time"

	"self-hosted-node/pkg/config"
	"self-hosted-node/pkg/store"
	"self-hosted-node/pkg/trisa/gds"
	"self-hosted-node/pkg/trisa/network"

	"github.com/rs/zerolog/log"
)

// Syncs the VASPs stored in the Global TRISA Directory (GDS) to local storage for
// easier counterparty lookup and analysis. This is a background routine that runs at
// a specified interval and ensures that only counterparties managed by the sync tool
// are updated in the local database.
type Sync struct {
	sync.Mutex
	conf    config.DirectorySyncConfig
	network network.Network
	store   store.CounterpartyStore
	echan   chan<- error
	stop    chan struct{}
	done    chan struct{}
}

// Creates a new directory synchronization service but does not run it.
func New(conf config.DirectorySyncConfig, network network.Network, store store.CounterpartyStore, echan chan<- error) (s *Sync, err error) {
	// Only return a sync stub if not enabled
	if !conf.Enabled {
		return &Sync{conf: conf}, nil
	}

	// Return a fully allocated sync service
	return &Sync{
		conf:    conf,
		network: network,
		store:   store,
		echan:   echan,
	}, nil
}

// Run the directory synchronization service.
func (s *Sync) Run() error {
	// Do not run the service if the directory sync is not enabled.
	if !s.conf.Enabled {
		return nil
	}

	// Lock the sync routine to initialize and start it.
	s.Lock()
	defer s.Unlock()

	if s.stop != nil {
		return ErrSyncAlreadyRunning
	}

	s.stop = make(chan struct{})
	s.done = make(chan struct{})
	go s.run()
	return nil
}

func (s *Sync) run() {
	ticker := time.NewTicker(s.conf.Interval)
	log.Info().Dur("sync_interval", s.conf.Interval).Msg("directory sync service running")

syncloop:
	for {
		select {
		case <-s.stop:
			break syncloop
		case <-ticker.C:
			if err := s.Sync(); err != nil {
				s.echan <- fmt.Errorf("directory synchronization fatal error: %w", err)
				break syncloop
			}
		}
	}

	close(s.done)
	log.Info().Msg("directory sync service stopped")
}

// Stop the directory synchronization service, blocking until the service is shutdown.
func (s *Sync) Stop() error {
	s.Lock()
	defer s.Unlock()

	if s.stop == nil {
		return ErrSyncNotRunning
	}

	// Send the stop signal and wait for routine to stop.
	close(s.stop)
	<-s.done

	s.stop = nil
	s.done = nil
	return nil
}

func (s *Sync) Sync() (err error) {
	// Fetch members from the GDS
	var gds gds.Directory
	if gds, err = s.network.Directory(); err != nil {
		return err
	}

	// Iterate over all members returned from the directory service
	iter := ListMembers(gds)
	for iter.Next() {
		// TODO: fetch details for member and populate in store.
	}

	// Check no errors were returned from iteration.
	if err = iter.Err(); err != nil {
		// Log error but return nil so the error is not fatal and we'll try again
		// during the next interval. If the error is configuration related (e.g. wrong
		// endpoint, bad certs) then the log should indicate the problem.
		log.Warn().Err(err).Msg("error fetching members from gds")
		return nil
	}

	// TODO: remove members that are no longer part of the GDS

	return nil
}
