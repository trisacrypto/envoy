package memks_test

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"self-hosted-node/pkg/trisa/keychain/keyerr"
	"self-hosted-node/pkg/trisa/keychain/memks"

	"github.com/stretchr/testify/require"
	api "github.com/trisacrypto/trisa/pkg/trisa/api/v1beta1"
)

func TestMemStore(t *testing.T) {
	store, err := memks.New()
	require.NoError(t, err, "could not create a new memstore")

	// Should not be able to get a key that doesn't exist
	key, ts, err := store.Get("notarealsignature")
	require.ErrorIs(t, err, keyerr.KeyNotFound, "expected key not found error")
	require.Zero(t, ts, "unexpected non-zero timestamp returned")
	require.Nil(t, key, "unexpected non-nil key returned")

	// Should not be able to put a key that doesn't return a signature
	err = store.Put(&TestKey{signature: errors.New("bad signature")})
	require.EqualError(t, err, "bad signature")

	// Should be able to put a key that has a signature
	expected := &TestKey{signature: "abc1234"}
	err = store.Put(expected)
	require.NoError(t, err, "could not put a key with a signature")

	// Should be able to get the key with that signature
	actual, ts, err := store.Get("abc1234")
	require.NoError(t, err, "could not get key from signature")
	require.WithinDuration(t, time.Now(), ts, 500*time.Millisecond)
	require.Equal(t, expected, actual, "incorrect key returned")

	// Should be able to put a different key with a different signature
	expected2 := &TestKey{signature: "12345678", marshaler: []byte("foo")}
	require.NoError(t, store.Put(expected2))

	// Should be able to get both keys
	actual, _, err = store.Get("abc1234")
	require.NoError(t, err, "could not get key")
	require.Equal(t, expected, actual)

	actual2, ts, err := store.Get("12345678")
	require.NoError(t, err, "could not get key")
	require.Equal(t, expected2, actual2)

	_, _, err = store.Get("still not a key")
	require.ErrorIs(t, err, keyerr.KeyNotFound, "expected key not found error")

	// Should not be able to overwrite key
	overwrite := &TestKey{signature: "12345678", marshaler: []byte("bar")}
	err = store.Put(overwrite)
	require.ErrorIs(t, err, keyerr.KeyOverwrite, "expected error overwriting key with same signature but different marshal data")

	// Should be able to overwrite key and update timestamp
	err = store.Put(expected2)
	require.NoError(t, err, "could not update same key into cache")
	_, ts2, _ := store.Get("12345678")
	require.True(t, ts2.After(ts), "timestamp was not updated in cache")

	// Should be able to delete a key
	err = store.Delete("12345678")
	require.NoError(t, err, "could not delete key")

	_, _, err = store.Get("12345678")
	require.ErrorIs(t, err, keyerr.KeyNotFound, "expected key not found error")

	// Should not have deleted other keys
	actual, _, err = store.Get("abc1234")
	require.NoError(t, err, "could not get key")
	require.Equal(t, expected, actual)

	// Should be able to overwrite key now that it is deleted
	err = store.Put(overwrite)
	require.NoError(t, err, "could not add different key after delete")

	// Should not get an error when deleting a key that isn't there
	err = store.Delete("ghostkeyisinvisible")
	require.NoError(t, err, "expected no error when deleting an unknown key")
}

// TestKey is a simple key that implements the keys.Key interface. Most the keys.Key
// methods are accessors, so the properties on the TestKey are returned from their
// relevant methods with a type check - if the type is an error, an error is returned,
// otherwise it is parsed into the correct type and panics if it's the wrong type.
type TestKey struct {
	isPrivate    bool        // Returned from IsPrivate()
	sealingKey   interface{} // Returned from SealingKey()
	proto        interface{} // Returned from Proto()
	unsealingKey interface{} // Returned from UnsealingKey()
	pkAlgorithm  string      // Returned from PublicKeyAlgorithm()
	signature    interface{} // returned from PublicKeySignature()
	marshaler    interface{} // returned from Marshal()
	unmarshaler  error       // returned from Unmarshal()
}

func (t *TestKey) IsPrivate() bool {
	return t.isPrivate
}

func (t *TestKey) SealingKey() (interface{}, error) {
	switch val := t.sealingKey.(type) {
	case error:
		return nil, val
	default:
		return t.sealingKey, nil
	}
}

func (t *TestKey) Proto() (*api.SigningKey, error) {
	switch val := t.proto.(type) {
	case *api.SigningKey:
		return val, nil
	case error:
		return nil, val
	default:
		panic(fmt.Errorf("unhandled type %T", val))
	}
}

func (t *TestKey) UnsealingKey() (interface{}, error) {
	switch val := t.unsealingKey.(type) {
	case error:
		return nil, val
	default:
		return t.unsealingKey, nil
	}
}

func (t *TestKey) PublicKeyAlgorithm() string {
	return t.pkAlgorithm
}

func (t *TestKey) PublicKeySignature() (string, error) {
	switch val := t.signature.(type) {
	case string:
		return val, nil
	case error:
		return "", val
	default:
		panic(fmt.Errorf("unhandled type %T", val))
	}
}

func (t *TestKey) Marshal() ([]byte, error) {
	switch val := t.marshaler.(type) {
	case []byte:
		return val, nil
	case error:
		return nil, val
	default:
		panic(fmt.Errorf("unhandled type %T", val))
	}
}

func (t *TestKey) Unmarshal(data []byte) error {
	return t.unmarshaler
}
