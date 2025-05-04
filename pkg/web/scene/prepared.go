package scene

import (
	"encoding/json"

	"github.com/trisacrypto/envoy/pkg/web/api/v1"
	"github.com/trisacrypto/trisa/pkg/ivms101"
	generic "github.com/trisacrypto/trisa/pkg/trisa/data/generic/v1beta1"
)

type Prepared struct {
	Routing     *api.Routing
	identity    *ivms101.IdentityPayload
	transaction *generic.Transaction
}

type Transfer struct {
	Originator   string
	Beneficiary  string
	Amount       float64
	VirtualAsset string
	AssetType    string
	TxID         string
	Tag          string
}

func (s Scene) Prepared() Prepared {
	if prepared, ok := s[APIData].(*api.Prepared); ok {
		return Prepared{
			Routing:     prepared.Routing,
			identity:    prepared.Identity,
			transaction: prepared.Transaction,
		}
	}
	return Prepared{}
}

func (s Prepared) Transfer() Transfer {
	if s.transaction == nil {
		return Transfer{}
	}
	return Transfer{
		Originator:   s.transaction.Originator,
		Beneficiary:  s.transaction.Beneficiary,
		Amount:       s.transaction.Amount,
		VirtualAsset: s.transaction.Network,
		AssetType:    s.transaction.AssetType,
		TxID:         s.transaction.Txid,
		Tag:          s.transaction.Tag,
	}
}

func (s Prepared) Originator() Person {
	if s.identity == nil || s.identity.Originator == nil || len(s.identity.Originator.OriginatorPersons) == 0 {
		return Person{}
	}
	return makePerson(s.identity.Originator.OriginatorPersons[0].GetNaturalPerson())
}

func (s Prepared) Beneficiary() Person {
	if s.identity == nil || s.identity.Beneficiary == nil || len(s.identity.Beneficiary.BeneficiaryPersons) == 0 {
		return Person{}
	}
	return makePerson(s.identity.Beneficiary.BeneficiaryPersons[0].GetNaturalPerson())
}

func (s Prepared) OriginatingVASP() Company {
	if s.identity == nil || s.identity.OriginatingVasp == nil || s.identity.OriginatingVasp.OriginatingVasp == nil {
		return Company{}
	}
	return makeCompany(s.identity.OriginatingVasp.OriginatingVasp.GetLegalPerson())
}

func (s Prepared) BeneficiaryVASP() Company {
	if s.identity == nil || s.identity.BeneficiaryVasp == nil || s.identity.BeneficiaryVasp.BeneficiaryVasp == nil {
		return Company{}
	}
	return makeCompany(s.identity.BeneficiaryVasp.BeneficiaryVasp.GetLegalPerson())
}

func (s Prepared) RoutingJSON() string {
	if s.Routing == nil {
		return ""
	}

	data, _ := json.Marshal(s.Routing)
	return string(data)
}

func (s Prepared) IdentityJSON() string {
	if s.identity == nil {
		return ""
	}

	data, _ := json.Marshal(s.identity)
	return string(data)
}

func (s Prepared) TransactionJSON() string {
	if s.transaction == nil {
		return ""
	}

	data, _ := json.Marshal(s.transaction)
	return string(data)
}
