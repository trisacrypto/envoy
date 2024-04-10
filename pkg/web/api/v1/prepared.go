package api

import (
	"fmt"
	"slices"
	"strings"

	"github.com/trisacrypto/trisa/pkg/ivms101"
	generic "github.com/trisacrypto/trisa/pkg/trisa/data/generic/v1beta1"
)

type Prepare struct {
	TravelAddress string    `json:"travel_address"`
	Originator    *Person   `json:"originator"`
	Beneficiary   *Person   `json:"beneficiary"`
	Transfer      *Transfer `json:"transfer"`
}

type Prepared struct {
	TravelAddress string                   `json:"travel_address"`
	Identity      *ivms101.IdentityPayload `json:"identity"`
	Transaction   *generic.Transaction     `json:"transaction"`
}

type Person struct {
	FirstName     string `json:"first_name"`
	LastName      string `json:"last_name"`
	CustomerID    string `json:"customer_id"`
	AddrLine1     string `json:"addr_line_1"`
	AddrLine2     string `json:"addr_line_2"`
	City          string `json:"city"`
	State         string `json:"state"`
	Country       string `json:"country"`
	CryptoAddress string `json:"crypto_address"`
}

type Transfer struct {
	Amount    float64 `json:"amount"`
	Network   string  `json:"network"`
	AssetType string  `json:"asset_type"`
	TxID      string  `json:"transaction_id"`
	Tag       string  `json:"tag"`
}

func (p *Prepare) Validate() error {
	if strings.TrimSpace(p.TravelAddress) == "" {
		return MissingField("travel_address")
	}

	return nil
}

func (p *Prepared) Validate() error {
	return nil
}

func (p *Prepare) Transaction() *generic.Transaction {
	return &generic.Transaction{
		Txid:        p.Transfer.TxID,
		Originator:  p.Originator.CryptoAddress,
		Beneficiary: p.Beneficiary.CryptoAddress,
		Amount:      p.Transfer.Amount,
		Network:     p.Transfer.Network,
		AssetType:   p.Transfer.AssetType,
		Tag:         p.Transfer.Tag,
		Timestamp:   "",
		ExtraJson:   "",
	}
}

func (p *Person) NaturalPerson() *ivms101.Person {
	addrLines := []string{
		strings.TrimSpace(p.AddrLine1),
		strings.TrimSpace(p.AddrLine2),
		strings.TrimSpace(fmt.Sprintf("%s, %s", p.City, p.State)),
	}

	for i, line := range addrLines {
		if line == "" || line == "," {
			addrLines = slices.Delete(addrLines, i, i)
		}
	}

	// NOTE: the country of the address is assigned country of residence
	return &ivms101.Person{
		Person: &ivms101.Person_NaturalPerson{
			NaturalPerson: &ivms101.NaturalPerson{
				Name: &ivms101.NaturalPersonName{
					NameIdentifiers: []*ivms101.NaturalPersonNameId{
						{
							PrimaryIdentifier:   p.LastName,
							SecondaryIdentifier: p.FirstName,
							NameIdentifierType:  ivms101.NaturalPersonLegal,
						},
					},
					LocalNameIdentifiers:    nil,
					PhoneticNameIdentifiers: nil,
				},
				GeographicAddresses: []*ivms101.Address{
					{
						AddressType: ivms101.AddressTypeHome,
						AddressLine: addrLines,
						Country:     p.Country,
					},
				},
				NationalIdentification: nil,
				CustomerIdentification: p.CustomerID,
				DateAndPlaceOfBirth:    nil,
				CountryOfResidence:     p.Country,
			},
		},
	}
}
