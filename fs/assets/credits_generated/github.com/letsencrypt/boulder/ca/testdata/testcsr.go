// Hack up the x509.CertificateRequest in here, run `go run testcsr.go`, and a
// DER-encoded CertificateRequest will be printed to stdout.
package main

import (
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"log"
	"os"
)

// A 2048-bit RSA private key
var rsaPrivateKey = `-----BEGIN RSA PRIVATE KEY-----
MIIEowIBAAKCAQEA5cpXqfCaUDD+hf93j5jxbrhK4jrJAzfAEjeZj/Lx5Rv/7eEO
uhS2DdCU2is82vR6yJ7EidUYVz/nUAjSTP7JIEsbyvfsfACABbqRyGltHlJnULVH
y/EMjt9xKZf17T8tOLHVUEAJTxsvjKn4TMIQJTNrAqm/lNrUXmCIR41Go+3RBGC6
YdAKEwcZMCzrjQGF06mC6/6xMmYMSMd6+VQRFIPpuPK/6BBp1Tgju2LleRC5uatj
QcFOoilGkfh1RnZp3GJ7q58KaqHiPmjl31rkY5vS3LP7yfU5TRBcxCSG8l8LKuRt
MArkbTEtj3PkDjbipL/SkLrZ28e5w9Egl4g1MwIDAQABAoIBABZqY5zPPK5f6SQ3
JHmciMitL5jb9SncMV9VjyRMpa4cyh1xW9dpF81HMI4Ls7cELEoPuspbQDGaqTzU
b3dVT1dYHFDzWF1MSzDD3162cg+IKE3mMSfCzt/NCiPtj+7hv86NAmr+pCnUVBIb
rn4GXD7UwjaTSn4Bzr+aGREpxd9Nr0JdNQwxVHZ75A92vTihCfaXyMCjhW3JEpF9
N89XehgidoGgtUxxeeb+WsO3nvVBpLv/HDxMTx/IDzvSA5nLlYMcqVzb7IJoeAQu
og0WJKlniYzvIdoQ6/hGydAW5sKd0qWh0JPYs7uLKAWrdAWvrFAp7//fYKVamalU
8pUu/WkCgYEA+tcTQ3qTnVh41O9YeM/7NULpIkuCAlR+PBRky294zho9nGQIPdaW
VNvyqqjLaHaXJVokYHbU4hDk6RbrhoWVd4Po/5g9cUkT1f6nrdZGRkg4XOCzHWvV
Yrqh3eYYX4bdiH5EhB78m0rrbjHfd7SF3cdYNzOUS2kJvCInYC6zPx8CgYEA6oRr
UhZFuoqRsEb28ELM8sHvdIMA/C3aWCu+nUGQ4gHSEb4uvuOD/7tQNuCaBioiXVPM
/4hjk9jHJcjYf5l33ANqIP7JiYAt4rzTWXF3iS6kQOhQhjksSlSnWqw0Uu1DtlpG
rzeG1ZkBuwH7Bx0yj4sGSz5sAvyF44aRsE6AC20CgYEArafWO0ISDb1hMbFdo44B
ELd45Pg3UluiZP+NZFWQ4cbC3pFWL1FvE+KNll5zK6fmLcLBKlM6QCOIBmKKvb+f
YXVeCg0ghFweMmkxNqUAU8nN02bwOa8ctFQWmaOhPgkFN2iLEJjPMsdkRA6c8ad1
gbtvNBAuWyKlzawrbGgISesCgYBkGEjGLINubx5noqJbQee/5U6S6CdPezKqV2Fw
NT/ldul2cTn6d5krWYOPKKYU437vXokst8XooKm/Us41CAfEfCCcHKNgcLklAXsj
ve5LOwEYQw+7ekORJjiX1tAuZN51wmpQ9t4x5LB8ZQgDrU6bPbdd/jKTw7xRtGoS
Wi8EsQKBgG8iGy3+kVBIjKHxrN5jVs3vj/l/fQL0WRMLCMmVuDBfsKyy3f9n8R1B
/KdwoyQFwsLOyr5vAjiDgpFurXQbVyH4GDFiJGS1gb6MNcinwSTpsbOLLV7zgibX
A2NgiQ+UeWMia16dZVd6gGDlY3lQpeyLdsdDd+YppNfy9vedjbvT
-----END RSA PRIVATE KEY-----`

// NISTP256 ECDSA private key
var ecdsaPrivateKey = `-----BEGIN EC PRIVATE KEY-----
MHcCAQEEIKwK8ik0Zgw26bWaGuNYa/QAtCDRwpOPS5FIhbwuFqWuoAoGCCqGSM49
AwEHoUQDQgAEfkxXCNEy4/zfwQ4arciDYQql7/+ftYvf51JTLCJAFu8kWKvNBENT
X8ays994FANu2VsJTF5Ud5JPYWHT87hjAA==
-----END EC PRIVATE KEY-----`

func main() {
	block, _ := pem.Decode([]byte(rsaPrivateKey))
	rsaPriv, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		log.Fatalf("Failed to parse private key: %s", err)
	}

	req := &x509.CertificateRequest{
		Subject: pkix.Name{
			CommonName: "CapiTalizedLetters.com",
		},
		DNSNames: []string{
			"moreCAPs.com",
			"morecaps.com",
			"evenMOREcaps.com",
			"Capitalizedletters.COM",
		},
	}
	csr, err := x509.CreateCertificateRequest(rand.Reader, req, rsaPriv)
	if err != nil {
		log.Fatalf("unable to create CSR: %s", err)
	}
	_, err = os.Stdout.Write(csr)
	if err != nil {
		log.Fatalf("unable to write to stdout: %s", err)
	}
}
