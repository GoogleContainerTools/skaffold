package phase

import (
	stderrors "errors"
	"net/http"
	"time"

	"github.com/buildpacks/imgutil"
	"github.com/google/go-containerregistry/pkg/v1/remote/transport"

	"github.com/buildpacks/lifecycle/log"
)

// topLayerDelays is hardcoded array of delays
var topLayerDelays = []time.Duration{
	100 * time.Millisecond,
	200 * time.Millisecond,
	500 * time.Millisecond,
	1 * time.Second,
	2 * time.Second,
}

// topLayerSleep is the function used for sleeping between retries.
// It can be replaced for testing.
var topLayerSleep = time.Sleep

// isRetryable returns true if the error is likely transient and should be retried.
// 401 Unauthorized and 403 Forbidden are not retryable as they indicate auth/config issues.
func isRetryable(err error) bool {
	if tErr, ok := stderrors.AsType[*transport.Error](err); ok {
		return tErr.StatusCode != http.StatusBadRequest &&
			tErr.StatusCode != http.StatusUnauthorized &&
			tErr.StatusCode != http.StatusForbidden &&
			tErr.StatusCode != http.StatusMethodNotAllowed &&
			tErr.StatusCode != http.StatusTooManyRequests
	}
	return true
}

// OpenRemoteImage opens a remote image with retry logic for registry mirror transient errors.
// go-containerregistry caches manifests, so each retry attempt creates a fresh image.
// Non-retryable errors (401, 403) are returned immediately without retry.
func OpenRemoteImage(logger log.Logger, newImage func() (imgutil.Image, error)) (imgutil.Image, error) {
	var lastErr error
	for attempt := 0; attempt <= len(topLayerDelays); attempt++ {
		img, err := newImage()
		if err == nil {
			if _, err = img.TopLayer(); err == nil {
				if attempt > 0 {
					logger.Infof("Successfully opened remote image after %d retries", attempt)
				}
				return img, nil
			}
		}
		lastErr = err
		if !isRetryable(err) {
			return nil, err
		}
		if attempt < len(topLayerDelays) {
			logger.Warnf("Failed to open remote image (attempt %d/%d): %v, retrying in %v", attempt+1, len(topLayerDelays)+1, err, topLayerDelays[attempt])
			topLayerSleep(topLayerDelays[attempt])
		}
	}
	return nil, lastErr
}
