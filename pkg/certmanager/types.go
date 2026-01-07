// Package certmanager provides domain logic for interacting with cert-manager resources.
package certmanager

import (
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// GVKs for cert-manager resources
var (
	CertificateGVK = schema.GroupVersionKind{
		Group:   "cert-manager.io",
		Version: "v1",
		Kind:    "Certificate",
	}
	CertificateRequestGVK = schema.GroupVersionKind{
		Group:   "cert-manager.io",
		Version: "v1",
		Kind:    "CertificateRequest",
	}
	IssuerGVK = schema.GroupVersionKind{
		Group:   "cert-manager.io",
		Version: "v1",
		Kind:    "Issuer",
	}
	ClusterIssuerGVK = schema.GroupVersionKind{
		Group:   "cert-manager.io",
		Version: "v1",
		Kind:    "ClusterIssuer",
	}
	OrderGVK = schema.GroupVersionKind{
		Group:   "acme.cert-manager.io",
		Version: "v1",
		Kind:    "Order",
	}
	ChallengeGVK = schema.GroupVersionKind{
		Group:   "acme.cert-manager.io",
		Version: "v1",
		Kind:    "Challenge",
	}
	SecretGVK = schema.GroupVersionKind{
		Group:   "",
		Version: "v1",
		Kind:    "Secret",
	}
	PodGVK = schema.GroupVersionKind{
		Group:   "",
		Version: "v1",
		Kind:    "Pod",
	}
)

// GVKs for cert-manager-operator resources
var (
	CertManagerOperatorGVK = schema.GroupVersionKind{
		Group:   "operator.openshift.io",
		Version: "v1alpha1",
		Kind:    "CertManager",
	}
)

// Standard labels used by cert-manager for resource relationships
const (
	// LabelCertificateName is the label cert-manager adds to CertificateRequests
	LabelCertificateName = "cert-manager.io/certificate-name"
	// LabelCertificateRevision is the label for the certificate revision
	LabelCertificateRevision = "cert-manager.io/certificate-revision"
	// LabelIssuerName is the label for the issuer name
	LabelIssuerName = "cert-manager.io/issuer-name"
	// LabelIssuerKind is the label for the issuer kind
	LabelIssuerKind = "cert-manager.io/issuer-kind"
	// LabelIssuerGroup is the label for the issuer group
	LabelIssuerGroup = "cert-manager.io/issuer-group"
)

// Cert-manager component names and namespace
const (
	CertManagerNamespace         = "cert-manager"
	CertManagerOperatorNamespace = "cert-manager-operator"
	ControllerDeploymentName     = "cert-manager"
	WebhookDeploymentName        = "cert-manager-webhook"
	CAInjectorDeploymentName     = "cert-manager-cainjector"
)

// Condition represents a Kubernetes-style condition
type Condition struct {
	Type               string
	Status             string
	Reason             string
	Message            string
	LastTransitionTime string
}

// IssuerRef represents a reference to an Issuer or ClusterIssuer
type IssuerRef struct {
	Name  string
	Kind  string
	Group string
}

// CertificateDetails contains a Certificate and all related resources
type CertificateDetails struct {
	Certificate         *unstructured.Unstructured
	CertificateRequests []*unstructured.Unstructured
	Issuer              *unstructured.Unstructured
	Orders              []*unstructured.Unstructured
	Challenges          []*unstructured.Unstructured
	Events              []Event
}

// IssuerDetails contains an Issuer and related information
type IssuerDetails struct {
	Issuer *unstructured.Unstructured
	Events []Event
}

// Event represents a Kubernetes Event
type Event struct {
	Type      string
	Reason    string
	Message   string
	Timestamp string
	Count     int32
}

// ComponentStatus represents the status of a cert-manager component
type ComponentStatus struct {
	Name              string
	Ready             bool
	AvailableReplicas int32
	DesiredReplicas   int32
	Message           string
}

// OperatorStatus represents the overall status of cert-manager operator
type OperatorStatus struct {
	Controller ComponentStatus
	Webhook    ComponentStatus
	CAInjector ComponentStatus
	Conditions []Condition
}
