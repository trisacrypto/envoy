package directory_test

import (
	"context"
	"encoding/json"
	"os"
	"testing"

	"self-hosted-node/pkg/bufconn"
	"self-hosted-node/pkg/config"
	"self-hosted-node/pkg/directory"
	"self-hosted-node/pkg/trisa/gds"
	mockgds "self-hosted-node/pkg/trisa/gds/mock"

	"github.com/stretchr/testify/require"
	members "github.com/trisacrypto/directory/pkg/gds/members/v1alpha1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
)

func TestPaginatedMembersIterator(t *testing.T) {
	bufnet := bufconn.New()
	defer bufnet.Close()

	conf := config.TRISAConfig{
		Directory: config.DirectoryConfig{
			Insecure:        true,
			Endpoint:        "bufnet",
			MembersEndpoint: "bufnet",
		},
	}

	gds := gds.New(conf)
	err := gds.Connect(grpc.WithContextDialer(bufnet.Dialer), grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err, "unable to connect to mock GDS via directory client")

	mock := mockgds.New(bufnet)

	t.Run("Error", func(t *testing.T) {
		mock.UseError(mockgds.ListRPC, codes.Internal, "something went wrong")

		iter := directory.ListMembers(gds)
		require.False(t, iter.Next(), "expected no iteration due to the error")
		require.EqualError(t, iter.Err(), "rpc error: code = Internal desc = something went wrong", "expected error to be reported in iterator")
		require.Nil(t, iter.Members(), "expected no members to be returned and no panic")
	})

	t.Run("Single", func(t *testing.T) {
		err := mock.UseFixture(mockgds.ListRPC, "testdata/single.pb.json")
		require.NoError(t, err, "could not load fixture")

		iter := directory.ListMembers(gds)

		// First page is also the last page
		require.True(t, iter.Next(), "expected a single page iteration")
		require.NoError(t, iter.Err(), "expected no error after iteration")

		page := iter.Members()
		require.Len(t, page, 3, "expected a single page of of results")

		// Ensure the last page is returned
		require.False(t, iter.Next(), "expected a single page iteration")
		require.NoError(t, iter.Err(), "expected no error after iteration")

		require.Equal(t, page, iter.Members(), "expected members to return last page of results")
	})

	t.Run("Multiple", func(t *testing.T) {
		vasps, err := loadVASPFixture("testdata/vasps.pb.json")
		require.NoError(t, err, "could not load vasps.pb.json fixture")
		require.Len(t, vasps, 12, "expected 12 vasps, has fixture changed?")

		mock.OnList = func(ctx context.Context, lr *members.ListRequest) (*members.ListReply, error) {
			switch lr.PageToken {
			case "":
				return &members.ListReply{
					NextPageToken: "pg1",
					Vasps:         vasps[0:4],
				}, nil
			case "pg1":
				return &members.ListReply{
					NextPageToken: "pg2",
					Vasps:         vasps[4:8],
				}, nil
			case "pg2":
				return &members.ListReply{
					NextPageToken: "",
					Vasps:         vasps[8:],
				}, nil
			default:
				return nil, status.Error(codes.NotFound, "undefined next page token")
			}
		}

		iter := directory.ListMembers(gds)
		pages := 0
		seen := make(map[string]struct{})

		for iter.Next() {
			pages++
			for _, vasp := range iter.Members() {
				seen[vasp.Name] = struct{}{}
			}
		}

		require.Equal(t, 3, pages, "expected 3 pages to be returned")
		require.Len(t, seen, 12, "expected 12 unique vasps returned")
		require.NoError(t, iter.Err(), "expected no error from iteration")
	})
}

func loadVASPFixture(path string) (vasps []*members.VASPMember, err error) {
	var data []byte
	if data, err = os.ReadFile(path); err != nil {
		return nil, err
	}

	items := make([]interface{}, 0)
	if err = json.Unmarshal(data, &items); err != nil {
		return nil, err
	}

	pbjson := protojson.UnmarshalOptions{
		AllowPartial:   true,
		DiscardUnknown: true,
	}

	vasps = make([]*members.VASPMember, 0, len(items))
	for _, item := range items {
		var pbdata []byte
		if pbdata, err = json.Marshal(item); err != nil {
			return nil, err
		}

		vasp := &members.VASPMember{}
		if err = pbjson.Unmarshal(pbdata, vasp); err != nil {
			return nil, err
		}
		vasps = append(vasps, vasp)
	}

	return vasps, nil
}
