package notmain

import (
	"bytes"
	"context"
	"crypto/x509"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"math"
	netmail "net/mail"
	"net/url"
	"os"
	"sort"
	"strings"
	"sync"
	"text/template"
	"time"

	"github.com/honeycombio/beeline-go"
	"github.com/jmhodges/clock"
	"google.golang.org/grpc"

	"github.com/letsencrypt/boulder/cmd"
	"github.com/letsencrypt/boulder/core"
	corepb "github.com/letsencrypt/boulder/core/proto"
	"github.com/letsencrypt/boulder/db"
	"github.com/letsencrypt/boulder/features"
	bgrpc "github.com/letsencrypt/boulder/grpc"
	blog "github.com/letsencrypt/boulder/log"
	bmail "github.com/letsencrypt/boulder/mail"
	"github.com/letsencrypt/boulder/metrics"
	"github.com/letsencrypt/boulder/sa"
	sapb "github.com/letsencrypt/boulder/sa/proto"
	"github.com/prometheus/client_golang/prometheus"
)

const (
	defaultExpirationSubject = "Let's Encrypt certificate expiration notice for domain {{.ExpirationSubject}}"
)

type regStore interface {
	GetRegistration(ctx context.Context, req *sapb.RegistrationID, _ ...grpc.CallOption) (*corepb.Registration, error)
}

type mailer struct {
	log             blog.Logger
	dbMap           *db.WrappedMap
	rs              regStore
	mailer          bmail.Mailer
	emailTemplate   *template.Template
	subjectTemplate *template.Template
	nagTimes        []time.Duration
	parallelSends   uint
	limit           int
	// Maximum number of rows to update in a single SQL UPDATE statement.
	updateChunkSize int
	clk             clock.Clock
	stats           mailerStats
}

type certDERWithRegID struct {
	DER   core.CertDER
	RegID int64
}

type mailerStats struct {
	sendDelay                         *prometheus.GaugeVec
	sendDelayHistogram                *prometheus.HistogramVec
	nagsAtCapacity                    *prometheus.GaugeVec
	errorCount                        *prometheus.CounterVec
	sendLatency                       prometheus.Histogram
	processingLatency                 prometheus.Histogram
	certificatesExamined              prometheus.Counter
	certificatesAlreadyRenewed        prometheus.Counter
	certificatesPerAccountNeedingMail prometheus.Histogram
}

