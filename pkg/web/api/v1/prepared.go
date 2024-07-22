package api

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/trisacrypto/trisa/pkg/ivms101"
	trisa "github.com/trisacrypto/trisa/pkg/trisa/api/v1beta1"
	generic "github.com/trisacrypto/trisa/pkg/trisa/data/generic/v1beta1"
	"google.golang.org/protobuf/types/known/anypb"
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
	FirstName      string          `json:"first_name"`
	LastName       string          `json:"last_name"`
	CustomerID     string          `json:"customer_id"`
	Identification *Identification `json:"identification"`
	AddrLine1      string          `json:"addr_line_1"`
	AddrLine2      string          `json:"addr_line_2"`
	City           string          `json:"city"`
	State          string          `json:"state"`
	PostalCode     string          `json:"post_code"`
	Country        string          `json:"country"`
	CryptoAddress  string          `json:"crypto_address"`
}

type Identification struct {
	TypeCode    string `json:"type_code"`
	Number      string `json:"number"`
	Country     string `json:"country"`
	DateOfBirth string `json:"dob"`
	BirthPlace  string `json:"birth_place"`
}

type Transfer struct {
	Amount    float64 `json:"amount"`
	Network   string  `json:"network"`
	AssetType string  `json:"asset_type"`
	TxID      string  `json:"transaction_id"`
	Tag       string  `json:"tag"`
}

func (p *Prepared) Dump() string {
	data, err := json.Marshal(p)
	if err != nil {
		log.Warn().Err(err).Msg("could not marshal prepared data")
		return ""
	}
	return string(data)
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
		strings.TrimSpace(fmt.Sprintf("%s, %s, %s", p.City, p.State, p.PostalCode)),
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
				NationalIdentification: &ivms101.NationalIdentification{
					NationalIdentifierType: p.Identification.NationalIdentifierTypeCode(),
					NationalIdentifier:     p.Identification.Number,
					CountryOfIssue:         p.Identification.Country,
				},
				CustomerIdentification: p.CustomerID,
				DateAndPlaceOfBirth: &ivms101.DateAndPlaceOfBirth{
					DateOfBirth:  p.Identification.DateOfBirth,
					PlaceOfBirth: p.Identification.BirthPlace,
				},
				CountryOfResidence: p.Country,
			},
		},
	}
}

func (p *Prepared) Payload() (payload *trisa.Payload, err error) {
	payload = &trisa.Payload{}

	if payload.Identity, err = anypb.New(p.Identity); err != nil {
		return nil, err
	}

	if payload.Transaction, err = anypb.New(p.Transaction); err != nil {
		return nil, err
	}

	payload.SentAt = time.Now().UTC().Format(time.RFC3339)
	return payload, nil
}

func (i *Identification) NationalIdentifierTypeCode() ivms101.NationalIdentifierTypeCode {
	i.TypeCode = strings.TrimSpace(i.TypeCode)
	if tc, err := ivms101.ParseNationalIdentifierTypeCode(i.TypeCode); err == nil {
		return tc
	}
	return ivms101.NationalIdentifierMISC
}
