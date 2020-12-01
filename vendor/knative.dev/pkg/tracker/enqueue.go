/*
Copyright 2018 The Knative Authors

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

package tracker

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/validation"

	"knative.dev/pkg/kmeta"
)

// New returns an implementation of Interface that lets a Reconciler
// register a particular resource as watching an ObjectReference for
// a particular lease duration.  This watch must be refreshed
// periodically (e.g. by a controller resync) or it will expire.
//
// When OnChanged is called by the informer for a particular
// GroupVersionKind, the provided callback is called with the "key"
// of each object actively watching the changed object.
func New(callback func(types.NamespacedName), lease time.Duration) Interface {
	return &impl{
		leaseDuration: lease,
		cb:            callback,
	}
}

type impl struct {
	m sync.Mutex
	// exact maps from an object reference to the set of
	// keys for objects watching it.
	exact map[Reference]set
	// inexact maps from a partial object reference (no name/selector) to
	// a map from watcher keys to the compiled selector and expiry.
	inexact map[Reference]matchers

	// The amount of time that an object may watch another
	// before having to renew the lease.
	leaseDuration time.Duration

	cb func(types.NamespacedName)
}

// Check that impl implements Interface.
var _ Interface = (*impl)(nil)

// set is a map from keys to expirations
type set map[types.NamespacedName]time.Time

// matchers maps the tracker's key to the matcher.
type matchers map[types.NamespacedName]matcher

// matcher holds the selector and expiry for matching tracked objects.
type matcher struct {
	// The selector to complete the match.
	selector labels.Selector

	// When this lease expires.
	expiry time.Time
}

// Track implements Interface.
func (i *impl) Track(ref corev1.ObjectReference, obj interface{}) error {
	return i.TrackReference(Reference{
		APIVersion: ref.APIVersion,
		Kind:       ref.Kind,
		Namespace:  ref.Namespace,
		Name:       ref.Name,
	}, obj)
}

func (i *impl) TrackReference(ref Reference, obj interface{}) error {
	invalidFields := map[string][]string{
		"APIVersion": validation.IsQualifiedName(ref.APIVersion),
		"Kind":       validation.IsCIdentifier(ref.Kind),
		"Namespace":  validation.IsDNS1123Label(ref.Namespace),
	}
	var selector labels.Selector
	fieldErrors := []string{}
	switch {
	case ref.Selector != nil && ref.Name != "":
		fieldErrors = append(fieldErrors, "cannot provide both Name and Selector")
	case ref.Name != "":
		invalidFields["Name"] = validation.IsDNS1123Subdomain(ref.Name)
	case ref.Selector != nil:
		ls, err := metav1.LabelSelectorAsSelector(ref.Selector)
		if err != nil {
			invalidFields["Selector"] = []string{err.Error()}
		}
		selector = ls
	default:
		fieldErrors = append(fieldErrors, "must provide either Name or Selector")
	}
	for k, v := range invalidFields {
		for _, msg := range v {
			fieldErrors = append(fieldErrors, fmt.Sprintf("%s: %s", k, msg))
		}
	}
	if len(fieldErrors) > 0 {
		sort.Strings(fieldErrors)
		return fmt.Errorf("invalid Reference:\n%s", strings.Join(fieldErrors, "\n"))
	}

	// Determine the key of the object tracking this reference.
	object, err := kmeta.DeletionHandlingAccessor(obj)
	if err != nil {
		return err
	}
	key := types.NamespacedName{Namespace: object.GetNamespace(), Name: object.GetName()}

	i.m.Lock()
	// Call the callback without the lock held.
	var keys []types.NamespacedName
	defer func(cb func(types.NamespacedName)) {
		for _, key := range keys {
			cb(key)
		}
	}(i.cb) // read i.cb with the lock held
	defer i.m.Unlock()
	if i.exact == nil {
		i.exact = make(map[Reference]set)
	}
	if i.inexact == nil {
		i.inexact = make(map[Reference]matchers)
	}

	// If the reference uses Name then it is an exact match.
	if selector == nil {
		l, ok := i.exact[ref]
		if !ok {
			l = set{}
		}

		if expiry, ok := l[key]; !ok || isExpired(expiry) {
			// When covering an uncovered key, immediately call the
			// registered callback to ensure that the following pattern
			// doesn't create problems:
			//    foo, err := lister.Get(key)
			//    // Later...
			//    err := tracker.TrackReference(fooRef, parent)
			// In this example, "Later" represents a window where "foo" may
			// have changed or been created while the Track is not active.
			// The simplest way of eliminating such a window is to call the
			// callback to "catch up" immediately following new
			// registrations.
			keys = append(keys, key)
		}
		// Overwrite the key with a new expiration.
		l[key] = time.Now().Add(i.leaseDuration)

		i.exact[ref] = l
		return nil
	}

	// Otherwise, it is an inexact match by selector.
	partialRef := Reference{
		APIVersion: ref.APIVersion,
		Kind:       ref.Kind,
		Namespace:  ref.Namespace,
		// Exclude the selector.
	}
	l, ok := i.inexact[partialRef]
	if !ok {
		l = matchers{}
	}

	if m, ok := l[key]; !ok || isExpired(m.expiry) {
		// When covering an uncovered key, immediately call the
		// registered callback to ensure that the following pattern
		// doesn't create problems:
		//    foo, err := lister.Get(key)
		//    // Later...
		//    err := tracker.TrackReference(fooRef, parent)
		// In this example, "Later" represents a window where "foo" may
		// have changed or been created while the Track is not active.
		// The simplest way of eliminating such a window is to call the
		// callback to "catch up" immediately following new
		// registrations.
		keys = append(keys, key)
	}
	// Overwrite the key with a new expiration.
	l[key] = matcher{
		selector: selector,
		expiry:   time.Now().Add(i.leaseDuration),
	}

	i.inexact[partialRef] = l
	return nil
}

func isExpired(expiry time.Time) bool {
	return time.Now().After(expiry)
}

// OnChanged implements Interface.
func (i *impl) OnChanged(obj interface{}) {
	observers := i.GetObservers(obj)

	for _, observer := range observers {
		i.cb(observer)
	}
}

// GetObservers implements Interface.
func (i *impl) GetObservers(obj interface{}) []types.NamespacedName {
	item, err := kmeta.DeletionHandlingAccessor(obj)
	if err != nil {
		return nil
	}

	or := kmeta.ObjectReference(item)
	ref := Reference{
		APIVersion: or.APIVersion,
		Kind:       or.Kind,
		Namespace:  or.Namespace,
		Name:       or.Name,
	}

	var keys []types.NamespacedName

	i.m.Lock()
	defer i.m.Unlock()

	// Handle exact matches.
	s, ok := i.exact[ref]
	if ok {
		for key, expiry := range s {
			// If the expiration has lapsed, then delete the key.
			if isExpired(expiry) {
				delete(s, key)
				continue
			}
			keys = append(keys, key)
		}
		if len(s) == 0 {
			delete(i.exact, ref)
		}
	}

	// Handle inexact matches.
	ref.Name = ""
	ms, ok := i.inexact[ref]
	if ok {
		ls := labels.Set(item.GetLabels())
		for key, m := range ms {
			// If the expiration has lapsed, then delete the key.
			if isExpired(m.expiry) {
				delete(ms, key)
				continue
			}
			if m.selector.Matches(ls) {
				keys = append(keys, key)
			}
		}
		if len(s) == 0 {
			delete(i.exact, ref)
		}
	}

	return keys
}

// OnChanged implements Interface.
func (i *impl) OnDeletedObserver(obj interface{}) {
	item, err := kmeta.DeletionHandlingAccessor(obj)
	if err != nil {
		return
	}

	key := types.NamespacedName{Namespace: item.GetNamespace(), Name: item.GetName()}

	i.m.Lock()
	defer i.m.Unlock()

	// Remove exact matches.
	for ref, matchers := range i.exact {
		delete(matchers, key)
		if len(matchers) == 0 {
			delete(i.exact, ref)
		}
	}

	// Remove inexact matches.
	for ref, matchers := range i.inexact {
		delete(matchers, key)
		if len(matchers) == 0 {
			delete(i.exact, ref)
		}
	}
}
