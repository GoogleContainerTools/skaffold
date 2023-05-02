package remote

import (
	"github.com/google/go-containerregistry/pkg/name"
)

type DigestIdentifier struct {
	Digest name.Digest
}

func (d DigestIdentifier) String() string {
	return d.Digest.String()
}