func (m *mailer) sendNags(conn bmail.Conn, contacts []string, certs []*x509.Certificate) error {
	if len(certs) == 0 {
		return errors.New("no certs given to send nags for")
	}
	emails := []string{}
	for _, contact := range contacts {
		parsed, err := url.Parse(contact)
		if err != nil {
			m.log.AuditErrf("parsing contact email %s: %s", contact, err)
			continue
		}
		if parsed.Scheme == "mailto" {
			emails = append(emails, parsed.Opaque)
		}
	}
	if len(emails) == 0 {
		return nil
	}

	expiresIn := time.Duration(math.MaxInt64)
	expDate := m.clk.Now()
	domains := []string{}
	serials := []string{}

	// Pick out the expiration date that is closest to being hit.
	for _, cert := range certs {
		domains = append(domains, cert.DNSNames...)
		serials = append(serials, core.SerialToString(cert.SerialNumber))
		possible := cert.NotAfter.Sub(m.clk.Now())
		if possible < expiresIn {
			expiresIn = possible
			expDate = cert.NotAfter
		}
	}
	domains = core.UniqueLowerNames(domains)
	sort.Strings(domains)

	const maxSerials = 100
	truncatedSerials := serials
	if len(truncatedSerials) > maxSerials {
		truncatedSerials = serials[0:maxSerials]
	}

	const maxDomains = 100
	truncatedDomains := domains
	if len(truncatedDomains) > maxDomains {
		truncatedDomains = domains[0:maxDomains]
	}

	// Construct the information about the expiring certificates for use in the
	// subject template
	expiringSubject := fmt.Sprintf("%q", domains[0])
	if len(domains) > 1 {
		expiringSubject += fmt.Sprintf(" (and %d more)", len(domains)-1)
	}

	// Execute the subjectTemplate by filling in the ExpirationSubject
	subjBuf := new(bytes.Buffer)
	err := m.subjectTemplate.Execute(subjBuf, struct {
		ExpirationSubject string
	}{
		ExpirationSubject: expiringSubject,
	})
	if err != nil {
		m.stats.errorCount.With(prometheus.Labels{"type": "SubjectTemplateFailure"}).Inc()
		return err
	}

	email := struct {
		ExpirationDate     string
		DaysToExpiration   int
		DNSNames           string
		TruncatedDNSNames  string
		NumDNSNamesOmitted int
	}{
		ExpirationDate:     expDate.UTC().Format(time.RFC822Z),
		DaysToExpiration:   int(expiresIn.Hours() / 24),
		DNSNames:           strings.Join(domains, "\n"),
		TruncatedDNSNames:  strings.Join(truncatedDomains, "\n"),
		NumDNSNamesOmitted: len(domains) - len(truncatedDomains),
	}
	msgBuf := new(bytes.Buffer)
	err = m.emailTemplate.Execute(msgBuf, email)
	if err != nil {
		m.stats.errorCount.With(prometheus.Labels{"type": "TemplateFailure"}).Inc()
		return err
	}

	logItem := struct {
		Rcpt              []string
		DaysToExpiration  int
		TruncatedDNSNames []string
		TruncatedSerials  []string
	}{
		Rcpt:              emails,
		DaysToExpiration:  email.DaysToExpiration,
		TruncatedDNSNames: truncatedDomains,
		TruncatedSerials:  truncatedSerials,
	}
	logStr, err := json.Marshal(logItem)
	if err != nil {
		m.log.Errf("logItem could not be serialized to JSON. Raw: %+v", logItem)
		return err
	}
	m.log.Infof("attempting send JSON=%s", string(logStr))

	startSending := m.clk.Now()
	err = conn.SendMail(emails, subjBuf.String(), msgBuf.String())
	if err != nil {
		m.log.Errf("failed send JSON=%s err=%s", string(logStr), err)
		return err
	}
	finishSending := m.clk.Now()
	elapsed := finishSending.Sub(startSending)
	m.stats.sendLatency.Observe(elapsed.Seconds())
	return nil
}

// updateLastNagTimestamps updates the lastExpirationNagSent column for every cert in
// the given list. Even though it can encounter errors, it only logs them and
// does not return them, because we always prefer to simply continue.
func (m *mailer) updateLastNagTimestamps(ctx context.Context, certs []*x509.Certificate) {
	for len(certs) > 0 {
		size := len(certs)
		if m.updateChunkSize > 0 && size > m.updateChunkSize {
			size = m.updateChunkSize
		}
		chunk := certs[0:size]
		certs = certs[size:]
		m.updateLastNagTimestampsChunk(ctx, chunk)
	}
}

// updateLastNagTimestampsChunk processes a single chunk (up to 65k) of certificates.
func (m *mailer) updateLastNagTimestampsChunk(ctx context.Context, certs []*x509.Certificate) {
	params := make([]interface{}, len(certs)+1)
	for i, cert := range certs {
		params[i+1] = core.SerialToString(cert.SerialNumber)
	}

	query := fmt.Sprintf(
		"UPDATE certificateStatus SET lastExpirationNagSent = ? WHERE serial IN (%s)",
		db.QuestionMarks(len(certs)),
	)
	params[0] = m.clk.Now()

	_, err := m.dbMap.WithContext(ctx).Exec(query, params...)
	if err != nil {
		m.log.AuditErrf("Error updating certificate status for %d certs: %s", len(certs), err)
		m.stats.errorCount.With(prometheus.Labels{"type": "UpdateCertificateStatus"}).Inc()
	}
}

func (m *mailer) certIsRenewed(ctx context.Context, names []string, issued time.Time) (bool, error) {
	namehash := sa.HashNames(names)

	var present bool
	err := m.dbMap.WithContext(ctx).SelectOne(
		&present,
		`SELECT EXISTS (SELECT id FROM fqdnSets WHERE setHash = ? AND issued > ? LIMIT 1)`,
		namehash,
		issued,
	)
	return present, err
}

type work struct {
	regID    int64
	certDERs []core.CertDER
}

