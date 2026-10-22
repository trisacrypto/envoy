package web_test

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/trisacrypto/envoy/pkg/enum"
	dberr "github.com/trisacrypto/envoy/pkg/store/errors"
	"github.com/trisacrypto/envoy/pkg/store/models"
	api "github.com/trisacrypto/envoy/pkg/web/api/v1"
	"go.rtnl.ai/ulid"
)

func onCreateCounterparty(ctx context.Context, in *models.Counterparty, log *models.ComplianceAuditLog) error {
	if !in.ID.IsZero() {
		return dberr.ErrNoIDOnCreate
	}

	if in.Source != enum.SourceUserEntry {
		return errors.New("source must be user entry")
	}

	if in.CommonName == "alice.vaspbot.com" || in.CommonName == "example.com" {
		return dberr.ErrAlreadyExists
	}

	in.ID = ulid.MakeSecure()
	in.Created = time.Now()
	in.Modified = in.Created

	return nil
}

func (w *webTestSuite) TestCreateCounterparty() {
	require := w.Require()
	client := w.ClientWithPermissions([]string{"counterparties:view", "counterparties:manage"})

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	w.Run("Sunrise", func() {
		w.store.OnCreateCounterparty = onCreateCounterparty

		in := &api.Counterparty{
			Protocol: "sunrise",
			Name:     "NorthEx CoinBank",
			Country:  "CA",
			Contacts: []*api.Contact{
				{
					Name:  "Compliance Officer",
					Email: "compliance@northex.ca",
					Role:  "Compliance",
				},
			},
		}

		out, err := client.CreateCounterparty(ctx, in)
		require.NoError(err, "could not create counterparty")
		require.NotNil(out, "counterparty should not be nil")

		require.NotZero(out.ID, "counterparty ID should be created")
		require.Equal(enum.SourceUserEntry.String(), out.Source, "counterparty source should be user entry")
		require.Equal("NorthEx CoinBank", out.Name, "counterparty name should be set")
		require.Equal("CA", out.Country, "counterparty country should be set")
		require.Equal(1, len(out.Contacts), "counterparty should have one contact")
		require.Equal("Compliance Officer", out.Contacts[0].Name, "counterparty contact name should be set")
		require.Equal("compliance@northex.ca", out.Contacts[0].Email, "counterparty contact email should be set")
		require.Equal("Compliance", out.Contacts[0].Role, "counterparty contact role should be set")
		require.Equal("northex.ca", out.CommonName, "counterparty common name should be set")
		require.Equal("mailto:compliance@northex.ca", out.Endpoint, "counterparty endpoint should be set")
		require.NotZero(out.Created, "counterparty created timestamp should be set")
		require.NotZero(out.Modified, "counterparty modified timestamp should be set")
	})

	w.Run("TRP", func() {
		w.store.OnCreateCounterparty = onCreateCounterparty

		in := &api.Counterparty{
			Protocol:      "trp",
			Name:          "NorthEx CoinBank",
			Country:       "CA",
			TravelAddress: "ta5afuBEVxJHJ2QvzbQC6ezGH2M",
		}

		out, err := client.CreateCounterparty(ctx, in)
		require.NoError(err, "could not create counterparty")
		require.NotNil(out, "counterparty should not be nil")

		require.NotZero(out.ID, "counterparty ID should be created")
		require.Equal(enum.SourceUserEntry.String(), out.Source, "counterparty source should be user entry")
		require.Equal("NorthEx CoinBank", out.Name, "counterparty name should be set")
		require.Equal("CA", out.Country, "counterparty country should be set")
		require.Equal(0, len(out.Contacts), "counterparty should have one contact")
		require.Equal("northex.ca", out.CommonName, "counterparty common name should be set")
		require.Equal("https://northex.ca?t=i", out.Endpoint, "counterparty endpoint should be set")
		require.NotZero(out.Created, "counterparty created timestamp should be set")
		require.NotZero(out.Modified, "counterparty modified timestamp should be set")
	})

	w.Run("Endpoint", func() {
		w.store.OnCreateCounterparty = onCreateCounterparty

		in := &api.Counterparty{
			Protocol:   "trp",
			Name:       "NorthEx CoinBank",
			Country:    "CA",
			CommonName: "northex.ca",
			Endpoint:   "https://northex.ca?t=i",
		}

		out, err := client.CreateCounterparty(ctx, in)
		require.NoError(err, "could not create counterparty")
		require.NotNil(out, "counterparty should not be nil")

		require.NotZero(out.ID, "counterparty ID should be created")
		require.Equal(enum.SourceUserEntry.String(), out.Source, "counterparty source should be user entry")
		require.Equal("NorthEx CoinBank", out.Name, "counterparty name should be set")
		require.Equal("CA", out.Country, "counterparty country should be set")
		require.Equal(0, len(out.Contacts), "counterparty should have one contact")
		require.Equal("northex.ca", out.CommonName, "counterparty common name should be set")
		require.Equal("https://northex.ca?t=i", out.Endpoint, "counterparty endpoint should be set")
		require.NotZero(out.Created, "counterparty created timestamp should be set")
		require.NotZero(out.Modified, "counterparty modified timestamp should be set")
	})

	w.Run("SourceDaybreak", func() {
		w.store.OnCreateCounterparty = onCreateCounterparty

		in := &api.Counterparty{
			Source:   "daybreak",
			Protocol: "sunrise",
			Name:     "NorthEx CoinBank",
			Country:  "CA",
			Contacts: []*api.Contact{
				{
					Name:  "Compliance Officer",
					Email: "compliance@northex.ca",
					Role:  "Compliance",
				},
			},
		}

		out, err := client.CreateCounterparty(ctx, in)
		require.EqualError(err, "[422] read-only field source: this field cannot be written by the user")
		require.Nil(out, "counterparty should not be nil")
	})

	w.Run("SourceGDS", func() {
		w.store.OnCreateCounterparty = onCreateCounterparty

		in := &api.Counterparty{
			Source:     "gds",
			Protocol:   "trisa",
			Name:       "Alice VASP",
			Country:    "GB",
			CommonName: "alice.vasp.com",
			Endpoint:   "alice.vasp.com:443",
		}

		out, err := client.CreateCounterparty(ctx, in)
		require.EqualError(err, "[422] 2 validation errors occurred")
		require.Nil(out, "counterparty should not be nil")
	})

	w.Run("ProtocolTRISA", func() {
		w.store.OnCreateCounterparty = onCreateCounterparty

		in := &api.Counterparty{
			Protocol:   "trisa",
			Name:       "NorthEx CoinBank",
			Country:    "CA",
			CommonName: "trisa.northex.ca",
			Endpoint:   "trisa.northex.ca:443",
		}

		out, err := client.CreateCounterparty(ctx, in)
		require.EqualError(err, "[422] invalid field protocol: cannot edit counterparties with the TRISA protocol")
		require.Nil(out, "counterparty should be nil")
	})

	w.Run("AlreadyExists", func() {
		w.store.OnCreateCounterparty = onCreateCounterparty

		in := &api.Counterparty{
			Protocol: "sunrise",
			Name:     "Example",
			Country:  "IN",
			Contacts: []*api.Contact{
				{
					Name:  "Compliance Officer",
					Email: "compliance@example.com",
					Role:  "Compliance",
				},
			},
		}

		out, err := client.CreateCounterparty(ctx, in)
		require.EqualError(err, "[409] counterparty already exists: example.com")
		require.Nil(out, "counterparty should be nil")
	})
}

