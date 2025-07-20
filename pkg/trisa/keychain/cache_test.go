package keychain_test

import (
	"testing"
	"time"

	"github.com/trisacrypto/envoy/pkg/trisa/keychain"
	"github.com/trisacrypto/envoy/pkg/trisa/keychain/keyerr"
	"github.com/trisacrypto/envoy/pkg/trisa/keychain/memks"

	"github.com/stretchr/testify/require"
)

func TestEmptyCache(t *testing.T) {
	chain, err := keychain.New()
	require.NoError(t, err, "could not create new keychain cache")
	require.IsType(t, &memks.MemStore{}, chain.(*keychain.Cache).InternalStore())
	require.IsType(t, &memks.MemStore{}, chain.(*keychain.Cache).ExternalStore())

	// No default key should return an error
	_, err = chain.SealingKey("bravo.trisa.dev", "")
	require.ErrorIs(t, err, keyerr.KeyNotMatched, "expected empty cache with no default keys to fail on cache miss")

	_, err = chain.UnsealingKey("", "bravo.trisa.dev")
	require.ErrorIs(t, err, keyerr.NoDefaultKeys, "expected empty cache with no default keys to fail on cache miss")

	_, err = chain.ExchangeKey("bravo.trisa.dev")
	require.ErrorIs(t, err, keyerr.NoDefaultKeys, "expected empty cache with no default keys to fail on cache miss")

	// Loading an internal key, without specifying it as a default or associated with common names should act as empty
	internalKey, err := loadKeyFixture(fixtureLocalKey)
	require.NoError(t, err, "could not load internal key fixture")

	err = chain.Store(internalKey, nil)
	require.NoError(t, err, "could not store internal key pair")

	_, err = chain.SealingKey("bravo.trisa.dev", "")
	require.ErrorIs(t, err, keyerr.KeyNotMatched, "expected empty cache with no default keys to fail when not matched to common name")

	_, err = chain.UnsealingKey("", "bravo.trisa.dev")
	require.ErrorIs(t, err, keyerr.NoDefaultKeys, "expected empty cache with no default keys to fail when not matched to common name")

	_, err = chain.ExchangeKey("bravo.trisa.dev")
	require.ErrorIs(t, err, keyerr.NoDefaultKeys, "expected empty cache with no default keys to fail when not matched to common name")

	// Should still be able to lookup internal key by signature
	internalPKS, err := internalKey.PublicKeySignature()
	require.NoError(t, err, "could not compute internal pks")

	keypair, err := chain.UnsealingKey(internalPKS, "")
	require.NoError(t, err, "expected lookup of internal key with signature")
	require.Equal(t, internalKey, keypair)
}

func TestDefaultCache(t *testing.T) {
	// Loading an internal key, specifying it as a default
	internalKey, err := loadKeyFixture(fixtureLocalKey)
	require.NoError(t, err, "could not load internal key fixture")

	chain, err := keychain.New(keychain.WithDefaultKey(internalKey))
	require.NoError(t, err, "could not create new keychain cache")
	require.IsType(t, &memks.MemStore{}, chain.(*keychain.Cache).InternalStore())
	require.IsType(t, &memks.MemStore{}, chain.(*keychain.Cache).ExternalStore())

	// No default key should return an error
	_, err = chain.SealingKey("bravo.trisa.dev", "")
	require.ErrorIs(t, err, keyerr.KeyNotMatched, "expected empty cache with no default keys to fail on cache miss")

	keypair, err := chain.UnsealingKey("", "bravo.trisa.dev")
	require.NoError(t, err, "expected default key returned as unsealing key")
	require.Equal(t, internalKey, keypair, "expected default keys returned")

	exchange, err := chain.ExchangeKey("bravo.trisa.dev")
	require.NoError(t, err, "expected default key returned for exchange key")
	require.Equal(t, internalKey, exchange, "expected default keys returned")

	// Cache an external key
	externalKey, err := loadKeyFixture(fixtureRemoteKey)
	require.NoError(t, err, "could not load external key fixture")
	err = chain.Cache("bravo.trisa.dev", externalKey, 0)
	require.NoError(t, err, "could not store external exchange key")

	outgoing, err := chain.SealingKey("bravo.trisa.dev", "")
	require.NoError(t, err, "could not fetch external exchange key by commmon name")
	require.Equal(t, externalKey, outgoing)

	keypair, err = chain.UnsealingKey("", "bravo.trisa.dev")
	require.NoError(t, err, "expected default key returned as unsealing key")
	require.Equal(t, internalKey, keypair, "expected default keys returned")

	exchange, err = chain.ExchangeKey("bravo.trisa.dev")
	require.NoError(t, err, "expected default key returned for exchange key")
	require.Equal(t, internalKey, exchange, "expected default keys returned")

	// Should still be able to lookup internal key by signature
	internalPKS, err := internalKey.PublicKeySignature()
	require.NoError(t, err, "could not compute internal pks")

	keypair, err = chain.UnsealingKey(internalPKS, "")
	require.NoError(t, err, "expected lookup of internal key with signature")
	require.Equal(t, internalKey, keypair)
}

