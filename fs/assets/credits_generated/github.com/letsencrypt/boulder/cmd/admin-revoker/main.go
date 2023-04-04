package notmain

import (
	"bufio"
	"context"
	"crypto"
	"crypto/sha256"
	"crypto/x509"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/user"
	"sort"
	"strconv"
	"sync"

	"github.com/jmhodges/clock"
	"github.com/letsencrypt/boulder/cmd"
	"github.com/letsencrypt/boulder/core"
	"github.com/letsencrypt/boulder/db"
	berrors "github.com/letsencrypt/boulder/errors"
	"github.com/letsencrypt/boulder/features"
	bgrpc "github.com/letsencrypt/boulder/grpc"
	blog "github.com/letsencrypt/boulder/log"
	"github.com/letsencrypt/boulder/metrics"
	"github.com/letsencrypt/boulder/privatekey"
	rapb "github.com/letsencrypt/boulder/ra/proto"
	"github.com/letsencrypt/boulder/revocation"
	"github.com/letsencrypt/boulder/sa"
	sapb "github.com/letsencrypt/boulder/sa/proto"
)

const usageString = `
usage:
  list-reasons           -config <path>
  serial-revoke          -config <path> <serial>           <reason-code>
  batched-serial-revoke  -config <path> <serial-file-path> <reason-code>   <parallelism>
  incident-table-revoke  -config <path> <table-name>       <reason-code>   <parallelism>
  reg-revoke             -config <path> <registration-id>  <reason-code>
  private-key-block      -config <path> -comment="<string>" -dry-run=<bool>    <priv-key-path>
  private-key-revoke     -config <path> -comment="<string>" -dry-run=<bool>    <priv-key-path>


descriptions:
  list-reasons           List all revocation reason codes
  serial-revoke          Revoke a single certificate by the hex serial number
  batched-serial-revoke  Revokes all certificates contained in a file of hex serial numbers
  incident-table-revoke  Revokes all certificates in the provided incident table
  reg-revoke             Revoke all certificates associated with a registration ID
  private-key-block      Adds the SPKI hash, derived from the provided private key, to the
                         blocked keys table. <priv-key-path> is expected to be the path
                         to a PEM formatted file containing an RSA or ECDSA private key
  private-key-revoke     Revokes all certificates matching the SPKI hash derived from the
                         provided private key. Then adds the hash to the blocked keys
                         table. <priv-key-path> is expected to be the path to a PEM
                         formatted file containing an RSA or ECDSA private key

flags:
  all:
    -config              File path to the configuration file for this service (required)

  private-key-block | private-key-revoke:
    -dry-run             true (default): only queries for affected certificates. false: will
                         perform the requested block or revoke action. Only implemented for
                         private-key-block and private-key-revoke.
    -comment             Comment to include in the blocked keys table entry. (default: "")
`

type Config struct {
	Revoker struct {
		DB cmd.DBConfig
		// Similarly, the Revoker needs a TLSConfig to set up its GRPC client
		// certs, but doesn't get the TLS field from ServiceConfig, so declares
		// its own.
		TLS cmd.TLSConfig

		RAService *cmd.GRPCClientConfig
		SAService *cmd.GRPCClientConfig

		Features map[string]bool
	}

	Syslog cmd.SyslogConfig
}

type revoker struct {
	rac   rapb.RegistrationAuthorityClient
	sac   sapb.StorageAuthorityClient
	dbMap *db.WrappedMap
	clk   clock.Clock
	log   blog.Logger
}

func newRevoker(c Config) *revoker {
	logger := cmd.NewLogger(c.Syslog)

	tlsConfig, err := c.Revoker.TLS.Load()
	cmd.FailOnError(err, "TLS config")

	clk := cmd.Clock()

	raConn, err := bgrpc.ClientSetup(c.Revoker.RAService, tlsConfig, metrics.NoopRegisterer, clk)
	cmd.FailOnError(err, "Failed to load credentials and create gRPC connection to RA")
	rac := rapb.NewRegistrationAuthorityClient(raConn)

	dbMap, err := sa.InitWrappedDb(c.Revoker.DB, nil, logger)
	cmd.FailOnError(err, "While initializing dbMap")

	saConn, err := bgrpc.ClientSetup(c.Revoker.SAService, tlsConfig, metrics.NoopRegisterer, clk)
	cmd.FailOnError(err, "Failed to load credentials and create gRPC connection to SA")
	sac := sapb.NewStorageAuthorityClient(saConn)

	return &revoker{
		rac:   rac,
		sac:   sac,
		dbMap: dbMap,
		clk:   clk,
		log:   logger,
	}
}

