package ctpolicy

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/letsencrypt/boulder/core"
	"github.com/letsencrypt/boulder/ctpolicy/loglist"
	berrors "github.com/letsencrypt/boulder/errors"
	blog "github.com/letsencrypt/boulder/log"
	pubpb "github.com/letsencrypt/boulder/publisher/proto"
	"github.com/prometheus/client_golang/prometheus"
)

const (
	succeeded = "succeeded"
	failed    = "failed"
)

// CTPolicy is used to hold information about SCTs required from various
// groupings
type CTPolicy struct {
	pub           pubpb.PublisherClient
	sctLogs       loglist.List
	infoLogs      loglist.List
	finalLogs     loglist.List
	stagger       time.Duration
	log           blog.Logger
	winnerCounter *prometheus.CounterVec
}

// New creates a new CTPolicy struct
func New(pub pubpb.PublisherClient, sctLogs loglist.List, infoLogs loglist.List, finalLogs loglist.List, stagger time.Duration, log blog.Logger, stats prometheus.Registerer) *CTPolicy {
	winnerCounter := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "sct_winner",
			Help: "Counter of logs which are selected for sct submission, by log URL and result (succeeded or failed).",
		},
		[]string{"url", "result"},
	)
	stats.MustRegister(winnerCounter)

	return &CTPolicy{
		pub:           pub,
		sctLogs:       sctLogs,
		infoLogs:      infoLogs,
		finalLogs:     finalLogs,
		stagger:       stagger,
		log:           log,
		winnerCounter: winnerCounter,
	}
}

type result struct {
	sct []byte
	url string
	err error
}

// GetSCTs retrieves exactly two SCTs from the total collection of configured
// log groups, with at most one SCT coming from each group. It expects that all
// logs run by a single operator (e.g. Google) are in the same group, to
// guarantee that SCTs from logs in different groups do not end up coming from
// the same operator. As such, it enforces Google's current CT Policy, which
// requires that certs have two SCTs from logs run by different operators.
func (ctp *CTPolicy) GetSCTs(ctx context.Context, cert core.CertDER, expiration time.Time) (core.SCTDERs, error) {
	// We'll cancel this sub-context when we have the two SCTs we need, to cause
	// any other ongoing submission attempts to quit.
	subCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	// This closure will be called in parallel once for each operator group.
	getOne := func(i int, g string) ([]byte, string, error) {
		// Sleep a little bit to stagger our requests to the later groups. Use `i-1`
		// to compute the stagger duration so that the first two groups (indices 0
		// and 1) get negative or zero (i.e. instant) sleep durations. If the
		// context gets cancelled (most likely because two logs from other operator
		// groups returned SCTs already) before the sleep is complete, quit instead.
		select {
		case <-subCtx.Done():
			return nil, "", subCtx.Err()
		case <-time.After(time.Duration(i-1) * ctp.stagger):
		}

		// Pick a random log from among those in the group. In practice, very few
		// operator groups have more than one log, so this loses little flexibility.
		url, key, err := ctp.sctLogs.PickOne(g, expiration)
		if err != nil {
			return nil, "", fmt.Errorf("unable to get log info: %w", err)
		}

		sct, err := ctp.pub.SubmitToSingleCTWithResult(ctx, &pubpb.Request{
			LogURL:       url,
			LogPublicKey: key,
			Der:          cert,
			Precert:      true,
		})
		if err != nil {
			return nil, url, fmt.Errorf("ct submission to %q (%q) failed: %w", g, url, err)
		}

		return sct.Sct, url, nil
	}

	// Ensure that this channel has a buffer equal to the number of goroutines
	// we're kicking off, so that they're all guaranteed to be able to write to
	// it and exit without blocking and leaking.
	results := make(chan result, len(ctp.sctLogs))

	// Kick off a collection of goroutines to try to submit the precert to each
	// log operator group. Randomize the order of the groups so that we're not
	// always trying to submit to the same two operators.
	for i, group := range ctp.sctLogs.Permute() {
		go func(i int, g string) {
			sctDER, url, err := getOne(i, g)
			results <- result{sct: sctDER, url: url, err: err}
		}(i, group)
	}

	go ctp.submitPrecertInformational(cert, expiration)

	// Finally, collect SCTs and/or errors from our results channel. We know that
	// we will collect len(ctp.sctLogs) results from the channel because every
	// goroutine is guaranteed to write one result to the channel.
	scts := make(core.SCTDERs, 0)
	errs := make([]string, 0)
	for i := 0; i < len(ctp.sctLogs); i++ {
		res := <-results
		if res.err != nil {
			errs = append(errs, res.err.Error())
			if res.url != "" {
				ctp.winnerCounter.WithLabelValues(res.url, failed).Inc()
			}
			continue
		}
		scts = append(scts, res.sct)
		ctp.winnerCounter.WithLabelValues(res.url, succeeded).Inc()
		if len(scts) >= 2 {
			return scts, nil
		}
	}

	// If we made it to the end of that loop, that means we never got two SCTs
	// to return. Error out instead.
	if ctx.Err() != nil {
		// We timed out (the calling function returned and canceled our context),
		// thereby causing all of our getOne sub-goroutines to be cancelled.
		return nil, berrors.MissingSCTsError("failed to get 2 SCTs before ctx finished: %s", ctx.Err())
	}
	return nil, berrors.MissingSCTsError("failed to get 2 SCTs, got %d error(s): %s", len(errs), strings.Join(errs, "; "))
}

// submitAllBestEffort submits the given certificate or precertificate to every
// log ("informational" for precerts, "final" for certs) configured in the policy.
// It neither waits for these submission to complete, nor tracks their success.
func (ctp *CTPolicy) submitAllBestEffort(blob core.CertDER, isPrecert bool, expiry time.Time) {
	logs := ctp.finalLogs
	if isPrecert {
		logs = ctp.infoLogs
	}

	for _, group := range logs {
		for _, log := range group {
			if log.StartInclusive.After(expiry) || log.EndExclusive.Equal(expiry) || log.EndExclusive.Before(expiry) {
				continue
			}

			go func(log loglist.Log) {
				_, err := ctp.pub.SubmitToSingleCTWithResult(
					context.Background(),
					&pubpb.Request{
						LogURL:       log.Url,
						LogPublicKey: log.Key,
						Der:          blob,
						Precert:      isPrecert,
					},
				)
				if err != nil {
					ctp.log.Warningf("ct submission of cert to log %q failed: %s", log.Url, err)
				}
			}(log)
		}
	}

}

// submitPrecertInformational submits precertificates to any configured
// "informational" logs, but does not care about success or returned SCTs.
func (ctp *CTPolicy) submitPrecertInformational(cert core.CertDER, expiration time.Time) {
	ctp.submitAllBestEffort(cert, true, expiration)
}

// SubmitFinalCert submits finalized certificates created from precertificates
// to any configured "final" logs, but does not care about success.
func (ctp *CTPolicy) SubmitFinalCert(cert core.CertDER, expiration time.Time) {
	ctp.submitAllBestEffort(cert, false, expiration)
}
