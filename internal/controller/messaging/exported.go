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

package messaging

import (
	"context"

	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/loafoe/provider-zitadel/internal/clients/zitadel"
	"github.com/loafoe/provider-zitadel/internal/controller/common"
)

// Exported names of this package's driver.
//
// The cluster scoped form of the kinds here is reconciled by the very same
// driver, so it needs a name it can refer to. These are aliases rather than
// renames: nothing about the driver changes, and everything that already uses it
// keeps working.

// SmtpDriver is the driver of this package, under a name another package can use.
type SmtpDriver = smtpDriver

// EmailHTTPDriver is the driver of this package, under a name another package can use.
type EmailHTTPDriver = emailHTTPDriver

// TwilioDriver is the driver of this package, under a name another package can use.
type TwilioDriver = twilioDriver

// SmsHTTPDriver is the driver of this package, under a name another package can use.
type SmsHTTPDriver = smsHTTPDriver

// NewActiveWebKeyExternal builds the external client for ActiveWebKey, under a
// name another package can use.
//
// The cluster scoped form of ActiveWebKey is reconciled by this same client.
func NewActiveWebKeyExternal(ctx context.Context, kube client.Client, _ resource.ModernManaged, zc *zitadel.Client) (managed.ExternalClient, error) {
	return newActiveWebKeyExternal(ctx, kube, nil, zc)
}

// The provider client factories, under names another package can use, so that
// the cluster scoped form of each is reconciled by the same client.

// SMTPProviderExternal builds the client for EmailProviderSMTP.
func SMTPProviderExternal() common.NewExternalClientFn { return newProviderExternal(smtpDriver{}) }

// EmailHTTPProviderExternal builds the client for EmailProviderHTTP.
func EmailHTTPProviderExternal() common.NewExternalClientFn {
	return newProviderExternal(emailHTTPDriver{})
}

// TwilioProviderExternal builds the client for SMSProviderTwilio.
func TwilioProviderExternal() common.NewExternalClientFn { return newProviderExternal(twilioDriver{}) }

// SMSHTTPProviderExternal builds the client for SMSProviderHTTP.
func SMSHTTPProviderExternal() common.NewExternalClientFn {
	return newProviderExternal(smsHTTPDriver{})
}
