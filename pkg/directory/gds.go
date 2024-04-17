package directory

import (
	"context"
	"time"

	"github.com/trisacrypto/envoy/pkg/trisa/gds"

	members "github.com/trisacrypto/directory/pkg/gds/members/v1alpha1"
)

const defaultTimeout = 20 * time.Second

type PaginatedMembersIterator struct {
	client  gds.Directory
	request *members.ListRequest
	reply   *members.ListReply
	err     error
}

func ListMembers(client gds.Directory) *PaginatedMembersIterator {
	return &PaginatedMembersIterator{
		client: client,
		request: &members.ListRequest{
			PageSize:  100,
			PageToken: "",
		},
	}
}

func (i *PaginatedMembersIterator) Next() bool {
	if i.reply == nil || i.reply.NextPageToken != "" {
		ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
		defer cancel()

		i.reply, i.err = i.client.List(ctx, i.request)
		if i.reply != nil {
			i.request.PageToken = i.reply.NextPageToken
		}

		return i.err == nil
	}
	return false
}

func (i *PaginatedMembersIterator) Members() []*members.VASPMember {
	if i.reply != nil {
		return i.reply.Vasps
	}
	return nil
}

func (i *PaginatedMembersIterator) Err() error {
	return i.err
}