func (m *mailer) processCerts(
	ctx context.Context,
	allCerts []certDERWithRegID,
	expiresIn time.Duration,
) error {
	regIDToCertDERs := make(map[int64][]core.CertDER)

	for _, cert := range allCerts {
		cs := regIDToCertDERs[cert.RegID]
		cs = append(cs, cert.DER)
		regIDToCertDERs[cert.RegID] = cs
	}

	parallelSends := m.parallelSends
	if parallelSends == 0 {
		parallelSends = 1
	}

	var wg sync.WaitGroup
	workChan := make(chan work, len(regIDToCertDERs))

	// Populate the work chan on a goroutine so work is available as soon
	// as one of the sender routines starts.
	go func(ch chan<- work) {
		for regID, certs := range regIDToCertDERs {
			ch <- work{regID, certs}
		}
		close(workChan)
	}(workChan)

	for senderNum := uint(0); senderNum < parallelSends; senderNum++ {
		// For politeness' sake, don't open more than 1 new connection per
		// second.
		if senderNum > 0 {
			time.Sleep(time.Second)
		}

		if ctx.Err() != nil {
			return ctx.Err()
		}

		conn, err := m.mailer.Connect()
		if err != nil {
			m.log.AuditErrf("connecting parallel sender %d: %s", senderNum, err)
			return err
		}
		wg.Add(1)
		go func(conn bmail.Conn, ch <-chan work) {
			defer wg.Done()
			for w := range ch {
				err := m.sendToOneRegID(ctx, conn, w.regID, w.certDERs, expiresIn)
				if err != nil {
					m.log.AuditErr(err.Error())
				}
			}
			conn.Close()
		}(conn, workChan)
	}
	wg.Wait()
	return nil
}

func (m *mailer) sendToOneRegID(ctx context.Context, conn bmail.Conn, regID int64, certDERs []core.CertDER, expiresIn time.Duration) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}
	if len(certDERs) == 0 {
		return errors.New("shouldn't happen: empty certificate list in sendToOneRegID")
	}
	reg, err := m.rs.GetRegistration(ctx, &sapb.RegistrationID{Id: regID})
	if err != nil {
		m.stats.errorCount.With(prometheus.Labels{"type": "GetRegistration"}).Inc()
		return fmt.Errorf("Error fetching registration %d: %s", regID, err)
	}

	parsedCerts := []*x509.Certificate{}
	for i, certDER := range certDERs {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		parsedCert, err := x509.ParseCertificate(certDER)
		if err != nil {
			// TODO(#1420): tell registration about this error
			m.log.AuditErrf("Error parsing certificate: %s. Body: %x", err, certDER)
			m.stats.errorCount.With(prometheus.Labels{"type": "ParseCertificate"}).Inc()
			continue
		}

		// The histogram version of send delay reports the worst case send delay for
		// a single regID in this cycle.
		if i == 0 {
			sendDelay := expiresIn - parsedCert.NotAfter.Sub(m.clk.Now())
			m.stats.sendDelayHistogram.With(prometheus.Labels{"nag_group": expiresIn.String()}).Observe(
				sendDelay.Truncate(time.Second).Seconds())
		}

		renewed, err := m.certIsRenewed(ctx, parsedCert.DNSNames, parsedCert.NotBefore)
		if err != nil {
			m.log.AuditErrf("expiration-mailer: error fetching renewal state: %v", err)
			// assume not renewed
		} else if renewed {
			m.log.Debugf("Cert %s is already renewed", core.SerialToString(parsedCert.SerialNumber))
			m.stats.certificatesAlreadyRenewed.Add(1)
			m.updateLastNagTimestamps(ctx, []*x509.Certificate{parsedCert})
			continue
		}

		parsedCerts = append(parsedCerts, parsedCert)
	}

	m.stats.certificatesPerAccountNeedingMail.Observe(float64(len(parsedCerts)))

	if len(parsedCerts) == 0 {
		// all certificates are renewed
		return nil
	}

	err = m.sendNags(conn, reg.Contact, parsedCerts)
	if err != nil {
		// Check to see if the error was due to the mail being undeliverable,
		// in which case we don't want to try again later.
		var badAddrErr *bmail.BadAddressSMTPError
		if ok := errors.As(err, &badAddrErr); ok {
			m.updateLastNagTimestamps(ctx, parsedCerts)
		}

		m.stats.errorCount.With(prometheus.Labels{"type": "SendNags"}).Inc()
		return fmt.Errorf("sending nag emails: %s", err)
	}

	m.updateLastNagTimestamps(ctx, parsedCerts)
	return nil
}

