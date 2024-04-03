package mock

import (
	"context"
	"fmt"
	"os"

	"self-hosted-node/pkg/bufconn"

	members "github.com/trisacrypto/directory/pkg/gds/members/v1alpha1"
	gds "github.com/trisacrypto/trisa/pkg/trisa/gds/api/v1beta1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
)

const (
	LookupRPC  = "trisa.gds.api.v1beta1.TRISADirectory/Lookup"
	SearchRPC  = "trisa.gds.api.v1beta1.TRISADirectory/Search"
	SummaryRPC = "gds.members.v1alpha1.TRISAMembers/Summary"
	ListRPC    = "gds.members.v1alpha1.TRISAMembers/List"
	DetailRPC  = "gds.members.v1alpha1.TRISAMembers/Details"
	StatusRPC  = "trisa.gds.api.v1beta1.TRISADirectory/Status"
)

// New creates a new mock GDS. If bufnet is nil, one is created for the user.
func New(bufnet *bufconn.Listener) *GDS {
	if bufnet == nil {
		bufnet = bufconn.New()
	}

	remote := &GDS{
		bufnet: bufnet,
		srv:    grpc.NewServer(),
		Calls:  make(map[string]int),
	}

	gds.RegisterTRISADirectoryServer(remote.srv, remote)
	members.RegisterTRISAMembersServer(remote.srv, remote)
	go remote.srv.Serve(remote.bufnet.Sock())
	return remote
}

// GDS implements a mock gRPC server for testing TRISA Global Directory Service client
// connections. The desired response of the directory service can be set by external
// callers using the OnRPC functions or the WithFixture or WithError functions. The
// Calls map can be used to count the number of times the remote peer PRC was called.
type GDS struct {
	gds.UnimplementedTRISADirectoryServer
	members.UnimplementedTRISAMembersServer
	bufnet    *bufconn.Listener
	srv       *grpc.Server
	Calls     map[string]int
	OnLookup  func(context.Context, *gds.LookupRequest) (*gds.LookupReply, error)
	OnSearch  func(context.Context, *gds.SearchRequest) (*gds.SearchReply, error)
	OnSummary func(context.Context, *members.SummaryRequest) (*members.SummaryReply, error)
	OnList    func(context.Context, *members.ListRequest) (*members.ListReply, error)
	OnDetail  func(context.Context, *members.DetailsRequest) (*members.MemberDetails, error)
	OnStatus  func(context.Context, *gds.HealthCheck) (*gds.ServiceState, error)
}

func (s *GDS) Channel() *bufconn.Listener {
	return s.bufnet
}

func (s *GDS) Shutdown() {
	s.srv.GracefulStop()
	s.bufnet.Close()
}

func (s *GDS) Reset() {
	for key := range s.Calls {
		s.Calls[key] = 0
	}

	s.OnLookup = nil
	s.OnSearch = nil
	s.OnSummary = nil
	s.OnList = nil
	s.OnDetail = nil
	s.OnStatus = nil
}

