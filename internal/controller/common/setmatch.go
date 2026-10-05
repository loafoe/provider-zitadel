/*
Copyright 2025 The Crossplane Authors.

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

package common

// Deciding what counts as drift
//
// A policy driver is asked the same question by every policy: does what Zitadel
// hold already match what the manifest asked for? The answer is not a plain
// comparison, because the manifest is allowed to leave fields out.
//
// Desired has to turn an unset field into something, and it turns it into the
// zero value. Comparing that against whatever Zitadel happens to hold reports
// drift on every poll, for a field the operator never asked to have managed, so
// the resource is rewritten forever and never settles. Every field of every
// policy here is optional in the CRD, so this is the normal case rather than an
// edge case.
//
// An unset field is therefore not compared. These helpers are where that rule
// lives, so that twenty-one drivers cannot each spell it slightly differently -
// and so that a driver that forgets it is visibly the odd one out.

// SetMatch is one field of a policy under comparison: the field's name, whether
// the manifest set it, what it asks for, and what Zitadel holds.
type SetMatch[T any] struct {
	Name string

	// Set reports whether the manifest gave this field a value. An unset field
	// is left alone.
	Set bool

	Want T
	Got  T
}

// Match builds one scalar field comparison.
//
// The name is only ever used in a failure message, and is carried so that one
// exists if a driver ever wants to report which field drifted.
func Match[T comparable](name string, set bool, want, got T) SetMatch[T] {
	return SetMatch[T]{Name: name, Set: set, Want: want, Got: got}
}

// MatchList builds one list field comparison, which compares by content rather
// than by identity.
func MatchList(name string, set bool, want, got []string) SetMatch[[]string] {
	return SetMatch[[]string]{Name: name, Set: set, Want: want, Got: got}
}

// AllSetMatch reports whether every field the manifest set matches what Zitadel
// holds. A field the manifest left unset is not drift, whatever Zitadel has.
//
// Asking for zero and asking for nothing are not the same request, and this
// keeps them apart: an explicit zero has Set true, so it is compared.
func AllSetMatch[T comparable](fields ...SetMatch[T]) bool {
	for _, f := range fields {
		if f.Set && f.Want != f.Got {
			return false
		}
	}

	return true
}

// AllSetMatchLists is AllSetMatch for a list field, where equality is by content
// rather than by identity.
//
// Order is significant, as it is for every list in this provider: Zitadel
// reports these in a stable order, and a manifest is expected to match it. A
// list the manifest left unset is not compared either.
func AllSetMatchLists(fields ...SetMatch[[]string]) bool {
	for _, f := range fields {
		if f.Set && !EqualStringSlices(f.Want, f.Got) {
			return false
		}
	}

	return true
}
