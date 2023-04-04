# crl/x509

The contents of this directory were forked from the Go stdlib's [crypto/x509](https://pkg.go.dev/crypto/x509) package at version [go1.19beta1](https://cs.opensource.google/go/go/+/refs/tags/go1.19beta1:src/crypto/x509/).

We created this fork in order to have greater agility and control over our CRL generation capabilities, including but not limited to:

* enforce that CRL Numbers are not more than 20 octets;
* raising the ReasonCode from an extension to a top-level field in each CRL Entry; and
* adding a streaming API which computes the CRL signature without having to hold all of the bytes of the CRL in memory at the same time.

The vast majority of our edits are in crl.go; the other files here are forked only to provide access to the private helper methods used internally by the CRL code.

We intend to upstream as many of our changes here as the Go maintainers will accept.
