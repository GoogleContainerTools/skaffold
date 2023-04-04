package lints

import (
	"time"

	"github.com/zmap/zlint/v3/lint"
)

const (
	// CABF Baseline Requirements 6.3.2 Certificate operational periods:
	// For the purpose of calculations, a day is measured as 86,400 seconds.
	// Any amount of time greater than this, including fractional seconds and/or
	// leap seconds, shall represent an additional day.
	BRDay time.Duration = 86400 * time.Second

	// Declare our own Sources for use in zlint registry filtering.
	LetsEncryptCPSAll          lint.LintSource = "LECPSAll"
	LetsEncryptCPSIntermediate lint.LintSource = "LECPSIntermediate"
	LetsEncryptCPSRoot         lint.LintSource = "LECPSRoot"
	LetsEncryptCPSSubscriber   lint.LintSource = "LECPSSubscriber"
	ChromeCTPolicy             lint.LintSource = "ChromeCT"
)

var (
	CPSV33Date = time.Date(2021, time.June, 8, 0, 0, 0, 0, time.UTC)
)
