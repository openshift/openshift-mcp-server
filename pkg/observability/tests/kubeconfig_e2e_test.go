//go:build observability_e2e

package observability

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

const (
	obsE2ESANamespace = "default"
	obsE2ESAName      = "obs-e2e"
	obsE2EBindingName = "obs-e2e-cluster-reader"
)

var (
	preparedKubeconfigOnce sync.Once
	preparedKubeconfigPath string
	preparedKubeconfigErr  error
)

// prepareObsE2EKubeconfig rewrites KUBECONFIG once per process:
//  1. Merges the OpenShift ingress CA so Route TLS verifies.
//  2. When the kubeconfig has only client certs, mints a bearer token
//     so OAuth proxies on Tempo/Loki gateways accept the request.
func prepareObsE2EKubeconfig(t *testing.T) {
	t.Helper()
	preparedKubeconfigOnce.Do(func() {
		preparedKubeconfigPath, preparedKubeconfigErr = buildObsE2EKubeconfig()
	})
	if preparedKubeconfigErr != nil {
		t.Fatalf("prepare observability e2e kubeconfig: %v", preparedKubeconfigErr)
	}
	if preparedKubeconfigPath == "" {
		return
	}
	if err := os.Setenv("KUBECONFIG", preparedKubeconfigPath); err != nil {
		t.Fatal(err)
	}
	t.Logf("using prepared observability e2e kubeconfig: %s", preparedKubeconfigPath)
}

func buildObsE2EKubeconfig() (string, error) {
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	cfg, err := loadingRules.Load()
	if err != nil {
		return "", fmt.Errorf("load kubeconfig: %w", err)
	}
	if cfg == nil || cfg.CurrentContext == "" {
		return "", fmt.Errorf("kubeconfig has no current context")
	}

	changed := false

	if caPEM, caErr := loadOpenShiftIngressCA(); caErr != nil {
		fmt.Fprintf(os.Stderr, "observability e2e: could not load OpenShift ingress CA: %v\n", caErr)
	} else if mergeIngressCA(cfg, caPEM) {
		changed = true
	}

	ctxEntry := cfg.Contexts[cfg.CurrentContext]
	if ctxEntry == nil {
		return "", fmt.Errorf("current context %q not found", cfg.CurrentContext)
	}
	auth := cfg.AuthInfos[ctxEntry.AuthInfo]
	if auth == nil {
		auth = clientcmdapi.NewAuthInfo()
		cfg.AuthInfos[ctxEntry.AuthInfo] = auth
	}
	// Mint only for client-cert kubeconfigs. Leave exec/AuthProvider alone —
	// those already supply bearer tokens (e.g. oc login, OIDC).
	hasBearer := strings.TrimSpace(auth.Token) != "" ||
		strings.TrimSpace(auth.TokenFile) != "" ||
		auth.Exec != nil || auth.AuthProvider != nil
	if !hasBearer {
		token, tokenErr := mintObsE2EToken()
		if tokenErr != nil {
			return "", fmt.Errorf("kubeconfig has no bearer token and minting obs-e2e SA token failed: %w", tokenErr)
		}
		auth.Token = token
		auth.ClientCertificate = ""
		auth.ClientCertificateData = nil
		auth.ClientKey = ""
		auth.ClientKeyData = nil
		changed = true
	}

	if !changed {
		return "", nil
	}

	dir, err := os.MkdirTemp("", "obs-e2e-kubeconfig-*")
	if err != nil {
		return "", err
	}
	out := filepath.Join(dir, "config")
	if err := clientcmd.WriteToFile(*cfg, out); err != nil {
		return "", fmt.Errorf("write prepared kubeconfig: %w", err)
	}
	return out, nil
}

func loadOpenShiftIngressCA() ([]byte, error) {
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	kubeConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, nil)
	restCfg, err := kubeConfig.ClientConfig()
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	client, err := dynamic.NewForConfig(restCfg)
	if err != nil {
		return nil, err
	}
	cm, err := client.Resource(schema.GroupVersionResource{
		Group: "", Version: "v1", Resource: "configmaps",
	}).Namespace("openshift-config-managed").Get(ctx, "default-ingress-cert", metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	raw, found, err := unstructured.NestedString(cm.Object, "data", "ca-bundle.crt")
	if err != nil {
		return nil, err
	}
	if !found || strings.TrimSpace(raw) == "" {
		return nil, fmt.Errorf("default-ingress-cert missing ca-bundle.crt")
	}
	return []byte(raw), nil
}

