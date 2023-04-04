//go:build integration

package integration

import (
	"os"
	"testing"

	"github.com/letsencrypt/boulder/test"
)

func TestDuplicateFQDNRateLimit(t *testing.T) {
	t.Parallel()
	domain := random_domain()
	os.Setenv("DIRECTORY", "http://boulder.service.consul:4001/directory")

	_, err := authAndIssue(nil, nil, []string{domain})
	test.AssertNotError(t, err, "Failed to issue first certificate")

	_, err = authAndIssue(nil, nil, []string{domain})
	test.AssertNotError(t, err, "Failed to issue second certificate")

	_, err = authAndIssue(nil, nil, []string{domain})
	test.AssertError(t, err, "Somehow managed to issue third certificate")
}
