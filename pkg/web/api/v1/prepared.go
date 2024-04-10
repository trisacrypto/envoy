package api

import (
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