// UseFixture loadsa a JSON fixture from disk (usually in a testdata folder) to use as
// the protocol buffer response to the specified RPC, simplifying handler mocking.
func (s *GDS) UseFixture(rpc, path string) (err error) {
	var data []byte
	if data, err = os.ReadFile(path); err != nil {
		return fmt.Errorf("could not read fixture: %v", err)
	}

	jsonpb := &protojson.UnmarshalOptions{
		AllowPartial:   true,
		DiscardUnknown: true,
	}

	switch rpc {
	case LookupRPC:
		out := &gds.LookupReply{}
		if err = jsonpb.Unmarshal(data, out); err != nil {
			return fmt.Errorf("could not unmarshal json into %T: %v", out, err)
		}
		s.OnLookup = func(context.Context, *gds.LookupRequest) (*gds.LookupReply, error) {
			return out, nil
		}
	case SearchRPC:
		out := &gds.SearchReply{}
		if err = jsonpb.Unmarshal(data, out); err != nil {
			return fmt.Errorf("could not unmarshal json into %T: %v", out, err)
		}
		s.OnSearch = func(context.Context, *gds.SearchRequest) (*gds.SearchReply, error) {
			return out, nil
		}
	case SummaryRPC:
		out := &members.SummaryReply{}
		if err = jsonpb.Unmarshal(data, out); err != nil {
			return fmt.Errorf("could not unmarshal json into %T: %v", out, err)
		}
		s.OnSummary = func(ctx context.Context, sr *members.SummaryRequest) (*members.SummaryReply, error) {
			return out, nil
		}
	case ListRPC:
		out := &members.ListReply{}
		if err = jsonpb.Unmarshal(data, out); err != nil {
			return fmt.Errorf("could not unmarshal json into %T: %v", out, err)
		}
		s.OnList = func(context.Context, *members.ListRequest) (*members.ListReply, error) {
			return out, nil
		}
	case DetailRPC:
		out := &members.MemberDetails{}
		if err = jsonpb.Unmarshal(data, out); err != nil {
			return fmt.Errorf("could not unmarshal json into %T: %v", out, err)
		}
		s.OnDetail = func(ctx context.Context, dr *members.DetailsRequest) (*members.MemberDetails, error) {
			return out, nil
		}
	case StatusRPC:
		out := &gds.ServiceState{}
		if err = jsonpb.Unmarshal(data, out); err != nil {
			return fmt.Errorf("could not unmarshal json into %T: %v", out, err)
		}
		s.OnStatus = func(context.Context, *gds.HealthCheck) (*gds.ServiceState, error) {
			return out, nil
		}
	default:
		return fmt.Errorf("unknown RPC %q", rpc)
	}

	return nil
}

// UseError allows you to specify a gRPC status error to return from the specified RPC.
func (s *GDS) UseError(rpc string, code codes.Code, msg string) error {
	switch rpc {
	case LookupRPC:
		s.OnLookup = func(context.Context, *gds.LookupRequest) (*gds.LookupReply, error) {
			return nil, status.Error(code, msg)
		}
	case SearchRPC:
		s.OnSearch = func(context.Context, *gds.SearchRequest) (*gds.SearchReply, error) {
			return nil, status.Error(code, msg)
		}
	case SummaryRPC:
		s.OnSummary = func(ctx context.Context, sr *members.SummaryRequest) (*members.SummaryReply, error) {
			return nil, status.Error(code, msg)
		}
	case ListRPC:
		s.OnList = func(context.Context, *members.ListRequest) (*members.ListReply, error) {
			return nil, status.Error(code, msg)
		}
	case DetailRPC:
		s.OnDetail = func(ctx context.Context, dr *members.DetailsRequest) (*members.MemberDetails, error) {
			return nil, status.Error(code, msg)
		}
	case StatusRPC:
		s.OnStatus = func(context.Context, *gds.HealthCheck) (*gds.ServiceState, error) {
			return nil, status.Error(code, msg)
		}
	default:
		return fmt.Errorf("unknown RPC %q", rpc)
	}
	return nil
}

func (s *GDS) Lookup(ctx context.Context, in *gds.LookupRequest) (*gds.LookupReply, error) {
	s.Calls[LookupRPC]++
	return s.OnLookup(ctx, in)
}

func (s *GDS) Search(ctx context.Context, in *gds.SearchRequest) (*gds.SearchReply, error) {
	s.Calls[SearchRPC]++
	return s.OnSearch(ctx, in)
}

func (s *GDS) Summary(ctx context.Context, in *members.SummaryRequest) (*members.SummaryReply, error) {
	s.Calls[SummaryRPC]++
	return s.OnSummary(ctx, in)
}

func (s *GDS) List(ctx context.Context, in *members.ListRequest) (*members.ListReply, error) {
	s.Calls[ListRPC]++
	return s.OnList(ctx, in)
}

func (s *GDS) Details(ctx context.Context, in *members.DetailsRequest) (*members.MemberDetails, error) {
	s.Calls[DetailRPC]++
	return s.OnDetail(ctx, in)
}

func (s *GDS) Status(ctx context.Context, in *gds.HealthCheck) (*gds.ServiceState, error) {
	s.Calls[StatusRPC]++
	return s.OnStatus(ctx, in)
}
