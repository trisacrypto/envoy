package directory

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"time"

	"github.com/trisacrypto/envoy/pkg/config"
	"github.com/trisacrypto/envoy/pkg/store"
	"github.com/trisacrypto/envoy/pkg/store/models"
	"github.com/trisacrypto/envoy/pkg/trisa/gds"
	"github.com/trisacrypto/envoy/pkg/trisa/network"

	"github.com/oklog/ulid/v2"
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

	// Network should only be nil for testing purposes; will panic during sync otherwise
	if network != nil {
		if s.gds, err = network.Directory(); err != nil {
			return nil, err
		}
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

	// Execute first sync at startup
	if err := s.Sync(); err != nil {
		s.echan <- fmt.Errorf("directory synchronization fatal error: %w", err)
		return
	}

	// Start directory sync interval
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
	// Do not stop the directory service if it is not enabled
	if !s.conf.Enabled {
		return nil
	}

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
	log.Debug().Msg("starting directory members sync")

	// Create the counterparty source map for upsert identification and deletion
	var local map[string]ulid.ULID
	if local, err = s.MakeSourceMap(); err != nil {
		// If the database cannot be accessed, this is a fatal error
		return err
	}

	// Track which members have been upserted to identify counterparties to remove
	upserts := make(map[ulid.ULID]struct{})
	updated, created, deleted := 0, 0, 0

	// Fetch members from the GDS and iterate over pages.
	iter := ListMembers(s.gds)
	for iter.Next() {
		for _, member := range iter.Members() {
			var vasp *models.Counterparty
			if vasp, err = s.Counterparty(member.Id); err != nil {
				log.Warn().Err(err).Str("vaspID", member.Id).Msg("could not fetch vasp member details")
				continue
			}

			// Lookup the counterparty in the local table to determine if update or create is required
			var update bool
			vasp.ID, update = local[member.Id]

			if update {
				if err = s.store.UpdateCounterparty(context.Background(), vasp); err != nil {
					// TODO: handle database specific errors including constraint violations
					log.Warn().Err(err).Str("id", vasp.ID.String()).Str("vaspID", member.Id).Msg("could not update vasp member counterparty")
				} else {
					updated++
				}
			} else {
				if err = s.store.CreateCounterparty(context.Background(), vasp); err != nil {
					// TODO: handle database specific errors including constraint violations
					log.Warn().Err(err).Str("vaspID", member.Id).Msg("could not create vasp member counterparty")
				} else {
					created++
				}
			}

			upserts[vasp.ID] = struct{}{}
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

	// Remove members that are no longer part of the GDS
	for _, cpID := range local {
		// If a local ID has not been created or updated then it is no longer in the
		// GDS and should be removed from the local database.
		if _, ok := upserts[cpID]; !ok {
			if err = s.store.DeleteCounterparty(context.Background(), cpID); err != nil {
				log.Warn().Err(err).Str("id", cpID.String()).Msg("could not delete counterparty")
			} else {
				deleted++
			}
		}
	}

	log.Info().Int("updated", updated).Int("created", created).Int("deleted", deleted).Msg("directory members sync complete")
	return nil
}

func (s *Sync) MakeSourceMap() (local map[string]ulid.ULID, err error) {
	var srcInfo []*models.CounterpartySourceInfo
	if srcInfo, err = s.store.ListCounterpartySourceInfo(context.Background(), models.SourceDirectorySync); err != nil {
		return nil, err
	}

	local = make(map[string]ulid.ULID)
	for _, info := range srcInfo {
		if !info.DirectoryID.Valid || info.DirectoryID.String == "" {
			continue
		}
		local[info.DirectoryID.String] = info.ID
	}

	return local, nil
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
		BusinessCategory:    sql.NullString{Valid: true, String: detail.MemberSummary.BusinessCategory.String()},
		VASPCategories:      models.VASPCategories(detail.MemberSummary.VaspCategories),
		IVMSRecord:          detail.LegalPerson,
	}

	// Parse the verifiedOn timestamp
	if detail.MemberSummary.VerifiedOn != "" {
		var verifiedOn time.Time
		if verifiedOn, err = time.Parse(time.RFC3339, detail.MemberSummary.VerifiedOn); err != nil {
			return nil, fmt.Errorf("could not parse verified_on timestamp: %w", err)
		}
		vasp.VerifiedOn = sql.NullTime{Valid: true, Time: verifiedOn.In(time.UTC)}
	}

	return vasp, nil
}
