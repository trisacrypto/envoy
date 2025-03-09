package models

import "go.rtnl.ai/ulid"

const DefaultPageSize = uint32(50)

type PageInfo struct {
	PageSize   uint32    `json:"page_size"`
	NextPageID ulid.ULID `json:"next_page_id"`
	PrevPageID ulid.ULID `json:"prev_page_id"`
}

type TransactionPage struct {
	Transactions []*Transaction       `json:"transactions"`
	Page         *TransactionPageInfo `json:"page"`
}

type SecureEnvelopePage struct {
	Envelopes []*SecureEnvelope `json:"envelopes"`
	Page      *PageInfo         `json:"page"`
}

type AccountsPage struct {
	Accounts []*Account `json:"accounts"`
	Page     *PageInfo  `json:"page"`
}

type CryptoAddressPage struct {
	CryptoAddresses []*CryptoAddress `json:"crypto_addresses"`
	Page            *PageInfo        `json:"page"`
}

type CounterpartyPage struct {
	Counterparties []*Counterparty       `json:"counterparties"`
	Page           *CounterpartyPageInfo `json:"page"`
}

type ContactsPage struct {
	Contacts []*Contact `json:"contacts"`
	Page     *PageInfo  `json:"page"`
}

type UserPage struct {
	Users []*User       `json:"users"`
	Page  *UserPageInfo `json:"page"`
}

type APIKeyPage struct {
	APIKeys []*APIKey `json:"api_keys"`
	Page    *PageInfo `json:"page"`
}

type SunrisePage struct {
	Messages []*Sunrise `json:"messages"`
	Page     *PageInfo  `json:"page"`
}

func PageInfoFrom(in *PageInfo) (out *PageInfo) {
	out = &PageInfo{
		PageSize: DefaultPageSize,
	}
	if in != nil && in.PageSize > 0 {
		out.PageSize = in.PageSize
	}
	return out
}