var counterparties = map[ulid.ULID]*models.Counterparty{
	ulid.MustParse("01M5JFVEAER0R39G3SZ0KHDR20"): {
		Model: models.Model{
			ID:       ulid.MustParse("01M5JFVEAER0R39G3SZ0KHDR20"),
			Created:  time.Date(2026, 10, 22, 19, 01, 52, 78, time.UTC),
			Modified: time.Date(2026, 10, 22, 19, 01, 52, 78, time.UTC),
		},
		Source:     enum.SourceUserEntry,
		Protocol:   enum.ProtocolSunrise,
		CommonName: "northex.ca",
		Endpoint:   "mailto:compliance@northex.ca",
		Name:       "NorthEx CoinBank",
		Country:    sql.NullString{Valid: true, String: "CA"},
	},
	ulid.MustParse("01M5JFVQYXMJTMVND5VTDDWBGK"): {
		Model: models.Model{
			ID:       ulid.MustParse("01M5JFVQYXMJTMVND5VTDDWBGK"),
			Created:  time.Date(2026, 10, 22, 19, 2, 1, 978282, time.UTC),
			Modified: time.Date(2026, 10, 22, 19, 2, 1, 978282, time.UTC),
		},
		Source:     enum.SourceUserEntry,
		Protocol:   enum.ProtocolTRP,
		CommonName: "unchanged.in",
		Endpoint:   "https://unchanged.in?t=i",
		Name:       "Unchanged India",
		Country:    sql.NullString{Valid: true, String: "IN"},
	},
	ulid.MustParse("01M5JFW33AE7J8ZF8S76F8FK11"): {
		Model: models.Model{
			ID:       ulid.MustParse("01M5JFW33AE7J8ZF8S76F8FK11"),
			Created:  time.Date(2026, 10, 22, 19, 2, 13, 35421, time.UTC),
			Modified: time.Date(2026, 10, 22, 19, 2, 13, 35421, time.UTC),
		},
		Source:     enum.SourceDaybreak,
		Protocol:   enum.ProtocolSunrise,
		CommonName: "rotational.io",
		Endpoint:   "mailto:compliance@rotational.io",
		Name:       "Rotational Labs, Inc.",
		Country:    sql.NullString{Valid: true, String: "US"},
	},
	ulid.MustParse("01M5JFWB6G96A8B8R5KCCN6EMZ"): {
		Model: models.Model{
			ID:       ulid.MustParse("01M5JFWB6G96A8B8R5KCCN6EMZ"),
			Created:  time.Date(2026, 10, 22, 19, 1, 21, 648123, time.UTC),
			Modified: time.Date(2026, 10, 22, 19, 1, 21, 648123, time.UTC),
		},
		Source:     enum.SourceDirectorySync,
		Protocol:   enum.ProtocolTRISA,
		CommonName: "trisa.debitcard.com",
		Endpoint:   "trisa.debitcard.com:443",
		Name:       "DebitCard, Inc.",
		Country:    sql.NullString{Valid: true, String: "US"},
	},
}

