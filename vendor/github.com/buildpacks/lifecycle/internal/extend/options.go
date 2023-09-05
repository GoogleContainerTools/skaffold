package extend

import "time"

type Options struct {
	BuildContext string
	IgnorePaths  []string
	CacheTTL     time.Duration
}