// mergeIngressCA appends the ingress CA PEM to the current context's cluster
// CA data so Route TLS verifies without insecure mode.
func mergeIngressCA(cfg *clientcmdapi.Config, caPEM []byte) bool {
	ctxEntry := cfg.Contexts[cfg.CurrentContext]
	if ctxEntry == nil {
		return false
	}
	cluster := cfg.Clusters[ctxEntry.Cluster]
	if cluster == nil {
		return false
	}
	existing := cluster.CertificateAuthorityData
	if len(existing) == 0 && cluster.CertificateAuthority != "" {
		data, err := os.ReadFile(cluster.CertificateAuthority)
		if err != nil {
			fmt.Fprintf(os.Stderr, "observability e2e: cannot read cluster CA file %s: %v\n", cluster.CertificateAuthority, err)
			return false
		}
		existing = data
	}
	var merged []byte
	merged = append(merged, existing...)
	if len(existing) > 0 && existing[len(existing)-1] != '\n' {
		merged = append(merged, '\n')
	}
	merged = append(merged, caPEM...)
	cluster.CertificateAuthorityData = merged
	cluster.CertificateAuthority = ""
	return true
}

func mintObsE2EToken() (string, error) {
	if err := ensureObsE2ESA(); err != nil {
		return "", err
	}
	cli, err := kubectlOrOc()
	if err != nil {
		return "", err
	}
	out, err := exec.Command(cli, "create", "token", obsE2ESAName,
		"-n", obsE2ESANamespace,
		"--duration=45m",
	).CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("%s create token: %w\n%s", cli, err, strings.TrimSpace(string(out)))
	}
	token := strings.TrimSpace(string(out))
	if token == "" {
		return "", fmt.Errorf("%s create token returned empty output", cli)
	}
	return token, nil
}

func ensureObsE2ESA() error {
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	kubeConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, nil)
	restCfg, err := kubeConfig.ClientConfig()
	if err != nil {
		return err
	}
	client, err := dynamic.NewForConfig(restCfg)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	saGVR := schema.GroupVersionResource{Group: "", Version: "v1", Resource: "serviceaccounts"}
	_, err = client.Resource(saGVR).Namespace(obsE2ESANamespace).Get(ctx, obsE2ESAName, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		sa := &unstructured.Unstructured{Object: map[string]any{
			"apiVersion": "v1",
			"kind":       "ServiceAccount",
			"metadata": map[string]any{
				"name":      obsE2ESAName,
				"namespace": obsE2ESANamespace,
			},
		}}
		if _, err := client.Resource(saGVR).Namespace(obsE2ESANamespace).Create(ctx, sa, metav1.CreateOptions{}); err != nil && !apierrors.IsAlreadyExists(err) {
			return fmt.Errorf("create ServiceAccount %s/%s: %w", obsE2ESANamespace, obsE2ESAName, err)
		}
	} else if err != nil {
		return fmt.Errorf("get ServiceAccount %s/%s: %w", obsE2ESANamespace, obsE2ESAName, err)
	}

	crbGVR := schema.GroupVersionResource{
		Group: "rbac.authorization.k8s.io", Version: "v1", Resource: "clusterrolebindings",
	}
	_, err = client.Resource(crbGVR).Get(ctx, obsE2EBindingName, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		crb := &unstructured.Unstructured{Object: map[string]any{
			"apiVersion": "rbac.authorization.k8s.io/v1",
			"kind":       "ClusterRoleBinding",
			"metadata": map[string]any{
				"name": obsE2EBindingName,
			},
			"roleRef": map[string]any{
				"apiGroup": "rbac.authorization.k8s.io",
				"kind":     "ClusterRole",
				"name":     "cluster-reader",
			},
			"subjects": []any{
				map[string]any{
					"kind":      "ServiceAccount",
					"name":      obsE2ESAName,
					"namespace": obsE2ESANamespace,
				},
			},
		}}
		if _, err := client.Resource(crbGVR).Create(ctx, crb, metav1.CreateOptions{}); err != nil && !apierrors.IsAlreadyExists(err) {
			return fmt.Errorf("create ClusterRoleBinding %s: %w", obsE2EBindingName, err)
		}
	} else if err != nil {
		return fmt.Errorf("get ClusterRoleBinding %s: %w", obsE2EBindingName, err)
	}
	return nil
}

func kubectlOrOc() (string, error) {
	if _, err := exec.LookPath("oc"); err == nil {
		return "oc", nil
	}
	if _, err := exec.LookPath("kubectl"); err == nil {
		return "kubectl", nil
	}
	return "", fmt.Errorf("neither oc nor kubectl found in PATH")
}