// findExpiringCertificates finds certificates that might need an expiration mail, filters them,
// groups by account, sends mail, and updates their status in the DB so we don't examine them again.
//
// Invariant: findExpiringCertificates should examine each certificate at most N times, where
// N is the number of reminders. For every certificate examined (barring errors), this function
// should update the lastExpirationNagSent field of certificateStatus, so it does not need to
// examine the same certificate again on the next go-round. This ensures we make forward progress
// and don't clog up the window of certificates to be examined.
func (m *mailer) findExpiringCertificates(ctx context.Context) error {
	now := m.clk.Now()
	// E.g. m.nagTimes = [2, 4, 8, 15] days from expiration
	for i, expiresIn := range m.nagTimes {
		left := now
		if i > 0 {
			left = left.Add(m.nagTimes[i-1])
		}
		right := now.Add(expiresIn)

		m.log.Infof("expiration-mailer: Searching for certificates that expire between %s and %s and had last nag >%s before expiry",
			left.UTC(), right.UTC(), expiresIn)

		var certs []certDERWithRegID
		var err error
		if features.Enabled(features.ExpirationMailerUsesJoin) {
			certs, err = m.getCertsWithJoin(ctx, left, right, expiresIn)
		} else {
			certs, err = m.getCerts(ctx, left, right, expiresIn)
		}
		if err != nil {
			return err
		}

		m.stats.certificatesExamined.Add(float64(len(certs)))

		// If the number of rows was exactly `m.limit` rows we need to increment
		// a stat indicating that this nag group is at capacity based on the
		// configured cert limit. If this condition continually occurs across mailer
		// runs then we will not catch up, resulting in under-sending expiration
		// mails. The effects of this were initially described in issue #2002[0].
		//
		// 0: https://github.com/letsencrypt/boulder/issues/2002
		atCapacity := float64(0)
		if len(certs) == m.limit {
			m.log.Infof("nag group %s expiring certificates at configured capacity (select limit %d)",
				expiresIn.String(), m.limit)
			atCapacity = float64(1)
		}
		m.stats.nagsAtCapacity.With(prometheus.Labels{"nag_group": expiresIn.String()}).Set(atCapacity)

		m.log.Infof("Found %d certificates expiring between %s and %s", len(certs),
			left.Format("2006-01-02 03:04"), right.Format("2006-01-02 03:04"))

		if len(certs) == 0 {
			continue // nothing to do
		}

		processingStarted := m.clk.Now()
		err = m.processCerts(ctx, certs, expiresIn)
		if err != nil {
			m.log.AuditErr(err.Error())
		}
		processingEnded := m.clk.Now()
		elapsed := processingEnded.Sub(processingStarted)
		m.stats.processingLatency.Observe(elapsed.Seconds())
	}

	return nil
}

func (m *mailer) getCertsWithJoin(ctx context.Context, left, right time.Time, expiresIn time.Duration) ([]certDERWithRegID, error) {
	// First we do a query on the certificateStatus table to find certificates
	// nearing expiry meeting our criteria for email notification. We later
	// sequentially fetch the certificate details. This avoids an expensive
	// JOIN.
	var certs []certDERWithRegID
	_, err := m.dbMap.WithContext(ctx).Select(
		&certs,
		`SELECT
				cert.der as der, cert.registrationID as regID
				FROM certificateStatus AS cs
				JOIN certificates as cert
				ON cs.serial = cert.serial
				AND cs.notAfter > :cutoffA
				AND cs.notAfter <= :cutoffB
				AND cs.status != "revoked"
				AND COALESCE(TIMESTAMPDIFF(SECOND, cs.lastExpirationNagSent, cs.notAfter) > :nagCutoff, 1)
				ORDER BY cs.notAfter ASC
				LIMIT :limit`,
		map[string]interface{}{
			"cutoffA":   left,
			"cutoffB":   right,
			"nagCutoff": expiresIn.Seconds(),
			"limit":     m.limit,
		},
	)
	if err != nil {
		m.log.AuditErrf("expiration-mailer: Error loading certificate serials: %s", err)
		return nil, err
	}
	m.log.Debugf("found %d certificates", len(certs))
	return certs, nil
}

