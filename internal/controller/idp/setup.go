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

package idp

import (
	"github.com/crossplane/crossplane-runtime/v2/pkg/controller"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/loafoe/provider-zitadel/apis/zitadel/v1alpha1"
	"github.com/loafoe/provider-zitadel/internal/controller/common"
)

// commonFields are the settings every identity provider carries.
//
// They are reached through an interface rather than read from each kind's own
// parameters, because there are twenty three of those and these four settings
// are identical in every one.
type commonFields interface {
	GetIsLinkingAllowed() *bool
	GetIsCreationAllowed() *bool
	GetIsAutoCreation() *bool
	GetIsAutoUpdate() *bool
	GetAutoLinking() *v1alpha1.IDPAutoLinking
	GetState() *v1alpha1.IDPState
}

// The wiring of every identity provider kind.
//
// Each entry says which Zitadel provider type it selects, which level it
// belongs to, and whether Zitadel reads it back - which is true only of an OIDC
// and a JWT provider, and decides whether its settings can be compared and
// changed.
//
// A JWT provider is organization only. Zitadel lets an organization accept one,
// but an instance cannot offer one to all of them, which is why the Terraform
// provider has an org_idp_jwt and no idp_jwt either.
//
//nolint:gocyclo // One branch per kind: a list of twenty three reads better than a dispatch.
func setups(mgr ctrl.Manager, o controller.Options) error {
	if err := common.SetupIDPController(mgr, o,
		v1alpha1.IDPOIDCGroupKind,
		v1alpha1.IDPOIDCGroupVersionKind,
		&v1alpha1.IDPOIDC{}, &v1alpha1.IDPOIDCList{},
		oidcDriver{base{"IDPOIDC", "OIDC", false, true}},
	); err != nil {
		return err
	}

	if err := common.SetupIDPController(mgr, o,
		v1alpha1.IDPOAuthGroupKind,
		v1alpha1.IDPOAuthGroupVersionKind,
		&v1alpha1.IDPOAuth{}, &v1alpha1.IDPOAuthList{},
		oauthDriver{base{"IDPOAuth", "OAuth", false, false}},
	); err != nil {
		return err
	}

	if err := common.SetupIDPController(mgr, o,
		v1alpha1.IDPAppleGroupKind,
		v1alpha1.IDPAppleGroupVersionKind,
		&v1alpha1.IDPApple{}, &v1alpha1.IDPAppleList{},
		appleDriver{base{"IDPApple", "Apple", false, false}},
	); err != nil {
		return err
	}

	if err := common.SetupIDPController(mgr, o,
		v1alpha1.IDPAzureADGroupKind,
		v1alpha1.IDPAzureADGroupVersionKind,
		&v1alpha1.IDPAzureAD{}, &v1alpha1.IDPAzureADList{},
		azureDriver{base{"IDPAzureAD", "AzureAD", false, false}},
	); err != nil {
		return err
	}

	if err := common.SetupIDPController(mgr, o,
		v1alpha1.IDPGitHubGroupKind,
		v1alpha1.IDPGitHubGroupVersionKind,
		&v1alpha1.IDPGitHub{}, &v1alpha1.IDPGitHubList{},
		githubDriver{base{"IDPGitHub", "GitHub", false, false}},
	); err != nil {
		return err
	}

	if err := common.SetupIDPController(mgr, o,
		v1alpha1.IDPGitHubEnterpriseServerGroupKind,
		v1alpha1.IDPGitHubEnterpriseServerGroupVersionKind,
		&v1alpha1.IDPGitHubEnterpriseServer{}, &v1alpha1.IDPGitHubEnterpriseServerList{},
		githubESDriver{base{"IDPGitHubEnterpriseServer", "GitHubEnterpriseServer", false, false}},
	); err != nil {
		return err
	}

	if err := common.SetupIDPController(mgr, o,
		v1alpha1.IDPGitLabGroupKind,
		v1alpha1.IDPGitLabGroupVersionKind,
		&v1alpha1.IDPGitLab{}, &v1alpha1.IDPGitLabList{},
		gitlabDriver{base{"IDPGitLab", "GitLab", false, false}},
	); err != nil {
		return err
	}

	if err := common.SetupIDPController(mgr, o,
		v1alpha1.IDPGitLabSelfHostedGroupKind,
		v1alpha1.IDPGitLabSelfHostedGroupVersionKind,
		&v1alpha1.IDPGitLabSelfHosted{}, &v1alpha1.IDPGitLabSelfHostedList{},
		gitlabSHDriver{base{"IDPGitLabSelfHosted", "GitLabSelfHosted", false, false}},
	); err != nil {
		return err
	}

	if err := common.SetupIDPController(mgr, o,
		v1alpha1.IDPGoogleGroupKind,
		v1alpha1.IDPGoogleGroupVersionKind,
		&v1alpha1.IDPGoogle{}, &v1alpha1.IDPGoogleList{},
		googleDriver{base{"IDPGoogle", "Google", false, false}},
	); err != nil {
		return err
	}

	if err := common.SetupIDPController(mgr, o,
		v1alpha1.IDPLDAPGroupKind,
		v1alpha1.IDPLDAPGroupVersionKind,
		&v1alpha1.IDPLDAP{}, &v1alpha1.IDPLDAPList{},
		ldapDriver{base{"IDPLDAP", "LDAP", false, false}},
	); err != nil {
		return err
	}

	if err := common.SetupIDPController(mgr, o,
		v1alpha1.IDPSAMLGroupKind,
		v1alpha1.IDPSAMLGroupVersionKind,
		&v1alpha1.IDPSAML{}, &v1alpha1.IDPSAMLList{},
		samlDriver{base{"IDPSAML", "SAML", false, false}},
	); err != nil {
		return err
	}

	if err := common.SetupIDPController(mgr, o,
		v1alpha1.OrgIDPOIDCGroupKind,
		v1alpha1.OrgIDPOIDCGroupVersionKind,
		&v1alpha1.OrgIDPOIDC{}, &v1alpha1.OrgIDPOIDCList{},
		orgOIDCDriver{base{"OrgIDPOIDC", "OIDC", true, true}},
	); err != nil {
		return err
	}

	if err := common.SetupIDPController(mgr, o,
		v1alpha1.OrgIDPOAuthGroupKind,
		v1alpha1.OrgIDPOAuthGroupVersionKind,
		&v1alpha1.OrgIDPOAuth{}, &v1alpha1.OrgIDPOAuthList{},
		orgOAuthDriver{base{"OrgIDPOAuth", "OAuth", true, false}},
	); err != nil {
		return err
	}

	if err := common.SetupIDPController(mgr, o,
		v1alpha1.OrgIDPJWTGroupKind,
		v1alpha1.OrgIDPJWTGroupVersionKind,
		&v1alpha1.OrgIDPJWT{}, &v1alpha1.OrgIDPJWTList{},
		jwtDriver{base{"OrgIDPJWT", "JWT", true, true}},
	); err != nil {
		return err
	}

	if err := common.SetupIDPController(mgr, o,
		v1alpha1.OrgIDPAppleGroupKind,
		v1alpha1.OrgIDPAppleGroupVersionKind,
		&v1alpha1.OrgIDPApple{}, &v1alpha1.OrgIDPAppleList{},
		orgAppleDriver{base{"OrgIDPApple", "Apple", true, false}},
	); err != nil {
		return err
	}

	if err := common.SetupIDPController(mgr, o,
		v1alpha1.OrgIDPAzureADGroupKind,
		v1alpha1.OrgIDPAzureADGroupVersionKind,
		&v1alpha1.OrgIDPAzureAD{}, &v1alpha1.OrgIDPAzureADList{},
		orgAzureDriver{base{"OrgIDPAzureAD", "AzureAD", true, false}},
	); err != nil {
		return err
	}

	if err := common.SetupIDPController(mgr, o,
		v1alpha1.OrgIDPGitHubGroupKind,
		v1alpha1.OrgIDPGitHubGroupVersionKind,
		&v1alpha1.OrgIDPGitHub{}, &v1alpha1.OrgIDPGitHubList{},
		orgGitHubDriver{base{"OrgIDPGitHub", "GitHub", true, false}},
	); err != nil {
		return err
	}

	if err := common.SetupIDPController(mgr, o,
		v1alpha1.OrgIDPGitHubEnterpriseServerGroupKind,
		v1alpha1.OrgIDPGitHubEnterpriseServerGroupVersionKind,
		&v1alpha1.OrgIDPGitHubEnterpriseServer{}, &v1alpha1.OrgIDPGitHubEnterpriseServerList{},
		orgGitHubESDriver{base{"OrgIDPGitHubEnterpriseServer", "GitHubEnterpriseServer", true, false}},
	); err != nil {
		return err
	}

	if err := common.SetupIDPController(mgr, o,
		v1alpha1.OrgIDPGitLabGroupKind,
		v1alpha1.OrgIDPGitLabGroupVersionKind,
		&v1alpha1.OrgIDPGitLab{}, &v1alpha1.OrgIDPGitLabList{},
		orgGitLabDriver{base{"OrgIDPGitLab", "GitLab", true, false}},
	); err != nil {
		return err
	}

	if err := common.SetupIDPController(mgr, o,
		v1alpha1.OrgIDPGitLabSelfHostedGroupKind,
		v1alpha1.OrgIDPGitLabSelfHostedGroupVersionKind,
		&v1alpha1.OrgIDPGitLabSelfHosted{}, &v1alpha1.OrgIDPGitLabSelfHostedList{},
		orgGitLabSHDriver{base{"OrgIDPGitLabSelfHosted", "GitLabSelfHosted", true, false}},
	); err != nil {
		return err
	}

	if err := common.SetupIDPController(mgr, o,
		v1alpha1.OrgIDPGoogleGroupKind,
		v1alpha1.OrgIDPGoogleGroupVersionKind,
		&v1alpha1.OrgIDPGoogle{}, &v1alpha1.OrgIDPGoogleList{},
		orgGoogleDriver{base{"OrgIDPGoogle", "Google", true, false}},
	); err != nil {
		return err
	}

	if err := common.SetupIDPController(mgr, o,
		v1alpha1.OrgIDPLDAPGroupKind,
		v1alpha1.OrgIDPLDAPGroupVersionKind,
		&v1alpha1.OrgIDPLDAP{}, &v1alpha1.OrgIDPLDAPList{},
		orgLDAPDriver{base{"OrgIDPLDAP", "LDAP", true, false}},
	); err != nil {
		return err
	}

	if err := common.SetupIDPController(mgr, o,
		v1alpha1.OrgIDPSAMLGroupKind,
		v1alpha1.OrgIDPSAMLGroupVersionKind,
		&v1alpha1.OrgIDPSAML{}, &v1alpha1.OrgIDPSAMLList{},
		orgSAMLDriver{base{"OrgIDPSAML", "SAML", true, false}},
	); err != nil {
		return err
	}

	return nil
}
