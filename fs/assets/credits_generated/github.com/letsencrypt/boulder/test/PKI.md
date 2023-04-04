Boulder's test environment contains four separate PKIs: one for WFE (simulating
the public WebPKI), one for gRPC (simulating an internal PKI), one for Redis
(simulating another internal PKI), and one for issuance. In live deployment,
the issuance PKI is a member of the global WebPKI, but we simulate them as
separate PKIs here.

The PKI used by WFE is under test/wfe-tls/, with test/wfe-tls/minica.pem serving
as the root. There are no intermediates. Setting test/wfe-tls/minica.pem as
a trusted root is sufficient to connect to the WFE over HTTPS. Currently there
is only one end-entity certificate in this PKI, and that's all we expect to
need. To validate HTTPS connections to a test-mode WFE in Python, set the environment
variable `REQUESTS_CA_BUNDLE`. For Node, set `NODE_EXTRA_CA_CERTS`. These
variables should be set to `/path/to/boulder/test/wfe-tls/minica.pem` (but only
in testing environments!). Note that in the Python case, setting this environment
variable may break HTTPS connections to non-WFE destinations. If causes problems
for you, you may need to create a combined bundle containing
`test/wfe-tls/minica.pem` in addition to the other relevant root certificates.

The gRPC PKI is under test/grpc-creds/. Each Boulder component has two
hostnames, each resolving to a different IP address in our test environment,
plus a third hostname that resolves to both IP addresses. Certificates for these
components contain all three hostnames, both test IP addresses, and are stored
under test/grpc-creds/SERVICE.boulder.

To issue new certificates in the WFE or gRPC PKI, install
https://github.com/jsha/minica, cd to the directory containing minica.pem for
the PKI you want to issue in, and run `minica -domains YOUR_NEW_DOMAINs`. If
you're updating the gRPC PKI, please make sure to update grpc-creds/generate.sh.

The issuance PKI consists of a root and two intermediates. These certificates and
their keys are generated using the `ceremony` tool during integration testing. The
private keys are stored in SoftHSM, and the public keys and certificates are
written out to /tmp. The test/test-*.key-pkcs11.json files contain the PKCS#11
variables needed to access these keys from boulder.

The following files are used in the generation/access of the root and intermediates.

Root:
   * test/test-root.key-pkcs11.json
   * test/cert-ceremonies/root-ceremony-rsa.yaml

Intermediate 1 `CA intermediate (RSA) A`:
   * test/test-ca.key-pkcs11.json
   * test/cert-ceremonies/intermediate-key-ceremony-rsa.yaml
   * test/cert-ceremonies/intermediate-ceremony-rsa-a.yaml

Intermediate 2 `CA intermediate (RSA) B`:
   * test/test-ca.key-pkcs11.json
   * test/cert-ceremonies/intermediate-key-ceremony-rsa.yaml
   * test/cert-ceremonies/intermediate-ceremony-rsa-b.yaml

Certificate test-example.pem, together with test-example.key are self-signed
certs used in integration tests. They were generated using:
   openssl req -x509 -newkey rsa:4096 -keyout test-example.key -out test-example.pem -days 36500 -nodes  -subj "/CN=www.example.com"