func onRetrieveCounterparty(ctx context.Context, id ulid.ULID) (counterparty *models.Counterparty, err error) {
	counterparty, ok := counterparties[id]
	if !ok {
		return nil, dberr.ErrNotFound
	}

	return counterparty, nil
}

func onUpdateCounterparty(ctx context.Context, in *models.Counterparty, log *models.ComplianceAuditLog) error {
	if in.ID.IsZero() {
		return dberr.ErrMissingID
	}

	in.Modified = time.Now()

	return nil
}

func (w *webTestSuite) TestUpdateCounterparties() {
	require := w.Require()
	client := w.ClientWithPermissions([]string{"counterparties:view", "counterparties:manage"})

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	w.Run("Sunrise", func() {
		w.store.OnRetrieveCounterparty = onRetrieveCounterparty
		w.store.OnUpdateCounterparty = onUpdateCounterparty

		in := &api.Counterparty{
			ID:         ulid.MustParse("01M5JFVEAER0R39G3SZ0KHDR20"),
			Protocol:   "sunrise",
			Name:       "NorthEx CoinBank, Ltd.",
			Website:    "https://northex.ca",
			CommonName: "northex.ca",
			Endpoint:   "mailto:compliance@northex.ca",
			Country:    "CA",
		}

		out, err := client.UpdateCounterparty(ctx, in)
		require.NoError(err, "could not update counterparty")
		require.NotNil(out, "counterparty should not be nil")

		require.NotZero(out.ID, "counterparty ID should be set")
		require.Equal(enum.SourceUserEntry.String(), out.Source, "counterparty source should be user entry")
		require.Equal("NorthEx CoinBank, Ltd.", out.Name, "counterparty name should be set")
		require.Equal("https://northex.ca", out.Website, "counterparty website should be set")
		require.Equal("northex.ca", out.CommonName, "counterparty common name should be set")
		require.Equal("mailto:compliance@northex.ca", out.Endpoint, "counterparty endpoint should be set")
		require.Equal("CA", out.Country, "counterparty country should be set")
	})

	w.Run("TRP", func() {
		w.store.OnRetrieveCounterparty = onRetrieveCounterparty
		w.store.OnUpdateCounterparty = onUpdateCounterparty

		in := &api.Counterparty{
			ID:         ulid.MustParse("01M5JFVQYXMJTMVND5VTDDWBGK"),
			Protocol:   "trp",
			Name:       "Unchanged India, Ltd.",
			Website:    "https://unchanged.in",
			CommonName: "unchanged.in",
			Endpoint:   "https://unchanged.in?t=i",
			Country:    "IN",
		}

		out, err := client.UpdateCounterparty(ctx, in)
		require.NoError(err, "could not update counterparty")
		require.NotNil(out, "counterparty should not be nil")

		require.NotZero(out.ID, "counterparty ID should be set")
		require.Equal(enum.SourceUserEntry.String(), out.Source, "counterparty source should be user entry")
		require.Equal("Unchanged India, Ltd.", out.Name, "counterparty name should be set")
		require.Equal("https://unchanged.in", out.Website, "counterparty website should be set")
		require.Equal("unchanged.in", out.CommonName, "counterparty common name should be set")
		require.Equal("https://unchanged.in?t=i", out.Endpoint, "counterparty endpoint should be set")
		require.Equal("IN", out.Country, "counterparty country should be set")
	})

	w.Run("Daybreak", func() {
		w.store.OnRetrieveCounterparty = onRetrieveCounterparty
		w.store.OnUpdateCounterparty = onUpdateCounterparty

		in := &api.Counterparty{
			ID:         ulid.MustParse("01M5JFW33AE7J8ZF8S76F8FK11"),
			Protocol:   "sunrise",
			Name:       "Rotational Labs, Inc.",
			Website:    "https://rotational.io",
			CommonName: "rotational.io",
			Endpoint:   "mailto:compliance@rotational.io",
			Country:    "US",
		}

		out, err := client.UpdateCounterparty(ctx, in)
		require.EqualError(err, "[409] only user created records can be edited")
		require.Nil(out, "counterparty should be nil")
	})

	w.Run("GDS", func() {
		w.store.OnRetrieveCounterparty = onRetrieveCounterparty
		w.store.OnUpdateCounterparty = onUpdateCounterparty

		in := &api.Counterparty{
			ID:         ulid.MustParse("01M5JFWB6G96A8B8R5KCCN6EMZ"),
			Protocol:   "trp",
			Name:       "DebitCard, Inc.",
			Website:    "https://debitcard.com",
			CommonName: "debitcard.com",
			Endpoint:   "https://debitcard.com:443",
			Country:    "US",
		}

		out, err := client.UpdateCounterparty(ctx, in)
		require.EqualError(err, "[409] only user created records can be edited")
		require.Nil(out, "counterparty should be nil")
	})
}

