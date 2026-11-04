/*
Copyright 2026 The Skaffold Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package cloudrun

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"syscall"
	"time"

	"github.com/cenkalti/backoff/v4"
	"google.golang.org/api/googleapi"

	sErrors "github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/errors"
	logger "github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/output/log"
	"github.com/GoogleContainerTools/skaffold/v2/proto/v1"
)

// Retry and status-poll backoff parameters for transient Cloud Run Admin API errors.
// Across 5 retries, nominal sleep intervals follow 1s -> 2s -> 4s -> 8s -> 10s
// (~25s nominal total sleep; 12.5s-37.5s with ±50% jitter via backoff.DefaultRandomizationFactor).
const (
	defaultRetryInitialInterval    = 1 * time.Second
	defaultRetryMaxInterval        = 10 * time.Second
	defaultRetryMultiplier         = 2.0
	defaultMaxRetries              = 5
	maxStatusPollBackoffMultiplier = 10
)

// newRetryBackOff constructs the retry backoff policy for Cloud Run Admin API mutations and lookups.
// Overridable in unit tests.
var newRetryBackOff = func(ctx context.Context) backoff.BackOff {
	b := backoff.NewExponentialBackOff(
		backoff.WithInitialInterval(defaultRetryInitialInterval),
		backoff.WithMaxInterval(defaultRetryMaxInterval),
		backoff.WithMultiplier(defaultRetryMultiplier),
		backoff.WithRandomizationFactor(backoff.DefaultRandomizationFactor),
		backoff.WithMaxElapsedTime(0),
	)
	return backoff.WithContext(backoff.WithMaxRetries(b, defaultMaxRetries), ctx)
}

// isRetryableError returns true for transient Cloud Run Admin API HTTP errors
// (429, 500, 502, 503, 504) and transient network/transport failures, and false
// for non-retryable client errors (e.g., 400, 401, 403, 404, 409, 412, 422) or
// context cancellation.
func isRetryableError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}

	var gErr *googleapi.Error
	if errors.As(err, &gErr) {
		switch gErr.Code {
		case http.StatusTooManyRequests,
			http.StatusInternalServerError,
			http.StatusBadGateway,
			http.StatusServiceUnavailable,
			http.StatusGatewayTimeout:
			return true
		default:
			return false
		}
	}

	if errors.Is(err, io.ErrUnexpectedEOF) ||
		errors.Is(err, io.EOF) ||
		errors.Is(err, syscall.ECONNRESET) ||
		errors.Is(err, syscall.ECONNREFUSED) ||
		errors.Is(err, syscall.EPIPE) {
		return true
	}

	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return true
	}

	return false
}

// deployResourceWithRetry executes the Cloud Run Get -> Create/Replace sequence with
// exponential backoff on transient errors. Re-evaluating Get on each retry ensures
// idempotency if an earlier Create succeeded backend-side before returning a 503.
func deployResourceWithRetry(
	ctx context.Context,
	resourceKind string,
	resourceName string,
	getErrCode proto.StatusCode,
	updateErrCode proto.StatusCode,
	getFn func(context.Context) error,
	createFn func(context.Context) error,
	replaceFn func(context.Context) error,
) error {
	lastErrCode := getErrCode
	err := backoff.RetryNotify(
		func() error {
			err := getFn(ctx)
			if err != nil {
				var gErr *googleapi.Error
				if !errors.As(err, &gErr) || gErr.Code != http.StatusNotFound {
					lastErrCode = getErrCode
					wrappedErr := sErrors.NewError(fmt.Errorf("error checking Cloud Run State: %w", err), &proto.ActionableErr{
						Message: err.Error(),
						ErrCode: getErrCode,
					})
					if isRetryableError(err) {
						return wrappedErr
					}
					return backoff.Permanent(wrappedErr)
				}
				// Resource does not exist yet (404); create it.
				err = createFn(ctx)
			} else {
				// Resource exists (200); replace it.
				err = replaceFn(ctx)
			}

			if err != nil {
				lastErrCode = updateErrCode
				wrappedErr := sErrors.NewError(fmt.Errorf("error deploying Cloud Run %s: %w", resourceKind, err), &proto.ActionableErr{
					Message: err.Error(),
					ErrCode: updateErrCode,
				})
				if isRetryableError(err) {
					return wrappedErr
				}
				return backoff.Permanent(wrappedErr)
			}
			return nil
		},
		newRetryBackOff(ctx),
		func(err error, next time.Duration) {
			logger.Entry(ctx).Debugf("Transient error deploying Cloud Run %s %q, retrying in %v: %v", resourceKind, resourceName, next, err)
		},
	)
	if err != nil {
		var sErr sErrors.Error
		if !errors.As(err, &sErr) {
			var wrapped error
			if lastErrCode == getErrCode {
				wrapped = fmt.Errorf("error checking Cloud Run State: %w", err)
			} else {
				wrapped = fmt.Errorf("error deploying Cloud Run %s: %w", resourceKind, err)
			}
			return sErrors.NewError(wrapped, &proto.ActionableErr{
				Message: err.Error(),
				ErrCode: lastErrCode,
			})
		}
		return err
	}
	return nil
}

// deleteResourceWithRetry executes a Cloud Run Delete call with exponential backoff
// on transient errors. If a retry receives 404 after an earlier transient error, it
// treats the resource as already deleted.
func deleteResourceWithRetry(
	ctx context.Context,
	resourceKind string,
	resourceName string,
	deleteErrCode proto.StatusCode,
	deleteFn func(context.Context) error,
) error {
	attempt := 0
	err := backoff.RetryNotify(
		func() error {
			currentAttempt := attempt
			attempt++
			err := deleteFn(ctx)
			if err == nil {
				return nil
			}
			var gErr *googleapi.Error
			if currentAttempt > 0 && errors.As(err, &gErr) && gErr.Code == http.StatusNotFound {
				return nil
			}
			wrappedErr := sErrors.NewError(fmt.Errorf("unable to delete Cloud Run %s", resourceKind), &proto.ActionableErr{
				Message: err.Error(),
				ErrCode: deleteErrCode,
			})
			if isRetryableError(err) {
				return wrappedErr
			}
			return backoff.Permanent(wrappedErr)
		},
		newRetryBackOff(ctx),
		func(err error, next time.Duration) {
			logger.Entry(ctx).Debugf("Transient error deleting Cloud Run %s %q, retrying in %v: %v", resourceKind, resourceName, next, err)
		},
	)
	if err != nil {
		var sErr sErrors.Error
		if !errors.As(err, &sErr) {
			return sErrors.NewError(fmt.Errorf("unable to delete Cloud Run %s", resourceKind), &proto.ActionableErr{
				Message: err.Error(),
				ErrCode: deleteErrCode,
			})
		}
		return err
	}
	return nil
}
