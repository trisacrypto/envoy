package keychain_test

import (
	"os"

	"github.com/trisacrypto/trisa/pkg/trisa/keys"
)

const (
	fixtureLocalKey  = "testdata/local.pem"
	fixtureRemoteKey = "testdata/remote.pem"
)

func loadKeyFixture(path string) (_ keys.Key, err error) {
	var data []byte
	if data, err = os.ReadFile(path); err != nil {
		return nil, err
	}

	certs := &keys.Certificate{}
	if err = certs.Unmarshal(data); err != nil {
		return nil, err
	}
	return certs, nil
}
