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

// Package cluster holds the cluster scoped forms of the instance wide managed
// resources.
//
// Each kind here manages state belonging to a whole Zitadel instance and is a
// singleton: one set of instance feature flags, one login policy default, one
// SMTP provider. Namespaced, two namespaces could each hold one and fight over
// the same settings, which the second writer wins. Cluster scoped, that conflict
// cannot be expressed.
//
// None of this is a second implementation. Every one of these kinds already has a
// controller that reconciles it, and that controller is reused unchanged: the
// cluster scoped resource is handed to it as the namespaced kind it mirrors, and
// the external name, finalizers and status it produces are copied back. The
// namespaced kinds are all kept, and all still work.
package cluster

import (
	"github.com/crossplane/crossplane-runtime/v2/pkg/controller"

	ctrl "sigs.k8s.io/controller-runtime"

	apisv1alpha1 "github.com/loafoe/provider-zitadel/apis/v1alpha1"
	zitadelv1alpha1 "github.com/loafoe/provider-zitadel/apis/zitadel/v1alpha1"
	"github.com/loafoe/provider-zitadel/internal/controller/common"
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
	"github.com/loafoe/provider-zitadel/internal/controller/instancesettings"
	"github.com/loafoe/provider-zitadel/internal/controller/messaging"
)

// Setup registers the cluster scoped controllers.
func Setup(mgr ctrl.Manager, o controller.Options) error {
	for _, s := range []func(ctrl.Manager, controller.Options) error{
		SetupClusterInstanceFeatures,
		SetupClusterSystemFeatures,
		SetupClusterInstanceRestrictions,
		SetupClusterInstanceSecretGenerator,
		SetupClusterActiveWebKey,
		SetupClusterEmailProviderSMTP,
		SetupClusterEmailProviderHTTP,
		SetupClusterSMSProviderTwilio,
		SetupClusterSMSProviderHTTP,
		SetupClusterDefaultLockoutPolicy,
		SetupClusterDefaultNotificationPolicy,
		SetupClusterDefaultPasswordAgePolicy,
		SetupClusterDefaultPasswordComplexityPolicy,
		SetupClusterDefaultPrivacyPolicy,
		SetupClusterDefaultDomainPolicy,
		SetupClusterDefaultLabelPolicy,
		SetupClusterDefaultLoginPolicy,
		SetupClusterDefaultOIDCSettings,
		SetupClusterDefaultSecuritySettings,
	} {
		if err := s(mgr, o); err != nil {
			return err
		}
	}

	return nil
}

// SetupClusterInstanceFeatures registers the ClusterInstanceFeatures controller.
//
// It is the same controller that reconciles InstanceFeatures, wrapped so that it is handed
// this kind instead. Nothing about how InstanceFeatures is reconciled is repeated here.
func SetupClusterInstanceFeatures(mgr ctrl.Manager, o controller.Options) error {
	return common.SetupManagedResourceController(
		mgr, o,
		apisv1alpha1.ClusterInstanceFeaturesGroupKind,
		apisv1alpha1.ClusterInstanceFeaturesGroupVersionKind,
		&apisv1alpha1.ClusterInstanceFeatures{},
		&apisv1alpha1.ClusterInstanceFeaturesList{},
		common.ReusingForCluster[*zitadelv1alpha1.InstanceFeatures](instancesettings.InstanceFeaturesClient()),
	)
}

// SetupClusterSystemFeatures registers the ClusterSystemFeatures controller.
//
// It is the same controller that reconciles SystemFeatures, wrapped so that it is handed
// this kind instead. Nothing about how SystemFeatures is reconciled is repeated here.
func SetupClusterSystemFeatures(mgr ctrl.Manager, o controller.Options) error {
	return common.SetupManagedResourceController(
		mgr, o,
		apisv1alpha1.ClusterSystemFeaturesGroupKind,
		apisv1alpha1.ClusterSystemFeaturesGroupVersionKind,
		&apisv1alpha1.ClusterSystemFeatures{},
		&apisv1alpha1.ClusterSystemFeaturesList{},
		common.ReusingForCluster[*zitadelv1alpha1.SystemFeatures](instancesettings.SystemFeaturesClient()),
	)
}