func (r *revoker) revokeCertificate(ctx context.Context, certObj core.Certificate, reasonCode revocation.Reason, skipBlockKey bool) error {
	if reasonCode < 0 || reasonCode == 7 || reasonCode > 10 {
		panic(fmt.Sprintf("Invalid reason code: %d", reasonCode))
	}
	u, err := user.Current()
	if err != nil {
		return err
	}

	var req *rapb.AdministrativelyRevokeCertificateRequest
	if certObj.DER != nil {
		cert, err := x509.ParseCertificate(certObj.DER)
		if err != nil {
			return err
		}
		req = &rapb.AdministrativelyRevokeCertificateRequest{
			Cert:         cert.Raw,
			Code:         int64(reasonCode),
			AdminName:    u.Username,
			SkipBlockKey: skipBlockKey,
		}
	} else {
		req = &rapb.AdministrativelyRevokeCertificateRequest{
			Serial:       certObj.Serial,
			Code:         int64(reasonCode),
			AdminName:    u.Username,
			SkipBlockKey: skipBlockKey,
		}
	}
	_, err = r.rac.AdministrativelyRevokeCertificate(ctx, req)
	if err != nil {
		return err
	}
	r.log.Infof("Revoked certificate %s with reason '%s'", certObj.Serial, revocation.ReasonToString[reasonCode])
	return nil
}

func (r *revoker) revokeBySerial(ctx context.Context, serial string, reasonCode revocation.Reason, skipBlockKey bool) error {
	certObj, err := sa.SelectPrecertificate(r.dbMap, serial)
	if err != nil {
		if db.IsNoRows(err) {
			return berrors.NotFoundError("precertificate with serial %q not found", serial)
		}
		return err
	}
	return r.revokeCertificate(ctx, certObj, reasonCode, skipBlockKey)
}

func (r *revoker) revokeSerialBatchFile(ctx context.Context, serialPath string, reasonCode revocation.Reason, parallelism int) error {
	file, err := os.Open(serialPath)
	if err != nil {
		return err
	}

	scanner := bufio.NewScanner(file)
	if err != nil {
		return err
	}

	wg := new(sync.WaitGroup)
	work := make(chan string, parallelism)
	for i := 0; i < parallelism; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for serial := range work {
				// handle newlines gracefully
				if serial == "" {
					continue
				}
				err := r.revokeBySerial(ctx, serial, reasonCode, false)
				if err != nil {
					r.log.Errf("failed to revoke %q: %s", serial, err)
				}
			}
		}()
	}

	for scanner.Scan() {
		serial := scanner.Text()
		if serial == "" {
			continue
		}
		work <- serial
	}
	close(work)
	wg.Wait()

	return nil
}

func (r *revoker) revokeIncidentTableSerials(ctx context.Context, tableName string, reasonCode revocation.Reason, parallelism int) error {
	wg := new(sync.WaitGroup)
	work := make(chan string, parallelism)
	for i := 0; i < parallelism; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for serial := range work {
				err := r.revokeBySerial(ctx, serial, reasonCode, false)
				if err != nil {
					r.log.Errf("failed to revoke %q: %s", serial, err)
				}
			}
		}()
	}

	stream, err := r.sac.SerialsForIncident(ctx, &sapb.SerialsForIncidentRequest{IncidentTable: tableName})
	if err != nil {
		return fmt.Errorf("setting up stream of serials from incident table %q: %s", tableName, err)
	}

	var atLeastOne bool
	for {
		is, err := stream.Recv()
		if err != nil {
			if err == io.EOF {
				break
			}
			return fmt.Errorf("streaming serials from incident table %q: %s", tableName, err)
		}
		atLeastOne = true
		work <- is.Serial
	}
	if !atLeastOne {
		r.log.AuditInfof("No serials found in incident table %q", tableName)
	}
	close(work)
	wg.Wait()

	return nil
}

