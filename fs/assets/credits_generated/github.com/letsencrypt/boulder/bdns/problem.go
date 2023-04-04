package bdns

import (
	"context"
	"fmt"
	"net"

	"github.com/miekg/dns"
)

// Error wraps a DNS error with various relevant information
type Error struct {
	recordType uint16
	hostname   string
	// Exactly one of rCode or underlying should be set.
	underlying error
	rCode      int
}

func (d Error) Error() string {
	var detail, additional string
	if d.underlying != nil {
		if netErr, ok := d.underlying.(*net.OpError); ok {
			if netErr.Timeout() {
				detail = detailDNSTimeout
			} else {
				detail = detailDNSNetFailure
			}
			// Note: we check d.underlying here even though `Timeout()` does this because the call to `netErr.Timeout()` above only
			// happens for `*net.OpError` underlying types!
		} else if d.underlying == context.DeadlineExceeded {
			detail = detailDNSTimeout
		} else if d.underlying == context.Canceled {
			detail = detailCanceled
		} else {
			detail = detailServerFailure
		}
	} else if d.rCode != dns.RcodeSuccess {
		detail = dns.RcodeToString[d.rCode]
		if explanation, ok := rcodeExplanations[d.rCode]; ok {
			additional = " - " + explanation
		}
	} else {
		detail = detailServerFailure
	}
	return fmt.Sprintf("DNS problem: %s looking up %s for %s%s", detail,
		dns.TypeToString[d.recordType], d.hostname, additional)
}

const detailDNSTimeout = "query timed out"
const detailCanceled = "query timed out (and was canceled)"
const detailDNSNetFailure = "networking error"
const detailServerFailure = "server failure at resolver"

// rcodeExplanations provide additional friendly explanatory text to be included in DNS
// error messages, for select inscrutable RCODEs.
var rcodeExplanations = map[int]string{
	dns.RcodeNameError:     "check that a DNS record exists for this domain",
	dns.RcodeServerFailure: "the domain's nameservers may be malfunctioning",
}