// SetupClusterInstanceRestrictions registers the ClusterInstanceRestrictions controller.
//
// It is the same controller that reconciles InstanceRestrictions, wrapped so that it is handed
// this kind instead. Nothing about how InstanceRestrictions is reconciled is repeated here.
func SetupClusterInstanceRestrictions(mgr ctrl.Manager, o controller.Options) error {
	return common.SetupManagedResourceController(
		mgr, o,
		apisv1alpha1.ClusterInstanceRestrictionsGroupKind,
		apisv1alpha1.ClusterInstanceRestrictionsGroupVersionKind,
		&apisv1alpha1.ClusterInstanceRestrictions{},
		&apisv1alpha1.ClusterInstanceRestrictionsList{},
		common.ReusingForCluster[*zitadelv1alpha1.InstanceRestrictions](instancesettings.InstanceRestrictionsClient()),
	)
}

// SetupClusterInstanceSecretGenerator registers the ClusterInstanceSecretGenerator controller.
//
// It is the same controller that reconciles InstanceSecretGenerator, wrapped so that it is handed
// this kind instead. Nothing about how InstanceSecretGenerator is reconciled is repeated here.
func SetupClusterInstanceSecretGenerator(mgr ctrl.Manager, o controller.Options) error {
	return common.SetupManagedResourceController(
		mgr, o,
		apisv1alpha1.ClusterInstanceSecretGeneratorGroupKind,
		apisv1alpha1.ClusterInstanceSecretGeneratorGroupVersionKind,
		&apisv1alpha1.ClusterInstanceSecretGenerator{},
		&apisv1alpha1.ClusterInstanceSecretGeneratorList{},
		common.ReusingForCluster[*zitadelv1alpha1.InstanceSecretGenerator](instancesettings.InstanceSecretGeneratorClient()),
	)
}

// SetupClusterActiveWebKey registers the ClusterActiveWebKey controller.
//
// It is the same controller that reconciles ActiveWebKey, wrapped so that it is handed
// this kind instead. Nothing about how ActiveWebKey is reconciled is repeated here.
func SetupClusterActiveWebKey(mgr ctrl.Manager, o controller.Options) error {
	return common.SetupManagedResourceController(
		mgr, o,
		apisv1alpha1.ClusterActiveWebKeyGroupKind,
		apisv1alpha1.ClusterActiveWebKeyGroupVersionKind,
		&apisv1alpha1.ClusterActiveWebKey{},
		&apisv1alpha1.ClusterActiveWebKeyList{},
		common.ReusingForCluster[*zitadelv1alpha1.ActiveWebKey](messaging.NewActiveWebKeyExternal),
	)
}

// SetupClusterEmailProviderSMTP registers the ClusterEmailProviderSMTP controller.
//
// It is the same controller that reconciles EmailProviderSMTP, wrapped so that it is handed
// this kind instead. Nothing about how EmailProviderSMTP is reconciled is repeated here.
func SetupClusterEmailProviderSMTP(mgr ctrl.Manager, o controller.Options) error {
	return common.SetupManagedResourceController(
		mgr, o,
		apisv1alpha1.ClusterEmailProviderSMTPGroupKind,
		apisv1alpha1.ClusterEmailProviderSMTPGroupVersionKind,
		&apisv1alpha1.ClusterEmailProviderSMTP{},
		&apisv1alpha1.ClusterEmailProviderSMTPList{},
		common.ReusingForCluster[*zitadelv1alpha1.EmailProviderSMTP](messaging.SMTPProviderExternal()),
	)
}

