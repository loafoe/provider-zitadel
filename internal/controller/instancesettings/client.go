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

package instancesettings

import (
	v1alpha1 "github.com/loafoe/provider-zitadel/apis/zitadel/v1alpha1"
	"github.com/loafoe/provider-zitadel/internal/clients/zitadel"
	"github.com/loafoe/provider-zitadel/internal/controller/common"
)

// InstanceFeaturesClient returns the external client factory for this package's driver, which is
// how InstanceFeatures is reconciled.
//
// The cluster scoped ClusterInstanceFeatures is set up from this, so both are reconciled by
// one client rather than by two that could drift apart.
func InstanceFeaturesClient() common.NewExternalClientFn {
	return common.PolicyExternalFactory[zitadel.FeatureFlags, zitadel.FeatureFlags, *v1alpha1.InstanceFeatures](featuresDriver{instance: true})
}

// SystemFeaturesClient returns the external client factory for this package's driver, which is
// how SystemFeatures is reconciled.
//
// The cluster scoped ClusterSystemFeatures is set up from this, so both are reconciled by
// one client rather than by two that could drift apart.
func SystemFeaturesClient() common.NewExternalClientFn {
	return common.PolicyExternalFactory[zitadel.FeatureFlags, zitadel.FeatureFlags, *v1alpha1.SystemFeatures](featuresDriver{})
}

// InstanceRestrictionsClient returns the external client factory for this package's driver, which is
// how InstanceRestrictions is reconciled.
//
// The cluster scoped ClusterInstanceRestrictions is set up from this, so both are reconciled by
// one client rather than by two that could drift apart.
func InstanceRestrictionsClient() common.NewExternalClientFn {
	return common.PolicyExternalFactory[zitadel.Restrictions, zitadel.Restrictions, *v1alpha1.InstanceRestrictions](restrictionsDriver{})
}

// InstanceSecretGeneratorClient returns the external client factory for this package's driver, which is
// how InstanceSecretGenerator is reconciled.
//
// The cluster scoped ClusterInstanceSecretGenerator is set up from this, so both are reconciled by
// one client rather than by two that could drift apart.
func InstanceSecretGeneratorClient() common.NewExternalClientFn {
	return common.PolicyExternalFactory[zitadel.SecretGenerator, zitadel.SecretGenerator, *v1alpha1.InstanceSecretGenerator](secretGeneratorDriver{})
}
