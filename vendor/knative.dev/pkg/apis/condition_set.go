/*
Copyright 2019 The Knative Authors

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

package apis

import (
	"reflect"
	"sort"
	"time"

	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Conditions is the interface for a Resource that implements the getter and
// setter for accessing a Condition collection.
// +k8s:deepcopy-gen=true
type ConditionsAccessor interface {
	GetConditions() Conditions
	SetConditions(Conditions)
}

// ConditionAccessor is used to access a condition through it's type
type ConditionAccessor interface {
	// GetCondition finds and returns the Condition that matches the ConditionType
	// It should return nil if the condition type is not present
	GetCondition(t ConditionType) *Condition
}

// ConditionSet is an abstract collection of the possible ConditionType values
// that a particular resource might expose.  It also holds the "happy condition"
// for that resource, which we define to be one of Ready or Succeeded depending
// on whether it is a Living or Batch process respectively.
// +k8s:deepcopy-gen=false
type ConditionSet struct {
	happy      ConditionType
	dependents []ConditionType
}

// ConditionManager allows a resource to operate on its Conditions using higher
// order operations.
type ConditionManager interface {
	ConditionAccessor

	// IsHappy looks at the happy condition and returns true if that condition is
	// set to true.
	IsHappy() bool

	// GetTopLevelCondition finds and returns the top level Condition (happy Condition).
	GetTopLevelCondition() *Condition

	// SetCondition sets or updates the Condition on Conditions for Condition.Type.
	// If there is an update, Conditions are stored back sorted.
	SetCondition(new Condition)

	// ClearCondition removes the non terminal condition that matches the ConditionType
	ClearCondition(t ConditionType) error

	// MarkTrue sets the status of t to true, and then marks the happy condition to
	// true if all dependents are true.
	MarkTrue(t ConditionType)

	// MarkTrueWithReason sets the status of t to true with the reason, and then marks the happy
	// condition to true if all dependents are true.
	MarkTrueWithReason(t ConditionType, reason, messageFormat string, messageA ...interface{})

	// MarkUnknown sets the status of t to Unknown and also sets the happy condition
	// to Unknown if no other dependent condition is in an error state.
	MarkUnknown(t ConditionType, reason, messageFormat string, messageA ...interface{})

	// MarkFalse sets the status of t and the happy condition to False.
	MarkFalse(t ConditionType, reason, messageFormat string, messageA ...interface{})

	// InitializeConditions updates all Conditions in the ConditionSet to Unknown
	// if not set.
	InitializeConditions()
}

// NewLivingConditionSet returns a ConditionSet to hold the conditions for the
// living resource. ConditionReady is used as the happy condition.
// The set of condition types provided are those of the terminal subconditions.
func NewLivingConditionSet(d ...ConditionType) ConditionSet {
	return newConditionSet(ConditionReady, d...)
}

// NewBatchConditionSet returns a ConditionSet to hold the conditions for the
// batch resource. ConditionSucceeded is used as the happy condition.
// The set of condition types provided are those of the terminal subconditions.
func NewBatchConditionSet(d ...ConditionType) ConditionSet {
	return newConditionSet(ConditionSucceeded, d...)
}

// newConditionSet returns a ConditionSet to hold the conditions that are
// important for the caller. The first ConditionType is the overarching status
// for that will be used to signal the resources' status is Ready or Succeeded.
func newConditionSet(happy ConditionType, dependents ...ConditionType) ConditionSet {
	var deps []ConditionType
	for _, d := range dependents {
		// Skip duplicates
		if d == happy || contains(deps, d) {
			continue
		}
		deps = append(deps, d)
	}
	return ConditionSet{
		happy:      happy,
		dependents: deps,
	}
}

func contains(ct []ConditionType, t ConditionType) bool {
	for _, c := range ct {
		if c == t {
			return true
		}
	}
	return false
}

// Check that conditionsImpl implements ConditionManager.
var _ ConditionManager = (*conditionsImpl)(nil)

// conditionsImpl implements the helper methods for evaluating Conditions.
// +k8s:deepcopy-gen=false
type conditionsImpl struct {
	ConditionSet
	accessor ConditionsAccessor
}

// Manage creates a ConditionManager from an accessor object using the original
// ConditionSet as a reference. Status must be a pointer to a struct.
func (r ConditionSet) Manage(status ConditionsAccessor) ConditionManager {
	return conditionsImpl{
		accessor:     status,
		ConditionSet: r,
	}
}

// IsHappy looks at the top level Condition (happy Condition) and returns true if that condition is
// set to true.
func (r conditionsImpl) IsHappy() bool {
	return r.GetTopLevelCondition().IsTrue()
}

// GetTopLevelCondition finds and returns the top level Condition (happy Condition).
func (r conditionsImpl) GetTopLevelCondition() *Condition {
	return r.GetCondition(r.happy)
}

// GetCondition finds and returns the Condition that matches the ConditionType
// previously set on Conditions.
func (r conditionsImpl) GetCondition(t ConditionType) *Condition {
	if r.accessor == nil {
		return nil
	}

	for _, c := range r.accessor.GetConditions() {
		if c.Type == t {
			return &c
		}
	}
	return nil
}

// SetCondition sets or updates the Condition on Conditions for Condition.Type.
// If there is an update, Conditions are stored back sorted.
func (r conditionsImpl) SetCondition(new Condition) {
	if r.accessor == nil {
		return
	}
	t := new.Type
	var conditions Conditions
	for _, c := range r.accessor.GetConditions() {
		if c.Type != t {
			conditions = append(conditions, c)
		} else {
			// If we'd only update the LastTransitionTime, then return.
			new.LastTransitionTime = c.LastTransitionTime
			if reflect.DeepEqual(&new, &c) {
				return
			}
		}
	}
	new.LastTransitionTime = VolatileTime{Inner: metav1.NewTime(time.Now())}
	conditions = append(conditions, new)
	// Sorted for convenience of the consumer, i.e. kubectl.
	sort.Slice(conditions, func(i, j int) bool { return conditions[i].Type < conditions[j].Type })
	r.accessor.SetConditions(conditions)
}

func (r conditionsImpl) isTerminal(t ConditionType) bool {
	for _, cond := range r.dependents {
		if cond == t {
			return true
		}
	}
	return t == r.happy
}

func (r conditionsImpl) severity(t ConditionType) ConditionSeverity {
	if r.isTerminal(t) {
		return ConditionSeverityError
	}
	return ConditionSeverityInfo
}

// RemoveCondition removes the non terminal condition that matches the ConditionType
// Not implemented for terminal conditions
func (r conditionsImpl) ClearCondition(t ConditionType) error {
	var conditions Conditions

	if r.accessor == nil {
		return nil
	}
	// Terminal conditions are not handled as they can't be nil
	if r.isTerminal(t) {
		return fmt.Errorf("Clearing terminal conditions not implemented")
	}
	cond := r.GetCondition(t)
	if cond == nil {
		return nil
	}
	for _, c := range r.accessor.GetConditions() {
		if c.Type != t {
			conditions = append(conditions, c)
		}
	}

	// Sorted for convenience of the consumer, i.e. kubectl.
	sort.Slice(conditions, func(i, j int) bool { return conditions[i].Type < conditions[j].Type })
	r.accessor.SetConditions(conditions)

	return nil
}

// MarkTrue sets the status of t to true, and then marks the happy condition to
// true if all other dependents are also true.
func (r conditionsImpl) MarkTrue(t ConditionType) {
	// Set the specified condition.
	r.SetCondition(Condition{
		Type:     t,
		Status:   corev1.ConditionTrue,
		Severity: r.severity(t),
	})
	r.recomputeHappiness(t)
}

// MarkTrueWithReason sets the status of t to true with the reason, and then marks the happy condition to
// true if all other dependents are also true.
func (r conditionsImpl) MarkTrueWithReason(t ConditionType, reason, messageFormat string, messageA ...interface{}) {
	// set the specified condition
	r.SetCondition(Condition{
		Type:     t,
		Status:   corev1.ConditionTrue,
		Reason:   reason,
		Message:  fmt.Sprintf(messageFormat, messageA...),
		Severity: r.severity(t),
	})
	r.recomputeHappiness(t)
}

// recomputeHappiness marks the happy condition to true if all other dependents are also true.
func (r conditionsImpl) recomputeHappiness(t ConditionType) {
	if c := r.findUnhappyDependent(); c != nil {
		// Propagate unhappy dependent to happy condition.
		r.SetCondition(Condition{
			Type:     r.happy,
			Status:   c.Status,
			Reason:   c.Reason,
			Message:  c.Message,
			Severity: r.severity(r.happy),
		})
	} else if t != r.happy {
		// Set the happy condition to true.
		r.SetCondition(Condition{
			Type:     r.happy,
			Status:   corev1.ConditionTrue,
			Severity: r.severity(r.happy),
		})
	}
}

func (r conditionsImpl) findUnhappyDependent() *Condition {
	// This only works if there are dependents.
	if len(r.dependents) == 0 {
		return nil
	}

	// Do not modify the accessors condition order.
	conditions := r.accessor.GetConditions().DeepCopy()

	// Filter based on terminal status.
	n := 0
	for _, c := range conditions {
		if c.Severity == ConditionSeverityError && c.Type != r.happy {
			conditions[n] = c
			n++
		}
	}
	conditions = conditions[:n]

	// Sort set conditions by time.
	sort.Slice(conditions, func(i, j int) bool {
		return conditions[i].LastTransitionTime.Inner.Time.After(conditions[j].LastTransitionTime.Inner.Time)
	})

	// First check the conditions with Status == False.
	for _, c := range conditions {
		// False conditions trump Unknown.
		if c.IsFalse() {
			return &c
		}
	}
	// Second check for conditions with Status == Unknown.
	for _, c := range conditions {
		if c.IsUnknown() {
			return &c
		}
	}

	// If something was not initialized.
	if len(r.dependents) > len(conditions) {
		return &Condition{
			Status: corev1.ConditionUnknown,
		}
	}

	// All dependents are fine.
	return nil
}

// MarkUnknown sets the status of t to Unknown and also sets the happy condition
// to Unknown if no other dependent condition is in an error state.
func (r conditionsImpl) MarkUnknown(t ConditionType, reason, messageFormat string, messageA ...interface{}) {
	// set the specified condition
	r.SetCondition(Condition{
		Type:     t,
		Status:   corev1.ConditionUnknown,
		Reason:   reason,
		Message:  fmt.Sprintf(messageFormat, messageA...),
		Severity: r.severity(t),
	})

	// check the dependents.
	isDependent := false
	for _, cond := range r.dependents {
		c := r.GetCondition(cond)
		// Failed conditions trump Unknown conditions
		if c.IsFalse() {
			// Double check that the happy condition is also false.
			happy := r.GetCondition(r.happy)
			if !happy.IsFalse() {
				r.MarkFalse(r.happy, reason, messageFormat, messageA...)
			}
			return
		}
		if cond == t {
			isDependent = true
		}
	}

	if isDependent {
		// set the happy condition, if it is one of our dependent subconditions.
		r.SetCondition(Condition{
			Type:     r.happy,
			Status:   corev1.ConditionUnknown,
			Reason:   reason,
			Message:  fmt.Sprintf(messageFormat, messageA...),
			Severity: r.severity(r.happy),
		})
	}
}

// MarkFalse sets the status of t and the happy condition to False.
func (r conditionsImpl) MarkFalse(t ConditionType, reason, messageFormat string, messageA ...interface{}) {
	types := []ConditionType{t}
	for _, cond := range r.dependents {
		if cond == t {
			types = append(types, r.happy)
		}
	}

	for _, t := range types {
		r.SetCondition(Condition{
			Type:     t,
			Status:   corev1.ConditionFalse,
			Reason:   reason,
			Message:  fmt.Sprintf(messageFormat, messageA...),
			Severity: r.severity(t),
		})
	}
}

// InitializeConditions updates all Conditions in the ConditionSet to Unknown
// if not set.
func (r conditionsImpl) InitializeConditions() {
	happy := r.GetCondition(r.happy)
	if happy == nil {
		happy = &Condition{
			Type:     r.happy,
			Status:   corev1.ConditionUnknown,
			Severity: ConditionSeverityError,
		}
		r.SetCondition(*happy)
	}
	// If the happy state is true, it implies that all of the terminal
	// subconditions must be true, so initialize any unset conditions to
	// true if our happy condition is true, otherwise unknown.
	status := corev1.ConditionUnknown
	if happy.Status == corev1.ConditionTrue {
		status = corev1.ConditionTrue
	}
	for _, t := range r.dependents {
		r.initializeTerminalCondition(t, status)
	}
}

// initializeTerminalCondition initializes a Condition to the given status if unset.
func (r conditionsImpl) initializeTerminalCondition(t ConditionType, status corev1.ConditionStatus) *Condition {
	if c := r.GetCondition(t); c != nil {
		return c
	}
	c := Condition{
		Type:     t,
		Status:   status,
		Severity: ConditionSeverityError,
	}
	r.SetCondition(c)
	return &c
}
