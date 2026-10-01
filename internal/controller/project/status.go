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

package project

import (
	"github.com/loafoe/provider-zitadel/apis/zitadel/v1alpha1"
	"github.com/loafoe/provider-zitadel/internal/clients/zitadel"
	"github.com/loafoe/provider-zitadel/internal/controller/common"
)

// updateStatus copies the observed state of p into the status of cr.
func updateStatus(cr *v1alpha1.Project, p *zitadel.Project) {
	cr.Status.AtProvider.ID = common.StringPtr(p.ProjectID)
	cr.Status.AtProvider.OrganizationID = common.StringPtr(p.OrganizationID)
	cr.Status.AtProvider.Name = common.StringPtr(p.Name)

	if p.State != "" {
		state := v1alpha1.ProjectState(p.State)
		cr.Status.AtProvider.State = &state
	}

	cr.Status.AtProvider.ProjectRoleAssertion = common.BoolPtr(p.ProjectRoleAssertion)
	cr.Status.AtProvider.AuthorizationRequired = common.BoolPtr(p.AuthorizationRequired)
	cr.Status.AtProvider.ProjectAccessRequired = common.BoolPtr(p.ProjectAccessRequired)

	if p.PrivateLabelingSetting != "" {
		pls := v1alpha1.PrivateLabelingSetting(p.PrivateLabelingSetting)
		cr.Status.AtProvider.PrivateLabelingSetting = &pls
	}

	if p.GrantedOrganizationID != "" {
		cr.Status.AtProvider.GrantedOrganizationID = common.StringPtr(p.GrantedOrganizationID)
	}

	if p.GrantedState != "" {
		cr.Status.AtProvider.GrantedState = common.StringPtr(p.GrantedState)
	}

	cr.Status.AtProvider.CreationDate = common.ParseTime(p.CreationDate)
	cr.Status.AtProvider.ChangeDate = common.ParseTime(p.ChangeDate)
}

// isUpToDate reports whether the remote project matches the desired state
// described by cr.
//
//nolint:gocyclo // flat comparison of the optional forProvider fields
func isUpToDate(cr *v1alpha1.Project, p *zitadel.Project) bool {
	fp := cr.Spec.ForProvider

	if fp.Name != p.Name {
		return false
	}

	if fp.ProjectRoleAssertion != nil && *fp.ProjectRoleAssertion != p.ProjectRoleAssertion {
		return false
	}

	if fp.AuthorizationRequired != nil && *fp.AuthorizationRequired != p.AuthorizationRequired {
		return false
	}

	if fp.ProjectAccessRequired != nil && *fp.ProjectAccessRequired != p.ProjectAccessRequired {
		return false
	}

	if fp.PrivateLabelingSetting != nil && string(*fp.PrivateLabelingSetting) != p.PrivateLabelingSetting {
		return false
	}

	if fp.State != nil && string(*fp.State) != p.State {
		return false
	}

	return true
}
