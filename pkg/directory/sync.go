package directory

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"time"

	"self-hosted-node/pkg/config"
	"self-hosted-node/pkg/store"
	"self-hosted-node/pkg/store/models"
	"self-hosted-node/pkg/trisa/gds"
	"self-hosted-node/pkg/trisa/network"

	"github.com/rs/zerolog/log"
	members "github.com/trisacrypto/directory/pkg/gds/members/v1alpha1"
)

// Syncs the VASPs stored in the Global TRISA Directory (GDS) to local storage for
// easier counterparty lookup and analysis. This is a background routine that runs at
// a specified interval and ensures that only counterparties managed by the sync tool
// are updated in the local database.
type Sync struct {
	sync.Mutex
	conf  config.DirectorySyncConfig
	gds   gds.Directory
	store store.CounterpartyStore
	echan chan<- error
	stop  chan struct{}
	done  chan struct{}
}

// Creates a new directory synchronization service but does not run it.
func New(conf config.DirectorySyncConfig, network network.Network, store store.CounterpartyStore, echan chan<- error) (s *Sync, err error) {
	// Only return a sync stub if not enabled
	if !conf.Enabled {
		return &Sync{conf: conf}, nil
	}

	s = &Sync{
		conf:  conf,
		store: store,
		echan: echan,
	}

	if s.gds, err = network.Directory(); err != nil {
		return nil, err
	}

	return s, nil
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
	// Fetch members from the GDS and iterate over pages.
	iter := ListMembers(s.gds)
	for iter.Next() {
		for _, member := range iter.Members() {
			var vasp *models.Counterparty
			if vasp, err = s.Counterparty(member.Id); err != nil {
				log.Warn().Err(err).Str("vaspID", member.Id).Msg("could not fetch vasp member details")
				continue
			}

			if err = s.store.CreateCounterparty(context.Background(), vasp); err != nil {
				// TODO: handle database specific errors including constraint violations
				log.Warn().Err(err).Str("vaspID", member.Id).Msg("could not create vasp member counterparty")
				continue
			}
		}
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

func (s *Sync) Counterparty(vaspID string) (vasp *models.Counterparty, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	var detail *members.MemberDetails
	if detail, err = s.gds.Detail(ctx, &members.DetailsRequest{MemberId: vaspID}); err != nil {
		return nil, err
	}

	// Create counterparty from detail reply
	vasp = &models.Counterparty{
		Source:              models.SourceDirectorySync,
		DirectoryID:         sql.NullString{Valid: true, String: detail.MemberSummary.Id},
		RegisteredDirectory: sql.NullString{Valid: true, String: detail.MemberSummary.RegisteredDirectory},
		Protocol:            models.ProtocolTRISA,
		CommonName:          detail.MemberSummary.CommonName,
		Endpoint:            detail.MemberSummary.Endpoint,
		Name:                detail.MemberSummary.Name,
		Website:             sql.NullString{Valid: true, String: detail.MemberSummary.Website},
		Country:             detail.MemberSummary.Country,
		BusinessCategory:    detail.MemberSummary.BusinessCategory.String(),
		VASPCategories:      models.VASPCategories(detail.MemberSummary.VaspCategories),
		IVMSRecord:          detail.LegalPerson,
	}

	// Parse the verifiedOn timestamp
	if detail.MemberSummary.VerifiedOn != "" {
		var verifiedOn time.Time
		if verifiedOn, err = time.Parse(time.RFC3339, detail.MemberSummary.VerifiedOn); err != nil {
			return nil, fmt.Errorf("could not parse verified_on timestamp: %w", err)
		}
		vasp.VerifiedOn = sql.NullTime{Valid: true, Time: verifiedOn}
	}

	return vasp, nil
}
