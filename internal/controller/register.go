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

	"github.com/loafoe/provider-zitadel/internal/controller/config"
	"github.com/loafoe/provider-zitadel/internal/controller/humanuser"
	"github.com/loafoe/provider-zitadel/internal/controller/oidcapplication"
	"github.com/loafoe/provider-zitadel/internal/controller/organization"
	"github.com/loafoe/provider-zitadel/internal/controller/personalaccesstoken"
	"github.com/loafoe/provider-zitadel/internal/controller/project"
	"github.com/loafoe/provider-zitadel/internal/controller/projectrole"
	"github.com/loafoe/provider-zitadel/internal/controller/serviceaccount"
)

// Setup creates all Zitadel controllers and adds them to the supplied manager.
func Setup(mgr ctrl.Manager, o controller.Options) error {
	for _, setup := range []func(ctrl.Manager, controller.Options) error{
		config.Setup,
		organization.Setup,
		project.Setup,
		projectrole.Setup,
		oidcapplication.Setup,
		humanuser.Setup,
		serviceaccount.Setup,
		personalaccesstoken.Setup,
	} {
		if err := setup(mgr, o); err != nil {
			return err
		}
	}

	return nil
}