func onDeleteCounterparty(ctx context.Context, id ulid.ULID, _ *models.ComplianceAuditLog) error {
	_, ok := counterparties[id]
	if !ok {
		return dberr.ErrNotFound
	}
	return nil
}

func (w *webTestSuite) TestDeleteCounterparty() {
	require := w.Require()
	client := w.ClientWithPermissions([]string{"counterparties:view", "counterparties:manage"})

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	w.Run("Sunrise", func() {
		w.store.OnRetrieveCounterparty = onRetrieveCounterparty
		w.store.OnDeleteCounterparty = onDeleteCounterparty

		err := client.DeleteCounterparty(ctx, ulid.MustParse("01M5JFVEAER0R39G3SZ0KHDR20"))
		require.NoError(err, "could not delete counterparty")
	})

	w.Run("TRP", func() {
		w.store.OnRetrieveCounterparty = onRetrieveCounterparty
		w.store.OnDeleteCounterparty = onDeleteCounterparty

		err := client.DeleteCounterparty(ctx, ulid.MustParse("01M5JFVQYXMJTMVND5VTDDWBGK"))
		require.NoError(err, "could not delete counterparty")
	})

	w.Run("Daybreak", func() {
		w.store.OnRetrieveCounterparty = onRetrieveCounterparty
		w.store.OnDeleteCounterparty = onDeleteCounterparty

		err := client.DeleteCounterparty(ctx, ulid.MustParse("01M5JFW33AE7J8ZF8S76F8FK11"))
		require.EqualError(err, "[403] this record cannot be edited")
	})

	w.Run("GDS", func() {
		w.store.OnRetrieveCounterparty = onRetrieveCounterparty
		w.store.OnDeleteCounterparty = onDeleteCounterparty

		err := client.DeleteCounterparty(ctx, ulid.MustParse("01M5JFWB6G96A8B8R5KCCN6EMZ"))
		require.EqualError(err, "[403] this record cannot be edited")
	})

	w.Run("NotFound", func() {
		w.store.OnRetrieveCounterparty = onRetrieveCounterparty
		w.store.OnDeleteCounterparty = onDeleteCounterparty

		err := client.DeleteCounterparty(ctx, ulid.MustParse("01M5JH36M7EWMCBJNRWJM9NSM9"))
		require.EqualError(err, "[404] counterparty not found")
	})
}
