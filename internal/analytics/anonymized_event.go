package analytics

import (
	"k8s.io/utils/strings/slices"

	"github.com/kubeshop/botkube/pkg/event"
)

// allowedApiVersions contains a list of resource apiVersion that can be collected
// without a risk of leaking private data.
//
// This is a raw output of the `kubectl api-versions` command, run on:
// k3d cluster (Kubernetes 1.22) with the following components installed:
// - Istio 1.14.1
// - `kube-prometheus-stack` 0.57.0
// - Argo CD 2.4.6
var allowedAPIVersions = []string{
	"admissionregistration.k8s.io/v1",
	"apiextensions.k8s.io/v1",
	"apiregistration.k8s.io/v1",
	"apps/v1",
	"argoproj.io/v1alpha1",
	"authentication.k8s.io/v1",
	"authorization.k8s.io/v1",
	"autoscaling/v1",
	"autoscaling/v2beta1",
	"autoscaling/v2beta2",
	"batch/v1",
	"batch/v1beta1",
	"certificates.k8s.io/v1",
	"coordination.k8s.io/v1",
	"discovery.k8s.io/v1",
	"discovery.k8s.io/v1beta1",
	"events.k8s.io/v1",
	"events.k8s.io/v1beta1",
	"extensions.istio.io/v1alpha1",
	"flowcontrol.apiserver.k8s.io/v1beta1",
	"helm.cattle.io/v1",
	"install.istio.io/v1alpha1",
	"k3s.cattle.io/v1",
	"metrics.k8s.io/v1beta1",
	"monitoring.coreos.com/v1",
	"monitoring.coreos.com/v1alpha1",
	"networking.istio.io/v1alpha3",
	"networking.istio.io/v1beta1",
	"networking.k8s.io/v1",
	"node.k8s.io/v1",
	"node.k8s.io/v1beta1",
	"policy/v1",
	"policy/v1beta1",
	"rbac.authorization.k8s.io/v1",
	"scheduling.k8s.io/v1",
	"security.istio.io/v1beta1",
	"storage.k8s.io/v1",
	"storage.k8s.io/v1beta1",
	"telemetry.istio.io/v1alpha1",
	"traefik.containo.us/v1alpha1",
	"v1",
}

const (
	anonymizedResourceAPIVersion = "other"
	anonymizedResourceKind       = "other"
)

// AnonymizedEventDetailsFrom returns anonymized data about a given event.
func AnonymizedEventDetailsFrom(event event.Event) EventDetails {
	apiVersion := event.APIVersion
	kind := event.Kind

	if !slices.Contains(allowedAPIVersions, apiVersion) {
		// unknown resource, let's anonymize the data to prevent leaking potentially private data
		apiVersion = anonymizedResourceAPIVersion
		kind = anonymizedResourceKind
	}

	return EventDetails{
		Type:       event.Type,
		APIVersion: apiVersion,
		Kind:       kind,
	}
}
