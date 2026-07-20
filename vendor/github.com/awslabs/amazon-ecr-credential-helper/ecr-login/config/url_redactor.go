// Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License"). You may
// not use this file except in compliance with the License. A copy of the
// License is located at
//
//	http://aws.amazon.com/apache2.0/
//
// or in the "license" file accompanying this file. This file is distributed
// on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either
// express or implied. See the License for the specific language governing
// permissions and limitations under the License.

package config

import (
	"errors"
	"net/url"

	"github.com/sirupsen/logrus"
)

// URLRedactorHook is a logrus hook that sanitizes URL fields in log entries
// to prevent leaking sensitive information like userinfo (passwords) and query parameters.
type URLRedactorHook struct{}

// Levels returns all log levels that this hook should be applied to
func (hook *URLRedactorHook) Levels() []logrus.Level {
	return logrus.AllLevels
}

// Fire is called when a log event is fired. It sanitizes the serverURL field
// and error fields that may contain URLs.
func (hook *URLRedactorHook) Fire(entry *logrus.Entry) error {
	// Redact serverURL field
	if value, exists := entry.Data["serverURL"]; exists {
		if strValue, ok := value.(string); ok {
			entry.Data["serverURL"] = RedactURL(strValue)
		}
	}

	// Redact URLs in error field
	if value, exists := entry.Data[logrus.ErrorKey]; exists {
		if err, ok := value.(error); ok {
			entry.Data[logrus.ErrorKey] = RedactURLFromError(err)
		}
	}

	return nil
}

// RedactURL redacts sensitive information from a URL string including:
// - Password in userinfo (user:password@host)
// - Query parameter values
// Returns the original string unchanged if it's not a valid URL.
func RedactURL(rawURL string) string {
	if rawURL == "" {
		return rawURL
	}

	parsed, err := url.ParseRequestURI(rawURL)
	if err != nil {
		return rawURL
	}

	return redactParsedURL(parsed)
}

// redactParsedURL redacts password and query parameters from a parsed URL
func redactParsedURL(parsed *url.URL) string {
	if parsed.User != nil {
		if _, hasPassword := parsed.User.Password(); hasPassword {
			parsed.User = url.UserPassword(parsed.User.Username(), "xxxxx")
		}
	}

	if query := parsed.Query(); len(query) > 0 {
		for k := range query {
			query.Set(k, "redacted")
		}
		parsed.RawQuery = query.Encode()
	}

	return parsed.String()
}

// RedactURLFromError redacts URL query parameter values from url.Error.
// This handles cases where HTTP errors contain URLs with sensitive query parameters.
// Returns the original error if it's not a url.Error or cannot be parsed.
func RedactURLFromError(err error) error {
	var urlErr *url.Error

	if err != nil && errors.As(err, &urlErr) {
		parsedURL, urlParseErr := url.Parse(urlErr.URL)
		if urlParseErr == nil && parsedURL.Scheme != "" && parsedURL.Host != "" {
			urlErr.URL = redactParsedURL(parsedURL)
			return urlErr
		}
	}

	return err
}
