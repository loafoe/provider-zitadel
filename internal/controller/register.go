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

// Package controller sets up all Zitadel controllers.
package controller

import (
	"github.com/crossplane/crossplane-runtime/v2/pkg/controller"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/loafoe/provider-zitadel/internal/controller/action"
	"github.com/loafoe/provider-zitadel/internal/controller/actionexecution"
	"github.com/loafoe/provider-zitadel/internal/controller/actiontarget"
	"github.com/loafoe/provider-zitadel/internal/controller/actiontargetpublickey"
	"github.com/loafoe/provider-zitadel/internal/controller/apiapplication"
	"github.com/loafoe/provider-zitadel/internal/controller/config"
	"github.com/loafoe/provider-zitadel/internal/controller/default_domain_policy"
	"github.com/loafoe/provider-zitadel/internal/controller/default_label_policy"
	"github.com/loafoe/provider-zitadel/internal/controller/default_lockout_policy"
	"github.com/loafoe/provider-zitadel/internal/controller/default_login_policy"
	"github.com/loafoe/provider-zitadel/internal/controller/default_notification_policy"
	"github.com/loafoe/provider-zitadel/internal/controller/default_oidc_settings"
	"github.com/loafoe/provider-zitadel/internal/controller/default_password_age_policy"
	"github.com/loafoe/provider-zitadel/internal/controller/default_password_complexity_policy"
	"github.com/loafoe/provider-zitadel/internal/controller/default_privacy_policy"
	"github.com/loafoe/provider-zitadel/internal/controller/default_security_settings"
	"github.com/loafoe/provider-zitadel/internal/controller/domain_policy"
	"github.com/loafoe/provider-zitadel/internal/controller/humanuser"
	"github.com/loafoe/provider-zitadel/internal/controller/idp"
	"github.com/loafoe/provider-zitadel/internal/controller/instancemember"
	"github.com/loafoe/provider-zitadel/internal/controller/instancesettings"
	"github.com/loafoe/provider-zitadel/internal/controller/label_policy"
	"github.com/loafoe/provider-zitadel/internal/controller/lockout_policy"
	"github.com/loafoe/provider-zitadel/internal/controller/loginpolicy"
	"github.com/loafoe/provider-zitadel/internal/controller/machinekey"
	"github.com/loafoe/provider-zitadel/internal/controller/notification_policy"
	"github.com/loafoe/provider-zitadel/internal/controller/oidcapplication"
	"github.com/loafoe/provider-zitadel/internal/controller/organization"
	"github.com/loafoe/provider-zitadel/internal/controller/organizationmetadata"
	"github.com/loafoe/provider-zitadel/internal/controller/orgmember"
	"github.com/loafoe/provider-zitadel/internal/controller/password_age_policy"
	"github.com/loafoe/provider-zitadel/internal/controller/password_complexity_policy"
	"github.com/loafoe/provider-zitadel/internal/controller/personalaccesstoken"
	"github.com/loafoe/provider-zitadel/internal/controller/privacy_policy"
	"github.com/loafoe/provider-zitadel/internal/controller/project"
	"github.com/loafoe/provider-zitadel/internal/controller/projectgrant"
	"github.com/loafoe/provider-zitadel/internal/controller/projectgrantmember"
	"github.com/loafoe/provider-zitadel/internal/controller/projectrole"
	"github.com/loafoe/provider-zitadel/internal/controller/serviceaccount"
	"github.com/loafoe/provider-zitadel/internal/controller/triggeractions"
	"github.com/loafoe/provider-zitadel/internal/controller/usergrant"
	"github.com/loafoe/provider-zitadel/internal/controller/usermetadata"
)

// Setup creates all Zitadel controllers and adds them to the supplied manager.
func Setup(mgr ctrl.Manager, o controller.Options) error {
	for _, setup := range []func(ctrl.Manager, controller.Options) error{
		config.Setup,

		// Tenancy: the organization itself and who administers it.
		organization.Setup,
		instancemember.Setup,
		orgmember.Setup,

		// Projects, their roles, and sharing a project with another
		// organization.
		project.Setup,
		projectrole.Setup,
		projectgrant.Setup,
		projectgrantmember.Setup,

		// Applications a workload can authenticate with.
		oidcapplication.Setup,
		apiapplication.Setup,

		// Users, service accounts and their credentials.
		humanuser.Setup,
		serviceaccount.Setup,
		machinekey.Setup,
		personalaccesstoken.Setup,
		usergrant.Setup,

		// Attributes and policy.
		usermetadata.Setup,
		organizationmetadata.Setup,

		// Organization policy. Every one of these is a singleton of one
		// organization; see internal/controller/common/policy.go for the
		// lifecycle they share.
		loginpolicy.Setup,
		lockout_policy.Setup,
		notification_policy.Setup,
		password_age_policy.Setup,
		password_complexity_policy.Setup,
		privacy_policy.Setup,
		domain_policy.Setup,
		label_policy.Setup,

		// Identity providers, at both levels. One package, because all twenty
		// three kinds are one resource with a different field set.
		idp.Setup,

		// Actions and the things they are called at.
		action.Setup,
		actiontarget.Setup,
		actiontargetpublickey.Setup,
		actionexecution.Setup,
		triggeractions.Setup,

		// The policies an organization inherits until it sets its own. These
		// cannot be reset, so deleting one restores what it overwrote.
		default_lockout_policy.Setup,
		default_notification_policy.Setup,
		default_password_age_policy.Setup,
		default_password_complexity_policy.Setup,
		default_privacy_policy.Setup,
		default_domain_policy.Setup,
		default_label_policy.Setup,
		default_login_policy.Setup,
		default_oidc_settings.Setup,
		default_security_settings.Setup,
		instancesettings.Setup,
	} {
		if err := setup(mgr, o); err != nil {
			return err
		}
	}

	return nil
}
