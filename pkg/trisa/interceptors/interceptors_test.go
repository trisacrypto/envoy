package interceptors_test

import (
	"os"
	"testing"

	"github.com/trisacrypto/envoy/pkg/logger"
	"google.golang.org/grpc/resolver"
)

func init() {
	// Required for buffconn testing.
	resolver.SetDefaultScheme("passthrough")
}

func TestMain(m *testing.M) {
	logger.Discard()
	ec := m.Run()
	logger.ResetLogger()
	os.Exit(ec)
}
