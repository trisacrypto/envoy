package directory

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"time"

	"github.com/trisacrypto/envoy/pkg/config"
	"github.com/trisacrypto/envoy/pkg/enum"
	"github.com/trisacrypto/envoy/pkg/store"
	"github.com/trisacrypto/envoy/pkg/store/models"
	"github.com/trisacrypto/envoy/pkg/store/txn"
	"github.com/trisacrypto/envoy/pkg/trisa/gds"
	"github.com/trisacrypto/envoy/pkg/trisa/network"

	"github.com/rs/zerolog/log"
	members "github.com/trisacrypto/directory/pkg/gds/members/v1alpha1"
	trisa "github.com/trisacrypto/trisa/pkg/trisa/gds/models/v1beta1"
	"go.rtnl.ai/ulid"
)

// Syncs the VASPs stored in the Global TRISA Directory (GDS) to local storage for
// easier counterparty lookup and analysis. This is a background routine that runs at
// a specified interval and ensures that only counterparties managed by the sync tool
// are updated in the local database.
type Sync struct {
	sync.Mutex
	conf  config.DirectorySyncConfig
	gds   gds.Directory
	store store.Store
	echan chan<- error
	stop  chan struct{}
	done  chan struct{}
}

// Creates a new directory synchronization service but does not run it.
func New(conf config.DirectorySyncConfig, network network.Network, store store.Store, echan chan<- error) (s *Sync, err error) {
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

	// Create a closure to handle each member sync operation. This closure will
	// create a new transaction for each member to ensure isolation and rollback on
	// failure. It will also handle the creation and update of the counterparty
	// in the local database, including contacts.
	syncMember := func(member *members.VASPMember) error {
		// Create a new transaction for each member to ensure isolation
		var tx txn.Txn
		if tx, err = s.store.Begin(context.Background(), &sql.TxOptions{ReadOnly: false}); err != nil {
			return err
		}
		defer tx.Rollback()

		var vasp *models.Counterparty
		if vasp, err = s.Counterparty(member.Id); err != nil {
			log.Warn().Err(err).Str("vaspID", member.Id).Msg("could not fetch vasp member details")
			return nil
		}

		// Lookup the counterparty in the local table to determine if update or create is required
		var update bool
		vasp.ID, update = local[member.Id]

		// Remove the contacts from the counterparty so that we can handle them spearately
		// TODO: allow many to many contacts to counterparties
		contacts, _ := vasp.Contacts()
		vasp.SetContacts(nil)

		if update {
			if err = tx.UpdateCounterparty(vasp); err != nil {
				// TODO: handle database specific errors including constraint violations
				log.Warn().Err(err).Str("id", vasp.ID.String()).Str("vaspID", member.Id).Msg("could not update vasp member counterparty")
			} else {
				updated++
			}

			// TODO: update contacts for the counterparty

		} else {
			if err = tx.CreateCounterparty(vasp); err != nil {
				// TODO: handle database specific errors including constraint violations
				log.Warn().Err(err).Str("vaspID", member.Id).Msg("could not create vasp member counterparty")
			} else {
				created++
			}

			// Create the contacts for the counterparty
			for _, contact := range contacts {
				contact.CounterpartyID = vasp.ID
				if err = tx.CreateContact(contact); err != nil {
					log.Warn().Err(err).Str("vaspID", member.Id).Msg("could not create contact for vasp member counterparty")
				}
			}
		}

		if err = tx.Commit(); err != nil {
			return err
		}

		upserts[vasp.ID] = struct{}{}
		return nil
	}

	// Fetch members from the GDS and iterate over pages.
	iter := ListMembers(s.gds)
	for iter.Next() {
		for _, member := range iter.Members() {
			if err = syncMember(member); err != nil {
				return err
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
	if srcInfo, err = s.store.ListCounterpartySourceInfo(context.Background(), enum.SourceDirectorySync); err != nil {
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
		Source:              enum.SourceDirectorySync,
		DirectoryID:         sql.NullString{Valid: true, String: detail.MemberSummary.Id},
		RegisteredDirectory: sql.NullString{Valid: true, String: detail.MemberSummary.RegisteredDirectory},
		Protocol:            enum.ProtocolTRISA,
		CommonName:          detail.MemberSummary.CommonName,
		Endpoint:            detail.MemberSummary.Endpoint,
		Name:                detail.MemberSummary.Name,
		Website:             sql.NullString{Valid: detail.MemberSummary.Website != "", String: detail.MemberSummary.Website},
		Country:             sql.NullString{Valid: detail.MemberSummary.Country != "", String: detail.MemberSummary.Country},
		BusinessCategory:    sql.NullString{Valid: true, String: detail.MemberSummary.BusinessCategory.String()},
		VASPCategories:      models.VASPCategories(detail.MemberSummary.VaspCategories),
		IVMSRecord:          detail.LegalPerson,
	}

	// Add contacts to the counterparty
	contacts := make([]*models.Contact, 0, 4)
	if contact := makeContact(detail.Contacts.Administrative, "Administrative Representative"); contact != nil {
		contacts = append(contacts, contact)
	}

	if contact := makeContact(detail.Contacts.Technical, "Technical Support"); contact != nil {
		contacts = append(contacts, contact)
	}

	if contact := makeContact(detail.Contacts.Legal, "Compliance Officer"); contact != nil {
		contacts = append(contacts, contact)
	}

	if contact := makeContact(detail.Contacts.Billing, "Billing and Accounts"); contact != nil {
		contacts = append(contacts, contact)
	}

	if len(contacts) > 0 {
		vasp.SetContacts(contacts)
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

func makeContact(contact *trisa.Contact, role string) *models.Contact {
	if contact == nil || contact.Email == "" {
		return nil
	}

	return &models.Contact{
		Name:  contact.Name,
		Email: contact.Email,
		Role:  role,
	}
}
