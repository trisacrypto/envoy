package webhook

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"time"

	"github.com/trisacrypto/envoy/pkg/config"
	trisa "github.com/trisacrypto/trisa/pkg/trisa/api/v1beta1"
	generic "github.com/trisacrypto/trisa/pkg/trisa/data/generic/v1beta1"
)

const (
	mockURL    = "mock:///"
	mockScheme = "mock"
)

var MockConfig = config.WebhookConfig{
	URL: mockURL,
}

// Mock implements the webhook Handler and is used for testing webhook interactions.
type Mock struct {
	OnCallback func(context.Context, *Request) (*Reply, error)
	Callbacks  int
}

func NewMock() *Mock {
	return &Mock{}
}

func (m *Mock) Callback(ctx context.Context, req *Request) (*Reply, error) {
	m.Callbacks++
	if m.OnCallback != nil {
		return m.OnCallback(ctx, req)
	}
	return nil, errors.New("no mock callback configured")
}

func (m *Mock) UseError(err error) {
	m.OnCallback = func(context.Context, *Request) (*Reply, error) {
		return nil, err
	}
}

func (m *Mock) UseReply(rep *Reply) {
	m.OnCallback = func(context.Context, *Request) (*Reply, error) {
		return rep, nil
	}
}

func (m *Mock) UseFixture(path string) (err error) {
	var f *os.File
	if f, err = os.Open(path); err != nil {
		return err
	}
	defer f.Close()

	var rep *Reply
	if err = json.NewDecoder(f).Decode(&rep); err != nil {
		return err
	}

	m.UseReply(rep)
	return nil
}

func (m *Mock) Reset() {
	m.OnCallback = nil
	m.Callbacks = 0
}

func MockPendingReply(_ context.Context, req *Request) (*Reply, error) {
	return &Reply{
		TransactionID: req.TransactionID,
		Payload: &Payload{
			Identity:   req.Payload.Identity,
			SentAt:     req.Payload.SentAt,
			ReceivedAt: req.Payload.ReceivedAt,
			Pending: &generic.Pending{
				EnvelopeId:     req.TransactionID.String(),
				ReceivedBy:     "Mock Webhook Handler",
				ReceivedAt:     time.Now().Format(time.RFC3339),
				Message:        "This is a test handler for the Envoy webhook package",
				ReplyNotAfter:  time.Now().Add(10 * time.Minute).Format(time.RFC3339),
				ReplyNotBefore: time.Now().Add(1 * time.Minute).Format(time.RFC3339),
				Transaction:    req.Payload.Transaction,
			},
		},
	}, nil
}

func MockErrorReply(_ context.Context, req *Request) (*Reply, error) {
	return &Reply{
		TransactionID: req.TransactionID,
		Error: &trisa.Error{
			Code:    trisa.ComplianceCheckFail,
			Message: "mock rejection error",
			Retry:   false,
		},
	}, nil
}