func TestCounterpartyCache(t *testing.T) {
	// Loading an internal key, specifying it as a default
	internalKey, err := loadKeyFixture(fixtureLocalKey)
	require.NoError(t, err, "could not load internal key fixture")

	// Cache an external key
	externalKey, err := loadKeyFixture(fixtureRemoteKey)
	require.NoError(t, err, "could not load external key fixture")

	chain, err := keychain.New()
	require.NoError(t, err, "could not create new keychain cache")
	require.IsType(t, &memks.MemStore{}, chain.(*keychain.Cache).InternalStore())
	require.IsType(t, &memks.MemStore{}, chain.(*keychain.Cache).ExternalStore())

	// Store internal and external key associated with common names
	err = chain.Store(internalKey, &keychain.KeyOptions{Counterparties: []string{"bravo.trisa.dev", "charlie.trisa.dev"}})
	require.NoError(t, err, "could not store internal unsealing key")

	err = chain.Cache("bravo.trisa.dev", externalKey, 0)
	require.NoError(t, err, "could not cache external exchange key")

	// No errors should be returned on lookups
	seal, err := chain.SealingKey("bravo.trisa.dev", "")
	require.NoError(t, err, "expected key for common name as unsealing key")
	require.Equal(t, externalKey, seal, "expected key for common name returned")

	keypair, err := chain.UnsealingKey("", "bravo.trisa.dev")
	require.NoError(t, err, "expected key for common name returned as unsealing key")
	require.Equal(t, internalKey, keypair, "expected key for common name returned")

	exchange, err := chain.ExchangeKey("bravo.trisa.dev")
	require.NoError(t, err, "expected key for common name returned for exchange key")
	require.Equal(t, internalKey, exchange, "expected key for common name returned")

}

func TestKeyCacheExpiration(t *testing.T) {
	chain, err := keychain.New(keychain.WithCacheDuration(100 * time.Millisecond))
	require.NoError(t, err, "could not create new keychain cache")

	externalKey, err := loadKeyFixture(fixtureRemoteKey)
	require.NoError(t, err, "could not load external key fixture")

	// Cache a key with configured cache duration
	err = chain.Cache("bravo.trisa.dev", externalKey, 0)
	require.NoError(t, err, "could not cache key")

	// Cache hit should happen within 800ms
	pubkey, err := chain.SealingKey("bravo.trisa.dev", "")
	require.NoError(t, err, "could not fetch cached pubkey")
	require.Equal(t, externalKey, pubkey)

	// Wait until cache expires
	time.Sleep(150 * time.Millisecond)

	// Cache miss should occur
	pubkey, err = chain.SealingKey("bravo.trisa.dev", "")
	require.ErrorIs(t, err, keyerr.KeyNotFound)
	require.Nil(t, pubkey)

	// Should be able to (re)cache a key with a specific duration
	err = chain.Cache("bravo.trisa.dev", externalKey, 200*time.Millisecond)
	require.NoError(t, err, "could not cache key")

	// Cache should not expire
	time.Sleep(150 * time.Millisecond)
	pubkey, err = chain.SealingKey("bravo.trisa.dev", "")
	require.NoError(t, err, "could not fetch cached pubkey")
	require.Equal(t, externalKey, pubkey)

	// Cache should expire after second sleep
	time.Sleep(150 * time.Millisecond)
	_, err = chain.SealingKey("bravo.trisa.dev", "")
	require.ErrorIs(t, err, keyerr.KeyNotFound)
}

