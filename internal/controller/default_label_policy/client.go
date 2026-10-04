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

package default_label_policy

import (
	v1alpha1 "github.com/loafoe/provider-zitadel/apis/zitadel/v1alpha1"
	"github.com/loafoe/provider-zitadel/internal/clients/zitadel"
	"github.com/loafoe/provider-zitadel/internal/controller/common"
)

// DefaultLabelPolicyClient returns the external client factory for this package's driver, which is
// how DefaultLabelPolicy is reconciled.
//
// The cluster scoped ClusterDefaultLabelPolicy is set up from this, so both are reconciled by
// one client rather than by two that could drift apart.
func DefaultLabelPolicyClient() common.NewExternalClientFn {
	return common.PolicyExternalFactory[zitadel.DefaultLabelPolicyInput, zitadel.DefaultLabelPolicy, *v1alpha1.DefaultLabelPolicy](driver{})
}
