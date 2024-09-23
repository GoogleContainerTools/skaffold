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

package cache

import (
	"time"
)

type CredentialsCache interface {
	Get(registry string) *AuthEntry
	GetPublic() *AuthEntry
	Set(registry string, entry *AuthEntry)
	List() []*AuthEntry
	Clear()
}

type Service string

const (
	ServiceECR       Service = "ecr"
	ServiceECRPublic Service = "ecr-public"
)

type AuthEntry struct {
	AuthorizationToken string
	RequestedAt        time.Time
	ExpiresAt          time.Time
	ProxyEndpoint      string
	Service            Service
}

// IsValid checks if AuthEntry is still valid at testTime. AuthEntries expire at 1/2 of their original
// requested window.
func (authEntry *AuthEntry) IsValid(testTime time.Time) bool {
	validWindow := authEntry.ExpiresAt.Sub(authEntry.RequestedAt)
	refreshTime := authEntry.ExpiresAt.Add(-1 * validWindow / time.Duration(2))
	return testTime.Before(refreshTime)
}
