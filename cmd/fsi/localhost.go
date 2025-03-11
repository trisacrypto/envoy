package main

import (
	"context"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/trisacrypto/envoy/pkg/web/api/v1"
	pb "github.com/trisacrypto/trisa/pkg/trisa/gds/models/v1beta1"
)

func envoyVASP() (vasp *pb.VASP, err error) {
	vasp = new(pb.VASP)
	if err = unmarshalPBFixture("localhost/envoy.pb.json", vasp); err != nil {
		return nil, err
	}
	return vasp, nil
}

func counterpartyVASP() (vasp *pb.VASP, err error) {
	vasp = new(pb.VASP)
	if err = unmarshalPBFixture("localhost/counterparty.pb.json", vasp); err != nil {
		return nil, err
	}
	return vasp, nil
}

func createContacts(conn api.Client, country string) (err error) {
	loadPeople()
	loadAddresses()

	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	// Create an account
	paths, ok := people[strings.ToUpper(country)]
	if !ok || len(paths) == 0 {
		return fmt.Errorf("no people for country %s", country)
	}

	naccount, naddresses := 0, 0
	for _, path := range paths {
		key := strings.TrimSuffix(filepath.Base(path), ".json")

		person := &api.Person{}
		if err = unmarshalJSONFixture(path, person); err != nil {
			return fmt.Errorf("cannot unmarshal person from %s: %w", path, err)
		}

		account := &api.Account{
			CustomerID:      person.CustomerID,
			FirstName:       person.Forename,
			LastName:        person.Surname,
			CryptoAddresses: make([]*api.CryptoAddress, 0, 2),
		}

		encoding := &api.EncodingQuery{
			Encoding: "base64",
			Format:   "pb",
		}
		if err = encoding.Validate(); err != nil {
			return err
		}

		record := person.NaturalPerson()
		if record != nil {
			if account.IVMSRecord, err = encoding.Marshal(record); err != nil {
				return fmt.Errorf("cannot marshal IVMS record for %s: %w", key, err)
			}
		}

		// Load the crypto addresses into that account
		for network, address := range addresses {
			if cryptoAddress, ok := address[key]; ok {
				account.CryptoAddresses = append(account.CryptoAddresses, &api.CryptoAddress{
					CryptoAddress: cryptoAddress,
					Network:       network,
				})
			}
		}

		// Create the account
		if _, err = conn.CreateAccount(ctx, account); err != nil {
			if serr, ok := err.(*api.StatusError); ok {
				if serr.StatusCode == http.StatusConflict || serr.StatusCode == http.StatusUnprocessableEntity {
					continue
				}
			}
			return fmt.Errorf("cannot create account for %s: %w", key, err)
		}

		naccount++
		naddresses += len(account.CryptoAddresses)
	}

	fmt.Printf("created %d accounts and %d crypto addresses\n", naccount, naddresses)
	return nil
}
