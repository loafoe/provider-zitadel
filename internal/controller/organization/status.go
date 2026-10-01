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

package organization

import (
	"github.com/loafoe/provider-zitadel/apis/zitadel/v1alpha1"
	"github.com/loafoe/provider-zitadel/internal/clients/zitadel"
	"github.com/loafoe/provider-zitadel/internal/controller/common"
)

// updateStatus copies the observed state of org into the status of cr.
func updateStatus(cr *v1alpha1.Organization, org *zitadel.Organization) {
	cr.Status.AtProvider.ID = common.StringPtr(org.ID)
	cr.Status.AtProvider.Name = common.StringPtr(org.Name)
	if org.PrimaryDomain != "" {
		cr.Status.AtProvider.PrimaryDomain = common.StringPtr(org.PrimaryDomain)
	}
	if org.State != "" {
		state := v1alpha1.OrganizationState(org.State)
		cr.Status.AtProvider.State = &state
	}
	cr.Status.AtProvider.CreationDate = common.ParseTime(org.CreationDate)
	cr.Status.AtProvider.ChangeDate = common.ParseTime(org.ChangeDate)
}

// isUpToDate reports whether the remote organization matches the desired
// state described by cr.
func isUpToDate(cr *v1alpha1.Organization, org *zitadel.Organization) bool {
	fp := cr.Spec.ForProvider

	if fp.Name != org.Name {
		return false
	}

	if fp.State != nil && string(*fp.State) != org.State {
		return false
	}

	// PrimaryDomain is read-only: it is reported for convenience but never
	// managed by this provider.

	return true
}
