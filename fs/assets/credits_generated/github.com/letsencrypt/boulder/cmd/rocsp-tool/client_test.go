package notmain

import (
	"context"
	"fmt"
	"math/big"
	"testing"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/jmhodges/clock"
	capb "github.com/letsencrypt/boulder/ca/proto"
	"github.com/letsencrypt/boulder/cmd"
	"github.com/letsencrypt/boulder/core"
	blog "github.com/letsencrypt/boulder/log"
	"github.com/letsencrypt/boulder/metrics"
	"github.com/letsencrypt/boulder/rocsp"
	"github.com/letsencrypt/boulder/sa"
	"github.com/letsencrypt/boulder/test"
	"github.com/letsencrypt/boulder/test/vars"
	"golang.org/x/crypto/ocsp"
	"google.golang.org/grpc"
)

func makeClient() (*rocsp.RWClient, clock.Clock) {
	CACertFile := "../../test/redis-tls/minica.pem"
	CertFile := "../../test/redis-tls/boulder/cert.pem"
	KeyFile := "../../test/redis-tls/boulder/key.pem"
	tlsConfig := cmd.TLSConfig{
		CACertFile: &CACertFile,
		CertFile:   &CertFile,
		KeyFile:    &KeyFile,
	}
	tlsConfig2, err := tlsConfig.Load()
	if err != nil {
		panic(err)
	}

	rdb := redis.NewRing(&redis.RingOptions{
		Addrs: map[string]string{
			"shard1": "10.33.33.2:4218",
			"shard2": "10.33.33.3:4218",
		},
		Username:  "unittest-rw",
		Password:  "824968fa490f4ecec1e52d5e34916bdb60d45f8d",
		TLSConfig: tlsConfig2,
	})
	clk := clock.NewFake()
	return rocsp.NewWritingClient(rdb, 500*time.Millisecond, clk, metrics.NoopRegisterer), clk
}

func TestGetStartingID(t *testing.T) {
	clk := clock.NewFake()
	dbMap, err := sa.NewDbMap(vars.DBConnSAFullPerms, sa.DbSettings{})
	test.AssertNotError(t, err, "failed setting up db client")
	defer test.ResetBoulderTestDatabase(t)()
	sa.SetSQLDebug(dbMap, blog.Get())

	cs := core.CertificateStatus{
		Serial:   "1337",
		NotAfter: clk.Now().Add(12 * time.Hour),
	}
	err = dbMap.Insert(&cs)
	test.AssertNotError(t, err, "inserting certificate status")
	firstID := cs.ID

	cs = core.CertificateStatus{
		Serial:   "1338",
		NotAfter: clk.Now().Add(36 * time.Hour),
	}
	err = dbMap.Insert(&cs)
	test.AssertNotError(t, err, "inserting certificate status")
	secondID := cs.ID
	t.Logf("first ID %d, second ID %d", firstID, secondID)

	clk.Sleep(48 * time.Hour)

	startingID, err := getStartingID(context.Background(), clk, dbMap)
	test.AssertNotError(t, err, "getting starting ID")

	test.AssertEquals(t, startingID, secondID)
}

func TestStoreResponse(t *testing.T) {
	redisClient, clk := makeClient()

	issuer, err := core.LoadCert("../../test/hierarchy/int-e1.cert.pem")
	test.AssertNotError(t, err, "loading int-e1")

	issuerKey, err := test.LoadSigner("../../test/hierarchy/int-e1.key.pem")
	test.AssertNotError(t, err, "loading int-e1 key ")
	response, err := ocsp.CreateResponse(issuer, issuer, ocsp.Response{
		SerialNumber: big.NewInt(1337),
		Status:       0,
		ThisUpdate:   clk.Now(),
		NextUpdate:   clk.Now().Add(time.Hour),
	}, issuerKey)
	test.AssertNotError(t, err, "creating OCSP response")

	cl := client{
		redis:         redisClient,
		db:            nil,
		ocspGenerator: nil,
		clk:           clk,
		logger:        blog.NewMock(),
	}

	err = cl.storeResponse(context.Background(), response)
	test.AssertNotError(t, err, "storing response")
}

type mockOCSPGenerator struct{}

func (mog mockOCSPGenerator) GenerateOCSP(ctx context.Context, in *capb.GenerateOCSPRequest, opts ...grpc.CallOption) (*capb.OCSPResponse, error) {
	return &capb.OCSPResponse{
		Response: []byte("phthpbt"),
	}, nil

}

func TestLoadFromDB(t *testing.T) {
	redisClient, clk := makeClient()

	dbMap, err := sa.NewDbMap(vars.DBConnSA, sa.DbSettings{})
	if err != nil {
		t.Fatalf("Failed to create dbMap: %s", err)
	}

	defer test.ResetBoulderTestDatabase(t)

	for i := 0; i < 100; i++ {
		err = dbMap.Insert(&core.CertificateStatus{
			Serial:          fmt.Sprintf("%036x", i),
			OCSPResponse:    []byte("phthpbt"),
			NotAfter:        clk.Now().Add(200 * time.Hour),
			OCSPLastUpdated: clk.Now(),
		})
		if err != nil {
			t.Fatalf("Failed to insert certificateStatus: %s", err)
		}
	}

	rocspToolClient := client{
		redis:         redisClient,
		db:            dbMap,
		ocspGenerator: mockOCSPGenerator{},
		clk:           clk,
		scanBatchSize: 10,
		logger:        blog.NewMock(),
	}

	speed := ProcessingSpeed{
		RowsPerSecond: 10000,
		ParallelSigns: 100,
	}

	err = rocspToolClient.loadFromDB(context.Background(), speed, 0)
	if err != nil {
		t.Fatalf("loading from DB: %s", err)
	}
}
