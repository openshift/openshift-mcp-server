package netedge

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/containers/kubernetes-mcp-server/pkg/api"
	"github.com/containers/kubernetes-mcp-server/pkg/kubernetes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	discoveryv1 "k8s.io/api/discovery/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestGetServiceEndpoints(t *testing.T) {
	scheme := kubernetes.Scheme
	// Ensure discoveryv1 is registered
	_ = discoveryv1.AddToScheme(scheme)

	tests := []struct {
		name          string
		namespace     string
		service       string
		existingObjs  []client.Object
		expectedError string
		validate      func(t *testing.T, result string)
	}{
		{
			name:      "successful retrieval",
			namespace: "default",
			service:   "myservice",
			existingObjs: []client.Object{
				&discoveryv1.EndpointSlice{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "myservice-1",
						Namespace: "default",
						Labels: map[string]string{
							"kubernetes.io/service-name": "myservice",
						},
					},
					Endpoints: []discoveryv1.Endpoint{
						{
							Addresses: []string{"1.2.3.4"},
						},
					},
					Ports: []discoveryv1.EndpointPort{
						{
							Port: ptr.To(int32(80)),
						},
					},
				},
			},
			validate: func(t *testing.T, result string) {
				var eps []discoveryv1.EndpointSlice
				err := json.Unmarshal([]byte(result), &eps)
				require.NoError(t, err)
				assert.Len(t, eps, 1)
				assert.Equal(t, "myservice-1", eps[0].Name)
				assert.Equal(t, "1.2.3.4", eps[0].Endpoints[0].Addresses[0])
			},
		},
		{
			name:          "endpoints not found",
			namespace:     "default",
			service:       "missing",
			existingObjs:  []client.Object{},
			expectedError: "no EndpointSlices found",
		},
		{
			name:          "missing arguments",
			namespace:     "",
			service:       "",
			expectedError: "parameter required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Override the client creation function
			oldNewClientFunc := newClientFunc
			newClientFunc = func(config *rest.Config, options client.Options) (client.Client, error) {
				return fake.NewClientBuilder().WithScheme(scheme).WithObjects(tt.existingObjs...).Build(), nil
			}
			defer func() { newClientFunc = oldNewClientFunc }()

			// Create mock params
			args := make(map[string]any)
			if tt.namespace != "" {
				args["namespace"] = tt.namespace
			}
			if tt.service != "" {
				args["service"] = tt.service
			}

			// Mock ToolHandlerParams
			mockReq := &mockToolCallRequest{args: args}

			// We need a non-nil RESTConfig to pass the check in the handler
			params := api.ToolHandlerParams{
				Context:          context.Background(),
				ToolCallRequest:  mockReq,
				KubernetesClient: &mockKubernetesClient{restConfig: &rest.Config{}},
			}

			result, err := getServiceEndpoints(params)

			// If the handler returns an error in ToolCallResult (which is mostly what it does for logic errors),
			// result.Error will be set. `err` return is usually nil unless panic/protocol error.

			// However, our handler returns `api.NewToolCallResult("", err)`.
			// So we check result.Error.

			if tt.expectedError != "" {
				assert.NoError(t, err) // The handler doesn't return Go error
				require.NotNil(t, result)
				require.Error(t, result.Error)
				assert.Contains(t, result.Error.Error(), tt.expectedError)
			} else {
				assert.NoError(t, err)
				require.NotNil(t, result)
				assert.NoError(t, result.Error)
				if tt.validate != nil {
					tt.validate(t, result.Content)
				}
			}
		})
	}
}

type mockToolCallRequest struct {
	args map[string]any
}

func (m *mockToolCallRequest) GetArguments() map[string]any {
	return m.args
}
