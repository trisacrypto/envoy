package secrets

import (
	"context"
	"fmt"
	"time"

	secretmanager "cloud.google.com/go/secretmanager/apiv1"
	"cloud.google.com/go/secretmanager/apiv1/secretmanagerpb"
	"google.golang.org/api/iterator"
)

const (
	DefaultTimeout        = 30 * time.Second
	DefaultPageSize       = 100
	AnnotationContentType = "content_type"
)

func NewGCP() (sm *GCP, err error) {
	sm = &GCP{}
	if sm.client, err = secretmanager.NewClient(context.Background()); err != nil {
		return nil, err
	}
	return sm, nil
}

type GCP struct {
	client *secretmanager.Client
}

func (g *GCP) Close() error {
	return g.client.Close()
}

func (g *GCP) ListSecrets(ctx context.Context, namespace string) (Iterator, error) {
	// Create the list secrets request
	req := &secretmanagerpb.ListSecretsRequest{
		Parent:    g.SecretParent(&Secret{Namespace: namespace}),
		PageSize:  int32(DefaultPageSize),
		PageToken: "",
	}

	// Create an internal api context with a deadline since a failed API call could hang
	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, DefaultTimeout)
		defer cancel()
	}

	iter := &GoogleSecretIterator{
		namespace: namespace,
		iter:      g.client.ListSecrets(ctx, req),
	}

	return iter, nil
}

func (g *GCP) CreateSecret(ctx context.Context, secret *Secret) (err error) {
	// Create an internal api context with a deadline since a failed API call could hang
	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, DefaultTimeout)
		defer cancel()
	}

	// Build the create secret request
	if err = g.createSecret(ctx, secret); err != nil {
		return err
	}

	// Add the data as a secret version
	if err = g.createSecretVersion(ctx, secret); err != nil {
		return err
	}
	return nil
}

func (g *GCP) createSecret(ctx context.Context, secret *Secret) (err error) {
	req := &secretmanagerpb.CreateSecretRequest{
		Parent:   g.SecretParent(secret),
		SecretId: secret.Name,
		Secret: &secretmanagerpb.Secret{
			Replication: &secretmanagerpb.Replication{
				Replication: &secretmanagerpb.Replication_Automatic_{
					Automatic: &secretmanagerpb.Replication_Automatic{},
				},
			},
			Annotations: map[string]string{
				AnnotationContentType: secret.ContentType,
			},
		},
	}

	var s *secretmanagerpb.Secret
	if s, err = g.client.CreateSecret(ctx, req); err != nil {
		return err
	}

	// Add the created time to the secret
	secret.Created = s.CreateTime.AsTime()
	return nil
}

func (g *GCP) createSecretVersion(ctx context.Context, secret *Secret) (err error) {
	req := &secretmanagerpb.AddSecretVersionRequest{
		Parent: g.SecretName(secret),
		Payload: &secretmanagerpb.SecretPayload{
			Data: secret.Data,
		},
	}

	var v *secretmanagerpb.SecretVersion
	if v, err = g.client.AddSecretVersion(ctx, req); err != nil {
		return err
	}

	secret.Created = v.CreateTime.AsTime()
	return nil
}

func (g *GCP) RetrieveSecret(ctx context.Context, secret *Secret) (err error) {
	// Create an internal api context with a deadline since a failed API call could hang
	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, DefaultTimeout)
		defer cancel()
	}

	// Get the secret metadata
	if err = g.retrieveSecret(ctx, secret); err != nil {
		return err
	}

	// Get the latest secret version payload
	if err = g.retrieveSecretVersion(ctx, secret); err != nil {
		return err
	}
	return nil
}

func (g *GCP) retrieveSecret(ctx context.Context, secret *Secret) (err error) {
	req := &secretmanagerpb.GetSecretRequest{
		Name: g.SecretName(secret),
	}

	var s *secretmanagerpb.Secret
	if s, err = g.client.GetSecret(ctx, req); err != nil {
		return err
	}

	secret.ContentType = s.Annotations["content_type"]
	secret.Created = s.CreateTime.AsTime()
	return nil
}

func (g *GCP) retrieveSecretVersion(ctx context.Context, secret *Secret) (err error) {
	req := &secretmanagerpb.AccessSecretVersionRequest{
		Name: g.SecretName(secret),
	}

	var reply *secretmanagerpb.AccessSecretVersionResponse
	if reply, err = g.client.AccessSecretVersion(ctx, req); err != nil {
		return err
	}

	secret.Data = reply.Payload.Data
	return nil
}

func (g *GCP) DeleteSecret(ctx context.Context, secret *Secret) (err error) {
	// Create an internal api context with a deadline since a failed API call could hang
	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, DefaultTimeout)
		defer cancel()
	}

	req := &secretmanagerpb.DeleteSecretRequest{
		Name: g.SecretName(secret),
	}

	if err = g.client.DeleteSecret(ctx, req); err != nil {
		return err
	}
	return nil
}

func (g *GCP) SecretParent(secret *Secret) string {
	return fmt.Sprintf("projects/%s", secret.Namespace)
}

func (g *GCP) SecretName(secret *Secret) string {
	return fmt.Sprintf("projects/%s/secrets/%s", secret.Namespace, secret.Name)
}

//===========================================================================
// Iterator
//===========================================================================

type GoogleSecretIterator struct {
	namespace string
	iter      *secretmanager.SecretIterator
	current   *secretmanagerpb.Secret
	err       error
}

func (i *GoogleSecretIterator) Close() error {
	i.iter = nil
	return nil
}

func (i *GoogleSecretIterator) Next() bool {
	var err error
	if i.current, err = i.iter.Next(); err != nil {
		if err != iterator.Done {
			i.err = err
		}
		return false
	}
	return true
}

func (i *GoogleSecretIterator) Err() error {
	return i.err
}

func (i *GoogleSecretIterator) Secret() *Secret {
	if i.current != nil {
		return &Secret{
			Namespace:   i.namespace,
			Name:        i.current.Name,
			ContentType: i.current.Annotations[AnnotationContentType],
			Created:     i.current.CreateTime.AsTime(),
		}
	}
	return nil
}
