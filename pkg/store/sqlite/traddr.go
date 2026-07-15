package sqlite

import (
	"context"
	"database/sql"

	"github.com/rs/zerolog/log"
	"github.com/trisacrypto/envoy/pkg/store/errors"
	"github.com/trisacrypto/envoy/pkg/store/models"
)

const (
	selectAccountTravelAddressSQL       = "SELECT id, travel_address FROM accounts"
	selectCryptoAddressTravelAddressSQL = "SELECT id, travel_address FROM crypto_addresses"
	updateAccountTravelAddressSQL       = "UPDATE accounts SET travel_address = :travel_address WHERE id = :id"
	updateCryptoAddressTravelAddressSQL = "UPDATE crypto_addresses SET travel_address = :travel_address WHERE id = :id"
)

func (s *Store) RegenerateTravelAddresses(ctx context.Context) (err error) {
	if s.readonly {
		return errors.ErrReadOnly
	}

	if s.mkta == nil {
		return errors.ErrMissingTravelAddressFactory
	}

	var tx *sql.Tx
	if tx, err = s.conn.BeginTx(ctx, &sql.TxOptions{ReadOnly: false}); err != nil {
		return err
	}
	defer tx.Rollback()

	if err = regenerateAccountTravelAddresses(tx, s.mkta); err != nil {
		return err
	}

	if err = regenerateCryptoAddressTravelAddresses(tx, s.mkta); err != nil {
		return err
	}

	if err = tx.Commit(); err != nil {
		return err
	}
	return nil
}

func regenerateAccountTravelAddresses(tx *sql.Tx, mkta models.TravelAddressFactory) (err error) {
	var rows *sql.Rows
	if rows, err = tx.Query(selectAccountTravelAddressSQL); err != nil {
		return dbe(err)
	}
	defer rows.Close()

	for rows.Next() {
		account := &models.Account{}
		if err = rows.Scan(&account.ID, &account.TravelAddress); err != nil {
			return dbe(err)
		}

		var travelAddress string
		if travelAddress, err = mkta(account); err != nil {
			return err
		}

		// Do nothing if the travel address is the same as the current one.
		if account.TravelAddress.Valid && account.TravelAddress.String == travelAddress {
			log.Debug().Str("type", "account").Str("id", account.ID.String()).Str("travel_address", travelAddress).Msg("travel address unchanged, skipping update")
			continue
		}

		log.Info().Str("type", "account").Str("id", account.ID.String()).Str("original", account.TravelAddress.String).Str("updated", travelAddress).Msg("updating travel address")
		account.TravelAddress = sql.NullString{Valid: travelAddress != "", String: travelAddress}
		if _, err = tx.Exec(updateAccountTravelAddressSQL, sql.Named("id", account.ID), sql.Named("travel_address", account.TravelAddress)); err != nil {
			return dbe(err)
		}
	}

	return dbe(rows.Err())
}

func regenerateCryptoAddressTravelAddresses(tx *sql.Tx, mkta models.TravelAddressFactory) (err error) {
	var rows *sql.Rows
	if rows, err = tx.Query(selectCryptoAddressTravelAddressSQL); err != nil {
		return dbe(err)
	}
	defer rows.Close()

	for rows.Next() {
		cryptoAddress := &models.CryptoAddress{}
		if err = rows.Scan(&cryptoAddress.ID, &cryptoAddress.TravelAddress); err != nil {
			return dbe(err)
		}

		var travelAddress string
		if travelAddress, err = mkta(cryptoAddress); err != nil {
			return err
		}

		// Do nothing if the travel address is the same as the current one.
		if cryptoAddress.TravelAddress.Valid && cryptoAddress.TravelAddress.String == travelAddress {
			log.Debug().Str("type", "crypto_address").Str("id", cryptoAddress.ID.String()).Str("travel_address", travelAddress).Msg("travel address unchanged, skipping update")
			continue
		}

		log.Info().Str("type", "crypto_address").Str("id", cryptoAddress.ID.String()).Str("original", cryptoAddress.TravelAddress.String).Str("updated", travelAddress).Msg("updating travel address")
		cryptoAddress.TravelAddress = sql.NullString{Valid: travelAddress != "", String: travelAddress}
		if _, err = tx.Exec(updateCryptoAddressTravelAddressSQL, sql.Named("id", cryptoAddress.ID), sql.Named("travel_address", cryptoAddress.TravelAddress)); err != nil {
			return dbe(err)
		}
	}

	return dbe(rows.Err())
}

const (
	countAccountsSQL        = "SELECT COUNT(id) FROM accounts"
	countCryptoAddressesSQL = "SELECT COUNT(id) FROM crypto_addresses"
)

func (s *Store) CountTravelAddresses(ctx context.Context) (_ int64, err error) {
	var tx *sql.Tx
	if tx, err = s.conn.BeginTx(ctx, &sql.TxOptions{ReadOnly: true}); err != nil {
		return 0, err
	}
	defer tx.Rollback()

	var numAccounts int64
	if err = tx.QueryRowContext(ctx, countAccountsSQL).Scan(&numAccounts); err != nil {
		return 0, err
	}

	var numCryptoAddresses int64
	if err = tx.QueryRowContext(ctx, countCryptoAddressesSQL).Scan(&numCryptoAddresses); err != nil {
		return 0, err
	}

	return numAccounts + numCryptoAddresses, nil
}