func (m *mailer) getCerts(ctx context.Context, left, right time.Time, expiresIn time.Duration) ([]certDERWithRegID, error) {
	// First we do a query on the certificateStatus table to find certificates
	// nearing expiry meeting our criteria for email notification. We later
	// sequentially fetch the certificate details. This avoids an expensive
	// JOIN.
	var serials []string
	_, err := m.dbMap.WithContext(ctx).Select(
		&serials,
		`SELECT
				cs.serial
				FROM certificateStatus AS cs
				WHERE cs.notAfter > :cutoffA
				AND cs.notAfter <= :cutoffB
				AND cs.status != "revoked"
				AND COALESCE(TIMESTAMPDIFF(SECOND, cs.lastExpirationNagSent, cs.notAfter) > :nagCutoff, 1)
				ORDER BY cs.notAfter ASC
				LIMIT :limit`,
		map[string]interface{}{
			"cutoffA":   left,
			"cutoffB":   right,
			"nagCutoff": expiresIn.Seconds(),
			"limit":     m.limit,
		},
	)
	if err != nil {
		m.log.AuditErrf("expiration-mailer: Error loading certificate serials: %s", err)
		return nil, err
	}
	m.log.Debugf("found %d certificates", len(serials))

	// Now we can sequentially retrieve the certificate details for each of the
	// certificate status rows
	var certs []certDERWithRegID
	for i, serial := range serials {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		var cert core.Certificate
		cert, err := sa.SelectCertificate(m.dbMap.WithContext(ctx), serial)
		if err != nil {
			// We can get a NoRowsErr when processing a serial number corresponding
			// to a precertificate with no final certificate. Since this certificate
			// is not being used by a subscriber, we don't send expiration email about
			// it.
			if db.IsNoRows(err) {
				m.log.Infof("no rows for serial %q", serial)
				continue
			}
			m.log.AuditErrf("expiration-mailer: Error loading cert %q: %s", cert.Serial, err)
			continue
		}
		certs = append(certs, certDERWithRegID{
			DER:   cert.DER,
			RegID: cert.RegistrationID,
		})
		if i == 0 {
			// Report the send delay metric. Note: this is the worst-case send delay
			// of any certificate in this batch because it's based on the first (oldest).
			sendDelay := expiresIn - cert.Expires.Sub(m.clk.Now())
			m.stats.sendDelay.With(prometheus.Labels{"nag_group": expiresIn.String()}).Set(
				sendDelay.Truncate(time.Second).Seconds())
		}
	}

	return certs, nil
}

type durationSlice []time.Duration

func (ds durationSlice) Len() int {
	return len(ds)
}

func (ds durationSlice) Less(a, b int) bool {
	return ds[a] < ds[b]
}

func (ds durationSlice) Swap(a, b int) {
	ds[a], ds[b] = ds[b], ds[a]
}

type Config struct {
	Mailer struct {
		cmd.ServiceConfig
		DB cmd.DBConfig
		cmd.SMTPConfig

		// From is the "From" address for reminder messages.
		From string

		// Subject is the Subject line of reminder messages.
		// This is a Go template with a single variable: ExpirationSubject,
		// which contains a list of affectd hostnames, possible truncated.
		Subject string

		// CertLimit is the maximum number of certificates to investigate in a
		// single batch.
		CertLimit int

		// UpdateChunkSize is the maximum number of rows to update in a single
		// SQL UPDATE statement.
		UpdateChunkSize int

		NagTimes []string

		// TODO(#6097): Remove this
		NagCheckInterval string

		// Path to a text/template email template
		EmailTemplate string

		// How often to process a batch of certificates
		Frequency cmd.ConfigDuration

		// How many parallel goroutines should process each batch of emails
		ParallelSends uint

		TLS       cmd.TLSConfig
		SAService *cmd.GRPCClientConfig

		// Path to a file containing a list of trusted root certificates for use
		// during the SMTP connection (as opposed to the gRPC connections).
		SMTPTrustedRootFile string

		Features map[string]bool
	}

	Syslog  cmd.SyslogConfig
	Beeline cmd.BeelineConfig
}