// SetupClusterEmailProviderHTTP registers the ClusterEmailProviderHTTP controller.
//
// It is the same controller that reconciles EmailProviderHTTP, wrapped so that it is handed
// this kind instead. Nothing about how EmailProviderHTTP is reconciled is repeated here.
func SetupClusterEmailProviderHTTP(mgr ctrl.Manager, o controller.Options) error {
	return common.SetupManagedResourceController(
		mgr, o,
		apisv1alpha1.ClusterEmailProviderHTTPGroupKind,
		apisv1alpha1.ClusterEmailProviderHTTPGroupVersionKind,
		&apisv1alpha1.ClusterEmailProviderHTTP{},
		&apisv1alpha1.ClusterEmailProviderHTTPList{},
		common.ReusingForCluster[*zitadelv1alpha1.EmailProviderHTTP](messaging.EmailHTTPProviderExternal()),
	)
}

// SetupClusterSMSProviderTwilio registers the ClusterSMSProviderTwilio controller.
//
// It is the same controller that reconciles SMSProviderTwilio, wrapped so that it is handed
// this kind instead. Nothing about how SMSProviderTwilio is reconciled is repeated here.
func SetupClusterSMSProviderTwilio(mgr ctrl.Manager, o controller.Options) error {
	return common.SetupManagedResourceController(
		mgr, o,
		apisv1alpha1.ClusterSMSProviderTwilioGroupKind,
		apisv1alpha1.ClusterSMSProviderTwilioGroupVersionKind,
		&apisv1alpha1.ClusterSMSProviderTwilio{},
		&apisv1alpha1.ClusterSMSProviderTwilioList{},
		common.ReusingForCluster[*zitadelv1alpha1.SMSProviderTwilio](messaging.TwilioProviderExternal()),
	)
}

// SetupClusterSMSProviderHTTP registers the ClusterSMSProviderHTTP controller.
//
// It is the same controller that reconciles SMSProviderHTTP, wrapped so that it is handed
// this kind instead. Nothing about how SMSProviderHTTP is reconciled is repeated here.
func SetupClusterSMSProviderHTTP(mgr ctrl.Manager, o controller.Options) error {
	return common.SetupManagedResourceController(
		mgr, o,
		apisv1alpha1.ClusterSMSProviderHTTPGroupKind,
		apisv1alpha1.ClusterSMSProviderHTTPGroupVersionKind,
		&apisv1alpha1.ClusterSMSProviderHTTP{},
		&apisv1alpha1.ClusterSMSProviderHTTPList{},
		common.ReusingForCluster[*zitadelv1alpha1.SMSProviderHTTP](messaging.SMSHTTPProviderExternal()),
	)
}

// SetupClusterDefaultLockoutPolicy registers the ClusterDefaultLockoutPolicy controller.
//
// It is the same controller that reconciles DefaultLockoutPolicy, wrapped so that it is handed
// this kind instead. Nothing about how DefaultLockoutPolicy is reconciled is repeated here.
func SetupClusterDefaultLockoutPolicy(mgr ctrl.Manager, o controller.Options) error {
	return common.SetupManagedResourceController(
		mgr, o,
		apisv1alpha1.ClusterDefaultLockoutPolicyGroupKind,
		apisv1alpha1.ClusterDefaultLockoutPolicyGroupVersionKind,
		&apisv1alpha1.ClusterDefaultLockoutPolicy{},
		&apisv1alpha1.ClusterDefaultLockoutPolicyList{},
		common.ReusingForCluster[*zitadelv1alpha1.DefaultLockoutPolicy](default_lockout_policy.DefaultLockoutPolicyClient()),
	)
}

// SetupClusterDefaultNotificationPolicy registers the ClusterDefaultNotificationPolicy controller.
//
// It is the same controller that reconciles DefaultNotificationPolicy, wrapped so that it is handed
// this kind instead. Nothing about how DefaultNotificationPolicy is reconciled is repeated here.
func SetupClusterDefaultNotificationPolicy(mgr ctrl.Manager, o controller.Options) error {
	return common.SetupManagedResourceController(
		mgr, o,
		apisv1alpha1.ClusterDefaultNotificationPolicyGroupKind,
		apisv1alpha1.ClusterDefaultNotificationPolicyGroupVersionKind,
		&apisv1alpha1.ClusterDefaultNotificationPolicy{},
		&apisv1alpha1.ClusterDefaultNotificationPolicyList{},
		common.ReusingForCluster[*zitadelv1alpha1.DefaultNotificationPolicy](default_notification_policy.DefaultNotificationPolicyClient()),
	)
}

