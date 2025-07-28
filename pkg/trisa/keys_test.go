package trisa_test

import (
	"context"
	"crypto/md5"
	"encoding/base64"
	"os"

	"github.com/trisacrypto/trisa/pkg/trisa/keys"
)

func (s *trisaTestSuite) TestKeyExchange() {
	require := s.Require()

	// Create a public key to exchange with the TRISA node
	outgoing, err := loadSealingKeys("testdata/certs/client.trisatest.dev.pem")
	require.NoError(err, "could not load client.trisatest.dev.pem fixture")

	req, err := outgoing.Proto()
	require.NoError(err, "could not create outgoing signing key protocol buffer")

	// Execute key exchange request
	rep, err := s.client.KeyExchange(context.Background(), req)
	require.NoError(err, "Could not make key exchange request")

	// Ensure default keys were returned
	require.Equal(int64(3), rep.Version)
	require.Equal("SHA256-RSA", rep.SignatureAlgorithm)
	require.Equal("RSA", rep.PublicKeyAlgorithm)
	require.Equal("2025-05-03T15:26:27Z", rep.NotBefore)
	require.Equal("2055-04-26T15:26:27Z", rep.NotAfter)
	require.False(rep.Revoked)

	checksum := md5.Sum(rep.Signature)
	require.Equal("G3u+xTl6HMaQdP8tUmNb0g==", base64.StdEncoding.EncodeToString(checksum[:]))

	checksum = md5.Sum(rep.Data)
	require.Equal("fQ0GhWCvdoigJfc5EIT+uw==", base64.StdEncoding.EncodeToString(checksum[:]))

	// Check to ensure that the key was cached
	chain, err := s.network.KeyChain()
	require.NoError(err, "could not get keychain")

	seal, err := chain.SealingKey("client.trisatest.dev")
	require.NoError(err, "could not get keys that were just exchanged")
	sks, err := seal.PublicKeySignature()
	require.NoError(err, "could not get cached key signature")

	oks, err := outgoing.PublicKeySignature()
	require.NoError(err, "could not get outgoing key signature")

	require.Equal(oks, sks, "cached exchange keys do not match keys sent in RPC")
}

func loadSealingKeys(path string) (_ keys.Key, err error) {
	var data []byte
	if data, err = os.ReadFile(path); err != nil {
		return nil, err
	}

	key := &keys.Certificate{}
	if err = key.Unmarshal(data); err != nil {
		return nil, err
	}

	return key, nil
}