func TestKeyStoreExpiration(t *testing.T) {
	chain, err := keychain.New(keychain.WithCacheDuration(100 * time.Millisecond))
	require.NoError(t, err, "could not create new keychain cache")

	internalKey, err := loadKeyFixture(fixtureLocalKey)
	require.NoError(t, err, "could not load internal key fixture")
	keysig, err := internalKey.PublicKeySignature()
	require.NoError(t, err, "could not compute public key signature")

	err = chain.Store(internalKey, &keychain.KeyOptions{Counterparties: []string{"bravo.trisa.dev"}})
	require.NoError(t, err, "could not store internal key fixture")

	// cache duration should not apply to internal keys
	time.Sleep(150 * time.Millisecond)
	keypair, err := chain.UnsealingKey(keysig, "")
	require.NoError(t, err, "internal keys should not be subject to key cache")
	require.Equal(t, internalKey, keypair)

	exchange, err := chain.ExchangeKey("bravo.trisa.dev")
	require.NoError(t, err, "internal keys should not be subject to key cache")
	require.Equal(t, internalKey, exchange)

	// add a key that does expire
	err = chain.Store(internalKey, &keychain.KeyOptions{ExpiresOn: time.Now().Add(200 * time.Millisecond), Counterparties: []string{"bravo.trisa.dev"}})
	require.NoError(t, err, "could not store internal key with expiration")

	// should be able to immediately fetch keys
	keypair, err = chain.UnsealingKey(keysig, "")
	require.NoError(t, err, "internal keys should not be subject to key cache")
	require.Equal(t, internalKey, keypair)

	exchange, err = chain.ExchangeKey("bravo.trisa.dev")
	require.NoError(t, err, "internal keys should not be subject to key cache")
	require.Equal(t, internalKey, exchange)

	// cache should expire and no keys should return
	time.Sleep(250 * time.Millisecond)
	keypair, err = chain.UnsealingKey(keysig, "")
	require.ErrorIs(t, err, keyerr.KeyExpired, "expected internal key cache to have expired")
	require.Nil(t, keypair, "expected no key to be returned after cache expiration")

	exchange, err = chain.ExchangeKey("bravo.trisa.dev")
	require.ErrorIs(t, err, keyerr.KeyExpired, "expected internal key cache to have expired")
	require.Nil(t, exchange)
}