// SetupClusterDefaultPasswordAgePolicy registers the ClusterDefaultPasswordAgePolicy controller.
//
// It is the same controller that reconciles DefaultPasswordAgePolicy, wrapped so that it is handed
// this kind instead. Nothing about how DefaultPasswordAgePolicy is reconciled is repeated here.
func SetupClusterDefaultPasswordAgePolicy(mgr ctrl.Manager, o controller.Options) error {
	return common.SetupManagedResourceController(
		mgr, o,
		apisv1alpha1.ClusterDefaultPasswordAgePolicyGroupKind,
		apisv1alpha1.ClusterDefaultPasswordAgePolicyGroupVersionKind,
		&apisv1alpha1.ClusterDefaultPasswordAgePolicy{},
		&apisv1alpha1.ClusterDefaultPasswordAgePolicyList{},
		common.ReusingForCluster[*zitadelv1alpha1.DefaultPasswordAgePolicy](default_password_age_policy.DefaultPasswordAgePolicyClient()),
	)
}

// SetupClusterDefaultPasswordComplexityPolicy registers the ClusterDefaultPasswordComplexityPolicy controller.
//
// It is the same controller that reconciles DefaultPasswordComplexityPolicy, wrapped so that it is handed
// this kind instead. Nothing about how DefaultPasswordComplexityPolicy is reconciled is repeated here.
func SetupClusterDefaultPasswordComplexityPolicy(mgr ctrl.Manager, o controller.Options) error {
	return common.SetupManagedResourceController(
		mgr, o,
		apisv1alpha1.ClusterDefaultPasswordComplexityPolicyGroupKind,
		apisv1alpha1.ClusterDefaultPasswordComplexityPolicyGroupVersionKind,
		&apisv1alpha1.ClusterDefaultPasswordComplexityPolicy{},
		&apisv1alpha1.ClusterDefaultPasswordComplexityPolicyList{},
		common.ReusingForCluster[*zitadelv1alpha1.DefaultPasswordComplexityPolicy](default_password_complexity_policy.DefaultPasswordComplexityPolicyClient()),
	)
}

// SetupClusterDefaultPrivacyPolicy registers the ClusterDefaultPrivacyPolicy controller.
//
// It is the same controller that reconciles DefaultPrivacyPolicy, wrapped so that it is handed
// this kind instead. Nothing about how DefaultPrivacyPolicy is reconciled is repeated here.
func SetupClusterDefaultPrivacyPolicy(mgr ctrl.Manager, o controller.Options) error {
	return common.SetupManagedResourceController(
		mgr, o,
		apisv1alpha1.ClusterDefaultPrivacyPolicyGroupKind,
		apisv1alpha1.ClusterDefaultPrivacyPolicyGroupVersionKind,
		&apisv1alpha1.ClusterDefaultPrivacyPolicy{},
		&apisv1alpha1.ClusterDefaultPrivacyPolicyList{},
		common.ReusingForCluster[*zitadelv1alpha1.DefaultPrivacyPolicy](default_privacy_policy.DefaultPrivacyPolicyClient()),
	)
}

// SetupClusterDefaultDomainPolicy registers the ClusterDefaultDomainPolicy controller.
//
// It is the same controller that reconciles DefaultDomainPolicy, wrapped so that it is handed
// this kind instead. Nothing about how DefaultDomainPolicy is reconciled is repeated here.
func SetupClusterDefaultDomainPolicy(mgr ctrl.Manager, o controller.Options) error {
	return common.SetupManagedResourceController(
		mgr, o,
		apisv1alpha1.ClusterDefaultDomainPolicyGroupKind,
		apisv1alpha1.ClusterDefaultDomainPolicyGroupVersionKind,
		&apisv1alpha1.ClusterDefaultDomainPolicy{},
		&apisv1alpha1.ClusterDefaultDomainPolicyList{},
		common.ReusingForCluster[*zitadelv1alpha1.DefaultDomainPolicy](default_domain_policy.DefaultDomainPolicyClient()),
	)
}

