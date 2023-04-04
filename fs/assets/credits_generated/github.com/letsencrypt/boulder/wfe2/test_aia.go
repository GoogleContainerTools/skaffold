//go:build integration

package wfe2

import (
	"context"
	"net/http"
	"strconv"

	"github.com/letsencrypt/boulder/issuance"
	"github.com/letsencrypt/boulder/probs"
	"github.com/letsencrypt/boulder/web"
)

// Issuer returns the Issuer Cert identified by the path (its IssuerNameID).
// Used by integration tests to handle requests for the AIA Issuer URL.
func (wfe *WebFrontEndImpl) Issuer(ctx context.Context, logEvent *web.RequestEvent, response http.ResponseWriter, request *http.Request) {
	idStr := request.URL.Path
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		wfe.sendError(response, logEvent, probs.Malformed("Issuer ID must be an integer"), err)
		return
	}

	issuer, ok := wfe.issuerCertificates[issuance.IssuerNameID(id)]
	if !ok {
		wfe.sendError(response, logEvent, probs.NotFound("Issuer ID did not match any known issuer"), nil)
		return
	}

	response.Header().Set("Content-Type", "application/pkix-cert")
	response.WriteHeader(http.StatusOK)
	_, err = response.Write(issuer.Certificate.Raw)
	if err != nil {
		wfe.log.Warningf("Could not write response: %s", err)
	}
}
