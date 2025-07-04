package scene

import (
	"github.com/trisacrypto/envoy/pkg/store/models"
	"github.com/trisacrypto/envoy/pkg/web/api/v1"
)

type LocalpartyScene struct {
	Localparty *api.Counterparty
	Error      error
}

// WithLocalparty returns a new scene with the local party information
func (s Scene) WithLocalparty(localparty *models.Counterparty, err error) Scene {
	data := &LocalpartyScene{
		Error: err,
	}

	if localparty != nil && err == nil {
		data.Localparty, data.Error = api.NewCounterparty(localparty, &api.EncodingQuery{})
	}

	s["Localparty"] = data
	return s
}

func (s Scene) Localparty() *LocalpartyScene {
	if lp, ok := s["Localparty"]; ok {
		if localparty, ok := lp.(*LocalpartyScene); ok {
			return localparty
		}
	}
	return nil
}

func (s *LocalpartyScene) Company() Company {
	if vasp, err := s.Localparty.IVMS101(); err == nil {
		return makeCompany(vasp)
	}
	return Company{}
}
