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

package humanuser

import (
	"github.com/loafoe/provider-zitadel/apis/zitadel/v1alpha1"
	"github.com/loafoe/provider-zitadel/internal/clients/zitadel"
	"github.com/loafoe/provider-zitadel/internal/controller/common"
)

// profile maps the desired profile of cr onto the client representation.
// Fields that are not set are left as "unset" so that the client can skip them.
func profile(fp v1alpha1.HumanUserParameters) *zitadel.SetHumanProfile {
	p := &zitadel.SetHumanProfile{
		GivenName:  fp.GivenName,
		FamilyName: fp.FamilyName,
	}

	if fp.NickName != nil {
		p.NickName = *fp.NickName
		p.HasNickName = true
	}

	if fp.DisplayName != nil {
		p.DisplayName = *fp.DisplayName
		p.HasDisplayName = true
	}

	if fp.PreferredLanguage != nil {
		p.PreferredLanguage = *fp.PreferredLanguage
		p.HasLanguage = true
	}

	if fp.Gender != nil {
		g, err := zitadel.GenderToProto(string(*fp.Gender))
		if err == nil {
			p.Gender = g
			p.HasGender = true
		}
	}

	return p
}

// metadata maps the desired metadata of cr onto the client representation.
func metadata(fp v1alpha1.HumanUserParameters) []zitadel.MetadataEntry {
	out := make([]zitadel.MetadataEntry, 0, len(fp.Metadata))
	for _, m := range fp.Metadata {
		out = append(out, zitadel.MetadataEntry{Key: m.Key, Value: m.Value})
	}

	return out
}

// updateStatus copies the observed state of u into the status of cr.
func updateStatus(cr *v1alpha1.HumanUser, u *zitadel.User) {
	cr.Status.AtProvider.ID = common.StringPtr(u.UserID)
	cr.Status.AtProvider.UserName = common.StringPtr(u.UserName)
	cr.Status.AtProvider.PreferredLoginName = common.StringPtr(u.PreferredLoginName)
	cr.Status.AtProvider.LoginNames = u.LoginNames
	cr.Status.AtProvider.CreationDate = common.ParseTime(u.CreationDate)
	cr.Status.AtProvider.ChangeDate = common.ParseTime(u.ChangeDate)

	if u.State != "" {
		state := v1alpha1.UserState(u.State)
		cr.Status.AtProvider.State = &state
	}

	if u.Human == nil {
		return
	}

	cr.Status.AtProvider.Email = common.StringPtr(u.Human.Email)
	cr.Status.AtProvider.EmailVerified = common.BoolPtr(u.Human.EmailVerified)

	if u.Human.Phone != "" {
		cr.Status.AtProvider.Phone = common.StringPtr(u.Human.Phone)
		cr.Status.AtProvider.PhoneVerified = common.BoolPtr(u.Human.PhoneVerified)
	}

	if u.Human.DisplayName != "" {
		cr.Status.AtProvider.DisplayName = common.StringPtr(u.Human.DisplayName)
	}

	if u.Human.NickName != "" {
		cr.Status.AtProvider.NickName = common.StringPtr(u.Human.NickName)
	}

	cr.Status.AtProvider.PasswordChangeRequired = common.BoolPtr(u.Human.PasswordChangeRequire)
}

// isUpToDate reports whether the remote user matches the desired state
// described by cr.
//
// Zitadel never returns the initial password, idp links or metadata of a user
// after creation, so those fields are create-only and cannot be drift detected.
//
//nolint:gocyclo // flat comparison of the optional forProvider fields
func isUpToDate(cr *v1alpha1.HumanUser, u *zitadel.User) bool {
	fp := cr.Spec.ForProvider

	if u.Human == nil {
		return false
	}

	if fp.UserName != nil && *fp.UserName != u.UserName {
		return false
	}

	if u.Human.GivenName != fp.GivenName || u.Human.FamilyName != fp.FamilyName {
		return false
	}

	if fp.NickName != nil && *fp.NickName != u.Human.NickName {
		return false
	}

	if fp.DisplayName != nil && *fp.DisplayName != u.Human.DisplayName {
		return false
	}

	if fp.PreferredLanguage != nil && *fp.PreferredLanguage != u.Human.PreferredLanguage {
		return false
	}

	if fp.Gender != nil && string(*fp.Gender) != u.Human.Gender {
		return false
	}

	if u.Human.Email != fp.Email {
		return false
	}

	if fp.Phone != nil && *fp.Phone != u.Human.Phone {
		return false
	}

	if fp.State != nil && string(*fp.State) != u.State {
		return false
	}

	return true
}
