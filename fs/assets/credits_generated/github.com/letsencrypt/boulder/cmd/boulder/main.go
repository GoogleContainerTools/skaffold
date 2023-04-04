package main

import (
	"fmt"
	"os"
	"path"

	_ "github.com/letsencrypt/boulder/cmd/admin-revoker"
	_ "github.com/letsencrypt/boulder/cmd/akamai-purger"
	_ "github.com/letsencrypt/boulder/cmd/bad-key-revoker"
	_ "github.com/letsencrypt/boulder/cmd/boulder-ca"
	_ "github.com/letsencrypt/boulder/cmd/boulder-observer"
	_ "github.com/letsencrypt/boulder/cmd/boulder-publisher"
	_ "github.com/letsencrypt/boulder/cmd/boulder-ra"
	_ "github.com/letsencrypt/boulder/cmd/boulder-sa"
	_ "github.com/letsencrypt/boulder/cmd/boulder-va"
	_ "github.com/letsencrypt/boulder/cmd/boulder-wfe2"
	_ "github.com/letsencrypt/boulder/cmd/caa-log-checker"
	_ "github.com/letsencrypt/boulder/cmd/ceremony"
	_ "github.com/letsencrypt/boulder/cmd/cert-checker"
	_ "github.com/letsencrypt/boulder/cmd/contact-auditor"
	_ "github.com/letsencrypt/boulder/cmd/crl-checker"
	_ "github.com/letsencrypt/boulder/cmd/crl-storer"
	_ "github.com/letsencrypt/boulder/cmd/crl-updater"
	_ "github.com/letsencrypt/boulder/cmd/expiration-mailer"
	_ "github.com/letsencrypt/boulder/cmd/id-exporter"
	_ "github.com/letsencrypt/boulder/cmd/log-validator"
	_ "github.com/letsencrypt/boulder/cmd/nonce-service"
	_ "github.com/letsencrypt/boulder/cmd/notify-mailer"
	_ "github.com/letsencrypt/boulder/cmd/ocsp-responder"
	_ "github.com/letsencrypt/boulder/cmd/ocsp-updater"
	_ "github.com/letsencrypt/boulder/cmd/orphan-finder"
	_ "github.com/letsencrypt/boulder/cmd/reversed-hostname-checker"
	_ "github.com/letsencrypt/boulder/cmd/rocsp-tool"

	"github.com/letsencrypt/boulder/cmd"
)

func main() {
	cmd.LookupCommand(path.Base(os.Args[0]))()
}

func init() {
	cmd.RegisterCommand("boulder", func() {
		if len(os.Args) <= 1 {
			fmt.Fprintf(os.Stderr, "Call with --list to list available subcommands. Run them like boulder <subcommand>.\n")
			return
		}
		subcommand := cmd.LookupCommand(os.Args[1])
		if subcommand == nil {
			fmt.Fprintf(os.Stderr, "Unknown subcommand '%s'.\n", os.Args[1])
			return
		}
		os.Args = os.Args[1:]
		subcommand()
	})
	cmd.RegisterCommand("--list", func() {
		for _, c := range cmd.AvailableCommands() {
			if c != "boulder" && c != "--list" {
				fmt.Println(c)
			}
		}
	})
}
