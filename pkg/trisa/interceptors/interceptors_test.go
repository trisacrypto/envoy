package interceptors_test

import (
	"os"
	"self-hosted-node/pkg/logger"
	"testing"
)

func TestMain(m *testing.M) {
	logger.Discard()
	ec := m.Run()
	logger.ResetLogger()
	os.Exit(ec)
}
