package models

import "github.com/oklog/ulid/v2"

type PageInfo struct {
	PageSize   uint32    `json:"page_size"`
	NextPageID ulid.ULID `json:"next_page_id"`
	PrevPageID ulid.ULID `json:"prev_page_id"`
}

type AccountsPage struct {
	Accounts []*Account `json:"accounts"`
	Page     *PageInfo  `json:"page"`
}

type CryptoAddressPage struct {
	CryptoAddresses []*CryptoAddress `json:"crypto_addresses"`
	Page            *PageInfo        `json:"page"`
}