func initStats(stats prometheus.Registerer) mailerStats {
	sendDelay := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "send_delay",
			Help: "For the last batch of certificates, difference between the idealized send time and actual send time. Will always be nonzero, bigger numbers are worse",
		},
		[]string{"nag_group"})
	stats.MustRegister(sendDelay)

	sendDelayHistogram := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "send_delay_histogram",
			Help:    "For each mail sent, difference between the idealized send time and actual send time. Will always be nonzero, bigger numbers are worse",
			Buckets: prometheus.LinearBuckets(86400, 86400, 10),
		},
		[]string{"nag_group"})
	stats.MustRegister(sendDelayHistogram)

	nagsAtCapacity := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "nags_at_capacity",
			Help: "Count of nag groups at capcacity",
		},
		[]string{"nag_group"})
	stats.MustRegister(nagsAtCapacity)

	errorCount := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "errors",
			Help: "Number of errors",
		},
		[]string{"type"})
	stats.MustRegister(errorCount)

	sendLatency := prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "send_latency",
			Help:    "Time the mailer takes sending messages in seconds",
			Buckets: metrics.InternetFacingBuckets,
		})
	stats.MustRegister(sendLatency)

	processingLatency := prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "processing_latency",
			Help:    "Time the mailer takes processing certificates in seconds",
			Buckets: []float64{30, 60, 75, 90, 120, 600, 3600},
		})
	stats.MustRegister(processingLatency)

	certificatesExamined := prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "certificates_examined",
			Help: "Number of certificates looked at that are potentially due for an expiration mail",
		})
	stats.MustRegister(certificatesExamined)

	certificatesAlreadyRenewed := prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "certificates_already_renewed",
			Help: "Number of certificates from certificates_examined that were ignored because they were already renewed",
		})
	stats.MustRegister(certificatesAlreadyRenewed)

	accountsNeedingMail := prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "certificates_per_account_needing_mail",
			Help:    "After ignoring certificates_already_renewed and grouping the remaining certificates by account, how many accounts needed to get an email; grouped by how many certificates each account needed",
			Buckets: []float64{0, 1, 2, 100, 1000, 10000, 100000},
		})
	stats.MustRegister(accountsNeedingMail)

	return mailerStats{
		sendDelay:                         sendDelay,
		sendDelayHistogram:                sendDelayHistogram,
		nagsAtCapacity:                    nagsAtCapacity,
		errorCount:                        errorCount,
		sendLatency:                       sendLatency,
		processingLatency:                 processingLatency,
		certificatesExamined:              certificatesExamined,
		certificatesAlreadyRenewed:        certificatesAlreadyRenewed,
		certificatesPerAccountNeedingMail: accountsNeedingMail,
	}
}

