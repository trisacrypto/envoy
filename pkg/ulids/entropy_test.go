package ulids_test

import (
	"sync"
	"testing"

	"github.com/trisacrypto/envoy/pkg/ulids"

	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
)

func TestPoolEntropy(t *testing.T) {
	wg := sync.WaitGroup{}
	entropy := ulids.NewPool()

	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 128; i++ {
				uu, err := ulid.New(ulid.Now(), entropy)
				require.NoError(t, err, "could not create ulid")
				require.NotEqual(t, 0, uu.Compare(ulids.Null), "expected ulid to not be null")
			}
		}()
	}

	wg.Wait()
}
