/*
Copyright 2026 The Kubernetes Authors.

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

package smb

import (
	"k8s.io/component-base/metrics"
	"k8s.io/component-base/metrics/legacyregistry"
)

// Metrics of the renewal of the kerberos tickets of volumes mounted with a
// keytab, exported by the node plugin with --metrics-address. A renewal
// failure is otherwise only visible in the log, and only has an effect once
// the ticket has expired and the connection to the server is lost.
const (
	krb5MetricsSubsystem = "csi_smb_kerberos"

	krb5RenewalResultSuccess = "success"
	krb5RenewalResultFailure = "failure"
)

var (
	krb5TicketRenewals = metrics.NewCounterVec(&metrics.CounterOpts{
		Subsystem:      krb5MetricsSubsystem,
		Name:           "ticket_renewals_total",
		Help:           "Number of renewals of the kerberos ticket of a principal with volumes mounted with a keytab, by result.",
		StabilityLevel: metrics.ALPHA,
	}, []string{"result"})

	krb5KeytabPrincipals = metrics.NewGauge(&metrics.GaugeOpts{
		Subsystem:      krb5MetricsSubsystem,
		Name:           "keytab_principals",
		Help:           "Number of kerberos principals with volumes mounted with a keytab on the node, at the last renewal.",
		StabilityLevel: metrics.ALPHA,
	})

	krb5KeytabUpdates = metrics.NewCounterVec(&metrics.CounterOpts{
		Subsystem:      krb5MetricsSubsystem,
		Name:           "keytab_updates_total",
		Help:           "Number of attempts to replace the stored keytab of a principal with a new keytab from the secret of one of its volumes, by result.",
		StabilityLevel: metrics.ALPHA,
	}, []string{"result"})

	krb5FailingRenewals = metrics.NewGauge(&metrics.GaugeOpts{
		Subsystem:      krb5MetricsSubsystem,
		Name:           "ticket_renewal_failing_principals",
		Help:           "Number of kerberos principals with volumes mounted with a keytab whose ticket could not be renewed at the last renewal.",
		StabilityLevel: metrics.ALPHA,
	})
)

func init() {
	legacyregistry.MustRegister(krb5TicketRenewals, krb5KeytabUpdates, krb5KeytabPrincipals, krb5FailingRenewals)
}