func main() {
	configFile := flag.String("config", "", "File path to the configuration file for this service")
	certLimit := flag.Int("cert_limit", 0, "Count of certificates to process per expiration period")
	reconnBase := flag.Duration("reconnectBase", 1*time.Second, "Base sleep duration between reconnect attempts")
	reconnMax := flag.Duration("reconnectMax", 5*60*time.Second, "Max sleep duration between reconnect attempts after exponential backoff")
	daemon := flag.Bool("daemon", false, "Run in daemon mode")

	flag.Parse()

	if *configFile == "" {
		flag.Usage()
		os.Exit(1)
	}

	var c Config
	err := cmd.ReadConfigFile(*configFile, &c)
	cmd.FailOnError(err, "Reading JSON config file into config structure")
	err = features.Set(c.Mailer.Features)
	cmd.FailOnError(err, "Failed to set feature flags")

	bc, err := c.Beeline.Load()
	cmd.FailOnError(err, "Failed to load Beeline config")
	beeline.Init(bc)
	defer beeline.Close()

	scope, logger := cmd.StatsAndLogging(c.Syslog, c.Mailer.DebugAddr)
	defer logger.AuditPanic()
	logger.Info(cmd.VersionString())

	if *certLimit > 0 {
		c.Mailer.CertLimit = *certLimit
	}
	// Default to 100 if no certLimit is set
	if c.Mailer.CertLimit == 0 {
		c.Mailer.CertLimit = 100
	}

	dbMap, err := sa.InitWrappedDb(c.Mailer.DB, scope, logger)
	cmd.FailOnError(err, "While initializing dbMap")

	tlsConfig, err := c.Mailer.TLS.Load()
	cmd.FailOnError(err, "TLS config")

	clk := cmd.Clock()

	conn, err := bgrpc.ClientSetup(c.Mailer.SAService, tlsConfig, scope, clk)
	cmd.FailOnError(err, "Failed to load credentials and create gRPC connection to SA")
	sac := sapb.NewStorageAuthorityClient(conn)

	var smtpRoots *x509.CertPool
	if c.Mailer.SMTPTrustedRootFile != "" {
		pem, err := os.ReadFile(c.Mailer.SMTPTrustedRootFile)
		cmd.FailOnError(err, "Loading trusted roots file")
		smtpRoots = x509.NewCertPool()
		if !smtpRoots.AppendCertsFromPEM(pem) {
			cmd.FailOnError(nil, "Failed to parse root certs PEM")
		}
	}

	// Load email template
	emailTmpl, err := os.ReadFile(c.Mailer.EmailTemplate)
	cmd.FailOnError(err, fmt.Sprintf("Could not read email template file [%s]", c.Mailer.EmailTemplate))
	tmpl, err := template.New("expiry-email").Parse(string(emailTmpl))
	cmd.FailOnError(err, "Could not parse email template")

	// If there is no configured subject template, use a default
	if c.Mailer.Subject == "" {
		c.Mailer.Subject = defaultExpirationSubject
	}
	// Load subject template
	subjTmpl, err := template.New("expiry-email-subject").Parse(c.Mailer.Subject)
	cmd.FailOnError(err, "Could not parse email subject template")

	fromAddress, err := netmail.ParseAddress(c.Mailer.From)
	cmd.FailOnError(err, fmt.Sprintf("Could not parse from address: %s", c.Mailer.From))

	smtpPassword, err := c.Mailer.PasswordConfig.Pass()
	cmd.FailOnError(err, "Failed to load SMTP password")
	mailClient := bmail.New(
		c.Mailer.Server,
		c.Mailer.Port,
		c.Mailer.Username,
		smtpPassword,
		smtpRoots,
		*fromAddress,
		logger,
		scope,
		*reconnBase,
		*reconnMax)

	var nags durationSlice
	for _, nagDuration := range c.Mailer.NagTimes {
		dur, err := time.ParseDuration(nagDuration)
		if err != nil {
			logger.AuditErrf("Failed to parse nag duration string [%s]: %s", nagDuration, err)
			return
		}
		// Add some padding to the nag times so we send _before_ the configured
		// time rather than after. See https://github.com/letsencrypt/boulder/pull/1029
		adjustedInterval := dur + c.Mailer.Frequency.Duration
		nags = append(nags, adjustedInterval)
	}
	// Make sure durations are sorted in increasing order
	sort.Sort(nags)

	if c.Mailer.UpdateChunkSize > 65535 {
		// MariaDB limits the number of placeholders parameters to max_uint16:
		// https://github.com/MariaDB/server/blob/10.5/sql/sql_prepare.cc#L2629-L2635
		cmd.Fail(fmt.Sprintf("UpdateChunkSize of %d is too big", c.Mailer.UpdateChunkSize))
	}

	m := mailer{
		log:             logger,
		dbMap:           dbMap,
		rs:              sac,
		mailer:          mailClient,
		subjectTemplate: subjTmpl,
		emailTemplate:   tmpl,
		nagTimes:        nags,
		limit:           c.Mailer.CertLimit,
		updateChunkSize: c.Mailer.UpdateChunkSize,
		parallelSends:   c.Mailer.ParallelSends,
		clk:             clk,
		stats:           initStats(scope),
	}

	// Prefill this labelled stat with the possible label values, so each value is
	// set to 0 on startup, rather than being missing from stats collection until
	// the first mail run.
	for _, expiresIn := range nags {
		m.stats.nagsAtCapacity.With(prometheus.Labels{"nag_group": expiresIn.String()}).Set(0)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go cmd.CatchSignals(logger, func() {
		cancel()
		select {} // wait for the `findExpiringCertificates` calls below to exit
	})

	if *daemon {
		if c.Mailer.Frequency.Duration == 0 {
			fmt.Fprintln(os.Stderr, "mailer.Frequency is not set in the JSON config")
			os.Exit(1)
		}
		t := time.NewTicker(c.Mailer.Frequency.Duration)
		for {
			select {
			case <-t.C:
				err = m.findExpiringCertificates(ctx)
				cmd.FailOnError(err, "expiration-mailer has failed")
			case <-ctx.Done():
				os.Exit(0)
			}
		}
	} else {
		err = m.findExpiringCertificates(ctx)
		cmd.FailOnError(err, "expiration-mailer has failed")
	}
}

func init() {
	cmd.RegisterCommand("expiration-mailer", main)
}
