package api

import "github.com/google/uuid"

//===========================================================================
// Transaction Resource
//===========================================================================

type Transaction struct {
	ID uuid.UUID `json:"id"`
}

type SecureEnvelope struct {
}

type DecryptedEnvelope struct {
}

type TransactionsList struct {
	Page         *PageQuery     `json:"page"`
	Transactions []*Transaction `json:"transactions"`
}

type SecureEnvelopesList struct {
	Page      *PageQuery        `json:"page"`
	Envelopes []*SecureEnvelope `json:"envelopes"`
}

type DecryptedEnvelopesList struct {
	Page      *PageQuery           `json:"page"`
	Envelopes []*DecryptedEnvelope `json:"envelopes"`
}
