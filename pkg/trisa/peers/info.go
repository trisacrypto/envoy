package peers

import (
	"database/sql"
	"time"

	"github.com/trisacrypto/envoy/pkg/store/models"
	"github.com/trisacrypto/envoy/pkg/web/api/v1"
)

// Info provides detailed VASP counterparty information that can be looked up via the
// TRISA Global Directory Service (GDS). This data structure can also be used for
// identifying counterparties in TRISA transactions or other interactions with the GDS.
type Info struct {
	ID                  string    `json:"vasp_id"`
	RegisteredDirectory string    `json:"registered_directory"`
	CommonName          string    `json:"common_name"`
	Endpoint            string    `json:"endpoint"`
	Name                string    `json:"name"`
	Country             string    `json:"country"`
	VerifiedOn          time.Time `json:"verified_on"`
}

// Validate the info struct contains enough information for Peer operations.
func (i *Info) Validate() error {
	if i.CommonName == "" {
		return ErrNoCommonName
	}

	if i.Endpoint == "" {
		return ErrNoEndpoint
	}
	return nil
}

// Returns a counterparty data structure for API purposes.
func (i *Info) Counterparty() *api.Counterparty {
	return &api.Counterparty{
		DirectoryID:         i.ID,
		RegisteredDirectory: i.RegisteredDirectory,
		CommonName:          i.CommonName,
		Endpoint:            i.Endpoint,
		Name:                i.Name,
		Country:             i.Country,
		VerifiedOn:          i.VerifiedOn,
	}
}

// Returns a counterparty model data structure for TRISA interactions
func (i *Info) Model() *models.Counterparty {
	return &models.Counterparty{
		Source:              models.SourcePeer,
		DirectoryID:         sql.NullString{Valid: i.ID != "", String: i.ID},
		RegisteredDirectory: sql.NullString{Valid: i.RegisteredDirectory != "", String: i.RegisteredDirectory},
		Protocol:            models.ProtocolTRISA,
		CommonName:          i.CommonName,
		Endpoint:            i.Endpoint,
		Name:                i.Name,
		Country:             sql.NullString{Valid: i.Country != "", String: i.Country},
		VerifiedOn:          sql.NullTime{Valid: !i.VerifiedOn.IsZero(), Time: i.VerifiedOn},
	}
}