func (r *revoker) revokeByReg(ctx context.Context, regID int64, reasonCode revocation.Reason) error {
	_, err := r.sac.GetRegistration(ctx, &sapb.RegistrationID{Id: regID})
	if err != nil {
		return fmt.Errorf("couldn't fetch registration: %w", err)
	}

	certObjs, err := sa.SelectPrecertificates(r.dbMap, "WHERE registrationID = :regID", map[string]interface{}{"regID": regID})
	if err != nil {
		return err
	}
	for _, certObj := range certObjs {
		err = r.revokeCertificate(ctx, certObj.Certificate, reasonCode, false)
		if err != nil {
			return err
		}
	}
	return nil
}

func (r *revoker) revokeMalformedBySerial(ctx context.Context, serial string, reasonCode revocation.Reason) error {
	return r.revokeCertificate(ctx, core.Certificate{Serial: serial}, reasonCode, false)
}

// blockByPrivateKey blocks future issuance for certificates with a a public key
// matching the SubjectPublicKeyInfo hash generated from the PublicKey embedded
// in privateKey. The embedded PublicKey will be verified as an actual match for
// the provided private key before any blocking takes place. This method does
// not revoke any certificates directly. However, 'bad-key-revoker', which
// references the 'blockedKeys' table, will eventually revoke certificates with
// a matching SPKI hash.
func (r *revoker) blockByPrivateKey(ctx context.Context, comment string, privateKey string) error {
	_, publicKey, err := privatekey.Load(privateKey)
	if err != nil {
		return err
	}

	spkiHash, err := getPublicKeySPKIHash(publicKey)
	if err != nil {
		return err
	}

	u, err := user.Current()
	if err != nil {
		return err
	}

	dbcomment := fmt.Sprintf("%s: %s", u.Username, comment)

	req := &sapb.AddBlockedKeyRequest{
		KeyHash:   spkiHash,
		Added:     r.clk.Now().UnixNano(),
		Source:    "admin-revoker",
		Comment:   dbcomment,
		RevokedBy: 0,
	}

	_, err = r.sac.AddBlockedKey(ctx, req)
	if err != nil {
		return err
	}
	return nil
}

// revokeByPrivateKey revokes all certificates with a public key matching the
// SubjectPublicKeyInfo hash generated from the PublicKey embedded in
// privateKey. The embedded PublicKey will be verified as an actual match for the
// provided private key before any revocation takes place. The provided key will
// not be added to the 'blockedKeys' table. This is done to avoid a race between
// 'admin-revoker' and 'bad-key-revoker'. You MUST call blockByPrivateKey after
// calling this function, on pain of violating the BRs.
func (r *revoker) revokeByPrivateKey(ctx context.Context, privateKey string) error {
	_, publicKey, err := privatekey.Load(privateKey)
	if err != nil {
		return err
	}

	spkiHash, err := getPublicKeySPKIHash(publicKey)
	if err != nil {
		return err
	}

	matches, err := r.getCertsMatchingSPKIHash(spkiHash)
	if err != nil {
		return err
	}

	for i, match := range matches {
		resp, err := r.sac.GetCertificateStatus(ctx, &sapb.Serial{Serial: match})
		if err != nil {
			return fmt.Errorf(
				"failed to get status for serial %q. Entry %d of %d affected certificates: %w",
				match,
				(i + 1),
				len(matches),
				err,
			)
		}

		if resp.Status != string(core.OCSPStatusGood) {
			r.log.AuditInfof("serial %q is already revoked, skipping", match)
			continue
		}

		err = r.revokeBySerial(ctx, match, revocation.Reason(1), true)
		if err != nil {
			return fmt.Errorf(
				"failed to revoke serial %q. Entry %d of %d affected certificates: %w",
				match,
				(i + 1),
				len(matches),
				err,
			)
		}
	}
	return nil
}

func (r *revoker) spkiHashInBlockedKeys(spkiHash []byte) (bool, error) {
	var count int
	err := r.dbMap.SelectOne(&count, "SELECT COUNT(*) as count FROM blockedKeys WHERE keyHash = ?", spkiHash)
	if err != nil {
		return false, err
	}

	if count > 0 {
		return true, nil
	}
	return false, nil
}

func (r *revoker) countCertsMatchingSPKIHash(spkiHash []byte) (int, error) {
	var count int
	err := r.dbMap.SelectOne(&count, "SELECT COUNT(*) as count FROM keyHashToSerial WHERE keyHash = ?", spkiHash)
	if err != nil {
		return 0, err
	}
	return count, nil
}