func TestLookup(t *testing.T) {
	chain, err := keychain.New()
	require.NoError(t, err, "could not create new keychain cache")

	externalKey, err := loadKeyFixture(fixtureRemoteKey)
	require.NoError(t, err, "could not load external key fixture")
	err = chain.Cache("bravo.trisa.dev", externalKey, 0)
	require.NoError(t, err, "could not cache external key fixture")

	externalSignature, err := externalKey.PublicKeySignature()
	require.NoError(t, err, "could not get external key signature")

	internalKey, err := loadKeyFixture(fixtureLocalKey)
	require.NoError(t, err, "could not load internal key fixture")
	err = chain.Store(internalKey, &keychain.KeyOptions{Counterparties: []string{"bravo.trisa.dev"}})
	require.NoError(t, err, "could not store internal key fixture")

	internalSignature, err := internalKey.PublicKeySignature()
	require.NoError(t, err, "could not get internal key signature")

	testCases := []struct {
		commonName string
		source     keychain.Source
		signature  string
		err        error
		message    string
	}{
		{"bravo.trisa.dev", keychain.ExternalSource, externalSignature, nil, "expected successful lookup of external key"},
		{"bravo.trisa.dev", keychain.InternalSource, internalSignature, nil, "expected successful lookup of internal key mapped to counterparty"},
		{"charlie.trisa.dev", keychain.ExternalSource, "", keyerr.KeyNotMatched, "expected unsuccessful lookup of external key"},
		{"charlie.trisa.dev", keychain.InternalSource, "", keyerr.NoDefaultKeys, "expected unsuccessful lookup of internal without default"},
	}

	// Test case where there are no default keys
	for _, tc := range testCases {
		actual, err := chain.(*keychain.Cache).Lookup(tc.commonName, tc.source)
		require.Equal(t, tc.signature, actual, tc.message)
		require.ErrorIs(t, err, tc.err, tc.message)
	}

	// Test case when there are default keys
	err = chain.Store(internalKey, &keychain.KeyOptions{IsDefault: true})
	require.NoError(t, err, "could not store default internal key fixture")

	testCases = []struct {
		commonName string
		source     keychain.Source
		signature  string
		err        error
		message    string
	}{
		{"bravo.trisa.dev", keychain.ExternalSource, externalSignature, nil, "expected successful lookup of external key"},
		{"bravo.trisa.dev", keychain.InternalSource, internalSignature, nil, "expected successful lookup of internal key mapped to counterparty"},
		{"charlie.trisa.dev", keychain.ExternalSource, "", keyerr.KeyNotMatched, "expected unsuccessful lookup of external key"},
		{"charlie.trisa.dev", keychain.InternalSource, internalSignature, nil, "expected uuccessful lookup of internal with default"},
	}

	// Test case where there are no default keys
	for _, tc := range testCases {
		actual, err := chain.(*keychain.Cache).Lookup(tc.commonName, tc.source)
		require.Equal(t, tc.signature, actual, tc.message)
		require.ErrorIs(t, err, tc.err, tc.message)
	}
}

func TestSourceMap(t *testing.T) {
	names := keychain.NewSourceMap()
	require.Contains(t, names, keychain.ExternalSource, "expected source map to contain external")
	require.Contains(t, names, keychain.InternalSource, "expected source map to contain internal")
	require.NotContains(t, names, keychain.UnknownSource, "expected source map not to contain unknown")

	for _, source := range []keychain.Source{keychain.ExternalSource, keychain.InternalSource} {
		// Should be able to directly access a value through source
		name, ok := names[source]["foo"]
		require.False(t, ok, "unexpected foo in names")
		require.Empty(t, name, "unexpected foo in names")

		names[source]["foo"] = "bar"
		name, ok = names[source]["foo"]
		require.True(t, ok, "expected foo in names")
		require.Equal(t, "bar", name, "expected foo in names")

		delete(names[source], "foo")
		name, ok = names[source]["foo"]
		require.False(t, ok, "unexpected foo in names")
		require.Empty(t, name, "unexpected foo in names")

		names[source]["foo"] = "zap"
		name, ok = names[source]["foo"]
		require.True(t, ok, "expected foo in names")
		require.Equal(t, "zap", name, "expected foo in names")
	}

	require.Len(t, names, 2)
	require.Len(t, names[keychain.ExternalSource], 1)
	require.Len(t, names[keychain.InternalSource], 1)
}

func TestSourceType(t *testing.T) {
	require.Equal(t, "unknown", keychain.UnknownSource.String())
	require.Equal(t, "internal", keychain.InternalSource.String())
	require.Equal(t, "external", keychain.ExternalSource.String())
}
