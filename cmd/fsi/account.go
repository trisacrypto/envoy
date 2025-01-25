package main

import (
	crand "crypto/rand"
	"encoding/hex"
	"fmt"
	"io/fs"
	"math"
	"math/rand"
	"path/filepath"
	"strings"
	"sync"

	"github.com/trisacrypto/envoy/pkg/web/api/v1"
)

var (
	precision = math.Pow10(8)
	networks  = []string{"BTC", "ETH"}

	people    map[string][]string
	addresses map[string]map[string]string

	peoplemu sync.Once
	cryptomu sync.Once
)

const (
	valueBTC   = 105137.20
	valueETH   = 3312.20
	maxdollars = 1000000
)

func makePrepare(travelAddress string) *api.Prepare {
	network := networks[rand.Intn(len(networks))]
	return &api.Prepare{
		TravelAddress: travelAddress,
		Originator:    makePerson("US", network),
		Beneficiary:   makePerson("DE", network),
		Transfer:      makeTransfer(network),
	}
}

func makePerson(country, network string) *api.Person {
	loadPeople()
	loadAddresses()

	paths, ok := people[strings.ToUpper(country)]
	if !ok || len(paths) == 0 {
		panic(fmt.Errorf("no people for country %s", country))
	}

	path := paths[rand.Intn(len(paths))]
	key := strings.TrimSuffix(filepath.Base(path), ".json")

	person := &api.Person{}
	if err := unmarshalJSONFixture(path, person); err != nil {
		panic(fmt.Errorf("cannot load person: %w", err))
	}

	person.CryptoAddress = addresses[network][key]
	if person.CryptoAddress == "" {
		panic(fmt.Errorf("no crypto address for %q in %s", key, network))
	}

	return person
}

func makeTransfer(network string) *api.Transfer {
	if network == "" {
		network = networks[rand.Intn(len(networks))]
	}

	transfer := &api.Transfer{
		Network: network,
	}

	switch transfer.Network {
	case "BTC":
		transfer.Amount = randomBTC()
	case "ETH":
		transfer.Amount = randomETH()
	default:
		panic(fmt.Errorf("unknown network %s", transfer.Network))
	}

	return transfer
}

func randomBTC() float64 {
	value := randDollars() / valueBTC
	return math.Ceil(value*precision) / precision
}

func randomETH() float64 {
	value := randDollars() / valueETH
	return math.Ceil(value*precision) / precision
}

func randDollars() float64 {
	dollars := float64(rand.Intn(maxdollars) + 1)
	cents := float64(rand.Intn(100)) / 100.0
	if cents < 1 {
		return dollars + cents
	}
	return dollars
}

func randDoB() string {
	year := rand.Intn(100) + 1910
	month := rand.Intn(12) + 1
	day := rand.Intn(28) + 1
	return fmt.Sprintf("%04d-%02d-%02d", year, month, day)
}

func randTxID() string {
	data := make([]byte, 32)
	crand.Read(data)
	return hex.EncodeToString(data)
}

func loadPeople() {
	peoplemu.Do(func() {
		people = make(map[string][]string)
		err := fs.WalkDir(fixtures, "fixtures/persons", func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}

			if d.IsDir() {
				people[strings.ToUpper(d.Name())] = make([]string, 0)
				return nil
			}

			if strings.HasSuffix(path, ".json") {
				key := strings.ToUpper(filepath.Base(filepath.Dir(path)))
				people[key] = append(people[key], path)
			}

			return nil
		})

		if err != nil {
			panic(fmt.Errorf("could not walk fixtures: %w", err))
		}
	})
}

func loadAddresses() {
	cryptomu.Do(func() {
		addresses = make(map[string]map[string]string)

		btc := make(map[string]string)
		if err := unmarshalJSONFixture("crypto_addresses/btc.json", &btc); err != nil {
			panic(fmt.Errorf("could not load btc addresses: %w", err))
		}
		addresses["BTC"] = btc

		eth := make(map[string]string)
		if err := unmarshalJSONFixture("crypto_addresses/eth.json", &eth); err != nil {
			panic(fmt.Errorf("could not load eth addresses: %w", err))
		}
		addresses["ETH"] = eth

		if len(btc) == 0 || len(eth) != len(btc) {
			panic("either no crypto addresses or different number of addresses")
		}
	})
}
