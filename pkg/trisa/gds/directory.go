package gds

import (
	"context"

	members "github.com/trisacrypto/directory/pkg/gds/members/v1alpha1"
	gds "github.com/trisacrypto/trisa/pkg/trisa/gds/api/v1beta1"
	"google.golang.org/grpc"
)

// Directory implements a client-side selection of interactions with the TRISA Global
// Directory Service, e.g. Lookup and Search from the TRISADirectoryClient and List from
// the TRISAMembersClient interfaces. A Directory is used to manage TRISA network peers.
type Directory interface {
	Lookup(ctx context.Context, in *gds.LookupRequest, opts ...grpc.CallOption) (*gds.LookupReply, error)
	Search(ctx context.Context, in *gds.SearchRequest, opts ...grpc.CallOption) (*gds.SearchReply, error)
	List(ctx context.Context, in *members.ListRequest, opts ...grpc.CallOption) (*members.ListReply, error)
	Status(ctx context.Context, in *gds.HealthCheck, opts ...grpc.CallOption) (*gds.ServiceState, error)
	Connect(opts ...grpc.DialOption) error
	Close() error
}
