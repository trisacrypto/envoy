package api

import (
	"strings"

	"github.com/trisacrypto/envoy/pkg/store/models"
	"github.com/trisacrypto/trisa/pkg/ivms101"
	trisa "github.com/trisacrypto/trisa/pkg/trisa/api/v1beta1"
	generic "github.com/trisacrypto/trisa/pkg/trisa/data/generic/v1beta1"
)

type Sunrise struct {
	Email        string    `json:"email"`
	Counterparty string    `json:"counterparty"`
	Originator   *Person   `json:"originator"`
	Beneficiary  *Person   `json:"beneficiary"`
	Transfer     *Transfer `json:"transfer"`
}

func (s *Sunrise) Validate() (err error) {
	if strings.TrimSpace(s.Email) == "" {
		err = ValidationError(err, MissingField("email"))
	}

	if strings.TrimSpace(s.Counterparty) == "" {
		err = ValidationError(err, MissingField("counterparty"))
	}

	if s.Originator == nil {
		err = ValidationError(err, MissingField("originator"))
	} else {
		if s.Originator.CryptoAddress == "" {
			err = ValidationError(err, MissingField("originator.crypto_address"))
		}

		if s.Originator.Identification == nil {
			s.Originator.Identification = &Identification{}
		}
	}

	if s.Beneficiary == nil {
		err = ValidationError(err, MissingField("beneficiary"))
	} else {
		if s.Beneficiary.CryptoAddress == "" {
			err = ValidationError(err, MissingField("beneficiary.crypto_address"))
		}

		if s.Beneficiary.Identification == nil {
			s.Beneficiary.Identification = &Identification{}
		}
	}

	if s.Transfer == nil {
		err = ValidationError(err, MissingField("transfer"))
	}

	return err
}

func (in *Sunrise) Payload(originatorVASP, beneficiaryVASP *models.Counterparty) (*trisa.Payload, error) {
	prepared := &Prepared{
		Identity: &ivms101.IdentityPayload{
			Originator: &ivms101.Originator{
				OriginatorPersons: []*ivms101.Person{
					in.Originator.NaturalPerson(),
				},
				AccountNumbers: []string{
					in.Originator.CryptoAddress,
				},
			},
			Beneficiary: &ivms101.Beneficiary{
				BeneficiaryPersons: []*ivms101.Person{
					in.Beneficiary.NaturalPerson(),
				},
				AccountNumbers: []string{
					in.Beneficiary.CryptoAddress,
				},
			},
			OriginatingVasp: &ivms101.OriginatingVasp{
				OriginatingVasp: &ivms101.Person{
					Person: &ivms101.Person_LegalPerson{
						LegalPerson: originatorVASP.IVMSRecord,
					},
				},
			},
			BeneficiaryVasp: &ivms101.BeneficiaryVasp{
				BeneficiaryVasp: &ivms101.Person{
					Person: &ivms101.Person_LegalPerson{
						LegalPerson: beneficiaryVASP.IVMSRecord,
					},
				},
			},
			TransferPath:    nil,
			PayloadMetadata: nil,
		},
		Transaction: in.Transaction(),
	}

	return prepared.Payload()
}

func (s *Sunrise) Transaction() *generic.Transaction {
	return &generic.Transaction{
		Txid:        s.Transfer.TxID,
		Originator:  s.Originator.CryptoAddress,
		Beneficiary: s.Beneficiary.CryptoAddress,
		Amount:      s.Transfer.Amount,
		Network:     s.Transfer.Network,
		AssetType:   s.Transfer.AssetType,
		Tag:         s.Transfer.Tag,
		Timestamp:   "",
		ExtraJson:   "",
	}
}