// TODO(#5899) Use an non-wrapped sql.Db client to iterate over results and
// return them on a channel.
func (r *revoker) getCertsMatchingSPKIHash(spkiHash []byte) ([]string, error) {
	var h []string
	_, err := r.dbMap.Select(&h, "SELECT certSerial FROM keyHashToSerial WHERE keyHash = ?", spkiHash)
	if err != nil {
		if db.IsNoRows(err) {
			return nil, berrors.NotFoundError("no certificates with a matching SPKI hash were found")
		}
		return nil, err
	}
	return h, nil
}

// This abstraction is needed so that we can use sort.Sort below
type revocationCodes []revocation.Reason

func (rc revocationCodes) Len() int           { return len(rc) }
func (rc revocationCodes) Less(i, j int) bool { return rc[i] < rc[j] }
func (rc revocationCodes) Swap(i, j int)      { rc[i], rc[j] = rc[j], rc[i] }

func privateKeyBlock(r *revoker, dryRun bool, comment string, count int, spkiHash []byte, keyPath string) error {
	keyExists, err := r.spkiHashInBlockedKeys(spkiHash)
	if err != nil {
		return fmt.Errorf("while checking if the provided key already exists in the 'blockedKeys' table: %s", err)
	}
	if keyExists {
		return errors.New("the provided key already exists in the 'blockedKeys' table")
	}

	if dryRun {
		r.log.AuditInfof(
			"To block issuance for this key and revoke %d certificates via bad-key-revoker, run with -dry-run=false",
			count,
		)
		r.log.AuditInfo("No keys were blocked or certificates revoked, exiting...")
		return nil
	}

	r.log.AuditInfo("Attempting to block issuance for the provided key")
	err = r.blockByPrivateKey(context.Background(), comment, keyPath)
	if err != nil {
		return fmt.Errorf("while attempting to block issuance for the provided key: %s", err)
	}
	r.log.AuditInfo("Issuance for the provided key has been successfully blocked, exiting...")
	return nil
}

func privateKeyRevoke(r *revoker, dryRun bool, comment string, count int, keyPath string) error {
	if dryRun {
		r.log.AuditInfof(
			"To immediately revoke %d certificates and block issuance for this key, run with -dry-run=false",
			count,
		)
		r.log.AuditInfo("No keys were blocked or certificates revoked, exiting...")
		return nil
	}

	if count <= 0 {
		// Do not revoke.
		return nil
	}

	// Revoke certificates.
	r.log.AuditInfof("Attempting to revoke %d certificates", count)
	err := r.revokeByPrivateKey(context.Background(), keyPath)
	if err != nil {
		return fmt.Errorf("while attempting to revoke certificates for the provided key: %s", err)
	}
	r.log.AuditInfo("All certificates matching using the provided key have been successfully")

	// Block future issuance.
	r.log.AuditInfo("Attempting to block issuance for the provided key")
	err = r.blockByPrivateKey(context.Background(), comment, keyPath)
	if err != nil {
		return fmt.Errorf("while attempting to block issuance for the provided key: %s", err)
	}
	r.log.AuditInfo("All certificates have been successfully revoked and issuance blocked, exiting...")
	return nil
}

// getPublicKeySPKIHash returns a hash of the SubjectPublicKeyInfo for the
// provided public key.
func getPublicKeySPKIHash(pubKey crypto.PublicKey) ([]byte, error) {
	rawSubjectPublicKeyInfo, err := x509.MarshalPKIXPublicKey(pubKey)
	if err != nil {
		return nil, err
	}
	spkiHash := sha256.Sum256(rawSubjectPublicKeyInfo)
	return spkiHash[:], nil
}

