package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"

	"github.com/letsencrypt/boulder/cmd"
	"github.com/letsencrypt/boulder/core"
	"github.com/letsencrypt/boulder/crl/crl_x509"
	"github.com/letsencrypt/boulder/revocation"
)

type s3TestSrv struct {
	sync.RWMutex
	allSerials map[string]revocation.Reason
}

func (srv *s3TestSrv) handleUpload(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(500)
		w.Write([]byte("failed to read request body"))
		return
	}

	crl, err := crl_x509.ParseRevocationList(body)
	if err != nil {
		w.WriteHeader(500)
		w.Write([]byte(fmt.Sprintf("failed to parse body: %s", err)))
		return
	}

	srv.Lock()
	defer srv.Unlock()
	for _, rc := range crl.RevokedCertificates {
		reason := 0
		if rc.ReasonCode != nil {
			reason = *rc.ReasonCode
		}
		srv.allSerials[core.SerialToString(rc.SerialNumber)] = revocation.Reason(reason)
	}

	w.WriteHeader(200)
	w.Write([]byte("{}"))
}

func (srv *s3TestSrv) handleClear(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		w.WriteHeader(405)
		return
	}

	srv.Lock()
	defer srv.Unlock()
	srv.allSerials = make(map[string]revocation.Reason)
}

func (srv *s3TestSrv) handleQuery(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		w.WriteHeader(405)
		return
	}

	serial := r.URL.Query().Get("serial")
	if serial == "" {
		w.WriteHeader(400)
		return
	}

	srv.RLock()
	defer srv.RUnlock()
	reason, ok := srv.allSerials[serial]
	if !ok {
		w.WriteHeader(404)
		return
	}

	w.WriteHeader(200)
	w.Write([]byte(fmt.Sprintf("%d", reason)))
}

func main() {
	listenAddr := flag.String("listen", "0.0.0.0:7890", "Address to listen on")
	flag.Parse()

	srv := s3TestSrv{allSerials: make(map[string]revocation.Reason)}

	http.HandleFunc("/", srv.handleUpload)
	http.HandleFunc("/clear", srv.handleClear)
	http.HandleFunc("/query", srv.handleQuery)

	// The gosec linter complains that timeouts cannot be set here. That's fine,
	// because this is test-only code.
	////nolint:gosec
	go log.Fatal(http.ListenAndServe(*listenAddr, nil))
	cmd.CatchSignals(nil, nil)
}