// SetupClusterDefaultLabelPolicy registers the ClusterDefaultLabelPolicy controller.
//
// It is the same controller that reconciles DefaultLabelPolicy, wrapped so that it is handed
// this kind instead. Nothing about how DefaultLabelPolicy is reconciled is repeated here.
func SetupClusterDefaultLabelPolicy(mgr ctrl.Manager, o controller.Options) error {
	return common.SetupManagedResourceController(
		mgr, o,
		apisv1alpha1.ClusterDefaultLabelPolicyGroupKind,
		apisv1alpha1.ClusterDefaultLabelPolicyGroupVersionKind,
		&apisv1alpha1.ClusterDefaultLabelPolicy{},
		&apisv1alpha1.ClusterDefaultLabelPolicyList{},
		common.ReusingForCluster[*zitadelv1alpha1.DefaultLabelPolicy](default_label_policy.DefaultLabelPolicyClient()),
	)
}

// SetupClusterDefaultLoginPolicy registers the ClusterDefaultLoginPolicy controller.
//
// It is the same controller that reconciles DefaultLoginPolicy, wrapped so that it is handed
// this kind instead. Nothing about how DefaultLoginPolicy is reconciled is repeated here.
func SetupClusterDefaultLoginPolicy(mgr ctrl.Manager, o controller.Options) error {
	return common.SetupManagedResourceController(
		mgr, o,
		apisv1alpha1.ClusterDefaultLoginPolicyGroupKind,
		apisv1alpha1.ClusterDefaultLoginPolicyGroupVersionKind,
		&apisv1alpha1.ClusterDefaultLoginPolicy{},
		&apisv1alpha1.ClusterDefaultLoginPolicyList{},
		common.ReusingForCluster[*zitadelv1alpha1.DefaultLoginPolicy](default_login_policy.DefaultLoginPolicyClient()),
	)
}

// SetupClusterDefaultOIDCSettings registers the ClusterDefaultOIDCSettings controller.
//
// It is the same controller that reconciles DefaultOIDCSettings, wrapped so that it is handed
// this kind instead. Nothing about how DefaultOIDCSettings is reconciled is repeated here.
func SetupClusterDefaultOIDCSettings(mgr ctrl.Manager, o controller.Options) error {
	return common.SetupManagedResourceController(
		mgr, o,
		apisv1alpha1.ClusterDefaultOIDCSettingsGroupKind,
		apisv1alpha1.ClusterDefaultOIDCSettingsGroupVersionKind,
		&apisv1alpha1.ClusterDefaultOIDCSettings{},
		&apisv1alpha1.ClusterDefaultOIDCSettingsList{},
		common.ReusingForCluster[*zitadelv1alpha1.DefaultOIDCSettings](default_oidc_settings.DefaultOIDCSettingsClient()),
	)
}

// SetupClusterDefaultSecuritySettings registers the ClusterDefaultSecuritySettings controller.
//
// It is the same controller that reconciles DefaultSecuritySettings, wrapped so that it is handed
// this kind instead. Nothing about how DefaultSecuritySettings is reconciled is repeated here.
func SetupClusterDefaultSecuritySettings(mgr ctrl.Manager, o controller.Options) error {
	return common.SetupManagedResourceController(
		mgr, o,
		apisv1alpha1.ClusterDefaultSecuritySettingsGroupKind,
		apisv1alpha1.ClusterDefaultSecuritySettingsGroupVersionKind,
		&apisv1alpha1.ClusterDefaultSecuritySettings{},
		&apisv1alpha1.ClusterDefaultSecuritySettingsList{},
		common.ReusingForCluster[*zitadelv1alpha1.DefaultSecuritySettings](default_security_settings.DefaultSecuritySettingsClient()),
	)
}