func main() {
	usage := func() {
		fmt.Fprint(os.Stderr, usageString)
		os.Exit(1)
	}
	if len(os.Args) <= 2 {
		usage()
	}

	command := os.Args[1]
	flagSet := flag.NewFlagSet(command, flag.ContinueOnError)
	configFile := flagSet.String("config", "", "File path to the configuration file for this service")
	dryRun := flagSet.Bool(
		"dry-run",
		true,
		"true (default): only queries for affected certificates. false: will perform the requested block or revoke action",
	)
	comment := flagSet.String("comment", "", "Comment to include in the blocked key database entry ")
	err := flagSet.Parse(os.Args[2:])
	cmd.FailOnError(err, "Error parsing flagset")

	if *configFile == "" {
		usage()
	}

	var c Config
	err = cmd.ReadConfigFile(*configFile, &c)
	cmd.FailOnError(err, "Reading JSON config file into config structure")
	err = features.Set(c.Revoker.Features)
	cmd.FailOnError(err, "Failed to set feature flags")

	ctx := context.Background()
	r := newRevoker(c)
	defer r.log.AuditPanic()

	args := flagSet.Args()
	switch {
	case command == "serial-revoke" && len(args) == 2:
		// 1: serial,  2: reasonCode
		serial := args[0]
		reasonCode, err := strconv.Atoi(args[1])
		cmd.FailOnError(err, "Reason code argument must be an integer")

		err = r.revokeBySerial(ctx, serial, revocation.Reason(reasonCode), false)
		cmd.FailOnError(err, "Couldn't revoke certificate by serial")

	case command == "batched-serial-revoke" && len(args) == 3:
		// 1: serial file path,  2: reasonCode, 3: parallelism
		serialPath := args[0]
		reasonCode, err := strconv.Atoi(args[1])
		cmd.FailOnError(err, "Reason code argument must be an integer")
		parallelism, err := strconv.Atoi(args[2])
		cmd.FailOnError(err, "parallelism argument must be an integer")
		if parallelism < 1 {
			cmd.Fail("parallelism argument must be >= 1")
		}

		err = r.revokeSerialBatchFile(ctx, serialPath, revocation.Reason(reasonCode), parallelism)
		cmd.FailOnError(err, "Batch revocation failed")

	case command == "reg-revoke" && len(args) == 2:
		// 1: registration ID,  2: reasonCode
		regID, err := strconv.ParseInt(args[0], 10, 64)
		cmd.FailOnError(err, "Registration ID argument must be an integer")
		reasonCode, err := strconv.Atoi(args[1])
		cmd.FailOnError(err, "Reason code argument must be an integer")

		err = r.revokeByReg(ctx, regID, revocation.Reason(reasonCode))
		cmd.FailOnError(err, "Couldn't revoke certificate by registration")

	case command == "malformed-revoke" && len(args) == 3:
		// 1: serial, 2: reasonCode
		serial := args[0]
		reasonCode, err := strconv.Atoi(args[1])
		cmd.FailOnError(err, "Reason code argument must be an integer")

		err = r.revokeMalformedBySerial(ctx, serial, revocation.Reason(reasonCode))
		cmd.FailOnError(err, "Couldn't revoke certificate by serial")

	case command == "list-reasons":
		var codes revocationCodes
		for k := range revocation.ReasonToString {
			codes = append(codes, k)
		}
		sort.Sort(codes)
		fmt.Printf("Revocation reason codes\n-----------------------\n\n")
		for _, k := range codes {
			fmt.Printf("%d: %s\n", k, revocation.ReasonToString[k])
		}

	case (command == "private-key-block" || command == "private-key-revoke") && len(args) == 1:
		// 1: keyPath
		keyPath := args[0]

		_, publicKey, err := privatekey.Load(keyPath)
		cmd.FailOnError(err, "Failed to load the provided private key")
		r.log.AuditInfo("The provided private key has been successfully verified")

		spkiHash, err := getPublicKeySPKIHash(publicKey)
		cmd.FailOnError(err, "While obtaining the SPKI hash for the provided key")

		count, err := r.countCertsMatchingSPKIHash(spkiHash)
		cmd.FailOnError(err, "While retrieving a count of certificates matching the provided key")
		r.log.AuditInfof("Found %d certificates matching the provided key", count)

		if command == "private-key-block" {
			err := privateKeyBlock(r, *dryRun, *comment, count, spkiHash, keyPath)
			cmd.FailOnError(err, "")
		}

		if command == "private-key-revoke" {
			err := privateKeyRevoke(r, *dryRun, *comment, count, keyPath)
			cmd.FailOnError(err, "")
		}

	case command == "incident-table-revoke" && len(args) == 3:
		// 1: tableName, 2: reasonCode, 3: parallelism
		tableName := args[0]

		reasonCode, err := strconv.Atoi(args[1])
		cmd.FailOnError(err, "Reason code argument must be an integer")

		parallelism, err := strconv.Atoi(args[2])
		cmd.FailOnError(err, "parallelism argument must be an integer")
		if parallelism < 1 {
			cmd.Fail("parallelism argument must be >= 1")
		}
		r.revokeIncidentTableSerials(ctx, tableName, revocation.Reason(reasonCode), parallelism)

	default:
		usage()
	}
}

func init() {
	cmd.RegisterCommand("admin-revoker", main)
}
