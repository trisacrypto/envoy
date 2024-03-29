package ulids

import (
	"io"
	"math/rand"
	"sync"
	"time"
)

// PoolEntropy is a thread-safe source of entropy that is not cryptographically secure
// but provides fast, concurrent access to random data generation.
type PoolEntropy struct {
	sync.Pool
}

func NewPool() *PoolEntropy {
	return &PoolEntropy{
		Pool: sync.Pool{
			New: func() any {
				return rand.New(rand.NewSource(time.Now().UnixNano()))
			},
		},
	}
}

func (e *PoolEntropy) Read(p []byte) (n int, err error) {
	r := e.Get().(io.Reader)
	n, err = r.Read(p)
	e.Put(r)
	return n, err
}
