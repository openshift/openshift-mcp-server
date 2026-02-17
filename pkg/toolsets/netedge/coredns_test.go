package netedge

import (
	"context"
	"testing"

	"github.com/containers/kubernetes-mcp-server/pkg/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestGetCoreDNSConfig(t *testing.T) {
	tests := []struct {
		name           string
		configMap      *corev1.ConfigMap
		expectedOutput string
		expectError    bool
		errorContains  string
	}{
		{
			name: "success - corefile found",
			configMap: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "dns-default",
					Namespace: "openshift-dns",
				},
				Data: map[string]string{
					"Corefile": "example corefile content",
				},
			},
			expectedOutput: "example corefile content",
			expectError:    false,
		},
		{
			name:           "failure - configmap not found",
			configMap:      nil,
			expectedOutput: "",
			expectError:    true,
			errorContains:  "failed to get dns-default ConfigMap",
		},
		{
			name: "failure - Corefile key missing",
			configMap: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "dns-default",
					Namespace: "openshift-dns",
				},
				Data: map[string]string{
					"OtherData": "some data",
				},
			},
			expectedOutput: "",
			expectError:    true,
			errorContains:  "corefile not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mock client
			objs := []runtime.Object{}
			if tt.configMap != nil {
				objs = append(objs, tt.configMap)
			}

			// Override newClientFunc to return fake client
			originalNewClientFunc := newClientFunc
			defer func() { newClientFunc = originalNewClientFunc }()

			newClientFunc = func(config *rest.Config, options client.Options) (client.Client, error) {
				return fake.NewClientBuilder().WithRuntimeObjects(objs...).Build(), nil
			}

			// Call handler
			params := api.ToolHandlerParams{
				Context:          context.Background(),
				KubernetesClient: &mockKubernetesClient{restConfig: &rest.Config{}},
			}

			result, err := getCoreDNSConfig(params)

			if tt.expectError {
				require.NoError(t, err) // Handler returns error in result, not as return value
				require.NotNil(t, result)
				require.Error(t, result.Error)
				assert.Contains(t, result.Error.Error(), tt.errorContains)
			} else {
				require.NoError(t, err)
				require.NotNil(t, result)
				require.NoError(t, result.Error)
				assert.Equal(t, tt.expectedOutput, result.Content)
			}
		})
	}
}
