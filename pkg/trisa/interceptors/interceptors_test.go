package interceptors_test

import (
	"os"
	"testing"

	"github.com/trisacrypto/envoy/pkg/logger"
)

func TestMain(m *testing.M) {
	logger.Discard()
	ec := m.Run()
	logger.ResetLogger()
	os.Exit(ec)
}
