package netedge

import (
	"encoding/json"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/dynamic/fake"
)

func (s *NetEdgeTestSuite) TestInspectRoute() {

	tests := []struct {
		name          string
		namespace     string
		route         string
		existingObjs  []runtime.Object
		expectedError string
		validate      func(result string)
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
			validate: func(result string) {
				var r map[string]interface{}
				err := json.Unmarshal([]byte(result), &r)
				s.Require().NoError(err)
				s.Assert().Equal("my-route", r["metadata"].(map[string]interface{})["name"])
				s.Assert().Equal("example.com", r["spec"].(map[string]interface{})["host"])
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
		s.Run(tt.name, func() {
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

			// Set args using suite helper
			s.SetArgs(args)
			s.SetDynamicClient(dynClient)

			result, err := inspectRoute(s.params)

			if tt.expectedError != "" {
				s.Assert().NoError(err)
				s.Require().NotNil(result)
				s.Require().Error(result.Error)
				s.Assert().Contains(result.Error.Error(), tt.expectedError)
			} else {
				s.Assert().NoError(err)
				s.Require().NotNil(result)
				s.Assert().NoError(result.Error)
				if tt.validate != nil {
					tt.validate(result.Content)
				}
			}
		})
	}
}
