package secrets

import (
	"io"
	"time"
)

const (
	PEMFile         = "application/x-pem-file"
	X509Certificate = "application/x-x509-user-cert"
)

// Secret represents a generic blob of data that can be stored in a secrets manager such
// as Hashicorp Vault or Google Secret Manager. The name and optional namespace are
// used to uniquely identify the secret and the content type is used to parse the
// secret data blob.
type Secret struct {
	Namespace   string    `json:"namespace,omitempty"`
	Name        string    `json:"name"`
	ContentType string    `json:"content_type"`
	Data        []byte    `json:"data"`
	Created     time.Time `json:"created,omitempty"`
}

// Iterator allows listing secrets from the key management store or database.
type Iterator interface {
	io.Closer
	Next() bool
	Err() error
	Secret() *Secret
}

type NamespaceFactory func(name, contentType string, data []byte) *Secret

func WithNamespace(namespace string) NamespaceFactory {
	return func(name, contentType string, data []byte) *Secret {
		return &Secret{
			Namespace:   namespace,
			Name:        name,
			ContentType: contentType,
			Data:        data,
		}
	}
}
