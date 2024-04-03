package gds

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"sync"

	"self-hosted-node/pkg/config"

	members "github.com/trisacrypto/directory/pkg/gds/members/v1alpha1"
	gds "github.com/trisacrypto/trisa/pkg/trisa/gds/api/v1beta1"
	"github.com/trisacrypto/trisa/pkg/trisa/mtls"
	"github.com/trisacrypto/trisa/pkg/trust"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
)

// GDS implements the Directory interface to interact with the TRISA Global Directory Service.
type GDS struct {
	sync.RWMutex
	conf    config.TRISAConfig
	conn    []*grpc.ClientConn
	client  gds.TRISADirectoryClient
	members members.TRISAMembersClient
}

// Ensure GDS implements the Directory interface
var _ Directory = &GDS{}

func New(conf config.TRISAConfig) *GDS {
	return &GDS{conf: conf}
}

// String returns the name of the directory service being connected to by parsing the
// root domain from the endpoint and stripping the port. E.g. api.trisatest.net:443
// becomes trisatest.net and localhost:4436 becomes localhost.
func (g *GDS) String() string {
	return g.conf.Directory.Network()
}

// Connect to the directory service by dialing the configured endpoints with the
// specified options and credentials. If no options are supplied, the Connect function
// attempts to connect using default options. The endpoints connected to are defined by
// the directory configuration. Returns an error if the GDS is already connected.
func (g *GDS) Connect(opts ...grpc.DialOption) (err error) {
	g.Lock()
	defer g.Unlock()
	if len(g.conn) > 0 {
		return ErrAlreadyConnected
	}

	defer func() {
		// Cleanup connections if there is an error
		if err != nil {
			g.conn = nil
			g.client = nil
			g.members = nil
		}
	}()

	g.conn = make([]*grpc.ClientConn, 2)
	if g.conn[0], err = g.connectDirectory(opts...); err != nil {
		return err
	}
	if g.conn[1], err = g.connectMembers(opts...); err != nil {
		return err
	}

	g.client = gds.NewTRISADirectoryClient(g.conn[0])
	g.members = members.NewTRISAMembersClient(g.conn[1])
	return nil
}

func (g *GDS) connectDirectory(opts ...grpc.DialOption) (cc *grpc.ClientConn, err error) {
	if len(opts) == 0 {
		opts = make([]grpc.DialOption, 0, 1)
		if g.conf.Directory.Insecure {
			opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
		} else {
			// Connecting to the GDS only requires default TLS credentials.
			opts = append(opts, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{})))
		}
	}

	if cc, err = grpc.Dial(g.conf.Directory.Endpoint, opts...); err != nil {
		return nil, fmt.Errorf("could not connect to %s: %s", g.conf.Directory.Endpoint, err)
	}
	return cc, nil
}

func (g *GDS) connectMembers(opts ...grpc.DialOption) (cc *grpc.ClientConn, err error) {
	if len(opts) == 0 {
		opts = make([]grpc.DialOption, 0, 1)
		if g.conf.Directory.Insecure {
			opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
		} else {
			// Connecting to the Members API requires TRISA mTLS credentials
			var certs *trust.Provider
			if certs, err = g.conf.LoadCerts(); err != nil {
				return nil, err
			}

			var pool trust.ProviderPool
			if pool, err = g.conf.LoadPool(); err != nil {
				return nil, err
			}

			var creds grpc.DialOption
			if creds, err = mtls.ClientCreds(g.conf.Directory.MembersEndpoint, certs, pool); err != nil {
				return nil, err
			}
			opts = append(opts, creds)
		}
	}

	if cc, err = grpc.Dial(g.conf.Directory.MembersEndpoint, opts...); err != nil {
		return nil, fmt.Errorf("could not connect to %s: %s", g.conf.Directory.MembersEndpoint, err)
	}
	return cc, nil
}

func (g *GDS) Close() (err error) {
	g.Lock()
	defer g.Unlock()
	for _, cc := range g.conn {
		if cc != nil {
			if cerr := cc.Close(); cerr != nil {
				err = errors.Join(err, cerr)
			}
		}
	}

	g.client = nil
	g.members = nil
	g.conn = nil
	return err
}

func (g *GDS) Lookup(ctx context.Context, in *gds.LookupRequest, opts ...grpc.CallOption) (*gds.LookupReply, error) {
	g.RLock()
	defer g.RUnlock()
	if g.client == nil {
		return nil, ErrNotConnected
	}
	return g.client.Lookup(ctx, in, opts...)
}

func (g *GDS) Search(ctx context.Context, in *gds.SearchRequest, opts ...grpc.CallOption) (*gds.SearchReply, error) {
	g.RLock()
	defer g.RUnlock()
	if g.client == nil {
		return nil, ErrNotConnected
	}
	return g.client.Search(ctx, in, opts...)
}

func (g *GDS) Summary(ctx context.Context, in *members.SummaryRequest, opts ...grpc.CallOption) (*members.SummaryReply, error) {
	g.RLock()
	defer g.RUnlock()
	if g.client == nil {
		return nil, ErrNotConnected
	}
	return g.members.Summary(ctx, in, opts...)
}

func (g *GDS) List(ctx context.Context, in *members.ListRequest, opts ...grpc.CallOption) (*members.ListReply, error) {
	g.RLock()
	defer g.RUnlock()
	if g.client == nil {
		return nil, ErrNotConnected
	}
	return g.members.List(ctx, in, opts...)
}

func (g *GDS) Detail(ctx context.Context, in *members.DetailsRequest, opts ...grpc.CallOption) (*members.MemberDetails, error) {
	g.RLock()
	defer g.RUnlock()
	if g.client == nil {
		return nil, ErrNotConnected
	}
	return g.members.Details(ctx, in, opts...)
}

func (g *GDS) Status(ctx context.Context, in *gds.HealthCheck, opts ...grpc.CallOption) (*gds.ServiceState, error) {
	g.RLock()
	defer g.RUnlock()
	if g.client == nil {
		return nil, ErrNotConnected
	}
	return g.client.Status(ctx, in, opts...)
}
