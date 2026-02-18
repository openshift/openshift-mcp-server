package netedge

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/containers/kubernetes-mcp-server/pkg/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/dynamic/fake"
	"k8s.io/client-go/rest"
)

func TestInspectRoute(t *testing.T) {
	tests := []struct {
		name          string
		namespace     string
		route         string
		existingObjs  []runtime.Object
		expectedError string
		validate      func(t *testing.T, result string)
	}{
		{
			name:      "successful retrieval",
			namespace: "default",
			route:     "my-route",
			existingObjs: []runtime.Object{
				&unstructured.Unstructured{
					Object: map[string]interface{}{
						"apiVersion": "route.openshift.io/v1",
						"kind":       "Route",
						"metadata": map[string]interface{}{
							"name":      "my-route",
							"namespace": "default",
						},
						"spec": map[string]interface{}{
							"host": "example.com",
						},
					},
				},
			},
			validate: func(t *testing.T, result string) {
				var r map[string]interface{}
				err := json.Unmarshal([]byte(result), &r)
				require.NoError(t, err)
				assert.Equal(t, "my-route", r["metadata"].(map[string]interface{})["name"])
				assert.Equal(t, "example.com", r["spec"].(map[string]interface{})["host"])
			},
		},
		{
			name:          "route not found",
			namespace:     "default",
			route:         "missing",
			existingObjs:  []runtime.Object{},
			expectedError: "failed to get route",
		},
		{
			name:          "missing arguments",
			namespace:     "",
			route:         "",
			expectedError: "parameter required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create fake dynamic client
			scheme := runtime.NewScheme()
			dynClient := fake.NewSimpleDynamicClient(scheme, tt.existingObjs...)

			// Create mock params
			args := make(map[string]any)
			if tt.namespace != "" {
				args["namespace"] = tt.namespace
			}
			if tt.route != "" {
				args["route"] = tt.route
			}

			// Mock ToolHandlerParams
			mockReq := &mockToolCallRequest{args: args}

			// We need a non-nil RESTConfig to pass the check in the handler
			params := api.ToolHandlerParams{
				Context:         context.Background(),
				ToolCallRequest: mockReq,
				KubernetesClient: &mockKubernetesClient{
					restConfig:    &rest.Config{},
					dynamicClient: dynClient,
				},
			}

			result, err := inspectRoute(params)

			if tt.expectedError != "" {
				assert.NoError(t, err)
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
