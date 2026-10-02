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

package v1alpha1

// Small helpers for the optional fields every managed resource is made of.
//
// They live here rather than in the controller package because the generated
// deepcopy code and the API types need them, and a managed resource kind is
// also where the shared policy harness's methods live.

// StringPtr returns a pointer to s.
func StringPtr(s string) *string { return &s }

// BoolPtr returns a pointer to b.
func BoolPtr(b bool) *bool { return &b }

// Int64Ptr returns a pointer to i.
func Int64Ptr(i int64) *int64 { return &i }

// Deref returns the value p points at, or the zero value when p is nil.
//
// It is what makes an optional field safe to read: a field the operator never
// set reads as its zero value rather than panicking.
func Deref(p *string) string {
	if p == nil {
		return ""
	}

	return *p
}

// DerefBool returns the value p points at, or false when p is nil.
func DerefBool(p *bool) bool {
	if p == nil {
		return false
	}

	return *p
}

// DerefInt64 returns the value p points at, or zero when p is nil.
func DerefInt64(p *int64) int64 {
	if p == nil {
		return 0
	}

	return *p
}
