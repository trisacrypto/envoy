package trisa

import (
	"context"
	"time"

	api "github.com/trisacrypto/trisa/pkg/trisa/api/v1beta1"
)

// Status implements the TRISAHealth gRPC interface and is used for GDS health checks.
// TODO: allow user to configure not before/after time window
func (s *Server) Status(ctx context.Context, in *api.HealthCheck) (out *api.ServiceState, err error) {
	out = &api.ServiceState{
		Status:    api.ServiceState_HEALTHY,
		NotBefore: time.Now().Add(5 * time.Minute).Format(time.RFC3339),
		NotAfter:  time.Now().Add(12 * time.Hour).Format(time.RFC3339),
	}

	if s.conf.Maintenance {
		out.Status = api.ServiceState_MAINTENANCE
	}

	return out, nil
}
