//go:build !integration

package wfe2

import (
	"context"
	"net/http"

	"github.com/letsencrypt/boulder/web"
)

// Issuer returns a 404, because production Boulder does not actually serve
// AIA Issuer URL content.
func (wfe *WebFrontEndImpl) Issuer(ctx context.Context, logEvent *web.RequestEvent, response http.ResponseWriter, request *http.Request) {
	// Use the same mechanism to return a 404 as wfe.Index does for paths other
	// than "/", so that the result is indistinguishable.
	logEvent.AddError("AIA Issuer URL requested")
	http.NotFound(response, request)
	response.Header().Set("Content-Type", "application/problem+json")
}
