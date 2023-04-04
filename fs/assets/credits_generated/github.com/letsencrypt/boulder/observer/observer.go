package observer

import (
	blog "github.com/letsencrypt/boulder/log"
	_ "github.com/letsencrypt/boulder/observer/probers/crl"
	_ "github.com/letsencrypt/boulder/observer/probers/dns"
	_ "github.com/letsencrypt/boulder/observer/probers/http"
	_ "github.com/letsencrypt/boulder/observer/probers/tls"
)

// Observer is the steward of goroutines started for each `monitor`.
type Observer struct {
	logger   blog.Logger
	monitors []*monitor
}

// Start spins off a goroutine for each monitor and then runs forever.
func (o Observer) Start() {
	for _, mon := range o.monitors {
		go mon.start(o.logger)
	}
	select {}
}
